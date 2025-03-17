package application

import (
	"context"
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"
	"github.com/stretchr/testify/assert"
)

const testName log.ContextField = "testName"

type mockAppClient struct {
	deleteErr bool
}

func (m mockAppClient) DeleteConsumer(ctx context.Context, id string) error {
	if m.deleteErr {
		return fmt.Errorf("error")
	}
	return nil
}

type mockApplicationRequest struct {
	values map[string]string
	name   string
	id     string
}

func (m mockApplicationRequest) GetApplicationDetailsValue(key string) string {
	if m.values == nil {
		return ""
	}
	if val, ok := m.values[key]; ok {
		return val
	}
	return ""
}

func (m mockApplicationRequest) GetManagedApplicationName() string {
	return m.name
}

func (m mockApplicationRequest) GetID() string {
	return m.id
}

func TestProvision(t *testing.T) {
	testCases := map[string]struct {
		client       mockAppClient
		request      mockApplicationRequest
		expectStatus provisioning.Status
	}{
		"success when provisioning a managed application with no op": {
			expectStatus: provisioning.Success,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testName, name)

			result := NewApplicationProvisioner(ctx, tc.client, &tc.request, []string{common.DefaultWorkspace}).Provision()
			assert.Equal(t, tc.expectStatus, result.GetStatus())
		})
	}
}

func TestDeleteConsumer(t *testing.T) {
	appIDAttr := common.WksPrefixName("default", common.AttrAppID)
	testCases := map[string]struct {
		client       mockAppClient
		request      mockApplicationRequest
		expectStatus provisioning.Status
	}{
		"expect error when no consumer id set": {
			request:      mockApplicationRequest{},
			expectStatus: provisioning.Error,
		},
		"expect error when delete consumer fails": {
			client: mockAppClient{
				deleteErr: true,
			},
			request: mockApplicationRequest{
				name: "appName",
				id:   "appID",
				values: map[string]string{
					appIDAttr: "consumerID",
				},
			},
			expectStatus: provisioning.Error,
		},
		"success deprovisioning a managed application": {
			client: mockAppClient{},
			request: mockApplicationRequest{
				name: "appName",
				id:   "appID",
				values: map[string]string{
					appIDAttr: "consumerID",
				},
			},
			expectStatus: provisioning.Success,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testName, name)

			result := NewApplicationProvisioner(ctx, tc.client, &tc.request, []string{common.DefaultWorkspace}).Deprovision()
			assert.Equal(t, tc.expectStatus, result.GetStatus())
		})
	}
}
