package agent

import (
	"context"
	"fmt"

	"github.com/Axway/agents-kong/pkg/discovery/kong"
	klib "github.com/kong/go-kong/kong"
)

type mockKongClient struct {
	// Provisioning
	CreateConsumerMock func(context.Context, string, string) (*klib.Consumer, error)
	AddConsumerACLMock func(context.Context, string) error
	DeleteConsumerMock func(context.Context, string) error
	// Credential
	DeleteOauth2Mock    func(context.Context, string, string) error
	DeleteHttpBasicMock func(context.Context, string, string) error
	DeleteAuthKeyMock   func(context.Context, string, string) error
	CreateHttpBasicMock func(context.Context, string, *klib.BasicAuth) (*klib.BasicAuth, error)
	CreateOauth2Mock    func(context.Context, string, *klib.Oauth2Credential) (*klib.Oauth2Credential, error)
	CreateAuthKeyMock   func(context.Context, string, *klib.KeyAuth) (*klib.KeyAuth, error)
	// Access Request
	AddRouteACLMock    func(context.Context, string, string) error
	RemoveRouteACLMock func(context.Context, string, string) error
	AddQuotaMock       func(context.Context, string, string, string, int) error
	// Discovery
	ListServicesMock         func(context.Context) ([]*klib.Service, error)
	ListRoutesForServiceMock func(context.Context, string) ([]*klib.Route, error)
	GetSpecForServiceMock    func(context.Context, *klib.Service) ([]byte, error)
	GetKongPluginsMock       func() *kong.Plugins
}

func (m *mockKongClient) CreateConsumer(ctx context.Context, id, name string) (*klib.Consumer, error) {
	if m.CreateConsumerMock != nil {
		return m.CreateConsumerMock(ctx, id, name)
	}
	return nil, fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) AddConsumerACL(ctx context.Context, id string) error {
	if m.AddConsumerACLMock != nil {
		return m.AddConsumerACLMock(ctx, id)
	}
	return fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) DeleteConsumer(ctx context.Context, id string) error {
	if m.DeleteConsumerMock != nil {
		return m.DeleteConsumerMock(ctx, id)
	}
	return fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) DeleteOauth2(ctx context.Context, consumerID, clientID string) error {
	if m.DeleteOauth2Mock != nil {
		return m.DeleteOauth2Mock(ctx, consumerID, clientID)
	}
	return fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) DeleteHttpBasic(ctx context.Context, consumerID, username string) error {
	if m.DeleteHttpBasicMock != nil {
		return m.DeleteHttpBasicMock(ctx, consumerID, username)
	}
	return fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) DeleteAuthKey(ctx context.Context, consumerID, authKey string) error {
	if m.DeleteAuthKeyMock != nil {
		return m.DeleteAuthKeyMock(ctx, consumerID, authKey)
	}
	return fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) CreateHttpBasic(ctx context.Context, consumerID string, basicAuth *klib.BasicAuth) (*klib.BasicAuth, error) {
	if m.CreateHttpBasicMock != nil {
		return m.CreateHttpBasicMock(ctx, consumerID, basicAuth)
	}
	return nil, fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) CreateOauth2(ctx context.Context, consumerID string, oauth2 *klib.Oauth2Credential) (*klib.Oauth2Credential, error) {
	if m.CreateOauth2Mock != nil {
		return m.CreateOauth2Mock(ctx, consumerID, oauth2)
	}
	return nil, fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) CreateAuthKey(ctx context.Context, consumerID string, keyAuth *klib.KeyAuth) (*klib.KeyAuth, error) {
	if m.CreateAuthKeyMock != nil {
		return m.CreateAuthKeyMock(ctx, consumerID, keyAuth)
	}
	return nil, fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) AddRouteACL(ctx context.Context, routeID, allowedID string) error {
	if m.AddRouteACLMock != nil {
		return m.AddRouteACLMock(ctx, routeID, allowedID)
	}
	return fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) RemoveRouteACL(ctx context.Context, routeID, revokedID string) error {
	if m.RemoveRouteACLMock != nil {
		return m.RemoveRouteACLMock(ctx, routeID, revokedID)
	}
	return fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) AddQuota(ctx context.Context, routeID, allowedID, quotaInterval string, quotaLimit int) error {
	if m.AddQuotaMock != nil {
		return m.AddQuotaMock(ctx, routeID, allowedID, quotaInterval, quotaLimit)
	}
	return fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) ListServices(ctx context.Context) ([]*klib.Service, error) {
	if m.ListServicesMock != nil {
		return m.ListServicesMock(ctx)
	}
	return nil, fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) ListRoutesForService(ctx context.Context, serviceId string) ([]*klib.Route, error) {
	if m.ListRoutesForServiceMock != nil {
		return m.ListRoutesForServiceMock(ctx, serviceId)
	}
	return nil, fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) GetSpecForService(ctx context.Context, service *klib.Service) ([]byte, error) {
	if m.GetSpecForServiceMock != nil {
		return m.GetSpecForServiceMock(ctx, service)
	}
	return nil, fmt.Errorf("unimplemented test func")
}

func (m *mockKongClient) GetKongPlugins() *kong.Plugins {
	if m.GetKongPluginsMock != nil {
		return m.GetKongPluginsMock()
	}
	return nil
}

type mockPluginLister struct {
	plugins []*klib.Plugin
}

func (m *mockPluginLister) ListAll(ctx context.Context) ([]*klib.Plugin, error) {
	if m.plugins == nil {
		return nil, fmt.Errorf("not implemented by test")
	}
	return m.plugins, nil
}
