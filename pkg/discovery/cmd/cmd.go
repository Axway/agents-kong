package discovery

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/apic"
	corecmd "github.com/Axway/agent-sdk/pkg/cmd"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agents-kong/pkg/discovery/agent"
	"github.com/Axway/agents-kong/pkg/discovery/config"
)

var DiscoveryCmd corecmd.AgentRootCmd
var agentConfig config.AgentConfig

func init() {
	// Create new root command with callbacks to initialize the agent config and command execution.
	// The first parameter identifies the name of the yaml file that agent will look for to load the config
	DiscoveryCmd = corecmd.NewRootCmd(
		"kong_discovery_agent",
		"Kong Discovery Agent",
		initConfig,
		run,
		corecfg.DiscoveryAgent,
	)

	// set the dataplane type that will be added to the agent spec
	corecfg.AgentDataPlaneType = apic.Kong.String()

	// Get the root command properties and bind the config property in YAML definition
	rootProps := DiscoveryCmd.GetProperties()
	config.AddKongProperties(rootProps)
}

// Callback that agent will call to process the execution
func run() error {
	var err error
	stopChan := make(chan struct{})

	kongAgent, err := agent.NewAgent(agentConfig)
	if err != nil {
		return err
	}

	go func() {
		for {
			err = kongAgent.DiscoverAPIs()
			if err != nil {
				log.Errorf("error in processing: %v", err)
				stopChan <- struct{}{}
			}
			log.Infof("next poll in %s", agentConfig.CentralCfg.GetPollInterval())
			time.Sleep(agentConfig.CentralCfg.GetPollInterval())
		}
	}()

	<-stopChan
	log.Info("Received signal to stop processing")

	return err
}

// Callback that agent will call to initialize the config. CentralConfig is parsed by Agent SDK
// and passed to the callback allowing the agent code to access the central config
func initConfig(centralConfig corecfg.CentralConfig) (interface{}, error) {
	rootProps := DiscoveryCmd.GetProperties()
	agentConfig = config.AgentConfig{
		CentralCfg:     centralConfig,
		KongGatewayCfg: config.ParseProperties(rootProps),
	}
	return agentConfig, nil
}
