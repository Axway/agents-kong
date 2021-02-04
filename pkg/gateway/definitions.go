package gateway

import (
	"github.com/Axway/agents-kong/pkg/kong"
	"github.com/Axway/agents-kong/pkg/subscription"

	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	corecfg "github.com/Axway/agent-sdk/pkg/config"

	config "github.com/Axway/agents-kong/pkg/config/discovery"
)

type InstanceEndpoint = v1alpha1.ApiServiceInstanceSpecEndpoint

// Headers - Type for request/response headers
type Headers map[string]string

// GwTransaction - Type for gateway transaction detail
type GwTransaction struct {
	ID              string  `json:"id"`
	SourceHost      string  `json:"srcHost"`
	SourcePort      int     `json:"srcPort"`
	DesHost         string  `json:"destHost"`
	DestPort        int     `json:"destPort"`
	URI             string  `json:"uri"`
	Method          string  `json:"method"`
	StatusCode      int     `json:"statusCode"`
	RequestHeaders  Headers `json:"requestHeaders"`
	ResponseHeaders Headers `json:"responseHeaders"`
	RequestBytes    int     `json:"requestByte"`
	ResponseBytes   int     `json:"responseByte"`
}

// GwTrafficLogEntry - Represents the structure of log entry the agent will receive
type GwTrafficLogEntry struct {
	TraceID             string        `json:"traceId"`
	APIName             string        `json:"apiName"`
	InboundTransaction  GwTransaction `json:"inbound"`
	OutboundTransaction GwTransaction `json:"outbound"`
}

type Client struct {
	centralCfg          corecfg.CentralConfig
	kongGatewayCfg      *config.KongGatewayConfig
	kongClient          kong.KongAPIClient
	apicClient          CentralClient
	subscriptionManager *subscription.Manager
}

type KongAPI struct {
	swaggerSpec      []byte
	id               string
	name             string
	description      string
	version          string
	url              string
	documentation    []byte
	resourceType     string
	endpoints        []InstanceEndpoint
	subscriptionInfo subscription.Info
	nameToPush       string
}

type CachedService struct {
	kongServiceId   string
	kongServiceName string
	hash            string
	centralName     string
}
