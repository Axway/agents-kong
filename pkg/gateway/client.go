package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	"github.com/kong/go-kong/kong"
)

func NewClient(gatewayCfg *config.GatewayConfig) (*Client, error) {
	clientBase := &http.Client{}
	defaultTransport := http.DefaultTransport.(*http.Transport)
	clientBase.Transport = defaultTransport

	headers := make(http.Header)
	headers.Set("Kong-Admin-Token", gatewayCfg.Token)
	client := kong.HTTPClientWithHeaders(clientBase, headers)

	kongClient, err := kong.NewClient(&gatewayCfg.AdminEndpoint, &client)
	if err != nil {
		return nil, err
	}
	// kongClient.SetDebugMode(true)

	return &Client{
		cfg:        gatewayCfg,
		kongClient: kongClient,
		baseClient: client,
	}, nil
}

// DiscoverAPIs - Process the API discovery
func (gc *Client) DiscoverAPIs() error {
	ctx := context.Background()
	services, err := gc.getAllServices(ctx)
	if err != nil {
		log.Errorf("failed to get services: %s", err)
		return err
	}

	return gc.processKongServicesList(ctx, services)
}

func (gc *Client) processKongServicesList(ctx context.Context, services []*kong.Service) error {
	var e error
	for _, service := range services {
		serviceBody, err := gc.processKongAPI(ctx, service)
		if err != nil {
			log.Error(err)
			e = err
			continue
		}

		err = agent.PublishAPI(*serviceBody)
		if err != nil {
			log.Errorf("failed to publish api: %s", err)
			e = err
			continue
		}
		log.Info("Published API " + serviceBody.APIName + " to AMPLIFY Central")
	}
	return e
}

// buildServiceBody - creates the service definition
func (gc *Client) buildServiceBody(kongAPI KongAPI) (apic.ServiceBody, error) {
	serviceAttribute := make(map[string]string)
	serviceAttribute["kong-api-hash"] = "asdf123"
	// serviceAttribute["kong-api-hash"] = convertUnitToString(apiHash)
	serviceAttribute["kong-resource-id"] = kongAPI.id
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

func (gc *Client) getServiceSpec(ctx context.Context, serviceId string) ([]byte, error) {
	// build out get request
	endpoint := fmt.Sprintf("%s/services/%s/document_objects", gc.cfg.AdminEndpoint, serviceId)
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
	return gc.getSpec(ctx, documents.Data[0].Path)
}

func (gc *Client) getSpec(ctx context.Context, path string) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/default/files/%s", gc.cfg.AdminEndpoint, path)
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

	serviceSpec := &ServiceSpec{}
	err = json.Unmarshal(data, serviceSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %s", err)
	}

	return []byte(serviceSpec.Contents), nil
}

func (gc *Client) processKongAPI(ctx context.Context, service *kong.Service) (*apic.ServiceBody, error) {
	// Get hash from API Spec
	// Compare Spec to cache
	// Publish if there is a change
	swaggerSpec, err := gc.getServiceSpec(ctx, *service.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get spec: %s", err)
	}
	oasSpec := Openapi{
		spec: string(swaggerSpec),
	}

	serviceBody, err := gc.buildServiceBody(newKongAPI(service, oasSpec))
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
