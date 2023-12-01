package config

import (
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

const (
	cfgKongHTTPLogsPath = "kong.logs.http.path"
	cfgKongHTTPLogsPort = "kong.logs.http.port"
)

func AddKongProperties(rootProps properties.Properties) {
	rootProps.AddStringProperty(cfgKongHTTPLogsPath, "/requestlogs", "Path on which the HTTP Log plugin sends request logs")
	rootProps.AddIntProperty(cfgKongHTTPLogsPort, 9000, "Port that listens for request logs from HTTP Log plugin")
}

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralCfg     corecfg.CentralConfig `config:"central"`
	KongGatewayCfg KongGatewayConfig     `config:"kong"`
}

// KongGatewayConfig - represents the config for gateway
type KongGatewayConfig struct {
	corecfg.IConfigValidator
	Logs KongLogsConfig `config:"logs"`
}

type KongLogsConfig struct {
	HTTP KongLogsHTTPConfig `config:"http"`
}

type KongLogsHTTPConfig struct {
	Path string `config:"path"`
	Port int    `config:"port"`
}

var agentConfig *AgentConfig

func SetAgentConfig(cfg *AgentConfig) {
	agentConfig = cfg
}

func GetAgentConfig() *AgentConfig {
	return agentConfig
}

func ParseProperties(rootProps properties.Properties) KongGatewayConfig {
	// Parse the config from bound properties and setup gateway config
	return KongGatewayConfig{
		Logs: KongLogsConfig{
			HTTP: KongLogsHTTPConfig{
				Path: rootProps.StringPropertyValue(cfgKongHTTPLogsPath),
				Port: rootProps.IntPropertyValue(cfgKongHTTPLogsPort),
			},
		},
	}
}
