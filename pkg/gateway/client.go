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

type Client struct {
	cfg        *config.GatewayConfig
	kongClient *kong.Client
	baseClient http.Client
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
	// kongClient.SetDebugMode(true)

	return &Client{
		cfg:        gatewayCfg,
		kongClient: kongClient,
		baseClient: client,
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
	svcs, err := gc.GetAllServices(ctx)
	if err != nil {
		log.Errorf("failed to get services: %s", err)
	}
	for _, svc := range svcs {
		swaggerSpec := gc.GetServiceSpec(ctx, *svc.ID)
		// swaggerSpec, err := gc.getSpec()
		// if err != nil {
		// 	log.Infof("Failed to load sample API specification %s ", err.Error())
		// }

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
	}

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
	return []byte("{}"), nil
}

func (gc *Client) GetService(ctx context.Context, service string) (*kong.Service, error) {
	servicesClient := gc.kongClient.Services
	return servicesClient.Get(ctx, &service)
}

func (gc *Client) GetAllServices(ctx context.Context) ([]*kong.Service, error) {
	servicesClient := gc.kongClient.Services
	return servicesClient.ListAll(ctx)
}

func (gc *Client) GetServiceSpec(ctx context.Context, serviceId string) []byte {
	// build out get request
	endpoint := fmt.Sprintf("%s/services/%s/document_objects", gc.cfg.AdminEndpoint, serviceId)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		log.Errorf("failed to create request: %s", err)
	}
	res, err := gc.baseClient.Do(req)
	if err != nil {
		log.Errorf("failed to execute request: %s", err)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("failed to read body: %s", err)
	}
	documents := &DocumentObjects{}
	err = json.Unmarshal(data, documents)
	if err != nil {
		log.Errorf("failed to unmarshal: %s", err)
	}

	endpoint = fmt.Sprintf("%s/default/files/%s", gc.cfg.AdminEndpoint, documents.Data[0].Path)
	req, err = http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		log.Errorf("failed to create request: %s", err)
	}
	res, err = gc.baseClient.Do(req)
	if err != nil {
		log.Errorf("failed to execute request: %s", err)
	}
	data, err = ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("failed to read body: %s", err)
	}
	serviceSpec := &ServiceSpec{}
	err = json.Unmarshal(data, serviceSpec)
	if err != nil {
		log.Errorf("failed to unmarshal: %s", err)
	}
	log.Infof("%+v", serviceSpec)
	return []byte(serviceSpec.Contents)
}

type DocumentObjects struct {
	Data []DocumentObject `json:"data,omitempty"`
	Next string           `json:"next,omitempty"`
}

type DocumentObject struct {
	CreatedAt int    `json:"created_at,omitempty"`
	ID        string `json:"id,omitempty"`
	Path      string `json:"path,omitempty"`
	Service   struct {
		ID string `json:"id,omitempty"`
	} `json:"service,omitempty"`
}

type ServiceSpec struct {
	Contents  string `json:"contents"`
	CreatedAt int    `json:"created_at"`
	ID        string `json:"id"`
	Path      string `json:"path"`
}
