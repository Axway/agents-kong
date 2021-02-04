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

// KongGatewayConfig - represents the config for gateway
type KongGatewayConfig struct {
	corecfg.IConfigValidator
	AdminEndpoint        string `config:"adminEndpoint"`
	Token                string `config:"token"`
	User                 string `config:"user"`
	ProxyEndpoint        string `config:"proxyEndpoint"`
	ProxyHttpPort        int    `config:"proxyHttpPort"`
	ProxyHttpsPort       int    `config:"proxyHttpsPort"`
	SpecHomePath         string `config:"specHomePath"`
	SpecDevPortalEnabled bool   `config:"specDevPortalEnabled"`
}

// ValidateCfg - Validates the gateway config
func (c *KongGatewayConfig) ValidateCfg() (err error) {
	if c.Token == "" {
		return fmt.Errorf("error: token is required")
	}
	if c.AdminEndpoint == "" {
		return fmt.Errorf("error: admin_endpoint is required")
	}
	if c.ProxyEndpoint == "" {
		return fmt.Errorf("error: proxy_endpoint is required")
	}
	if c.ProxyHttpPort == 0 || c.ProxyHttpsPort == 0 {
		return fmt.Errorf("error: proxy_endpoint_protocols requires at least one value of either http or https")
	}
	if c.User == "" {
		return fmt.Errorf("error: user is required")
	}
	return
}
