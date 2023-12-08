package gateway

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"sync"

	klib "github.com/kong/go-kong/kong"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/filter"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agents-kong/pkg/common"
	"github.com/Axway/agents-kong/pkg/discovery/config"
	kutil "github.com/Axway/agents-kong/pkg/discovery/kong"
	"github.com/Axway/agents-kong/pkg/discovery/subscription"
)

var kongToCRDMapper = map[string]string{
	"basic-auth": provisioning.BasicAuthCRD,
	"key-auth":   provisioning.APIKeyCRD,
	"oauth2":     provisioning.OAuthSecretCRD,
}

func NewClient(agentConfig config.AgentConfig) (*Client, error) {
	kongGatewayConfig := agentConfig.KongGatewayCfg
	clientBase := &http.Client{}
	kongClient, err := kutil.NewKongClient(clientBase, kongGatewayConfig)
	if err != nil {
		return nil, err
	}
	daCache := cache.New()
	logger := log.NewFieldLogger().WithField("component", "agent")

	plugins, err := kongClient.Plugins.ListAll(context.Background())
	if err != nil {
		return nil, err
	}

	discoveryFilter, err := filter.NewFilter(agentConfig.KongGatewayCfg.Spec.Filter)
	if err != nil {
		return nil, err
	}

	if err = hasGlobalACLEnabledInPlugins(logger, plugins, agentConfig.KongGatewayCfg.ACL.Disabled); err != nil {
		logger.WithError(err).Error("ACL Plugin configured as required, but none found in Kong plugins.")
		return nil, err
	}

	provisionLogger := log.NewFieldLogger().WithComponent("provision").WithPackage("kong")
	opts := []subscription.ProvisionerOption{}
	if agentConfig.KongGatewayCfg.ACL.Disabled {
		opts = append(opts, subscription.WithACLDisabled())
	}
	subscription.NewProvisioner(kongClient, provisionLogger, opts...)

	return &Client{
		logger:         logger,
		centralCfg:     agentConfig.CentralCfg,
		kongGatewayCfg: kongGatewayConfig,
		kongClient:     kongClient,
		cache:          daCache,
		mode:           common.Marketplace,
		filter:         discoveryFilter,
		aclDisabled:    fmt.Sprint(agentConfig.KongGatewayCfg.ACL.Disabled),
	}, nil
}

func pluginIsGlobal(p *klib.Plugin) bool {
	if p.Service == nil && p.Route == nil {
		return true
	}
	return false
}

// Returns no error in case a global ACL plugin which is enabled is found
func hasGlobalACLEnabledInPlugins(logger log.FieldLogger, plugins []*klib.Plugin, aclDisabled bool) error {
	if aclDisabled {
		logger.Warn("ACL Plugin disabled. Assuming global access is allowed for all services.")
		return nil
	}
	for _, plugin := range plugins {
		if *plugin.Name == "acl" && *plugin.Enabled && pluginIsGlobal(plugin) {
			return nil
		}
	}
	return fmt.Errorf("failed to find acl plugin is enabled and installed on the Kong Gateway. " +
		"Enable in on the Gateway or change the config to disable this check.")
}

func (gc *Client) DiscoverAPIs() error {
	gc.logger.Info("execute discovery process")

	ctx := context.Background()
	var err error

	plugins := kutil.Plugins{PluginLister: gc.kongClient.GetKongPlugins()}
	gc.plugins = plugins

	services, err := gc.kongClient.ListServices(ctx)
	if err != nil {
		gc.logger.WithError(err).Error("failed to get services")
		return err
	}

	gc.processKongServicesList(ctx, services)
	return nil
}

func (gc *Client) processKongServicesList(ctx context.Context, services []*klib.Service) {
	wg := new(sync.WaitGroup)
	for _, service := range services {
		if !gc.filter.Evaluate(toTagsMap(service)) {
			gc.logger.WithField(common.AttrServiceName, *service.Name).Info("Service not passing tag filters. Skipping discovery for this service.")
			continue
		}
		wg.Add(1)
		go func(service *klib.Service, wg *sync.WaitGroup) {
			defer wg.Done()
			err := gc.processSingleKongService(ctx, service)
			if err != nil {
				log.Error(err)
			}
		}(service, wg)
	}
	wg.Wait()
}

func toTagsMap(service *klib.Service) map[string]string {
	// The SDK currently only supports map[string]string format.
	filters := make(map[string]string)
	for i, t := range service.Tags {
		filters[fmt.Sprintf("t%d", i)] = *t
	}
	return filters
}

func (gc *Client) processSingleKongService(ctx context.Context, service *klib.Service) error {
	log := gc.logger.WithField(common.AttrServiceName, *service.Name)
	log.Info("processing service")

	routes, err := gc.kongClient.ListRoutesForService(ctx, *service.ID)
	if err != nil {
		log.WithError(err).Errorf("failed to get routes for service")
		return err
	}
	kongServiceSpec, err := gc.kongClient.GetSpecForService(ctx, service)
	if err != nil {
		log.WithError(err).Errorf("failed to get spec for service")
		return err
	}

	// don't publish an empty spec
	if kongServiceSpec == nil {
		log.Warn("no spec found")
		return nil
	}
	oasSpec := &Openapi{
		spec: string(kongServiceSpec),
	}

	for _, route := range routes {
		gc.specPreparation(ctx, route, service, oasSpec)
	}
	return nil
}

func (gc *Client) specPreparation(ctx context.Context, route *klib.Route, service *klib.Service, spec *Openapi) {
	log := gc.logger.WithField(common.AttrRouteID, *route.ID).
		WithField(common.AttrServiceID, *service.ID)

	apiPlugins, err := gc.plugins.GetEffectivePlugins(*route.ID, *service.ID)
	if err != nil {
		log.Warn("could not list plugins")
		return
	}

	endpoints := gc.processKongRoute(route)
	if len(endpoints) == 0 {
		log.Info("not processing route as no enabled endpoints detected")
		return
	}
	serviceBody, err := gc.processKongAPI(ctx, route, service, spec, endpoints, apiPlugins)
	if err != nil {
		log.WithError(err).Error("failed to process kong API")
		return
	}
	if serviceBody == nil {
		log.Info("not processing since no changes were detected")
		return
	}
	log = log.WithField("apiName", serviceBody.APIName)
	err = agent.PublishAPI(*serviceBody)
	if err != nil {
		log.WithError(err).Error("failed to publish api")
		return
	}

	log.Info("Successfully published to central")
}

func (gc *Client) processKongRoute(route *klib.Route) []apic.EndpointDefinition {
	if route == nil {
		return []apic.EndpointDefinition{}
	}

	kRoute := KongRoute{
		Route:       route,
		defaultHost: gc.kongGatewayCfg.Proxy.Host,
		httpPort:    gc.kongGatewayCfg.Proxy.Ports.HTTP.Value,
		httpsPort:   gc.kongGatewayCfg.Proxy.Ports.HTTPS.Value,
		basePath:    gc.kongGatewayCfg.Proxy.BasePath,
	}

	return kRoute.GetEndpoints()
}

func (gc *Client) processKongAPI(
	ctx context.Context,
	route *klib.Route,
	service *klib.Service,
	oasSpec *Openapi,
	endpoints []apic.EndpointDefinition,
	apiPlugins map[string]*klib.Plugin,
) (*apic.ServiceBody, error) {
	kongAPI := newKongAPI(route, service, oasSpec, endpoints)
	isAlreadyPublished, checksum := isPublished(&kongAPI, gc.cache)
	// If true, then the api is published and there were no changes detected
	if isAlreadyPublished {
		gc.logger.Debug("api is already published")
		return nil, nil
	}
	err := gc.cache.Set(checksum, kongAPI)
	if err != nil {
		gc.logger.WithError(err).Error("failed to save api to cache")
	}

	kongAPI.ard = provisioning.APIKeyARD
	kongAPI.crds = []string{}
	for k := range apiPlugins {
		if crd, ok := kongToCRDMapper[k]; ok {
			kongAPI.crds = append(kongAPI.crds, crd)
		}
	}

	agentDetails := map[string]string{
		common.AttrServiceID: *service.ID,
		common.AttrRouteID:   *route.ID,
		common.AttrChecksum:  checksum,
	}
	kongAPI.agentDetails = agentDetails
	serviceBody, err := kongAPI.buildServiceBody()
	if err != nil {
		gc.logger.WithError(err).Error("failed to build service body")
		return nil, err
	}
	return &serviceBody, nil
}

func newKongAPI(
	route *klib.Route,
	service *klib.Service,
	oasSpec *Openapi,
	endpoints []apic.EndpointDefinition,
) KongAPI {
	return KongAPI{
		id:            *service.Name,
		name:          *service.Name,
		description:   oasSpec.Description(),
		version:       oasSpec.Version(),
		url:           *service.Host,
		resourceType:  oasSpec.ResourceType(),
		documentation: []byte(*service.Name),
		swaggerSpec:   []byte(oasSpec.spec),
		endpoints:     endpoints,
		stage:         *route.Name,
	}
}

func (ka *KongAPI) buildServiceBody() (apic.ServiceBody, error) {
	tags := map[string]interface{}{}
	if ka.tags != nil {
		for _, tag := range ka.tags {
			tags[tag] = true
		}
	}

	serviceAttributes := map[string]string{
		"GatewayType": "Kong API Gateway",
	}

	return apic.NewServiceBodyBuilder().
		SetAPIName(ka.name).
		SetAPISpec(ka.swaggerSpec).
		SetAPIUpdateSeverity(ka.apiUpdateSeverity).
		SetDescription(ka.description).
		SetDocumentation(ka.documentation).
		SetID(ka.id).
		SetImage(ka.image).
		SetImageContentType(ka.imageContentType).
		SetResourceType(ka.resourceType).
		SetServiceAgentDetails(util.MapStringStringToMapStringInterface(ka.agentDetails)).
		SetServiceAttribute(serviceAttributes).
		SetStage(ka.stage).
		SetState(apic.PublishedStatus).
		SetStatus(apic.PublishedStatus).
		SetTags(tags).
		SetTitle(ka.name).
		SetURL(ka.url).
		SetVersion(ka.version).
		SetServiceEndpoints(ka.endpoints).
		SetAccessRequestDefinitionName(ka.ard, false).
		SetCredentialRequestDefinitions(ka.crds).Build()
}

// makeChecksum generates a makeChecksum for the api for change detection
func makeChecksum(val interface{}) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%v", val)))
	return fmt.Sprintf("%x", sum)
}

// isPublished checks if an api is published with the latest changes. Returns true if it is, and false if it is not.
func isPublished(api *KongAPI, c cache.Cache) (bool, string) {
	// Change detection (asset + policies)
	checksum := makeChecksum(api)
	item, err := c.Get(checksum)
	if err != nil || item == nil {
		return false, checksum
	}
	return true, checksum
}
