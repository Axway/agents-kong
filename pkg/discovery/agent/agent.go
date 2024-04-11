package agent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"sync"

	klib "github.com/kong/go-kong/kong"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/cache"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/filter"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agents-kong/pkg/common"
	"github.com/Axway/agents-kong/pkg/discovery/config"
	"github.com/Axway/agents-kong/pkg/discovery/kong"
	"github.com/Axway/agents-kong/pkg/discovery/subscription"
)

var kongToCRDMapper = map[string]string{
	kong.BasicAuthPlugin: provisioning.BasicAuthCRD,
	kong.KeyAuthPlugin:   provisioning.APIKeyCRD,
	kong.OAuthPlugin:     provisioning.OAuthSecretCRD,
}

type kongClient interface {
	// Provisioning
	CreateConsumer(ctx context.Context, id, name string) (*klib.Consumer, error)
	AddConsumerACL(ctx context.Context, id string) error
	DeleteConsumer(ctx context.Context, id string) error
	// Credential
	DeleteOauth2(ctx context.Context, consumerID, clientID string) error
	DeleteHttpBasic(ctx context.Context, consumerID, username string) error
	DeleteAuthKey(ctx context.Context, consumerID, authKey string) error
	CreateHttpBasic(ctx context.Context, consumerID string, basicAuth *klib.BasicAuth) (*klib.BasicAuth, error)
	CreateOauth2(ctx context.Context, consumerID string, oauth2 *klib.Oauth2Credential) (*klib.Oauth2Credential, error)
	CreateAuthKey(ctx context.Context, consumerID string, keyAuth *klib.KeyAuth) (*klib.KeyAuth, error)
	// Access Request
	AddRouteACL(ctx context.Context, routeID, allowedID string) error
	RemoveRouteACL(ctx context.Context, routeID, revokedID string) error
	AddQuota(ctx context.Context, routeID, allowedID, quotaInterval string, quotaLimit int) error
	// Discovery
	ListServices(ctx context.Context) ([]*klib.Service, error)
	ListRoutesForService(ctx context.Context, serviceId string) ([]*klib.Route, error)
	GetSpecForService(ctx context.Context, service *klib.Service) ([]byte, error)
	GetKongPlugins() *kong.Plugins
}

type Agent struct {
	logger         log.FieldLogger
	centralCfg     corecfg.CentralConfig
	kongGatewayCfg *config.KongGatewayConfig
	kongClient     kongClient
	plugins        kong.Plugins
	cache          cache.Cache
	filter         filter.Filter
}

func NewAgent(agentConfig config.AgentConfig, agentOpts ...func(a *Agent)) (*Agent, error) {
	ka := &Agent{
		logger:         log.NewFieldLogger().WithComponent("agent").WithPackage("kongAgent"),
		centralCfg:     agentConfig.CentralCfg,
		kongGatewayCfg: agentConfig.KongGatewayCfg,
		cache:          cache.New(),
	}
	for _, o := range agentOpts {
		o(ka)
	}

	var err error
	if ka.kongClient == nil {
		ka.kongClient, err = kong.NewKongClient(ka.kongGatewayCfg)
	}
	if err != nil {
		return nil, err
	}

	pluginLister := ka.kongClient.GetKongPlugins()
	if pluginLister == nil {
		return nil, fmt.Errorf("could not get kong plugin lister")
	}
	plugins, err := ka.kongClient.GetKongPlugins().ListAll(context.Background())
	if err != nil {
		return nil, err
	}

	ka.filter, err = filter.NewFilter(agentConfig.KongGatewayCfg.Spec.Filter)
	if err != nil {
		return nil, err
	}

	if err = hasGlobalACLEnabledInPlugins(ka.logger, plugins, agentConfig.KongGatewayCfg.ACL.Disable); err != nil {
		ka.logger.WithError(err).Error("ACL Plugin configured as required, but none found in Kong plugins.")
		return nil, err
	}

	opts := []subscription.ProvisionerOption{}
	if agentConfig.KongGatewayCfg.ACL.Disable {
		opts = append(opts, subscription.WithACLDisable())
	}
	subscription.NewProvisioner(ka.kongClient, opts...)
	return ka, nil
}

func withKongClient(kongClient kongClient) func(a *Agent) {
	return func(a *Agent) {
		a.kongClient = kongClient
	}
}

func pluginIsGlobal(p *klib.Plugin) bool {
	if p.Service == nil && p.Route == nil {
		return true
	}
	return false
}

// Returns no error in case a global ACL plugin which is enabled is found
func hasGlobalACLEnabledInPlugins(logger log.FieldLogger, plugins []*klib.Plugin, aclDisable bool) error {
	if aclDisable {
		logger.Warn("ACL Plugin check disabled. Assuming global access is allowed for all services.")
		return nil
	}
	for _, plugin := range plugins {
		if *plugin.Name == "acl" && *plugin.Enabled && pluginIsGlobal(plugin) {
			return nil
		}
	}
	return fmt.Errorf("acl plugin is not enabled/installed, install and enable or change the config to disable this check")
}

func (gc *Agent) DiscoverAPIs() error {
	gc.logger.Info("execute discovery process")

	ctx := context.Background()
	var err error

	plugins := kong.Plugins{PluginLister: gc.kongClient.GetKongPlugins()}
	gc.plugins = plugins

	services, err := gc.kongClient.ListServices(ctx)
	if err != nil {
		gc.logger.WithError(err).Error("failed to get services")
		return err
	}

	gc.processKongServicesList(ctx, services)
	return nil
}

func (gc *Agent) processKongServicesList(ctx context.Context, services []*klib.Service) {
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
	for _, tag := range service.Tags {
		filters[*tag] = *tag
	}
	return filters
}

func (gc *Agent) processSingleKongService(ctx context.Context, service *klib.Service) error {
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

	// parse the spec file that was found and get the spec processor
	spec := apic.NewSpecResourceParser(kongServiceSpec, "")
	spec.Parse()

	wg := sync.WaitGroup{}
	wg.Add(len(routes))
	for _, r := range routes {
		func(route *klib.Route) {
			defer wg.Done()
			gc.specPreparation(ctx, route, service, spec.GetSpecProcessor())
		}(r)
	}
	wg.Wait()

	return nil
}

func (gc *Agent) specPreparation(ctx context.Context, route *klib.Route, service *klib.Service, spec apic.SpecProcessor) {
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

func (gc *Agent) processKongRoute(route *klib.Route) []apic.EndpointDefinition {
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

func (gc *Agent) processKongAPI(
	ctx context.Context,
	route *klib.Route,
	service *klib.Service,
	spec apic.SpecProcessor,
	endpoints []apic.EndpointDefinition,
	apiPlugins map[string]*klib.Plugin,
) (*apic.ServiceBody, error) {
	kongAPI := newKongAPI(route, service, spec, endpoints, apiPlugins)
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
	spec apic.SpecProcessor,
	endpoints []apic.EndpointDefinition,
	apiPlugins map[string]*klib.Plugin,
) KongAPI {
	resType := spec.GetResourceType()
	ka := &KongAPI{
		id:            *service.ID,
		name:          *service.Name,
		description:   spec.GetDescription(),
		version:       spec.GetVersion(),
		url:           *service.Host,
		resourceType:  resType,
		documentation: []byte(*service.Name),
		spec:          spec.GetSpecBytes(),
		endpoints:     endpoints,
		stageName:     *route.Name,
		stage:         *route.ID,
	}
	ka.processSpecSecurity(spec, apiPlugins)
	return *ka
}

func (ka *KongAPI) processSpecSecurity(spec apic.SpecProcessor, apiPlugins map[string]*klib.Plugin) {
	// strip any security from spec if it is an oas spec
	resType := spec.GetResourceType()
	if resType != apic.Oas2 && resType != apic.Oas3 {
		return
	}
	oasSpec := spec.(apic.OasSpecProcessor)
	oasSpec.StripSpecAuth()

	ka.ard = provisioning.APIKeyARD
	ka.crds = []string{}
	for k, plugin := range apiPlugins {
		if crd, ok := kongToCRDMapper[k]; ok {
			ka.crds = append(ka.crds, crd)
		}
		switch k {
		case kong.BasicAuthPlugin:
			oasSpec.AddSecuritySchemes(oasSpec.GetSecurityBuilder().HTTPBasic().Build())
		case kong.KeyAuthPlugin:
			ka.apiKeySecurity(oasSpec, plugin.Config)
		case kong.OAuthPlugin:
			ka.oAuthSecurity(oasSpec, plugin.Config)
		}
	}

	ka.spec = oasSpec.(apic.SpecProcessor).GetSpecBytes()
}

func (ka *KongAPI) apiKeySecurity(spec apic.OasSpecProcessor, config map[string]interface{}) {
	keyAuth, err := kong.NewKeyAuthPluginConfigFromMap(config)
	if err != nil {
		return
	}

	for _, key := range keyAuth.KeyNames {
		if keyAuth.KeyInQuery {
			spec.AddSecuritySchemes(spec.GetSecurityBuilder().APIKey().SetArgumentName(key).InQueryParam().Build())
		} else {
			// forcing header if not in query
			spec.AddSecuritySchemes(spec.GetSecurityBuilder().APIKey().SetArgumentName(key).InHeader().Build())
		}
	}
}

func (ka *KongAPI) oAuthSecurity(spec apic.OasSpecProcessor, config map[string]interface{}) {
	oAuth, err := kong.NewOAuthPluginConfigFromMap(config)
	if err != nil {
		return
	}

	builder := spec.GetSecurityBuilder().OAuth()

	s := url.URL{}
	for _, e := range ka.endpoints {
		if e.Protocol == httpsScheme {
			s = url.URL{
				Scheme: httpsScheme,
				Host:   fmt.Sprintf("%v:%v", e.Host, e.Port),
				Path:   e.BasePath,
			}
			break
		}
	}
	if s.Scheme == "" {
		return
	}
	tokenURL := fmt.Sprintf("%v/oauth2/token", s.String())
	authURL := fmt.Sprintf("%v/oauth2/authorize", s.String())
	scopes := map[string]string{}
	for _, n := range oAuth.Scopes {
		scopes[n] = n
	}

	if oAuth.EnableImplicitGrant {
		builder = builder.AddFlow(apic.NewOAuthFlowBuilder().SetScopes(scopes).SetAuthorizationURL(authURL).Implicit())
	}

	if oAuth.EnableAuthorizationCode {
		builder = builder.AddFlow(apic.NewOAuthFlowBuilder().SetScopes(scopes).SetAuthorizationURL(authURL).SetTokenURL(tokenURL).AuthorizationCode())
	}

	if oAuth.EnableClientCredentials {
		builder = builder.AddFlow(apic.NewOAuthFlowBuilder().SetScopes(scopes).SetTokenURL(tokenURL).ClientCredentials())
	}

	if oAuth.EnablePasswordGrant {
		builder = builder.AddFlow(apic.NewOAuthFlowBuilder().SetScopes(scopes).SetTokenURL(tokenURL).Password())
	}
	spec.AddSecuritySchemes(builder.Build())
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

	builder := apic.NewServiceBodyBuilder().
		SetAPIName(ka.name).
		SetAPISpec(ka.spec).
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
		SetStageDisplayName(ka.stageName).
		SetStageDescriptor("Route").
		SetState(apic.PublishedStatus).
		SetStatus(apic.PublishedStatus).
		SetTags(tags).
		SetTitle(ka.name).
		SetURL(ka.url).
		SetVersion(ka.version).
		SetServiceEndpoints(ka.endpoints).
		SetSourceDataplaneType(apic.Kong, false)

	if len(ka.crds) > 0 {
		return builder.SetAccessRequestDefinitionName(ka.ard, false).
			SetCredentialRequestDefinitions(ka.crds).Build()
	}
	return builder.SetAuthPolicy(apic.Passthrough).Build()
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
