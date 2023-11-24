package gateway

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"sync"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agents-kong/pkg/common"
	"github.com/Axway/agents-kong/pkg/subscription"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cache"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	kutil "github.com/Axway/agents-kong/pkg/kong"
	klib "github.com/kong/go-kong/kong"
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

	if err := hasACLEnabledInPlugins(plugins); err != nil {
		return nil, err
	}

	provisionLogger := log.NewFieldLogger().WithComponent("provision").WithPackage("kong")
	subscription.NewProvisioner(kongClient, provisionLogger)

	return &Client{
		logger:         logger,
		centralCfg:     agentConfig.CentralCfg,
		kongGatewayCfg: kongGatewayConfig,
		kongClient:     kongClient,
		cache:          daCache,
		mode:           common.Marketplace,
	}, nil
}

// Returns no error in case an ACL plugin which is enabled is found
func hasACLEnabledInPlugins(plugins []*klib.Plugin) error {
	for _, plugin := range plugins {
		if *plugin.Name == "acl" && *plugin.Enabled {
			return nil
		}
	}
	return fmt.Errorf("failed to find acl plugin is enabled and installed")
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

func (gc *Client) processSingleKongService(ctx context.Context, service *klib.Service) error {
	log := gc.logger.WithField("service-name", *service.Name)
	log.Infof("processing service")

	Host := gc.kongGatewayCfg.Proxy.Host
	httpPort := gc.kongGatewayCfg.Proxy.Ports.HTTP
	httpsPort := gc.kongGatewayCfg.Proxy.Ports.HTTPS

	routes, err := gc.kongClient.ListRoutesForService(ctx, *service.ID)
	if err != nil {
		log.WithError(err).Errorf("failed to get routes for service")
		return err
	}
	if len(routes) == 0 {
		//	gc.deleteCentralService(*service.ID, *service.Name)
		return nil
	}

	route := routes[0]
	log = log.WithField("route-id", *route.ID)

	apiPlugins, err := gc.plugins.GetEffectivePlugins(*route.ID, *service.ID)
	if err != nil {
		log.WithError(err).Errorf("failed to get plugins for route")
		return err
	}

	kongServiceSpec, err := gc.kongClient.GetSpecForService(ctx, service)
	if err != nil {
		log.WithError(err).Errorf("failed to get spec for service")
		return err
	}

	// don't publish an empty spec
	if kongServiceSpec == nil {
		log.Debug("no spec found")
		return nil
	}

	oasSpec := Openapi{
		spec: string(kongServiceSpec),
	}

	endpoints := gc.processKongRoute(Host, oasSpec.BasePath(), route, httpPort, httpsPort)
	serviceBody, err := gc.processKongAPI(ctx, *route.ID, service, oasSpec, endpoints, apiPlugins)
	if err != nil {
		return err
	}
	if serviceBody == nil {
		log.Debugf("not processing since no changes were detected")
		return nil
	}
	err = agent.PublishAPI(*serviceBody)
	if err != nil {
		log.WithError(err).Error("failed to publish api")
		return err
	}

	log.Infof("Published API '%s' to central", serviceBody.APIName)
	return nil
}

func (gc *Client) processKongRoute(defaultHost string, basePath string, route *klib.Route, httpPort, httpsPort int) []apic.EndpointDefinition {
	var endpoints []apic.EndpointDefinition
	if route == nil {
		return endpoints
	}

	hosts := route.Hosts
	hosts = append(hosts, &defaultHost)

	for _, host := range hosts {
		for _, path := range route.Paths {
			for _, protocol := range route.Protocols {
				port := httpPort
				if *protocol == "https" {
					port = httpsPort
				}

				routingBasePath := *path
				if *route.StripPath {
					routingBasePath = routingBasePath + basePath
				}
				endpoint := apic.EndpointDefinition{
					Host:     *host,
					Port:     int32(port),
					Protocol: *protocol,
					BasePath: routingBasePath,
				}
				endpoints = append(endpoints, endpoint)
			}
		}
	}

	return endpoints
}

func (gc *Client) processKongAPI(
	ctx context.Context,
	routeID string,
	service *klib.Service,
	oasSpec Openapi,
	endpoints []apic.EndpointDefinition,
	apiPlugins map[string]*klib.Plugin,
) (*apic.ServiceBody, error) {
	kongAPI := newKongAPI(routeID, service, oasSpec, endpoints)
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
		common.AttrServiceId: *service.ID,
		common.AttrRouteId:   routeID,
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
	routeID string,
	service *klib.Service,
	oasSpec Openapi,
	endpoints []apic.EndpointDefinition,
) KongAPI {
	return KongAPI{
		id:            routeID,
		name:          *service.Name,
		description:   oasSpec.Description(),
		version:       oasSpec.Version(),
		url:           *service.Host,
		resourceType:  oasSpec.ResourceType(),
		documentation: []byte(*service.Name),
		swaggerSpec:   []byte(oasSpec.spec),
		endpoints:     endpoints,
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

func isValidAuthTypeAndEnabled(p *klib.Plugin) bool {
	if !*p.Enabled {
		return false
	}
	for _, availableAuthName := range []string{"basic-auth", "oauth2", "key-auth"} {
		if *p.Name == availableAuthName {
			return true
		}
	}
	return false
}
