package gateway

import (
	"io/ioutil"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
)

// GatewayClient - Represents the Gateway client
type GatewayClient struct {
	cfg *config.GatewayConfig
}

// NewClient - Creates a new Gateway Client
func NewClient(gatewayCfg *config.GatewayConfig) (*GatewayClient, error) {
	return &GatewayClient{
		cfg: gatewayCfg,
	}, nil
}

// ExternalAPI - Sample struct representing the API definition in API gateway
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
func (a *GatewayClient) DiscoverAPIs() error {
	// Gateway specific implementation to get the details for discovered API goes here
	// Set the service definition
	// As sample the implementation reads the swagger for musical-instrument from local directory
	swaggerSpec, err := a.getSpec()
	if err != nil {
		log.Infof("Failed to load sample API specification from %s: %s ", a.cfg.SpecPath, err.Error())
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

	serviceBody, err := a.buildServiceBody(externalAPI)
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
func (a *GatewayClient) buildServiceBody(externalAPI ExternalAPI) (apic.ServiceBody, error) {
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

func (a *GatewayClient) getSpec() ([]byte, error) {
	bytes, err := ioutil.ReadFile(a.cfg.SpecPath)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
