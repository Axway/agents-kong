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

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cache"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	kutil "github.com/Axway/agents-kong/pkg/kong"
	klib "github.com/kong/go-kong/kong"

	_ "github.com/Axway/agents-kong/pkg/subscription/auth/apikey"    // needed for apikey subscription initialization
	_ "github.com/Axway/agents-kong/pkg/subscription/auth/basicauth" // needed for basicAuth subscription initialization
	_ "github.com/Axway/agents-kong/pkg/subscription/auth/oauth2"    // needed for oauth2 subscription initialization
)

func NewClient(agentConfig config.AgentConfig) (*Client, error) {
	kongGatewayConfig := agentConfig.KongGatewayCfg
	clientBase := &http.Client{}
	kongClient, err := kutil.NewKongClient(clientBase, kongGatewayConfig)
	if err != nil {
		return nil, err
	}
	apicClient := NewCentralClient(agent.GetCentralClient(), agentConfig.CentralCfg)
	daCache := cache.New()
	logger := logrus.WithFields(logrus.Fields{
		"component": "agent",
		"package":   "discovery",
	})

	plugins, err := kongClient.Plugins.ListAll(context.Background())
	if err != nil {
		return nil, err
	}
	for _, plugin := range plugins {
		if *plugin.Name == "acl" {
			if groups, ok := plugin.Config["allow"].([]interface{}); ok {
				allowedGroup := findACLGroup(groups)
				logrus.Infof("Allowed ACL group %s", allowedGroup)
				if allowedGroup == "" {
					return nil, fmt.Errorf("failed to find  acl with group value amplify.group under allow")
				}
			}
		}
	}
	subscription.NewProvisioner(kongClient.Client, logger)
	return &Client{
		ctx:            context.Background(),
		logger:         logger,
		centralCfg:     agentConfig.CentralCfg,
		kongGatewayCfg: kongGatewayConfig,
		kongClient:     kongClient,
		apicClient:     apicClient,
		cache:          daCache,
		mode:           common.Marketplace,
	}, nil
}

func findACLGroup(groups []interface{}) string {
	for _, group := range groups {
		fmt.Println(group)
		if groupStr, ok := group.(string); ok && groupStr == common.AclGroup {
			return groupStr
		}
	}
	return ""
}

func (gc *Client) DiscoverAPIs() error {
	gc.logger.Info("execute discovery process")

	plugins := kutil.Plugins{PluginLister: gc.kongClient.GetKongPlugins()}
	gc.plugins = plugins
	services, err := gc.kongClient.ListServices(gc.ctx)
	if err != nil {
		gc.logger.WithError(err).Error("failed to get services")
		return err
	}

	gc.processKongServicesList(services)
	return nil
}

func (gc *Client) processKongServicesList(services []*klib.Service) {
	wg := new(sync.WaitGroup)
	for _, service := range services {
		wg.Add(1)
		go func(service *klib.Service, wg *sync.WaitGroup) {
			defer wg.Done()
			err := gc.processSingleKongService(gc.ctx, service)
			if err != nil {
				log.Error(err)
			}
		}(service, wg)
	}
	wg.Wait()
}

func (gc *Client) processSingleKongService(ctx context.Context, service *klib.Service) error {
	gc.logger.Infof("processing service %s", *service.Name)

	proxyEndpoint := gc.kongGatewayCfg.ProxyEndpoint
	httpPort := gc.kongGatewayCfg.ProxyHttpPort
	httpsPort := gc.kongGatewayCfg.ProxyHttpsPort

	routes, err := gc.kongClient.ListRoutesForService(ctx, *service.ID)
	if err != nil {
		gc.logger.WithError(err).Errorf("failed to get routes for service %s", *service.Name)
		return err
	}
	if len(routes) == 0 {
		//	gc.deleteCentralService(*service.ID, *service.Name)
		return nil
	}

	route := routes[0]

	apiPlugins, err := gc.plugins.GetEffectivePlugins(*route.ID, *service.ID)
	if err != nil {
		gc.logger.WithError(err).Errorf("failed to get plugins for route %s", *route.ID)
		return err
	}

	kongServiceSpec, err := gc.kongClient.GetSpecForService(ctx, *service.Host)
	if err != nil {
		gc.logger.WithError(err).Errorf("failed to get spec for service %s", *service.Name)
		return err
	}

	oasSpec := Openapi{
		spec: string(kongServiceSpec),
	}
	endpoints := gc.processKongRoute(proxyEndpoint, oasSpec.BasePath(), route, httpPort, httpsPort)
	serviceBody, err := gc.processKongAPI(*route.ID, service, oasSpec, endpoints, apiPlugins)
	if err != nil {
		return err
	}
	if serviceBody == nil {
		gc.logger.Debugf("not processing '%s' since no changes were detected", *service.Name)
		return nil
	}
	err = agent.PublishAPI(*serviceBody)
	if err != nil {
		gc.logger.WithError(err).Error("failed to publish api")
		return err
	}

	gc.logger.Infof("Published API '%s' to central", serviceBody.APIName)
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
		gc.logger.Debug("api is already published")
		return nil, nil
	}
	err := gc.cache.Set(checksum, kongAPI)
	if err != nil {
		gc.logger.WithError(err).Error("failed to save api to cache")
	}

	ardName, crdName := getFirstAuthPluginArdAndCrd(apiPlugins)
	logrus.Infof("API %v Access Request Definition %s Credential Request Definition %s", kongAPI.name, ardName, crdName)
	kongAPI.accessRequestDefinition = ardName
	kongAPI.CRDs = []string{crdName}
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
		SetAccessRequestDefinitionName(ka.accessRequestDefinition, false).
		SetCredentialRequestDefinitions(ka.CRDs).Build()
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

func getFirstAuthPluginArdAndCrd(plugins map[string]*klib.Plugin) (string, string) {
	for key := range plugins {
		switch key {
		case "key-auth":
			return provisioning.APIKeyARD, provisioning.APIKeyCRD
		case "jwt":
			return "jwt", "jwt"
		case "basic-auth":
			return provisioning.BasicAuthARD, provisioning.BasicAuthCRD
		case "oauth2":
			return "oauth2", "oauth2"
		}

	}
	return "", ""
}
