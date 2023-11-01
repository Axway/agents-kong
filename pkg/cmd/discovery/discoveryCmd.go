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

	// Get the root command properties and bind the config property in YAML definition
	rootProps := DiscoveryCmd.GetProperties()
	rootProps.AddStringProperty("kong.token", "", "Token to authenticate with Kong Gateway")
	rootProps.AddStringProperty("kong.adminEndpoint", "", "The Kong admin endpoint")
	rootProps.AddStringProperty("kong.proxyEndpoint", "", "The Kong proxy endpoint")
	rootProps.AddIntProperty("kong.proxyEndpointProtocols.http", 80, "The Kong proxy http port")
	rootProps.AddIntProperty("kong.proxyEndpointProtocols.https", 443, "The Kong proxy https port")
}

// Callback that agent will call to process the execution
func run() error {
	var err error
	var stopChan chan struct{}
	stopChan = make(chan struct{})

	gatewayClient, err := gateway.NewClient(agentConfig)
	if err != nil {
		return err
	}

	go func() {
		for {
			err = gatewayClient.DiscoverAPIs()
			if err != nil {
				log.Errorf("error in processing: %v", err)
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
		AdminEndpoint:        rootProps.StringPropertyValue("kong.adminEndpoint"),
		Token:                rootProps.StringPropertyValue("kong.token"),
		ProxyEndpoint:        rootProps.StringPropertyValue("kong.proxyEndpoint"),
		ProxyHttpPort:        rootProps.IntPropertyValue("kong.proxyEndpointProtocols.http"),
		ProxyHttpsPort:       rootProps.IntPropertyValue("kong.proxyEndpointProtocols.https"),
		SpecHomePath:         rootProps.StringPropertyValue("kong.specHomePath"),
		SpecDevPortalEnabled: rootProps.BoolPropertyValue("kong.specDevPortalEnabled"),
		SpecDownloadPaths:    rootProps.StringSlicePropertyValue("kong.specDownloadPaths"),
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
