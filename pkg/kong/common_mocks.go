package kong

import (
	"context"

	"github.com/Axway/agents-kong/pkg/kong/specmanager"
	klib "github.com/kong/go-kong/kong"
)

type MockKongClient struct {
	*klib.Client
	BaseClient        DoRequest
	KongAdminEndpoint string
}

func (m MockKongClient) ListServices(ctx context.Context) ([]*klib.Service, error) {
	return []*klib.Service{}, nil
}

func (m MockKongClient) ListRoutesForService(ctx context.Context, serviceId string) ([]*klib.Route, error) {
	return []*klib.Route{}, nil
}

func (m MockKongClient) GetSpecForService(ctx context.Context, serviceId string) (*specmanager.KongServiceSpec, error) {
	return &specmanager.KongServiceSpec{}, nil
}

func (m MockKongClient) GetKongPlugins() *Plugins {
	return &Plugins{PluginLister: PluginsMock{}}
}

type PluginsMock []*klib.Plugin

func (pm PluginsMock) ListAll(_ context.Context) ([]*klib.Plugin, error) {
	return pm, nil
}
