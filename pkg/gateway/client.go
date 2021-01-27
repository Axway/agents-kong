package gateway

import (
	"context"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	"github.com/kong/go-kong/kong"
)

type Client struct {
	cfg        *config.GatewayConfig
	kongClient *kong.Client
}

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
	kongClient.SetDebugMode(true)

	return &Client{
		cfg:        gatewayCfg,
		kongClient: kongClient,
	}, nil
}

type ExternalAPI struct {
	swaggerSpec   []byte
	id            string
	name          string
	description   string
	version       string
	url           string
	documentation []byte
}

// DiscoverAPIs - Process the API discovery
func (gc *Client) DiscoverAPIs() error {
	ctx := context.Background()
	gc.GetAllServices(ctx)
	// Gateway specific implementation to get the details for discovered API goes here
	// Set the service definition
	// As sample the implementation reads the swagger for musical-instrument from local directory
	swaggerSpec, err := gc.getSpec()
	if err != nil {
		log.Infof("Failed to load sample API specification %s ", err.Error())
	}

	externalAPI := ExternalAPI{
		id:            "65c79285-f550-4617-bf6e-003e617841f2",
		name:          "Musical-Instrument-Sample",
		description:   "Sample for API discovery agent",
		version:       "1.0.0",
		url:           "",
		documentation: []byte("\"Sample documentation for API discovery agent\""),
		swaggerSpec:   swaggerSpec,
	}

	serviceBody, err := gc.buildServiceBody(externalAPI)
	if err != nil {
		return err
	}
	err = agent.PublishAPI(serviceBody)
	if err != nil {
		return err
	}
	log.Info("Published API " + serviceBody.APIName + "to AMPLIFY Central")
	return err
}

// buildServiceBody - creates the service definition
func (gc *Client) buildServiceBody(externalAPI ExternalAPI) (apic.ServiceBody, error) {
	return apic.NewServiceBodyBuilder().
		SetID(externalAPI.id).
		SetTitle(externalAPI.name).
		SetURL(externalAPI.url).
		SetDescription(externalAPI.description).
		SetAPISpec(externalAPI.swaggerSpec).
		SetVersion(externalAPI.version).
		SetAuthPolicy(apic.Passthrough).
		SetDocumentation(externalAPI.documentation).
		SetResourceType(apic.Oas2).
		Build()
}

func (gc *Client) getSpec() ([]byte, error) {
	var bytes []byte
	return bytes, nil
}

func (gc *Client) GetService(ctx context.Context, service string) (*kong.Service, error) {
	servicesClient := gc.kongClient.Services
	return servicesClient.Get(ctx, &service)
}

func (gc *Client) GetAllServices(ctx context.Context) ([]*kong.Service, error) {
	servicesClient := gc.kongClient.Services
	return servicesClient.ListAll(ctx)
}

func (gc *Client) GetServiceRoutes(service string) {
}

func (gc *Client) GetAllRoutes() {
}
