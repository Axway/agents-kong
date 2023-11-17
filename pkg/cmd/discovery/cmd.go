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
	cfgKongAdminURL             = "kong.admin.url"
	cfgKongAdminAPIKey          = "kong.admin.auth.apikey.value"
	cfgKongAdminAPIKeyHeader    = "kong.admin.auth.apikey.header"
	cfgKongProxyHost            = "kong.proxy.host"
	cfgKongProxyPortHttp        = "kong.proxy.port.http"
	cfgKongProxyPortHttps       = "kong.proxy.port.https"
	cfgKongSpecURLPaths         = "kong.spec.urlPaths"
	cfgKongSpecLocalPath        = "kong.spec.localPath"
	cfgKongSpecDevPortalEnabled = "kong.spec.devPortalEnabled"
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
	rootProps.AddStringProperty(cfgKongAdminURL, "", "The Kong admin endpoint")
	rootProps.AddStringProperty(cfgKongAdminAPIKey, "", "API Key value to authenticate with Kong Gateway")
	rootProps.AddStringProperty(cfgKongAdminAPIKeyHeader, "", "API Key header to authenticate with Kong Gateway")
	rootProps.AddStringProperty(cfgKongProxyHost, "", "The Kong proxy endpoint")
	rootProps.AddIntProperty(cfgKongProxyPortHttp, 80, "The Kong proxy http port")
	rootProps.AddIntProperty(cfgKongProxyPortHttps, 443, "The Kong proxy https port")
	rootProps.AddStringSliceProperty(cfgKongSpecURLPaths, []string{}, "URL paths that the agent will look in for spec files")
	rootProps.AddStringProperty(cfgKongSpecLocalPath, "", "Local paths where the agent will look for spec files")
	rootProps.AddBoolProperty(cfgKongSpecDevPortalEnabled, false, "Dev Portal is used to download spec files")
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
		Admin: config.KongAdminConfig{
			URL: rootProps.StringPropertyValue(cfgKongAdminURL),
			Auth: config.KongAdminAuthConfig{
				APIKey: config.KongAdminAuthAPIKeyConfig{
					Value:  rootProps.StringPropertyValue(cfgKongAdminAPIKey),
					Header: rootProps.StringPropertyValue(cfgKongAdminAPIKeyHeader),
				},
			},
		},
		Proxy: config.KongProxyConfig{
			Host: rootProps.StringPropertyValue(cfgKongProxyHost),
			Port: config.KongProxyPortConfig{
				HTTP:  rootProps.IntPropertyValue(cfgKongProxyPortHttp),
				HTTPS: rootProps.IntPropertyValue(cfgKongProxyPortHttps),
			},
		},
		Spec: config.KongSpecConfig{
			URLPaths:         rootProps.StringSlicePropertyValue(cfgKongSpecURLPaths),
			LocalPath:        rootProps.StringPropertyValue(cfgKongSpecLocalPath),
			DevPortalEnabled: rootProps.BoolPropertyValue(cfgKongSpecDevPortalEnabled),
		},
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
