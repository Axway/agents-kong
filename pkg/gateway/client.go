package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/Axway/agents-kong/pkg/kong/specmanager"
	"github.com/Axway/agents-kong/pkg/kong/specmanager/devportal"
	"github.com/Axway/agents-kong/pkg/kong/specmanager/localdir"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	kutil "github.com/Axway/agents-kong/pkg/kong"
	"github.com/Axway/agents-kong/pkg/subscription"
	klib "github.com/kong/go-kong/kong"

	_ "github.com/Axway/agents-kong/pkg/subscription/apikey" // needed for apikey subscription initialization
)

const kongHash = "kong-hash"
const kongServiceID = "kong-service-id"

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

	sm, err := initSubscriptionManager(kongClient.Client)
	if err != nil {
		return nil, err
	}

	return &Client{
		centralCfg:          agentConfig.CentralCfg,
		kongGatewayCfg:      kongGatewayConfig,
		kongClient:          kongClient,
		apicClient:          apicClient,
		subscriptionManager: sm,
	}, nil
}

func (gc *Client) DiscoverAPIs() error {
	apiServices, err := gc.apicClient.fetchCentralAPIServices(nil)
	if err != nil {
		log.Infof("failed to get central api services: %s", err)
	}
	// TODO: initCache should only run once
	initCache(apiServices)

	services, err := gc.kongClient.ListServices(context.Background())
	if err != nil {
		log.Errorf("failed to get services: %s", err)
		return err
	}

	gc.removeDeletedServices(services)
	gc.processKongServicesList(services)
	return nil
}

func (gc *Client) removeDeletedServices(services []*klib.Service) {
	specCache := cache.GetCache()
	log.Info("checking for deleted kong services")
	// TODO: add go funcs
	for _, serviceID := range specCache.GetKeys() {
		if !doesServiceExists(serviceID, services) {
			item, err := specCache.Get(serviceID)
			if err != nil {
				log.Errorf("failed to get cached service: %s", serviceID)
			}
			cachedService := item.(CachedService)
			err = gc.apicClient.deleteCentralAPIService(cachedService)
			if err != nil {
				log.Errorf("failed to delete service '%s': %s", cachedService.kongServiceName, err)
				continue
			}
			err = specCache.Delete(serviceID)
			if err != nil {
				log.Errorf("failed to delete service '%s' from the cache: %s", cachedService.kongServiceName, err)
			}
		}
	}
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
		gc.deleteCentralService(*service.ID, *service.Name)
		return nil
	}

	route := routes[0]

	plugins := kutil.Plugins{PluginLister: gc.kongClient.GetKongPlugins()}
	ep, err := plugins.GetEffectivePlugins(*route.ID, *service.ID)
	if err != nil {
		return fmt.Errorf("failed to get plugins for route %s: %w", *route.ID, err)
	}

	subscriptionInfo := gc.subscriptionManager.GetSubscriptionInfo(ep)

	kongServiceSpec, err := specmanager.GetSpecification(ctx, service)
	if err != nil {
		return fmt.Errorf("failed to get spec for %s: %s", *service.Name, err)
	}

	oasSpec := Openapi{
		spec: kongServiceSpec.Contents,
	}

	endpoints := gc.processKongRoute(proxyEndpoint, oasSpec.BasePath(), route, httpPort, httpsPort)

	serviceBody, err := gc.processKongAPI(*route.ID, service, oasSpec, endpoints, subscriptionInfo)
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

func (gc *Client) deleteCentralService(serviceID string, serviceName string) {
	log.Debugf("kong service '%s' has no routes.", serviceName)
	item, _ := cache.GetCache().Get(serviceID)

	if svc, ok := item.(CachedService); ok {
		err := gc.apicClient.deleteCentralAPIService(svc)

		if err != nil {
			log.Errorf("failed to delete service '%' from central: %s", err)
		} else {
			log.Warnf("deleted Kong service '%s' from central", serviceName)
		}

		cache.GetCache().Delete(serviceID)
	}
}

func (gc *Client) processKongRoute(defaultHost string, basePath string, route *klib.Route, httpPort, httpsPort int) []InstanceEndpoint {
	var endpoints []InstanceEndpoint
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
				endpoint := InstanceEndpoint{
					Host:     *host,
					Port:     int32(port),
					Protocol: *protocol,
					Routing:  v1alpha1.ApiServiceInstanceSpecRouting{BasePath: routingBasePath},
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
	endpoints []InstanceEndpoint,
	subscriptionInfo subscription.Info,
) (*apic.ServiceBody, error) {
	name := gc.centralCfg.GetEnvironmentName() + "." + *service.Name
	kongAPI := newKongAPI(routeID, service, oasSpec, endpoints, name, subscriptionInfo)

	serviceBody, err := kongAPI.buildServiceBody()
	if err != nil {
		return nil, fmt.Errorf("failed to build service body: %v", serviceBody)
	}

	serviceBodyHash, _ := util.ComputeHash(serviceBody)
	hash := fmt.Sprintf("%v", serviceBodyHash)
	serviceBody.ServiceAttributes[kongHash] = hash
	serviceBody.ServiceAttributes[kongServiceID] = *service.ID

	isCached := setCachedService(*service.ID, *service.Name, hash, serviceBody.APIName)
	if isCached {
		return nil, nil
	}

	return &serviceBody, nil
}

func newKongAPI(
	routeID string,
	service *klib.Service,
	oasSpec Openapi,
	endpoints []InstanceEndpoint,
	nameToPush string,
	info subscription.Info,
) KongAPI {
	return KongAPI{
		id:               routeID,
		name:             *service.Name,
		description:      oasSpec.Description(),
		version:          oasSpec.Version(),
		url:              *service.Host,
		resourceType:     oasSpec.ResourceType(),
		documentation:    []byte("\"Sample documentation for API discovery agent\""),
		swaggerSpec:      []byte(oasSpec.spec),
		endpoints:        endpoints,
		nameToPush:       nameToPush,
		subscriptionInfo: info,
	}
}

func (ka *KongAPI) buildServiceBody() (apic.ServiceBody, error) {
	serviceAttribute := make(map[string]string)
	body := apic.NewServiceBodyBuilder().
		SetAPIName(ka.name).
		SetAPISpec(ka.swaggerSpec).
		SetDescription(ka.description).
		SetDocumentation(ka.documentation).
		SetID(ka.id).
		SetResourceType(ka.resourceType).
		SetServiceAttribute(serviceAttribute).
		SetTitle(ka.name).
		SetURL(ka.url).
		SetVersion(ka.version).
		SetServiceEndpoints(ka.endpoints).
		SetAuthPolicy(ka.subscriptionInfo.APICPolicyName).
		SetSubscriptionName(ka.subscriptionInfo.SchemaName)

	sb, err := body.Build()

	// TODO: add set method for NameToPush
	if err == nil {
		sb.NameToPush = ka.nameToPush
	}

	return sb, err
}

func doesServiceExists(serviceId string, services []*klib.Service) bool {
	for _, srv := range services {
		if serviceId == *srv.ID {
			return true
		}
	}
	log.Infof("Kong service '%s' no longer exists.", serviceId)
	return false
}

func initSubscriptionManager(kc *klib.Client) (*subscription.Manager, error) {
	sm := subscription.New(
		logrus.StandardLogger(),
		agent.GetCentralClient(),
		agent.GetCentralClient(),
		kc)

	// register schemas
	for _, schema := range sm.Schemas() {
		if err := agent.GetCentralClient().RegisterSubscriptionSchema(schema); err != nil {
			return nil, fmt.Errorf("failed to register subscription schema %s: %w", schema.GetSubscriptionName(), err)
		}
		log.Infof("Schema registered: %s", schema.GetSubscriptionName())
	}

	agent.GetCentralClient().GetSubscriptionManager().RegisterValidator(sm.ValidateSubscription)
	// register validator and handlers
	agent.GetCentralClient().GetSubscriptionManager().RegisterProcessor(apic.SubscriptionApproved, sm.ProcessSubscribe)
	agent.GetCentralClient().GetSubscriptionManager().RegisterProcessor(apic.SubscriptionUnsubscribeInitiated, sm.ProcessUnsubscribe)

	// start polling for subscriptions
	agent.GetCentralClient().GetSubscriptionManager().Start()

	return sm, nil
}
