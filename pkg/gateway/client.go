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
	kutil "github.com/Axway/agents-kong/pkg/kong"
	"github.com/Axway/agents-kong/pkg/subscription"
	"github.com/getkin/kin-openapi/openapi3"
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

	plugins := gc.kongClient.GetKongPlugins()
	services, err := gc.kongClient.ListServices(context.Background())
	if err != nil {
		log.Errorf("failed to get services: %s", err)
		return err
	}

	gc.removeDeletedServices(services)
	gc.processKongServicesList(services, plugins)
	return nil
}

func (gc *Client) removeDeletedServices(services []*kong.Service) error {
	specCache := cache.GetCache()
	log.Info("checking for deleted kong services")
	// TODO: add go funcs
	for _, serviceId := range specCache.GetKeys() {
		if !doesServiceExists(serviceId, services) {
			item, err := specCache.Get(serviceId)
			if err != nil {
				log.Errorf("failed to get cached service: %s", serviceId)
				return err
			}
			cachedService := item.(CachedService)
			err = gc.apicClient.deleteCentralAPIService(cachedService)
			if err != nil {
				log.Errorf("failed to remove service '%s': %s", cachedService.kongServiceName, err)
				return err
			}
			err = specCache.Delete(serviceId)
			if err != nil {
				log.Errorf("failed to delete service '%s' from the cache: %s", cachedService.kongServiceName, err)
			}
		}
	}
	return nil
}

func (gc *Client) processKongServicesList(services []*kong.Service, plugins *kutil.Plugins) {
	wg := new(sync.WaitGroup)

	for _, service := range services {
		wg.Add(1)

		go func(service *kong.Service, wg *sync.WaitGroup) {
			defer wg.Done()

			err := gc.processSingleKongService(context.Background(), service, plugins)
			if err != nil {
				log.Error(err)
			}
		}(service, wg)
	}

	wg.Wait()
}

func (gc *Client) processSingleKongService(ctx context.Context, service *kong.Service, plugins *kutil.Plugins) error {
	routes, err := gc.kongClient.ListRoutesForService(ctx, *service.ID)
	if err != nil {
		return err
	}
	if len(routes) == 0 {
		log.Warnf("Kong service %s (%s) has no route and will be ignored", *service.Name, *service.ID)
		return nil
	}

	route := routes[0]
	subscriptionHandler, err := gc.getSubscriptionHandler(route, plugins)

	endpoints := gc.processKongRoute(gc.kongGatewayCfg.ProxyEndpoint, route)
	kongServiceSpec, err := gc.kongClient.GetSpecForService(ctx, *service.ID)
	if err != nil {
		// TODO: If no spec is found, then it was likely deleted, and should be deleted from central
		return fmt.Errorf("failed to get spec for %s: %s", *service.Name, err)
	}

	oasSpec := Openapi{
		spec: kongServiceSpec.Contents,
	}
	oasSpec.SetOas3Servers(endpointsToURL(endpoints))

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

func (gc *Client) getSubscriptionHandler(route *kong.Route, plugins *kutil.Plugins) (subscription.SubscriptionHandler, error) {
	return gc.subscriptionManager.GetEffectiveSubscriptionHandler(route.ID, route.Service.ID, plugins)
}

func (gc *Client) processKongRoute(defaultHost string, route *kong.Route) []InstanceEndpoint {
	var endpoints []InstanceEndpoint
	if route == nil {
		return endpoints
	}
	hosts := route.Hosts
	if len(route.Hosts) == 0 {
		hosts = []*string{&defaultHost}
	}
	for _, host := range hosts {
		for _, path := range route.Paths {
			for _, protocol := range route.Protocols {
				port := 80
				if *protocol == "https" {
					port = 443
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

func buildServiceBody(kongAPI KongAPI, name string, subscriptionHandler subscription.SubscriptionHandler) (apic.ServiceBody, error) {
	serviceAttribute := make(map[string]string)
	body := apic.NewServiceBodyBuilder().
		SetAPIName(kongAPI.name).
		SetAPISpec(kongAPI.swaggerSpec).
		SetDescription(kongAPI.description).
		SetDocumentation(kongAPI.documentation).
		SetID(kongAPI.id).
		SetResourceType(kongAPI.resourceType).
		SetServiceAttribute(serviceAttribute).
		SetTitle(kongAPI.name).
		SetURL(kongAPI.url).
		SetVersion(kongAPI.version)

	if kongAPI.resourceType == apic.Oas2 {
		body = body.SetServiceEndpoints(kongAPI.endpoints)
	}

	if subscriptionHandler == nil {
		body.SetAuthPolicy(apic.Passthrough)
	} else {
		body.SetAuthPolicy(subscriptionHandler.APICPolicy())
		body.SetSubscriptionName(subscriptionHandler.Name())
	}

	sb, err := body.Build()

	// TODO: add set method for NameToPush
	// Set add prefix for unique catalog item
	if err == nil {
		sb.NameToPush = name
	}

	return sb, err
}

func (gc *Client) processKongAPI(
	service *kong.Service,
	oasSpec Openapi,
	endpoints []InstanceEndpoint,
	subscriptionHandler subscription.SubscriptionHandler,
) (*apic.ServiceBody, error) {
	kongAPI := newKongAPI(service, oasSpec, endpoints)

	name := gc.centralCfg.GetEnvironmentName() + "." + kongAPI.name
	serviceBody, err := buildServiceBody(kongAPI, name, subscriptionHandler)
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

func newKongAPI(service *kong.Service, oasSpec Openapi, endpoints []InstanceEndpoint) KongAPI {
	return KongAPI{
		id:            *service.ID,
		name:          *service.Name,
		description:   oasSpec.Description(),
		version:       oasSpec.Version(),
		url:           *service.Host,
		resourceType:  oasSpec.ResourceType(),
		documentation: []byte("\"Sample documentation for API discovery agent\""),
		swaggerSpec:   []byte(oasSpec.spec),
		endpoints:     endpoints,
	}
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

func endpointsToURL(endpoints []InstanceEndpoint) openapi3.Servers {
	var urls openapi3.Servers
	for _, endpoint := range endpoints {
		url := fmt.Sprintf("%s://%s%s", endpoint.Protocol, endpoint.Host, endpoint.Routing.BasePath)
		urls = append(urls, &openapi3.Server{URL: url})
	}
	return urls
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
