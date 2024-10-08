package traceability

import (
	libcmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"

	"github.com/Axway/agent-sdk/pkg/apic"
	corecmd "github.com/Axway/agent-sdk/pkg/cmd"
	corecfg "github.com/Axway/agent-sdk/pkg/config"

	"github.com/Axway/agents-kong/pkg/traceability/beater"
	"github.com/Axway/agents-kong/pkg/traceability/config"
)

var TraceCmd corecmd.AgentRootCmd
var beatCmd *libcmd.BeatsRootCmd

func init() {
	name := "kong_traceability_agent"
	settings := instance.Settings{
		Name:            name,
		HasDashboards:   true,
		ConfigOverrides: corecfg.LogConfigOverrides(),
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

	// set the dataplane type that will be added to the agent spec
	corecfg.AgentDataPlaneType = apic.Kong.String()

	rootProps := TraceCmd.GetProperties()
	config.AddKongProperties(rootProps)
}

func run() error {
	return beatCmd.Execute()
}

// Callback that agent will call to initialize the config. CentralConfig is parsed by Agent SDK
// and passed to the callback allowing the agent code to access the central config
func initConfig(centralConfig corecfg.CentralConfig) (interface{}, error) {

	rootProps := TraceCmd.GetProperties()

	agentConfig := &config.AgentConfig{
		CentralCfg:     centralConfig,
		KongGatewayCfg: config.ParseProperties(rootProps),
	}

	config.SetAgentConfig(agentConfig)

	return agentConfig, nil
}
