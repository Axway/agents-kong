package config

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

const (
	cfgKongAdminURL          = "kong.admin.url"
	cfgKongAdminAPIKey       = "kong.admin.auth.apikey.value"
	cfgKongAdminAPIKeyHeader = "kong.admin.auth.apikey.header"
	cfgKongProxyHost         = "kong.proxy.host"
	cfgKongProxyPortHttp     = "kong.proxy.port.http"
	cfgKongProxyPortHttps    = "kong.proxy.port.https"
	cfgKongSpecURLPaths      = "kong.spec.urlPaths"
	cfgKongSpecLocalPath     = "kong.spec.localPath"
	cfgKongSpecFilter        = "kong.spec.filter"
)

func AddKongProperties(rootProps properties.Properties) {
	rootProps.AddStringProperty(cfgKongAdminURL, "", "The Kong admin endpoint")
	rootProps.AddStringProperty(cfgKongAdminAPIKey, "", "API Key value to authenticate with Kong Gateway")
	rootProps.AddStringProperty(cfgKongAdminAPIKeyHeader, "", "API Key header to authenticate with Kong Gateway")
	rootProps.AddStringProperty(cfgKongProxyHost, "", "The Kong proxy endpoint")
	rootProps.AddIntProperty(cfgKongProxyPortHttp, 0, "The Kong proxy http port")
	rootProps.AddIntProperty(cfgKongProxyPortHttps, 0, "The Kong proxy https port")
	rootProps.AddStringSliceProperty(cfgKongSpecURLPaths, []string{}, "URL paths that the agent will look in for spec files")
	rootProps.AddStringProperty(cfgKongSpecLocalPath, "", "Local paths where the agent will look for spec files")
	rootProps.AddStringProperty(cfgKongSpecFilter, "", "Which tags the routes must have in order to discover their specs. Empty means filters are ignored.")
}

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralCfg     corecfg.CentralConfig `config:"central"`
	KongGatewayCfg *KongGatewayConfig    `config:"kong"`
}

type KongAdminConfig struct {
	URL  string              `config:"url"`
	Auth KongAdminAuthConfig `config:"auth"`
}

type KongAdminAuthConfig struct {
	APIKey KongAdminAuthAPIKeyConfig `config:"apikey"`
}

type KongAdminAuthAPIKeyConfig struct {
	Header string `config:"header"`
	Value  string `config:"value"`
}

type KongProxyConfig struct {
	Host string              `config:"host"`
	Port KongProxyPortConfig `config:"port"`
}

type KongProxyPortConfig struct {
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

// ValidateCfg - Validates the gateway config
func (c *KongGatewayConfig) ValidateCfg() (err error) {
	if c.Admin.URL == "" {
		return fmt.Errorf("error: admin url is required")
	}
	if c.Proxy.Host == "" {
		return fmt.Errorf("error: proxy host is required")
	}
	if c.Proxy.Port.HTTP == 0 && c.Proxy.Port.HTTPS == 0 {
		return fmt.Errorf("error: at least one proxy port value of either http or https is required")
	}
	return
}

func ParseProperties(rootProps properties.Properties) *KongGatewayConfig {
	// Parse the config from bound properties and setup gateway config
	return &KongGatewayConfig{
		Admin: KongAdminConfig{
			URL: rootProps.StringPropertyValue(cfgKongAdminURL),
			Auth: KongAdminAuthConfig{
				APIKey: KongAdminAuthAPIKeyConfig{
					Value:  rootProps.StringPropertyValue(cfgKongAdminAPIKey),
					Header: rootProps.StringPropertyValue(cfgKongAdminAPIKeyHeader),
				},
			},
		},
		Proxy: KongProxyConfig{
			Host: rootProps.StringPropertyValue(cfgKongProxyHost),
			Port: KongProxyPortConfig{
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
