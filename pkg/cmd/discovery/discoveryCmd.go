package discovery

import (
	"time"

	corecmd "github.com/Axway/agent-sdk/pkg/cmd"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	"github.com/Axway/agents-kong/pkg/gateway"
)

var DiscoveryCmd corecmd.AgentRootCmd
var gatewayConfig *config.GatewayConfig

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

	// Get the root command properties and bind the config property in YAML definition
	rootProps := DiscoveryCmd.GetProperties()
	rootProps.AddStringProperty("kong.user", "", "Kong Gateway admin user")
	rootProps.AddStringProperty("kong.token", "", "Token to authenticate with Kong Gateway")
	rootProps.AddStringProperty("kong.admin_endpoint", "", "The Kong Admin endpoint")
}

// Callback that agent will call to process the execution
func run() error {
	var err error
	var stopChan chan struct{}
	stopChan = make(chan struct{})

	gatewayClient, err := gateway.NewClient(gatewayConfig)
	go func() {
		for {
			log.Info("preparing to poll APIs")
			err = gatewayClient.DiscoverAPIs()
			if err != nil {
				log.Error("error in processing: %s", err)
				stopChan <- struct{}{}
			}
			log.Infof("next poll in %s", gatewayConfig.PollInterval)
			time.Sleep(gatewayConfig.PollInterval)
		}
	}()

	select {
	case <-stopChan:
		log.Info("Received signal to stop processing")
		break
	}

	return err
}

// Callback that agent will call to initialize the config. CentralConfig is parsed by Agent SDK
// and passed to the callback allowing the agent code to access the central config
func initConfig(centralConfig corecfg.CentralConfig) (interface{}, error) {
	rootProps := DiscoveryCmd.GetProperties()
	centralConfig.GetPollInterval()
	// Parse the config from bound properties and setup gateway config
	gatewayConfig = &config.GatewayConfig{
		AdminEndpoint: rootProps.StringPropertyValue("kong.admin_endpoint"),
		Token:         rootProps.StringPropertyValue("kong.token"),
		User:          rootProps.StringPropertyValue("kong.user"),
		PollInterval:  rootProps.DurationPropertyValue("central.pollInterval"),
	}

	agentConfig := config.AgentConfig{
		CentralCfg: centralConfig,
		GatewayCfg: gatewayConfig,
	}
	return agentConfig, nil
}

func GetAgentConfig() *config.GatewayConfig {
	return gatewayConfig
}
