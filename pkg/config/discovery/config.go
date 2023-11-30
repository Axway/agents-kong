package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	cfgKongProxyHost         = "kong.proxy.host"
	cfgKongAdminUrl          = "kong.admin.url"
	cfgKongAdminAPIKey       = "kong.admin.auth.apiKey.value"
	cfgKongAdminAPIKeyHeader = "kong.admin.auth.apiKey.header"
	cfgKongAdminUsername     = "kong.admin.auth.basicauth.username"
	cfgKongAdminPassword     = "kong.admin.auth.basicauth.password"
	cfgKongProxyPortHttp     = "kong.proxy.ports.http"
	cfgKongProxyPortHttps    = "kong.proxy.ports.https"
	cfgKongSpecURLPaths      = "kong.spec.urlPaths"
	cfgKongSpecLocalPath     = "kong.spec.localPath"
	cfgKongSpecFilter        = "kong.spec.filter"
)

func AddKongProperties(rootProps properties.Properties) {
	rootProps.AddStringProperty(cfgKongAdminUrl, "", "The Admin API url")
	rootProps.AddStringProperty(cfgKongAdminAPIKey, "", "API Key value to authenticate with Kong Gateway")
	rootProps.AddStringProperty(cfgKongAdminAPIKeyHeader, "", "API Key header to authenticate with Kong Gateway")
	rootProps.AddStringProperty(cfgKongAdminUsername, "", "Username for basic auth to authenticate with Kong Admin API")
	rootProps.AddStringProperty(cfgKongAdminPassword, "", "Password for basic auth to authenticate with Kong Admin API")
	rootProps.AddStringProperty(cfgKongProxyHost, "", "The Kong proxy endpoint")
	rootProps.AddIntProperty(cfgKongProxyPortHttp, 0, "The Kong proxy http port")
	rootProps.AddIntProperty(cfgKongProxyPortHttps, 0, "The Kong proxy https port")
	rootProps.AddStringSliceProperty(cfgKongSpecURLPaths, []string{}, "URL paths that the agent will look in for spec files")
	rootProps.AddStringProperty(cfgKongSpecLocalPath, "", "Local paths where the agent will look for spec files")
	rootProps.AddStringProperty(cfgKongSpecFilter, "", "SDK Filter format. Empty means filters are ignored.")
}

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralCfg     corecfg.CentralConfig `config:"central"`
	KongGatewayCfg *KongGatewayConfig    `config:"kong"`
}

type KongAdminConfig struct {
	Url  string              `config:"url"`
	Auth KongAdminAuthConfig `config:"auth"`
}

type KongAdminAuthConfig struct {
	APIKey    KongAdminAuthAPIKeyConfig `config:"apiKey"`
	BasicAuth KongAdminBasicAuthConfig  `config:"basicAuth"`
}

type KongAdminBasicAuthConfig struct {
	Username string `config:"username"`
	Password string `config:"password"`
}

type KongAdminAuthAPIKeyConfig struct {
	Header string `config:"header"`
	Value  string `config:"value"`
}

type KongProxyConfig struct {
	Host  string         `config:"host"`
	Ports KongPortConfig `config:"ports"`
}

type KongPortConfig struct {
	HTTP  int `config:"http"`
	HTTPS int `config:"https"`
}

type KongSpecConfig struct {
	URLPaths         []string `config:"urlPaths"`
	LocalPath        string   `config:"localPath"`
	DevPortalEnabled bool     `config:"devPortalEnabled"`
	Filter           string   `config:"filter"`
}

// KongGatewayConfig - represents the config for gateway
type KongGatewayConfig struct {
	corecfg.IConfigValidator
	Admin KongAdminConfig `config:"admin"`
	Proxy KongProxyConfig `config:"proxy"`
	Spec  KongSpecConfig  `config:"spec"`
}

const (
	hostErr       = "Kong Host must be provided."
	proxyPortErr  = "Both proxy port values of http https are required"
	invalidUrlErr = "Invalid Admin API url provided. Must contain protocol + hostname + port." +
		"Examples: <http://kong.com:8001>, <https://kong.com:8444>"
	credentialConfigErr = "Invalid authorization configuration provided. " +
		"If provided, (Username and Password) or (ClientID and ClientSecret) must be non-empty"
)

// ValidateCfg - Validates the gateway config
func (c *KongGatewayConfig) ValidateCfg() (err error) {
	logger := log.NewFieldLogger().WithPackage("config").WithComponent("ValidateConfig")
	if c.Proxy.Host == "" {
		return fmt.Errorf(hostErr)
	}
	if c.Proxy.Ports.HTTP == 0 || c.Proxy.Ports.HTTPS == 0 {
		return fmt.Errorf(proxyPortErr)
	}
	if invalidAdminUrl(c.Admin.Url) {
		return fmt.Errorf(invalidUrlErr)
	}
	if noCredentialsProvided(c) {
		logger.Warn("No credentials provided. Assuming Kong Admin API requires no authorization.")
	}
	if invalidCredentialConfig(c) {
		return fmt.Errorf(credentialConfigErr)
	}
	return
}

func noCredentialsProvided(c *KongGatewayConfig) bool {
	apiKey := c.Admin.Auth.APIKey.Value
	user := c.Admin.Auth.BasicAuth.Username
	pass := c.Admin.Auth.BasicAuth.Password

	if apiKey == "" && user == "" && pass == "" {
		return true
	}
	return false
}

func invalidAdminUrl(u string) bool {
	parsedUrl, err := url.Parse(u)
	if err != nil {
		return true
	}
	if parsedUrl.Port() == "" ||
		strings.HasPrefix(parsedUrl.Host, "http://") || strings.HasPrefix(parsedUrl.Host, "https://") {
		return true
	}
	return false
}

func invalidCredentialConfig(c *KongGatewayConfig) bool {
	user := c.Admin.Auth.BasicAuth.Username
	pass := c.Admin.Auth.BasicAuth.Password

	if (user == "" && pass != "") ||
		(user != "" && pass == "") {
		return true
	}
	return false
}

func ParseProperties(rootProps properties.Properties) *KongGatewayConfig {
	// Parse the config from bound properties and setup gateway config
	return &KongGatewayConfig{
		Admin: KongAdminConfig{
			Url: rootProps.StringPropertyValue(cfgKongAdminUrl),
			Auth: KongAdminAuthConfig{
				APIKey: KongAdminAuthAPIKeyConfig{
					Value:  rootProps.StringPropertyValue(cfgKongAdminAPIKey),
					Header: rootProps.StringPropertyValue(cfgKongAdminAPIKeyHeader),
				},
				BasicAuth: KongAdminBasicAuthConfig{
					Username: rootProps.StringPropertyValue(cfgKongAdminUsername),
					Password: rootProps.StringPropertyValue(cfgKongAdminPassword),
				},
			},
		},
		Proxy: KongProxyConfig{
			Host: rootProps.StringPropertyValue(cfgKongProxyHost),
			Ports: KongPortConfig{
				HTTP:  rootProps.IntPropertyValue(cfgKongProxyPortHttp),
				HTTPS: rootProps.IntPropertyValue(cfgKongProxyPortHttps),
			},
		},
		Spec: KongSpecConfig{
			URLPaths:  rootProps.StringSlicePropertyValue(cfgKongSpecURLPaths),
			LocalPath: rootProps.StringPropertyValue(cfgKongSpecLocalPath),
			Filter:    rootProps.StringPropertyValue(cfgKongSpecFilter),
		},
	}
}
