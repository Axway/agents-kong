package traceability

import (
	corecmd "github.com/Axway/agent-sdk/pkg/cmd"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agents-kong/pkg/beater"
	config "github.com/Axway/agents-kong/pkg/config/traceability"
	libcmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
)

var TraceCmd corecmd.AgentRootCmd
var beatCmd *libcmd.BeatsRootCmd

func init() {
	name := "kong_traceability_agent"
	settings := instance.Settings{
		Name:          name,
		HasDashboards: true,
	}

	beatCmd = libcmd.GenRootCmdWithSettings(beater.New, settings)
	cmd := beatCmd.Command
	// Wrap the beat command with the agent command processor with callbacks to initialize the agent config and command execution.
	// The first parameter identifies the name of the yaml file that agent will look for to load the config
	TraceCmd = corecmd.NewCmd(
		&cmd,
		name,
		"Kong Traceability Agent",
		initConfig,
		run,
		corecfg.TraceabilityAgent,
	)

	// Get the root command properties and bind the config property in YAML definition

	rootProps := TraceCmd.GetProperties()
	rootProps.AddStringProperty("kong.user", "", "Kong Gateway admin user")
	rootProps.AddStringProperty("kong.token", "", "Token to authenticate with Kong Gateway")
	rootProps.AddStringProperty("kong.admin_endpoint", "", "The Kong Admin endpoint")
}

func run() error {
	return beatCmd.Execute()
}

// Callback that agent will call to initialize the config. CentralConfig is parsed by Agent SDK
// and passed to the callback allowing the agent code to access the central config
func initConfig(centralConfig corecfg.CentralConfig) (interface{}, error) {
	rootProps := TraceCmd.GetProperties()
	// Parse the config from bound properties and setup gateway config
	gatewayConfig := &config.GatewayConfig{
		// LogFile:        rootProps.StringPropertyValue("gateway-section.logFile"),
		// ProcessOnInput: rootProps.BoolPropertyValue("gateway-section.processOnInput"),
		AdminEndpoint: rootProps.StringPropertyValue("kong.admin_endpoint"),
		Token:         rootProps.StringPropertyValue("kong.token"),
		User:          rootProps.StringPropertyValue("kong.user"),
	}

	agentConfig := &config.AgentConfig{
		CentralCfg: centralConfig,
		GatewayCfg: gatewayConfig,
	}
	beater.SetGatewayConfig(gatewayConfig)

	return agentConfig, nil
}
