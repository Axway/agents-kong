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
	"github.com/sirupsen/logrus"

	"github.com/Axway/agents-kong/pkg/kong/specmanager"
	"github.com/Axway/agents-kong/pkg/kong/specmanager/devportal"
	"github.com/Axway/agents-kong/pkg/kong/specmanager/localdir"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cache"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	kutil "github.com/Axway/agents-kong/pkg/kong"
	klib "github.com/kong/go-kong/kong"

	"github.com/Axway/agents-kong/pkg/subscription/auth/apikey"    // needed for apikey subscription initialization
	"github.com/Axway/agents-kong/pkg/subscription/auth/basicauth" // needed for basicAuth subscription initialization
	"github.com/Axway/agents-kong/pkg/subscription/auth/oauth2"    // needed for oauth2 subscription initialization
)

var authTypes = []string{apikey.Name, basicauth.Name, oauth2.Name}

var isValidAuthTypeAndEnabled = func(p *klib.Plugin) bool {
	if *p.Enabled != true {
		return false
	}
	for _, availableAuthName := range authTypes {
		if *p.Name == availableAuthName {
			return true
		}
	}
	return false
}

func NewClient(agentConfig config.AgentConfig) (*Client, error) {
	kongGatewayConfig := agentConfig.KongGatewayCfg
	clientBase := &http.Client{}
	kongClient, err := kutil.NewKongClient(clientBase, kongGatewayConfig)
	if err != nil {
		return nil, err
	}
	apicClient := NewCentralClient(agent.GetCentralClient(), agentConfig.CentralCfg)
	if agentConfig.KongGatewayCfg.SpecDevPortalEnabled {
		specmanager.AddSource(devportal.NewSpecificationSource(kongClient))
	}
	if len(agentConfig.KongGatewayCfg.SpecHomePath) > 0 {
		specmanager.AddSource(localdir.NewSpecificationSource(agentConfig.KongGatewayCfg.SpecHomePath))
	}
	daCache := cache.New()
	logger := logrus.WithFields(logrus.Fields{
		"component": "agent",
	})

	plugins, err := kongClient.Plugins.ListAll(context.Background())
	if err != nil {
		return nil, err
	}
	daCache.Set("kong-plugins", plugins)

	if err := hasACLEnabledInPlugins(plugins); err != nil {
		return nil, err
	}

	subscription.NewProvisioner(kongClient.Client, logger)
	return &Client{
		centralCfg:     agentConfig.CentralCfg,
		kongGatewayCfg: kongGatewayConfig,
		kongClient:     kongClient,
		apicClient:     apicClient,
		cache:          daCache,
		mode:           common.Marketplace,
	}, nil
}

func (gc *Client) ExecuteDiscovery() error {
	gc.getPlugins()
	gc.createRequestDefinitions()
	return gc.discoverAPIs()
}

func (gc *Client) discoverAPIs() error {
	services, err := gc.kongClient.ListServices(context.Background())
	if err != nil {
		log.Errorf("failed to get services: %s", err)
		return err
	}
	gc.processKongServicesList(services)
	return nil
}

func (gc *Client) createRequestDefinitions() error {
	log.Debug("creating request definitions")
	gc.createAccessRequestDefinition()
	return gc.createCredentialRequestDefinition()
}

func (gc *Client) createAccessRequestDefinition() {
	gc.ard = provisioning.APIKeyARD
}

func (gc *Client) createCredentialRequestDefinition() error {
	allPlugins, err := gc.cache.Get("kong-plugin")
	if err != nil {
		allPlugins, err = gc.plugins.PluginLister.ListAll(context.Background())
		if err != nil {
			return fmt.Errorf("failed to fetch kong plugins")
		}
	}

	uniqueCrds := map[string]string{}
	for _, plugin := range allPlugins.([]*klib.Plugin) {
		if isValidAuthTypeAndEnabled(plugin) {
			uniqueCrds[*plugin.Name] = *plugin.Name
		}
	}

	for _, crd := range uniqueCrds {
		gc.crds = append(gc.crds, crd)
	}
	return nil
}

func (gc *Client) processKongServicesList(services []*klib.Service) {
	wg := new(sync.WaitGroup)
	for _, service := range services {
		wg.Add(1)
		go func(service *klib.Service, wg *sync.WaitGroup) {
			defer wg.Done()
			err := gc.processSingleKongService(context.Background(), service)
			if err != nil {
				log.Error(err)
			}
		}(service, wg)
	}
	wg.Wait()
}

func (gc *Client) processSingleKongService(ctx context.Context, service *klib.Service) error {
	proxyEndpoint := gc.kongGatewayCfg.ProxyEndpoint
	httpPort := gc.kongGatewayCfg.ProxyHttpPort
	httpsPort := gc.kongGatewayCfg.ProxyHttpsPort

	routes, err := gc.kongClient.ListRoutesForService(ctx, *service.ID)
	if err != nil {
		return err
	}
	if len(routes) == 0 {
		//	gc.deleteCentralService(*service.ID, *service.Name)
		return nil
	}

	route := routes[0]

	apiPlugins, err := gc.plugins.GetEffectivePlugins(*route.ID, *service.ID)
	if err != nil {
		return fmt.Errorf("failed to get plugins for route %s: %w", *route.ID, err)
	}

	kongServiceSpec, err := specmanager.GetSpecification(ctx, service)
	if err != nil {
		return fmt.Errorf("failed to get spec for %s: %s", *service.Name, err)
	}

	oasSpec := Openapi{
		spec: kongServiceSpec.Contents,
	}
	endpoints := gc.processKongRoute(proxyEndpoint, oasSpec.BasePath(), route, httpPort, httpsPort)
	serviceBody, err := gc.processKongAPI(*route.ID, service, oasSpec, endpoints, apiPlugins)

	if err != nil {
		return err
	}
	if serviceBody == nil {
		log.Debugf("not processing '%s' since no changes were detected", *service.Name)
		return nil
	}
	err = agent.PublishAPI(*serviceBody)
	if err != nil {
		return fmt.Errorf("failed to publish api: %s", err)
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
				if *route.StripPath == true {
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
		logrus.Debug("api is already published")
		return nil, nil
	}
	err := gc.cache.Set(checksum, kongAPI)
	if err != nil {
		logrus.Errorf("failed to save api to cache: %s", err)
	}
	kongAPI.ard = gc.ard
	kongAPI.crds = gc.crds
	agentDetails := map[string]string{
		common.AttrServiceId: *service.ID,
		common.AttrRouteId:   routeID,
		common.AttrChecksum:  checksum,
	}
	kongAPI.agentDetails = agentDetails
	serviceBody, err := kongAPI.buildServiceBody()
	if err != nil {
		return nil, fmt.Errorf("failed to build service body: %v", serviceBody)
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

//func getSpecType(specContent []byte) (string, error) {
//
//	if specContent != nil {
//		jsonMap := make(map[string]interface{})
//		err := json.Unmarshal(specContent, &jsonMap)
//		if err != nil {
//			logrus.Info("Not an swagger or openapi spec")
//			return "", nil
//		}
//		if _, isSwagger := jsonMap["swagger"]; isSwagger {
//			return apic.Oas2, nil
//		} else if _, isOpenAPI := jsonMap["openapi"]; isOpenAPI {
//			return apic.Oas3, nil
//		}
//	}
//	return "", nil
//}

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

// Returns no error in case an ACL plugin which is enabled is found
func hasACLEnabledInPlugins(plugins []*klib.Plugin) error {
	for _, plugin := range plugins {
		if *plugin.Name == "acl" && *plugin.Enabled == true {
			return nil
		}
	}
	return fmt.Errorf("failed to find acl plugin is enabled and installed")
}

func (gc *Client) getPlugins() {
	gc.plugins = kutil.Plugins{PluginLister: gc.kongClient.GetKongPlugins()}
}
