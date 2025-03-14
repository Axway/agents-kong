package application

import (
	"context"
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"
	"github.com/google/uuid"
	klib "github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
)

const testName log.ContextField = "testName"

type mockAppClient struct {
	createErr bool
	deleteErr bool
	addACLErr bool
	consumer  *klib.Consumer
}

func (m mockAppClient) CreateConsumer(ctx context.Context, id, name string) (*klib.Consumer, error) {
	if m.createErr {
		return nil, fmt.Errorf("error")
	}
	return m.consumer, nil
}

func (m mockAppClient) AddConsumerACL(ctx context.Context, id string) error {
	if m.addACLErr {
		return fmt.Errorf("error")
	}
	return nil
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
		"expect error when no app name set": {
			request: mockApplicationRequest{
				id: "appID",
			},
			expectStatus: provisioning.Error,
		},
		"expect error when create consumer fails": {
			client: mockAppClient{
				createErr: true,
			},
			request: mockApplicationRequest{
				name: "appName",
				id:   "appID",
			},
			expectStatus: provisioning.Error,
		},
		"success when provisioning a managed application even when acl call fails": {
			client: mockAppClient{
				addACLErr: true,
				consumer: &klib.Consumer{
					ID: klib.String(uuid.NewString()),
				},
			},
			request: mockApplicationRequest{
				name: "appName",
				id:   "appID",
			},
			expectStatus: provisioning.Success,
		},
		"success when provisioning a managed application": {
			client: mockAppClient{
				consumer: &klib.Consumer{
					ID: klib.String(uuid.NewString()),
				},
			},
			request: mockApplicationRequest{
				name: "appName",
				id:   "appID",
			},
			expectStatus: provisioning.Success,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testName, name)

			result := NewApplicationProvisioner(ctx, tc.client, &tc.request, []string{common.DefaultWorkspace}).Provision()
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

func TestDeleteConsumer(t *testing.T) {
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
					common.AttrAppID: "consumerID",
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
					common.AttrAppID: "consumerID",
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
