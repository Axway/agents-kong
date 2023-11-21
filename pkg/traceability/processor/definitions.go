package processor

import "github.com/Axway/agent-sdk/pkg/util/log"

const (
	CtxTransactionID log.ContextField = "transactionID"
	ctxEntryIndex    log.ContextField = "entryIndex"
)

func init() {
	log.RegisterContextField(CtxTransactionID, ctxEntryIndex)
}

// TrafficLogEntry - Represents the structure of log entry the agent will receive from Kong's HTTP Log Plugin
type TrafficLogEntry struct {
	ClientIP    string     `json:"client_ip"`
	StartedAt   int64      `json:"started_at"`
	UpstreamURI string     `json:"upstream_uri"`
	Latencies   *Latencies `json:"latencies"`
	Request     *Request   `json:"request"`
	Response    *Response  `json:"response"`
	Route       *Route     `json:"route"`
	Service     *Service   `json:"service"`
	Consumer    *Consumer  `json:"consumer"`
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
	UpdatedAt               int64             `json:"updated_at"`
	Protocols               []string          `json:"protocols"`
	StripPath               bool              `json:"strip_path"`
	CreatedAt               int64             `json:"created_at"`
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
	CreatedAt      int64  `json:"created_at"`
	ConnectTimeout int    `json:"connect_timeout"`
	ID             string `json:"id"`
	Protocol       string `json:"protocol"`
	Name           string `json:"name"`
	ReadTimeout    int    `json:"read_timeout"`
	Port           int    `json:"port"`
	Path           string `json:"path"`
	UpdatedAt      int64  `json:"updated_at"`
	WriteTimeout   int    `json:"write_timeout"`
	Retries        int    `json:"retries"`
	WsID           string `json:"ws_id"`
}

type TLS struct {
	Version                string `json:"version"`
	Cipher                 string `json:"cipher"`
	SupportedClientCiphers string `json:"supported_client_ciphers"`
	ClientVerify           string `json:"client_verify"`
}

type Consumer struct {
	CustomID  string   `json:"custom_id"`
	CreatedAt int64    `json:"created_at"`
	ID        string   `json:"id"`
	Tags      []string `json:"tags"`
	Username  string   `json:"username"`
}
