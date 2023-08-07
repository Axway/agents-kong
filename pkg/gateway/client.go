package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agents-kong/pkg/common"
	"github.com/Axway/agents-kong/pkg/subscription"
	"github.com/sirupsen/logrus"
	"net/http"
	"sync"

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

	_ "github.com/Axway/agents-kong/pkg/subscription/auth" // needed for apikey subscription initialization
)

const (
	AclGroup    = "amplify.group"
	marketplace = "marketplace"
	// CorsField -
	CorsField = "cors"
	// RedirectURLsField -
	RedirectURLsField = "redirectURLs"
	OauthServerField  = "oauthServer"

	OAuth2AuthType = "oauth2"

	ApplicationTypeField = "applicationType"
	// ClientTypeField -
	ClientTypeField = "clientType"
	AudienceField   = "audience"
	OauthScopes     = "oauthScopes"
)

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
		centralCfg:     agentConfig.CentralCfg,
		kongGatewayCfg: kongGatewayConfig,
		kongClient:     kongClient,
		apicClient:     apicClient,
		cache:          daCache,
		mode:           marketplace,
	}, nil
}

func findACLGroup(groups []interface{}) string {
	for _, group := range groups {
		if groupStr, ok := group.(string); ok && groupStr == AclGroup {
			return groupStr
		}
	}
	return ""
}

func (gc *Client) DiscoverAPIs() error {
	plugins := kutil.Plugins{PluginLister: gc.kongClient.GetKongPlugins()}
	gc.plugins = plugins
	services, err := gc.kongClient.ListServices(context.Background())
	if err != nil {
		log.Errorf("failed to get services: %s", err)
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
	specType, _ := getSpecType(kongAPI.swaggerSpec)
	logrus.Infof("Specification Type %s", specType)
	ardName, crdName := getFirstAuthPluginArdAndCrd(apiPlugins)
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
		"GatewayType": "webMethods",
	}

	return apic.NewServiceBodyBuilder().
		SetAPIName(ka.name).
		SetAPISpec(ka.swaggerSpec).
		SetAPIUpdateSeverity(ka.apiUpdateSeverity).
		SetAuthPolicy(ka.subscriptionInfo.APICPolicyName).
		SetDescription(ka.description).
		SetDocumentation(ka.documentation).
		SetID(ka.id).
		SetImage(ka.image).
		SetImageContentType(ka.imageContentType).
		SetResourceType(ka.resourceType).
		SetServiceAgentDetails(util.MapStringStringToMapStringInterface(ka.agentDetails)).
		SetServiceAttribute(serviceAttributes).
		SetStage(ka.stage).
		SetState(ka.state).
		SetStatus(ka.status).
		SetTags(tags).
		SetTitle(ka.name).
		SetURL(ka.url).
		SetVersion(ka.version).
		SetSubscriptionName(ka.subscriptionInfo.SchemaName).SetServiceEndpoints(ka.endpoints).
		SetAccessRequestDefinitionName(ka.accessRequestDefinition, false).
		SetCredentialRequestDefinitions(ka.CRDs).Build()
}

func getSpecType(specContent []byte) (string, error) {

	if specContent != nil {
		jsonMap := make(map[string]interface{})
		err := json.Unmarshal(specContent, &jsonMap)
		if err != nil {
			logrus.Info("Not an swagger or openapi spec")
			return "", nil
		}
		if _, isSwagger := jsonMap["swagger"]; isSwagger {
			return apic.Oas2, nil
		} else if _, isOpenAPI := jsonMap["openapi"]; isOpenAPI {
			return apic.Oas3, nil
		}
	}
	return "", nil
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

func GetCorsSchemaPropertyBuilder() provisioning.PropertyBuilder {
	return provisioning.NewSchemaPropertyBuilder().
		SetName(CorsField).
		SetLabel("Javascript Origins").
		IsArray().
		AddItem(
			provisioning.NewSchemaPropertyBuilder().
				SetName("Origins").
				IsString())
}
