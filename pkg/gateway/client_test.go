package gateway

import (
	"testing"

	corecfg "github.com/Axway/agent-sdk/pkg/config"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
)

func TestKongClient(t *testing.T) {
	gatewayConfig := &config.KongGatewayConfig{}
	_ = config.AgentConfig{
		CentralCfg:     corecfg.NewCentralConfig(corecfg.DiscoveryAgent),
		KongGatewayCfg: gatewayConfig,
	}
}
