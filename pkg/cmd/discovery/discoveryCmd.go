package discovery

import (
	"os"
	"time"

	corecmd "github.com/Axway/agent-sdk/pkg/cmd"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	"github.com/Axway/agents-kong/pkg/gateway"
)

var DiscoveryCmd corecmd.AgentRootCmd
var agentConfig config.AgentConfig

func Execute() {
	if err := DiscoveryCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

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
	rootProps.AddStringProperty("kong.admin_endpoint", "", "The Kong admin endpoint")
	rootProps.AddStringProperty("kong.proxy_endpoint", "", "The Kong proxy endpoint")
	rootProps.AddIntProperty("kong.proxy_endpoint_protocols.http", 80, "The Kong proxy http port")
	rootProps.AddIntProperty("kong.proxy_endpoint_protocols.https", 443, "The Kong proxy https port")
}

// Callback that agent will call to process the execution
func run() error {
	var err error
	var stopChan chan struct{}
	stopChan = make(chan struct{})

	gatewayClient, err := gateway.NewClient(agentConfig)
	go func() {
		for {
			err = gatewayClient.DiscoverAPIs()
			if err != nil {
				log.Error("error in processing: %s", err)
				stopChan <- struct{}{}
			}
			log.Infof("next poll in %s", agentConfig.CentralCfg.GetPollInterval())
			time.Sleep(agentConfig.CentralCfg.GetPollInterval())
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

	// Parse the config from bound properties and setup gateway config
	gatewayConfig := &config.KongGatewayConfig{
		AdminEndpoint:        rootProps.StringPropertyValue("kong.admin_endpoint"),
		Token:                rootProps.StringPropertyValue("kong.token"),
		User:                 rootProps.StringPropertyValue("kong.user"),
		ProxyEndpoint:        rootProps.StringPropertyValue("kong.proxy_endpoint"),
		ProxyHttpPort:        rootProps.IntPropertyValue("kong.proxy_endpoint_protocols.http"),
		ProxyHttpsPort:       rootProps.IntPropertyValue("kong.proxy_endpoint_protocols.https"),
		SpecHomePath:         rootProps.StringPropertyValue("kong.spec_home_path"),
		SpecDevPortalEnabled: rootProps.BoolPropertyValue("kong.spec_dev_portal_enabled"),
	}

	agentConfig = config.AgentConfig{
		CentralCfg:     centralConfig,
		KongGatewayCfg: gatewayConfig,
	}
	return agentConfig, nil
}

func GetAgentConfig() config.AgentConfig {
	return agentConfig
}
