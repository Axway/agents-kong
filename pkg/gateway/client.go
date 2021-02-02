package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	"github.com/kong/go-kong/kong"
)

const kongChecksum = "kong-api-checksum"
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
	apicClient := agent.GetCentralClient()
	return &Client{
		agentConfig: agentConfig,
		kongClient:  kongClient,
		baseClient:  client,
		apicClient:  apicClient,
	}, nil
}

// DiscoverAPIs - Process the API discovery
func (gc *Client) DiscoverAPIs() error {
	ctx := context.Background()
	gc.initCache()
	services, err := gc.getAllServices(ctx)
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
	for _, serviceId := range specCache.GetKeys() {
		if !gc.serviceExists(serviceId, services) {
			item, err := specCache.Get(serviceId)
			if err != nil {
				log.Errorf("failed to get cached service: %s", serviceId)
				return err
			}
			cachedService := item.(CachedService)
			err = gc.removeService(cachedService)
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

func (gc *Client) removeService(cachedService CachedService) error {
	envName := gc.agentConfig.CentralCfg.GetEnvironmentName()
	url := fmt.Sprintf("%s%s/apiservices/%s", gc.agentConfig.CentralCfg.GetAPIServerURL(), envName, cachedService.centralName)
	log.Info(url)
	// TODO: ExecuteAPI only returns a success when status code is 200
	gc.apicClient.ExecuteAPI(http.MethodDelete, url, nil, nil)

	log.Infof("service removed: %s", cachedService.kongServiceName)
	return nil
}

func (gc *Client) serviceExists(serviceId string, services []*kong.Service) bool {
	for _, srv := range services {
		if serviceId == *srv.ID {
			return true
		}
	}
	log.Infof("Kong service '%s' no longer exists.", serviceId)
	return false
}

func (gc *Client) initCache() {
	envName := gc.agentConfig.CentralCfg.GetEnvironmentName()
	url := fmt.Sprintf("%s%s/apiservices", gc.agentConfig.CentralCfg.GetAPIServerURL(), envName)
	data, err := gc.apicClient.ExecuteAPI(http.MethodGet, url, nil, nil)
	if err != nil {
		log.Errorf("failed to get apiservices for '%s': %s", envName, err)
		return
	}

	var centralAPIServices []*v1alpha1.APIService
	err = json.Unmarshal(data, &centralAPIServices)
	if err != nil {
		log.Errorf("failed to unmarshal apiservices: %s", err)
		return
	}

	for _, apiSvc := range centralAPIServices {
		cacheServiceChecksum(apiSvc.Attributes[externalAPIID], apiSvc.Title, apiSvc.Attributes[kongChecksum], apiSvc.Name)
	}
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
	serviceBody, err := gc.processKongAPI(ctx, service)
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

func (gc *Client) buildServiceBody(kongAPI KongAPI, checksum string) (apic.ServiceBody, error) {
	serviceAttribute := make(map[string]string)
	serviceAttribute[kongChecksum] = checksum
	return apic.NewServiceBodyBuilder().
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
		SetVersion(kongAPI.version).
		Build()
}

func (gc *Client) getAllServices(ctx context.Context) ([]*kong.Service, error) {
	servicesClient := gc.kongClient.Services
	return servicesClient.ListAll(ctx)
}

func (gc *Client) getServiceSpec(ctx context.Context, serviceId string) (*KongServiceSpec, error) {
	// build out get request
	endpoint := fmt.Sprintf("%s/services/%s/document_objects", gc.agentConfig.KongGatewayCfg.AdminEndpoint, serviceId)
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
	return gc.getSpec(ctx, documents.Data[0].Path)
}

func (gc *Client) getSpec(ctx context.Context, path string) (*KongServiceSpec, error) {
	endpoint := fmt.Sprintf("%s/default/files/%s", gc.agentConfig.KongGatewayCfg.AdminEndpoint, path)
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

func (gc *Client) processKongAPI(ctx context.Context, service *kong.Service) (*apic.ServiceBody, error) {
	kongServiceSpec, err := gc.getServiceSpec(ctx, *service.ID)
	if err != nil {
		// TODO: If no spec is found, then it was likely deleted, and should be deleted from central
		return nil, fmt.Errorf("failed to get spec for %s: %s", *service.Name, err)
	}

	oasSpec := Openapi{
		spec: kongServiceSpec.Contents,
	}

	serviceBody, err := gc.buildServiceBody(newKongAPI(service, oasSpec), kongServiceSpec.Checksum)
	isCached := cacheServiceChecksum(*service.ID, *service.Name, kongServiceSpec.Checksum, serviceBody.APIName)
	if isCached == true {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build service body: %s", serviceBody)
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

// If the item is cached, return true
func cacheServiceChecksum(kongServiceId string, kongServiceName string, checksum string, centralName string) bool {
	specCache := cache.GetCache()
	item, err := specCache.Get(kongServiceId)
	// if there is an error, then the item is not in the cache
	if err != nil {
		cachedService := CachedService{
			kongServiceId:   kongServiceId,
			kongServiceName: kongServiceName,
			checksum:        checksum,
			centralName:     centralName,
		}
		specCache.Set(kongServiceId, cachedService)
		log.Infof("adding to the cache: '%s'. centralName: '%s'", kongServiceName, centralName)
		return false
	}

	if item != nil {
		if cachedService, ok := item.(CachedService); ok {
			if cachedService.kongServiceId == kongServiceId && cachedService.checksum == checksum {
				cachedService.centralName = centralName
				cachedService.kongServiceName = kongServiceName
				specCache.Set(kongServiceId, cachedService)
				return true
			} else {
				cachedService.kongServiceName = kongServiceName
				cachedService.checksum = checksum
				specCache.Set(kongServiceId, cachedService)
				log.Infof("adding to the cache: '%s'. centralName: '%'", kongServiceName, centralName)
			}
		}
	}
	return false
}
