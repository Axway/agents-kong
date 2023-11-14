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

const (
	cfgKongToken             = "kong.token"
	cfgKongAdminEp           = "kong.adminEndpoint"
	cfgKongProxyEp           = "kong.proxyEndpoint"
	cfgKongProxyEpHttp       = "kong.proxyEndpointProtocols.http"
	cfgKongProxyEpHttps      = "kong.proxyEndpointProtocols.https"
	cfgKongSpecDownloadPaths = "kong.specDownloadPaths"
	cfgKongSpecLocalPaths    = "kong.specLocalPaths"
)

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
	rootProps.AddStringProperty(cfgKongToken, "", "Token to authenticate with Kong Gateway")
	rootProps.AddStringProperty(cfgKongAdminEp, "", "The Kong admin endpoint")
	rootProps.AddStringProperty(cfgKongProxyEp, "", "The Kong proxy endpoint")
	rootProps.AddIntProperty(cfgKongProxyEpHttp, 80, "The Kong proxy http port")
	rootProps.AddIntProperty(cfgKongProxyEpHttps, 443, "The Kong proxy https port")
	rootProps.AddStringSliceProperty(cfgKongSpecDownloadPaths, []string{}, "URL paths where the agent will look in for spec files")
	rootProps.AddStringSliceProperty(cfgKongSpecLocalPaths, []string{}, "Local paths where the agent will look for spec files")
}

// Callback that agent will call to process the execution
func run() error {
	var err error
	stopChan := make(chan struct{})

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

	<-stopChan
	log.Info("Received signal to stop processing")

	return err
}

// Callback that agent will call to initialize the config. CentralConfig is parsed by Agent SDK
// and passed to the callback allowing the agent code to access the central config
func initConfig(centralConfig corecfg.CentralConfig) (interface{}, error) {
	rootProps := DiscoveryCmd.GetProperties()

	// Parse the config from bound properties and setup gateway config
	gatewayConfig := &config.KongGatewayConfig{
		Token:             rootProps.StringPropertyValue(cfgKongToken),
		AdminEndpoint:     rootProps.StringPropertyValue(cfgKongAdminEp),
		ProxyEndpoint:     rootProps.StringPropertyValue(cfgKongProxyEp),
		ProxyHttpPort:     rootProps.IntPropertyValue(cfgKongProxyEpHttp),
		ProxyHttpsPort:    rootProps.IntPropertyValue(cfgKongProxyEpHttps),
		SpecDownloadPaths: rootProps.StringSlicePropertyValue(cfgKongSpecDownloadPaths),
		SpecLocalPaths:    rootProps.StringSlicePropertyValue(cfgKongSpecLocalPaths),
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
