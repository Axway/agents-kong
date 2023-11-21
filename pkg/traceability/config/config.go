package config

import (
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

const (
	cfgKongHTTPLogPluginPath = "kong.httpLogPlugin.path"
	cfgKongHTTPLogPluginPort = "kong.httpLogPlugin.port"
)

func AddKongProperties(rootProps properties.Properties) {
	rootProps.AddStringProperty(cfgKongHTTPLogPluginPath, "/requestlogs", "Path on which the HTTP Log plugin sends request logs")
	rootProps.AddIntProperty(cfgKongHTTPLogPluginPort, 9000, "Port that listens for request logs from HTTP Log plugin")
}

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralCfg          corecfg.CentralConfig    `config:"central"`
	HttpLogPluginConfig *KongHttpLogPluginConfig `config:"httpLogPlugin"`
}

type KongHttpLogPluginConfig struct {
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

func ParseProperties(rootProps properties.Properties) *KongHttpLogPluginConfig {
	// Parse the config from bound properties and setup gateway config
	return &KongHttpLogPluginConfig{
		Path: rootProps.StringPropertyValue(cfgKongHTTPLogPluginPath),
		Port: rootProps.IntPropertyValue(cfgKongHTTPLogPluginPort),
	}
}
