package credential

import (
	"context"
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"
	klib "github.com/kong/go-kong/kong"
)

const testName log.ContextField = "testName"

type mockCredentialClient struct{}

func (mockCredentialClient) DeleteOauth2(ctx context.Context, consumerID, clientID string) error {
	return nil
}

func (mockCredentialClient) DeleteHttpBasic(ctx context.Context, consumerID, username string) error {
	return nil
}

func (mockCredentialClient) DeleteAuthKey(ctx context.Context, consumerID, authKey string) error {
	return nil
}

func (mockCredentialClient) CreateHttpBasic(ctx context.Context, consumerID string, basicAuth *klib.BasicAuth) (*klib.BasicAuth, error) {
	return &klib.BasicAuth{}, nil
}

func (mockCredentialClient) CreateOauth2(ctx context.Context, consumerID string, oauth2 *klib.Oauth2Credential) (*klib.Oauth2Credential, error) {
	return &klib.Oauth2Credential{}, nil
}

func (mockCredentialClient) CreateAuthKey(ctx context.Context, consumerID string, keyAuth *klib.KeyAuth) (*klib.KeyAuth, error) {
	return &klib.KeyAuth{}, nil
}

type mockCredentialRequest struct{}

func (m *mockCredentialRequest) GetApplicationDetailsValue(key string) string {
	return ""
}

func (m *mockCredentialRequest) GetApplicationName() string {
	return ""
}
func (m *mockCredentialRequest) GetCredentialDetailsValue(key string) string {
	return ""
}
func (m *mockCredentialRequest) GetCredentialData() map[string]interface{} {
	return nil
}
func (m *mockCredentialRequest) GetCredentialType() string {
	return ""
}

func TestProvision(t *testing.T) {
	testCases := map[string]struct {
		client       mockCredentialClient
		request      mockCredentialRequest
		expectStatus provisioning.Status
	}{
		"expect error when no app name set": {
			request: mockCredentialRequest{},
			client:  mockCredentialClient{},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testName, name)

			_, _ = NewCredentialProvisioner(ctx, tc.client, &tc.request).Provision()
		})
	}
}
