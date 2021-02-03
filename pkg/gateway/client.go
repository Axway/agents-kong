package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	kutil "github.com/Axway/agents-kong/pkg/kong"
	"github.com/Axway/agents-kong/pkg/subscription"
	"github.com/kong/go-kong/kong"
)

const kongHash = "kong-hash"
const externalAPIID = "externalAPIID"

func NewClient(agentConfig config.AgentConfig) (*Client, error) {
	kongGatewayConfig := agentConfig.KongGatewayCfg
	clientBase := &http.Client{}
	defaultTransport := http.DefaultTransport.(*http.Transport)
	clientBase.Transport = defaultTransport

	headers := make(http.Header)
	headers.Set("Kong-Admin-Token", kongGatewayConfig.Token)
	client := kong.HTTPClientWithHeaders(clientBase, headers)

	kongClient, err := kong.NewClient(&kongGatewayConfig.AdminEndpoint, &client)
	if err != nil {
		return nil, err
	}

	sm := enableSubscription()

	apicClient := CentralClient{
		client:        agent.GetCentralClient(),
		envName:       agentConfig.CentralCfg.GetEnvironmentName(),
		apiServerHost: agentConfig.CentralCfg.GetAPIServerURL(),
	}
	return &Client{
		centralCfg:          agentConfig.CentralCfg,
		kongGatewayCfg:      agentConfig.KongGatewayCfg,
		kongClient:          kongClient,
		baseClient:          client,
		apicClient:          apicClient,
		subscriptionManager: sm,
	}, nil
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

func (gc *Client) DiscoverAPIs() error {
	ctx := context.Background()
	apiServices, err := gc.apicClient.fetchCentralAPIServices(nil)
	if err != nil {
		log.Infof("failed to get central api services: %s", err)
	}
	initCache(apiServices)

	plugins := gc.getKongPlugins()

	services, err := gc.kongClient.Services.ListAll(ctx)
	if err != nil {
		log.Errorf("failed to get services: %s", err)
		return err
	}

	gc.removeDeletedServices(services)
	gc.processKongServicesList(ctx, services, plugins)
	return nil
}

func (gc *Client) getKongPlugins() *kutil.Plugins {
	plugins := gc.kongClient.Plugins
	return &kutil.Plugins{PluginLister: plugins}
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

func (gc *Client) processKongServicesList(ctx context.Context, services []*kong.Service, plugins *kutil.Plugins) {
	wg := new(sync.WaitGroup)

	for _, service := range services {
		wg.Add(1)

		go func(service *kong.Service, wg *sync.WaitGroup) {
			defer wg.Done()

			err := gc.processSingleKongService(ctx, service, plugins)
			if err != nil {
				log.Error(err)
			}
		}(service, wg)
	}

	wg.Wait()
}

func (gc *Client) processSingleKongService(ctx context.Context, service *kong.Service, plugins *kutil.Plugins) error {
	routes, err := gc.getRoutesForService(ctx, *service.ID)
	if err != nil {
		return err
	}

	if len(routes) == 0 {
		log.Warnf("Kong service %s (%s) has no route and will be ignored", *service.Name, *service.ID)
		return nil
	}

	// just use the first route
	route := routes[0]

	subscriptionHandler, err := gc.getSubscriptionHandler(ctx, route, plugins)

	endpoints := gc.processKongRoute(gc.kongGatewayCfg.ProxyEndpoint, route)
	serviceBody, err := gc.processKongAPI(ctx, service, endpoints, subscriptionHandler)
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

func (gc *Client) getSubscriptionHandler(ctx context.Context, route *kong.Route, plugins *kutil.Plugins) (subscription.SubscriptionHandler, error) {
	return gc.subscriptionManager.GetEffectiveSubscriptionHandler(route.ID, route.Service.ID, plugins)
}

func (gc *Client) processKongRoute(defaultHost string, route *kong.Route) []v1alpha1.ApiServiceInstanceSpecEndpoint {
	var endpoints []v1alpha1.ApiServiceInstanceSpecEndpoint
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

func (gc *Client) buildServiceBody(kongAPI KongAPI, endpoints []v1alpha1.ApiServiceInstanceSpecEndpoint, subscriptionHandler subscription.SubscriptionHandler) (apic.ServiceBody, error) {
	serviceAttribute := make(map[string]string)
	sbb := apic.NewServiceBodyBuilder().
		SetAPIName(kongAPI.name).
		SetAPISpec(kongAPI.swaggerSpec).
		SetDescription(kongAPI.description).
		SetDocumentation(kongAPI.documentation).
		SetID(kongAPI.id).
		SetResourceType(kongAPI.resourceType).
		SetServiceAttribute(serviceAttribute).
		SetTitle(kongAPI.name).
		SetURL(kongAPI.url).
		SetVersion(kongAPI.version).
		SetServiceEndpoints(endpoints)

	if subscriptionHandler == nil {
		sbb.SetAuthPolicy(apic.Passthrough)
	} else {
		sbb.SetAuthPolicy(subscriptionHandler.APICPolicy())
		sbb.SetSubscriptionName(subscriptionHandler.Name())
	}

	sb, err := sbb.Build()

	// Set add prefix for unique catalog item
	if err == nil {
		sb.NameToPush = gc.centralCfg.GetEnvironmentName() + "." + kongAPI.name
	}

	return sb, err
}

func (gc *Client) getRoutesForService(ctx context.Context, serviceId string) ([]*kong.Route, error) {
	routes, _, err := gc.kongClient.Routes.ListForService(ctx, &serviceId, nil)
	return routes, err
}

func (gc *Client) getServiceSpec(ctx context.Context, serviceId string) (*KongServiceSpec, error) {
	endpoint := fmt.Sprintf("%s/services/%s/document_objects", gc.kongGatewayCfg.AdminEndpoint, serviceId)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %s", err)
	}
	res, err := gc.baseClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %s", err)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %s", err)
	}
	documents := &DocumentObjects{}
	err = json.Unmarshal(data, documents)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %s", err)
	}
	if len(documents.Data) < 1 {
		return nil, fmt.Errorf("no documents found")
	}
	spec, err := gc.getSpec(ctx, documents.Data[0].Path)

	return spec, err
}

func (gc *Client) getSpec(ctx context.Context, path string) (*KongServiceSpec, error) {
	endpoint := fmt.Sprintf("%s/default/files/%s", gc.kongGatewayCfg.AdminEndpoint, path)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %s", err)
	}

	res, err := gc.baseClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %s", err)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %s", err)
	}

	kongServiceSpec := &KongServiceSpec{}
	err = json.Unmarshal(data, kongServiceSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %s", err)
	}
	if len(kongServiceSpec.Contents) == 0 {
		return nil, fmt.Errorf("spec not found at '%s'", path)
	}
	return kongServiceSpec, nil
}

func (gc *Client) processKongAPI(
	ctx context.Context,
	service *kong.Service,
	endpoints []v1alpha1.ApiServiceInstanceSpecEndpoint,
	subscriptionHandler subscription.SubscriptionHandler,
) (*apic.ServiceBody, error) {
	kongServiceSpec, err := gc.getServiceSpec(ctx, *service.ID)
	if err != nil {
		// TODO: If no spec is found, then it was likely deleted, and should be deleted from central
		return nil, fmt.Errorf("failed to get spec for %s: %s", *service.Name, err)
	}

	oasSpec := Openapi{
		spec: kongServiceSpec.Contents,
	}

	kongAPI := newKongAPI(service, oasSpec)
	// TODO: delete api service if needed
	// If a kong route is deleted, and there are no more routes, then delete the api service?
	// if a kong route no longer has any paths defined, then delete the api service?
	// If an api spec is deleted from the service, then delete the api service?

	serviceBody, err := gc.buildServiceBody(kongAPI, endpoints, subscriptionHandler)
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

func newKongAPI(service *kong.Service, oasSpec Openapi) KongAPI {

	return KongAPI{
		id:            *service.ID,
		name:          *service.Name,
		description:   oasSpec.Description(),
		version:       oasSpec.Version(),
		url:           *service.Host,
		resourceType:  oasSpec.ResourceType(),
		documentation: []byte("\"Sample documentation for API discovery agent\""),
		swaggerSpec:   []byte(oasSpec.spec),
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
