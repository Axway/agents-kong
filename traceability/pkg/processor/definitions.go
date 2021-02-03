package processor

import (
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	"github.com/Axway/agents-kong/pkg/kong"
	"github.com/Axway/agents-kong/pkg/subscription"
)

// KongTrafficLogEntry - Represents the structure of log entry the agent will receive from Kong's HTTP Log Plugin
type KongTrafficLogEntry struct {
	ClientIP    string     `json:"client_ip"`
	StartedAt   int64      `json:"started_at"`
	UpstreamURI string     `json:"upstream_uri"`
	Latencies   *Latencies `json:"latencies"`
	Request     *Request   `json:"request"`
	Response    *Response  `json:"response"`
	Route       *Route     `json:"route"`
	Service     *Service   `json:"service"`
	Tries       []*Tries   `json:"tries"`
}

type Latencies struct {
	Request int `json:"request"`
	Kong    int `json:"kong"`
	Proxy   int `json:"proxy"`
}

type Request struct {
	QueryString map[string]string `json:"querystring"`
	Size        int               `json:"size"`
	URI         string            `json:"uri"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	Method      string            `json:"method"`
	TLS         *TLS              `json:"tls"`
}

type Response struct {
	Headers map[string]string `json:"headers"`
	Status  int               `json:"status"`
	Size    int               `json:"size"`
}

type Route struct {
	ID                      string            `json:"id"`
	UpdatedAt               int               `json:"updated_at"`
	Protocols               []string          `json:"protocols"`
	StripPath               bool              `json:"strip_path"`
	CreatedAt               int               `json:"created_at"`
	WsID                    string            `json:"ws_id"`
	Service                 map[string]string `json:"service"`
	Name                    string            `json:"name"`
	Hosts                   []string          `json:"hosts"`
	PreserveHost            bool              `json:"preserve_host"`
	RegexPriority           int               `json:"regex_priority"`
	Paths                   []string          `json:"paths"`
	ResponseBuffering       bool              `json:"response_buffering"`
	HttpsRedirectStatusCode int               `json:"https_redirect_status_code"`
	PathHandling            string            `json:"path_handling"`
	RequestBuffering        bool              `json:"request_buffering"`
}

type Service struct {
	Host           string `json:"host"`
	CreatedAt      int    `json:"created_at"`
	ConnectTimeout int    `json:"connect_timeout"`
	ID             string `json:"id"`
	Protocol       string `json:"protocol"`
	Name           string `json:"name"`
	ReadTimeout    int    `json:"read_timeout"`
	Port           int    `json:"port"`
	Path           string `json:"path"`
	UpdatedAt      int    `json:"updated_at"`
	WriteTimeout   int    `json:"write_timeout"`
	Retries        int    `json:"retries"`
	WsID           string `json:"ws_id"`
}

type Tries struct {
	BalancerLatency int    `json:"balancer_latency"`
	Port            int    `json:"port"`
	BalancerStart   int64  `json:"balancer_start"`
	IP              string `json:"ip"`
}

type TLS struct {
	Version                string `json:"version"`
	Cipher                 string `json:"cipher"`
	SupportedClientCiphers string `json:"supported_client_ciphers"`
	ClientVerify           string `json:"client_verify"`
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
