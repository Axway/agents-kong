package gateway

import (
	"net/http"

	"github.com/Axway/agent-sdk/pkg/apic"

	config "github.com/Axway/agents-kong/pkg/config/discovery"
	"github.com/kong/go-kong/kong"
)

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

type KongServiceSpec struct {
	Contents  string `json:"contents"`
	CreatedAt int    `json:"created_at"`
	ID        string `json:"id"`
	Path      string `json:"path"`
	Checksum  string `json:"checksum"`
}

type Client struct {
	agentConfig config.AgentConfig
	kongClient  *kong.Client
	baseClient  http.Client
	apicClient  apic.Client
}

type KongAPI struct {
	swaggerSpec   []byte
	id            string
	name          string
	description   string
	version       string
	url           string
	documentation []byte
	resourceType  string
}

type CachedService struct {
	serviceID   string
	serviceName string
	checksum    string
	centralID   string
}
