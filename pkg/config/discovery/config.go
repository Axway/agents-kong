package config

import (
	"fmt"

	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

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
	URLPaths []string `config:"urlPaths"`
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
