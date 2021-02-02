package config

import (
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
	AdminEndpoint string `config:"adminEndpoint"`
	Token         string `config:"token"`
	User          string `config:"user"`
}

// ValidateCfg - Validates the gateway config
func (c *KongGatewayConfig) ValidateCfg() (err error) {
	return
}
