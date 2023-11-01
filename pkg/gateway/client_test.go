package gateway

import (
	"context"
	"net/http"
	"testing"

	"github.com/Axway/agent-sdk/pkg/cache"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agents-kong/pkg/common"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	kutil "github.com/Axway/agents-kong/pkg/kong"
	klib "github.com/kong/go-kong/kong"
)

func TestKongClient(t *testing.T) {
	gatewayConfig := &config.KongGatewayConfig{
		AdminEndpoint: "http://localhost",
		ProxyEndpoint: "http://localhost",
	}
	_ = config.AgentConfig{
		CentralCfg:     corecfg.NewCentralConfig(corecfg.DiscoveryAgent),
		KongGatewayCfg: gatewayConfig,
	}
}

type mockLister struct{}

func (m mockLister) ListAll(ctx context.Context) ([]*klib.Plugin, error) {
	return []*klib.Plugin{}, nil
}

func TestKongAPI(t *testing.T) {
	// mockKongAPI := KongAPI{
	// 	swaggerSpec: []byte{},
	// 	crds:        []string{"api-key", "basic-auth"},
	// 	ard:         "api-key",
	// }
	mockAgentConfig := config.AgentConfig{}
	mockKlibClient := &klib.Client{}
	mockKongClient := &Client{
		centralCfg:     mockAgentConfig.CentralCfg,
		kongGatewayCfg: mockAgentConfig.KongGatewayCfg,
		kongClient: &kutil.MockKongClient{
			Client:            mockKlibClient,
			BaseClient:        &http.Client{},
			KongAdminEndpoint: "",
		},
		apicClient: CentralClient{},
		cache:      cache.New(),
		mode:       common.Marketplace,
		plugins:    kutil.Plugins{},
	}
	mockKongClient.plugins.PluginLister = mockLister{}
	mockKongClient.ExecuteDiscovery()
}
