package config

import (
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralCfg corecfg.CentralConfig `config:"central"`
	GatewayCfg *GatewayConfig        `config:"kong"`
}

// GatewayConfig - represents the config for gateway
type GatewayConfig struct {
	corecfg.IConfigValidator
	AdminEndpoint string `config:"adminEndpoint"`
	ProxyEndpoint string `config:"proxyEndpoint"`
	Token         string `config:"token"`
	User          string `config:"user"`
}

// ValidateCfg - Validates the gateway config
func (c *GatewayConfig) ValidateCfg() (err error) {
	return
}
