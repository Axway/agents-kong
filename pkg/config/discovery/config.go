package config

import (
	"fmt"

	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralCfg     corecfg.CentralConfig `config:"central"`
	KongGatewayCfg *KongGatewayConfig    `config:"kong"`
}

// KongGatewayConfig - represents the config for gateway
type KongGatewayConfig struct {
	corecfg.IConfigValidator
	AdminEndpoint        string   `config:"adminEndpoint"`
	Token                string   `config:"token"`
	ProxyEndpoint        string   `config:"proxyEndpoint"`
	ProxyHttpPort        int      `config:"proxyHttpPort"`
	ProxyHttpsPort       int      `config:"proxyHttpsPort"`
	SpecHomePath         string   `config:"specHomePath"`
	SpecDevPortalEnabled bool     `config:"specDevPortalEnabled"`
	SpecDownloadPaths    []string `config:"specDownloadPaths"`
}

// ValidateCfg - Validates the gateway config
func (c *KongGatewayConfig) ValidateCfg() (err error) {
	if c.AdminEndpoint == "" {
		return fmt.Errorf("error: adminEndpoint is required")
	}
	if c.ProxyEndpoint == "" {
		return fmt.Errorf("error: proxyEndpoint is required")
	}
	if c.ProxyHttpPort == 0 && c.ProxyHttpsPort == 0 {
		return fmt.Errorf("error: proxyEndpointProtocols requires at least one value of either http or https")
	}
	if c.Token == "" {
		log.Warn("no token set for authenticating with the kong admin endpoint")
	}
	return
}
