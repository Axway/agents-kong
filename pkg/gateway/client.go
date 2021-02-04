package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	"github.com/Axway/agents-kong/pkg/subscription"
	"github.com/kong/go-kong/kong"
)

const kongHash = "kong-hash"
const externalAPIID = "externalAPIID"

func NewClient(agentConfig config.AgentConfig) (*Client, error) {
	kongGatewayConfig := agentConfig.KongGatewayCfg
	clientBase := &http.Client{}
	kongClient, err := NewKongClient(clientBase, kongGatewayConfig)
	if err != nil {
		return nil, err
	}

	apicClient := NewCentralClient(agent.GetCentralClient(), agentConfig.CentralCfg)

	return &Client{
		centralCfg:          agentConfig.CentralCfg,
		kongGatewayCfg:      kongGatewayConfig,
		kongClient:          kongClient,
		apicClient:          apicClient,
		subscriptionManager: enableSubscription(),
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

func (gc *Client) removeDeletedServices(services []*kong.Service) {
	specCache := cache.GetCache()
	log.Info("checking for deleted kong services")
	// TODO: add go funcs
	for _, serviceId := range specCache.GetKeys() {
		if !doesServiceExists(serviceId, services) {
			item, err := specCache.Get(serviceId)
			if err != nil {
				log.Errorf("failed to get cached service: %s", serviceId)
			}
			cachedService := item.(CachedService)
			err = gc.apicClient.deleteCentralAPIService(cachedService)
			if err != nil {
				log.Errorf("failed to delete service '%s': %s", cachedService.kongServiceName, err)
				continue
			}
			err = specCache.Delete(serviceId)
			if err != nil {
				log.Errorf("failed to delete service '%s' from the cache: %s", cachedService.kongServiceName, err)
			}
		}
	}
}

func (gc *Client) processKongServicesList(services []*kong.Service) {
	wg := new(sync.WaitGroup)

	for _, service := range services {
		wg.Add(1)

		go func(service *kong.Service, wg *sync.WaitGroup) {
			defer wg.Done()

			err := gc.processSingleKongService(context.Background(), service)
			if err != nil {
				log.Error(err)
			}
		}(service, wg)
	}

	wg.Wait()
}

func (gc *Client) processSingleKongService(ctx context.Context, service *kong.Service) error {
	proxyEndpoint := gc.kongGatewayCfg.ProxyEndpoint
	httpPort := gc.kongGatewayCfg.ProxyHttpPort
	httpsPort := gc.kongGatewayCfg.ProxyHttpsPort

	routes, err := gc.kongClient.ListRoutesForService(ctx, *service.ID)
	if err != nil {
		return err
	}

	// delete services that have no routes
	if len(routes) == 0 {
		log.Warnf("kong service '%s' has no routes. Attempting to delete the service from central", *service.Name)
		item, _ := cache.GetCache().Get(*service.ID)

		if svc, ok := item.(CachedService); ok {
			err := gc.apicClient.deleteCentralAPIService(svc)

			if err != nil {
				log.Errorf("failed to delete service '%' from central: %s", err)
			} else {
				log.Warnf("deleted Kong service '%s' from central", *service.Name)
			}

			cache.GetCache().Delete(*service.ID)
		}
		return nil
	}

	route := routes[0]
	subscriptionHandler, err := gc.getSubscriptionHandler(route)

	endpoints := gc.processKongRoute(proxyEndpoint, route, httpPort, httpsPort)

	kongServiceSpec, err := gc.kongClient.GetSpecForService(ctx, *service.ID)
	if err != nil {
		// TODO: If no spec is found, then it was likely deleted, and should be deleted from central
		return fmt.Errorf("failed to get spec for %s: %s", *service.Name, err)
	}

	oasSpec := Openapi{
		spec: kongServiceSpec.Contents,
	}

	serviceBody, err := gc.processKongAPI(service, oasSpec, endpoints, subscriptionHandler)
	if err != nil {
		return err
	}
	if serviceBody == nil {
		return fmt.Errorf("not processing '%s' since no changes were detected", *service.Name)
	}

	err = agent.PublishAPI(*serviceBody)
	if err != nil {
		return fmt.Errorf("failed to publish api: %s", err)
	}

	log.Info("Published API " + serviceBody.APIName + " to AMPLIFY Central")

	return nil
}

func (gc *Client) getSubscriptionHandler(route *kong.Route) (subscription.SubscriptionHandler, error) {
	plugins := gc.kongClient.GetKongPlugins()
	return gc.subscriptionManager.GetEffectiveSubscriptionHandler(route.ID, route.Service.ID, plugins)
}

func (gc *Client) processKongRoute(defaultHost string, route *kong.Route, httpPort, httpsPort int) []InstanceEndpoint {
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
				endpoint := InstanceEndpoint{
					Host:     *host,
					Port:     int32(port),
					Protocol: *protocol,
					Routing:  v1alpha1.ApiServiceInstanceSpecRouting{BasePath: *path},
				}
				endpoints = append(endpoints, endpoint)
			}
		}
	}

	return endpoints
}

func (gc *Client) processKongAPI(
	service *kong.Service,
	oasSpec Openapi,
	endpoints []InstanceEndpoint,
	subscriptionHandler subscription.SubscriptionHandler,
) (*apic.ServiceBody, error) {
	name := gc.centralCfg.GetEnvironmentName() + "." + *service.Name
	kongAPI := newKongAPI(service, oasSpec, endpoints, name, subscriptionHandler)

	serviceBody, err := kongAPI.buildServiceBody()
	if err != nil {
		return nil, fmt.Errorf("failed to build service body: %v", serviceBody)
	}

	serviceBodyHash, _ := util.ComputeHash(serviceBody)
	hash := fmt.Sprintf("%v", serviceBodyHash)
	serviceBody.ServiceAttributes[kongHash] = hash

	isCached := setCachedService(*service.ID, *service.Name, hash, serviceBody.APIName)
	if isCached == true {
		return nil, nil
	}

	return &serviceBody, nil
}

func newKongAPI(
	service *kong.Service,
	oasSpec Openapi,
	endpoints []InstanceEndpoint,
	nameToPush string,
	handler subscription.SubscriptionHandler,
) KongAPI {
	return KongAPI{
		id:                  *service.ID,
		name:                *service.Name,
		description:         oasSpec.Description(),
		version:             oasSpec.Version(),
		url:                 *service.Host,
		resourceType:        oasSpec.ResourceType(),
		documentation:       []byte("\"Sample documentation for API discovery agent\""),
		swaggerSpec:         []byte(oasSpec.spec),
		endpoints:           endpoints,
		nameToPush:          nameToPush,
		subscriptionHandler: handler,
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
		SetServiceEndpoints(ka.endpoints)

	if ka.subscriptionHandler == nil {
		body.SetAuthPolicy(apic.Passthrough)
	} else {
		body.SetAuthPolicy(ka.subscriptionHandler.APICPolicy())
		body.SetSubscriptionName(ka.subscriptionHandler.Name())
	}

	sb, err := body.Build()

	// TODO: add set method for NameToPush
	if err == nil {
		sb.NameToPush = ka.nameToPush
	}

	return sb, err
}

func doesServiceExists(serviceId string, services []*kong.Service) bool {
	for _, srv := range services {
		if serviceId == *srv.ID {
			return true
		}
	}
	log.Infof("Kong service '%s' no longer exists.", serviceId)
	return false
}

func enableSubscription() *subscription.SubscriptionManager {
	// Configure subscription schemas
	sm := subscription.New(logrus.StandardLogger(), agent.GetCentralClient())

	// register schemas
	for _, schema := range sm.Schemas() {
		err := agent.GetCentralClient().RegisterSubscriptionSchema(schema)
		if err != nil {
			log.Errorf("Failed due: %s", err)
		}
		log.Infof("Schema registered: %s", schema.GetSubscriptionName())
	}

	agent.GetCentralClient().GetSubscriptionManager().RegisterValidator(sm.ValidateSubscription)
	// register validator and handlers
	agent.GetCentralClient().GetSubscriptionManager().RegisterProcessor(apic.SubscriptionApproved, sm.ProcessSubscribe)

	// start polling for subscriptions
	agent.GetCentralClient().GetSubscriptionManager().Start()

	return sm
}
