package traceabilityconfig

import (
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralCfg          corecfg.CentralConfig `config:"central"`
	HttpLogPluginConfig *HttpLogPluginConfig  `config:"http_log_plugin_config"`
}

type HttpLogPluginConfig struct {
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
