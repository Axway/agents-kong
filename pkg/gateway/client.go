package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
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
		centralCfg:     agentConfig.CentralCfg,
		kongGatewayCfg: kongGatewayConfig,
		kongClient:     kongClient,
		apicClient:     apicClient,
	}, nil
}

func (gc *Client) DiscoverAPIs() error {
	ctx := context.Background()
	apiServices, err := gc.apicClient.fetchCentralAPIServices(nil)
	if err != nil {
		log.Infof("failed to get central api services: %s", err)
	}
	// TODO: initCache should only run once
	initCache(apiServices)

	services, err := gc.kongClient.ListServices(ctx)
	if err != nil {
		log.Errorf("failed to get services: %s", err)
		return err
	}

	gc.removeDeletedServices(services)
	gc.processKongServicesList(ctx, services)
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
				log.Errorf("failed to delete service '%' from the cache: %s", cachedService.kongServiceName, err)
			}
		}
	}
	return nil
}

func (gc *Client) processKongServicesList(ctx context.Context, services []*kong.Service) {
	wg := new(sync.WaitGroup)

	for _, service := range services {
		wg.Add(1)

		go func(service *kong.Service, wg *sync.WaitGroup) {
			defer wg.Done()

			err := gc.processSingleKongService(ctx, service)
			if err != nil {
				log.Error(err)
			}
		}(service, wg)
	}

	wg.Wait()
}

func (gc *Client) processSingleKongService(ctx context.Context, service *kong.Service) error {
	routes, err := gc.kongClient.ListRoutesForService(ctx, *service.ID)
	if err != nil {
		return err
	}
	endpoints := gc.processKongRoute(gc.kongGatewayCfg.ProxyEndpoint, routes)

	kongServiceSpec, err := gc.kongClient.GetSpecForService(ctx, *service.ID)
	if err != nil {
		// TODO: If no spec is found, then it was likely deleted, and should be deleted from central
		return fmt.Errorf("failed to get spec for %s: %s", *service.Name, err)
	}
	oasSpec := Openapi{
		spec: kongServiceSpec.Contents,
	}
	oasSpec.SetOas3Servers(endpointsToURL(endpoints))
	serviceBody, err := gc.processKongAPI(service, oasSpec, endpoints)
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

func (gc *Client) processKongRoute(defaultHost string, routes []*kong.Route) []v1alpha1.ApiServiceInstanceSpecEndpoint {
	var endpoints []v1alpha1.ApiServiceInstanceSpecEndpoint
	if routes == nil || len(routes) == 0 {
		return endpoints
	}
	route := routes[0]
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
				endpoint := v1alpha1.ApiServiceInstanceSpecEndpoint{
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

func buildServiceBody(kongAPI KongAPI) (apic.ServiceBody, error) {
	serviceAttribute := make(map[string]string)
	body := apic.NewServiceBodyBuilder().
		SetAPIName(kongAPI.name).
		SetAPISpec(kongAPI.swaggerSpec).
		SetAuthPolicy(apic.Passthrough).
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

	return body.Build()
}

func (gc *Client) processKongAPI(service *kong.Service, oasSpec Openapi, endpoints []v1alpha1.ApiServiceInstanceSpecEndpoint) (*apic.ServiceBody, error) {
	kongAPI := newKongAPI(service, oasSpec, endpoints)
	// TODO: delete api service if needed
	// If a kong route is deleted, and there are no more routes, then delete the api service?
	// if a kong route no longer has any paths defined, then delete the api service?
	// If an api spec is deleted from the service, then delete the api service?

	serviceBody, err := buildServiceBody(kongAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to build service body: %s", serviceBody)
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

func newKongAPI(service *kong.Service, oasSpec Openapi, endpoints []v1alpha1.ApiServiceInstanceSpecEndpoint) KongAPI {
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

func endpointsToURL(endpoints []v1alpha1.ApiServiceInstanceSpecEndpoint) openapi3.Servers {
	var urls openapi3.Servers
	for _, endpoint := range endpoints {
		url := fmt.Sprintf("%s://%s%s", endpoint.Protocol, endpoint.Host, endpoint.Routing.BasePath)
		urls = append(urls, &openapi3.Server{URL: url})
	}
	return urls
}
