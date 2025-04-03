package agent

import (
	"context"
	"testing"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	"github.com/Axway/agent-sdk/pkg/cache"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/filter"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"
	config "github.com/Axway/agents-kong/pkg/discovery/config"
	"github.com/Axway/agents-kong/pkg/discovery/kong"
	klib "github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
)

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

func TestNewAgent(t *testing.T) {
	testCases := map[string]struct {
		gatewayConfig *config.KongGatewayConfig
		client        *mockKongClient
		expectErr     bool
	}{
		"error when plugin lister is not created": {
			gatewayConfig: &config.KongGatewayConfig{},
			client:        &mockKongClient{},
			expectErr:     true,
		},
		"error getting kong plugins using lister": {
			gatewayConfig: &config.KongGatewayConfig{},
			client: &mockKongClient{
				GetKongPluginsMock: func() *kong.Plugins {
					return &kong.Plugins{PluginLister: &mockPluginLister{}}
				},
			},
			expectErr: true,
		},
		"error hit because ACL was not installed": {
			gatewayConfig: &config.KongGatewayConfig{},
			client: &mockKongClient{
				GetKongPluginsMock: func() *kong.Plugins {
					return &kong.Plugins{PluginLister: &mockPluginLister{plugins: []*klib.Plugin{}}}
				},
			},
			expectErr: true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			cfg := config.AgentConfig{
				CentralCfg:     corecfg.NewCentralConfig(corecfg.DiscoveryAgent),
				KongGatewayCfg: tc.gatewayConfig,
			}
			agent.InitializeForTest(&mock.Client{}, agent.TestWithMarketplace())

			a, err := NewAgent(cfg, withKongClient(tc.client))
			if tc.expectErr {
				assert.Nil(t, a)
				assert.NotNil(t, err)
			}
		})
	}
}

func TestDiscovery(t *testing.T) {
	testCases := map[string]struct {
		client    *mockKongClient
		expectErr bool
	}{
		"expect error when services call fails": {
			client: &mockKongClient{
				GetKongPluginsMock: func() *kong.Plugins {
					return &kong.Plugins{PluginLister: &mockPluginLister{plugins: []*klib.Plugin{}}}
				},
			},
			expectErr: true,
		},
		"success when no services returned": {
			client: &mockKongClient{
				GetKongPluginsMock: func() *kong.Plugins {
					return &kong.Plugins{PluginLister: &mockPluginLister{plugins: []*klib.Plugin{}}}
				},
				ListServicesMock: func(context.Context) ([]*klib.Service, error) {
					return []*klib.Service{}, nil
				},
			},
		},
		"success when services returned but no routes": {
			client: &mockKongClient{
				GetKongPluginsMock: func() *kong.Plugins {
					return &kong.Plugins{PluginLister: &mockPluginLister{plugins: []*klib.Plugin{}}}
				},
				ListServicesMock: func(context.Context) ([]*klib.Service, error) {
					return []*klib.Service{
						{
							Enabled: boolPtr(true),
							Host:    stringPtr("petstore.com"),
							ID:      stringPtr("petstore-id"),
							Name:    stringPtr("PetStore"),
							Tags:    []*string{},
						},
					}, nil
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			f, _ := filter.NewFilter("")
			ka := &Agent{
				logger:     log.NewFieldLogger().WithComponent("agent").WithPackage("kongAgent"),
				centralCfg: corecfg.NewCentralConfig(corecfg.DiscoveryAgent),
				kongGatewayCfg: &config.KongGatewayConfig{
					Workspaces: []string{common.DefaultWorkspace},
				},
				cache:      cache.New(),
				kongClient: tc.client,
				filter:     f,
			}

			// agent.InitializeForTest()

			err := ka.DiscoverAPIs()
			if tc.expectErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
		})
	}
}
