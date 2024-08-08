package config

import (
	"fmt"
	"strings"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

type props interface {
	AddStringProperty(name string, defaultVal string, description string)
	AddIntProperty(name string, defaultVal int, description string, options ...properties.IntOpt)
	StringPropertyValue(name string) string
	IntPropertyValue(name string) int
}

const (
	cfgKongHTTPLogsPath = "kong.logs.http.path"
	cfgKongHTTPLogsPort = "kong.logs.http.port"
)

func AddKongProperties(rootProps props) {
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

const (
	pathErr = "a path for the http server to listen on is required"
	portErr = "a port for the http server to listen on is required"
)

// ValidateCfg - Validates the gateway config
func (c *KongGatewayConfig) ValidateCfg() (err error) {
	if c.Logs.HTTP.Path == "" {
		return fmt.Errorf(pathErr)
	}
	if c.Logs.HTTP.Port == 0 {
		return fmt.Errorf(portErr)
	}
	return
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

func ParseProperties(rootProps props) KongGatewayConfig {
	// Parse the config from bound properties and setup gateway config
	cfg := KongGatewayConfig{
		Logs: KongLogsConfig{
			HTTP: KongLogsHTTPConfig{
				Path: rootProps.StringPropertyValue(cfgKongHTTPLogsPath),
				Port: rootProps.IntPropertyValue(cfgKongHTTPLogsPort),
			},
		},
	}

	if !strings.HasPrefix(cfg.Logs.HTTP.Path, "/") {
		cfg.Logs.HTTP.Path = fmt.Sprintf("/%s", cfg.Logs.HTTP.Path)
	}

	return cfg
}
