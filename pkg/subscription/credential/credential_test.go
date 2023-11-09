package credential

import (
	"context"
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"
	"github.com/google/uuid"
	klib "github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
)

const testName log.ContextField = "testName"

type mockCredentialClient struct {
	consumer *klib.Consumer
	err      bool
	kongErr  bool
}

func (m mockCredentialClient) UpdateCredential(ctx context.Context, req CredRequest) (prov.RequestStatus, prov.Credential) {
	rs := prov.NewRequestStatusBuilder()
	cred := provisioning.NewCredentialBuilder().
		SetOAuthID("")

	if m.err {
		return rs.Failed(), cred
	}
	if m.kongErr {
		return rs.Failed(), cred
	}

	return rs.Success(), cred
}

func (m mockCredentialClient) DeleteCredential(ctx context.Context, req CredRequest) prov.RequestStatus {
	rs := prov.NewRequestStatusBuilder()

	if m.err {
		return rs.Failed()
	}
	if m.kongErr {
		return rs.Failed()
	}
	return rs.Success()
}

func (m mockCredentialClient) CreateCredential(ctx context.Context, req CredRequest) (prov.RequestStatus, prov.Credential) {
	rs := prov.NewRequestStatusBuilder()
	cred := provisioning.NewCredentialBuilder().
		SetOAuthID("")

	if m.err {
		return rs.Failed(), cred
	}
	if m.kongErr {
		return rs.Failed(), cred
	}
	return rs.Success(), cred
}

type mockCredentialRequest struct {
	name string
	id   string
}

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
		"expect error when create consumer fails": {
			client: mockCredentialClient{
				err: true,
			},
			request: mockCredentialRequest{
				name: "appName",
				id:   "appID",
			},
			expectStatus: provisioning.Error,
		},
		"success when provisioning a managed application even when acl call fails": {
			client: mockCredentialClient{
				consumer: &klib.Consumer{
					ID: klib.String(uuid.NewString()),
				},
			},
			request: mockCredentialRequest{
				name: "appName",
				id:   "appID",
			},
			expectStatus: provisioning.Success,
		},
		"success when provisioning a managed application": {
			client: mockCredentialClient{
				consumer: &klib.Consumer{
					ID: klib.String(uuid.NewString()),
				},
			},
			request: mockCredentialRequest{
				name: "appName",
				id:   "appID",
			},
			expectStatus: provisioning.Success,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testName, name)

			result, _ := NewCredentialProvisioner(ctx, tc.client, &tc.request).Provision()
			assert.Equal(t, tc.expectStatus, result.GetStatus())
			if tc.expectStatus == provisioning.Success {
				// validate consumerID set
				val, ok := result.GetProperties()[common.AttrAppID]
				assert.True(t, ok)
				assert.Equal(t, *tc.client.consumer.ID, val)
			}
		})
	}
}
