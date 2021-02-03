package discovery

import (
	"time"

	coreagent "github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	corecmd "github.com/Axway/agent-sdk/pkg/cmd"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	log "github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	"github.com/Axway/agents-kong/pkg/gateway"
)

var DiscoveryCmd corecmd.AgentRootCmd
var gatewayConfig *config.GatewayConfig
var agentConfig *config.AgentConfig

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
	gatewayClient, err := gateway.NewClient(gatewayConfig)
	//err = gatewayClient.DiscoverAPIs()
	apicClient := coreagent.GetCentralClient()
	err = apicClient.RegisterSubscriptionWebhook()
	if err != nil {
		log.Errorf("Unable to register subscription webhook: %s", err.Error())
		return err
	}

	err = createSubscriptionSchema(apicClient)
	if err != nil {
		log.Errorf("Unable to register subscription schema for API Key authentication: %s", err.Error())
		return err
	}

	go func() {
		for {
			err = gatewayClient.DiscoverAPIs()
			if err != nil {
				log.Error("Error in processing API discovery: " + err.Error())
				//stopChan <- struct{}{}
			}
			time.Sleep(time.Duration(agentConfig.GatewayCfg.PollInterval))
		}
	}()

	apicClient.GetSubscriptionManager().RegisterValidator(gatewayClient.ValidateSubscription)
	apicClient.GetSubscriptionManager().RegisterProcessor(apic.SubscriptionApproved, gatewayClient.ProcessSubscribe)
	apicClient.GetSubscriptionManager().RegisterProcessor(apic.SubscriptionUnsubscribeInitiated, gatewayClient.ProcessUnsubscribe)
	apicClient.GetSubscriptionManager().Start()
	return err
}

// Callback that agent will call to initialize the config. CentralConfig is parsed by Agent SDK
// and passed to the callback allowing the agent code to access the central config
func initConfig(centralConfig corecfg.CentralConfig) (interface{}, error) {
	rootProps := DiscoveryCmd.GetProperties()
	// Parse the config from bound properties and setup gateway config
	gatewayConfig = &config.GatewayConfig{
		AdminEndpoint: rootProps.StringPropertyValue("kong.admin_endpoint"),
		Token:         rootProps.StringPropertyValue("kong.token"),
		User:          rootProps.StringPropertyValue("kong.user"),
		PollInterval:  rootProps.IntPropertyValue("kong.PollInterval"),
	}

	agentConfig := config.AgentConfig{
		CentralCfg: centralConfig,
		GatewayCfg: gatewayConfig,
	}
	return agentConfig, nil
}

// GetAgentConfig - Returns the agent config
func GetAgentConfig() *config.GatewayConfig {
	return gatewayConfig
}

func createSubscriptionSchema(apicClient apic.Client) error {
	subscriptionSchema := apic.NewSubscriptionSchema(agentConfig.CentralCfg.GetEnvironmentName() + apic.SubscriptionSchemaNameSuffix)
	//subscriptionSchema.AddProperty("allowTracing", "boolean", "Allow tracing", "", true, make([]string, 0))
	return apicClient.RegisterSubscriptionSchema(subscriptionSchema)
}
