package config

import (
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralCfg corecfg.CentralConfig `config:"central"`
}

type GatewayConfig struct {
	corecfg.IConfigValidator
	LogFile        string `config:"logFile"`
	ProcessOnInput bool   `config:"processOnInput"`
	AdminEndpoint  string `config:"adminEndpoint"`
	Token          string `config:"token"`
	User           string `config:"user"`
}
