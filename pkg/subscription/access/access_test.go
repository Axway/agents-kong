package access

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

type mockAccessClient struct {
	addManagedAppErr    bool
	removeManagedAppErr bool
	addQuotaErr         bool
}

func (c mockAccessClient) AddManagedAppACL(ctx context.Context, managedAppID, routeID string) error {
	if c.addManagedAppErr {
		return fmt.Errorf("error")
	}
	return nil
}

func (c mockAccessClient) RemoveManagedAppACL(ctx context.Context, routeID, managedAppID string) error {
	if c.removeManagedAppErr {
		return fmt.Errorf("error")
	}
	return nil
}

func (c mockAccessClient) AddQuota(ctx context.Context, routeID, managedAppID, quotaInterval string, quotaLimit int) error {
	if c.addQuotaErr {
		return fmt.Errorf("error")
	}
	return nil
}

type mockAccessRequest struct {
	values  map[string]string
	details map[string]interface{}
	quota   provisioning.Quota
}

func (a mockAccessRequest) GetApplicationDetailsValue(key string) string {
	if a.values == nil {
		return ""
	}
	if val, ok := a.values[key]; ok {
		return val
	}
	return ""
}
func (a mockAccessRequest) GetInstanceDetails() map[string]interface{} {
	if a.details == nil {
		return nil
	}
	return a.details
}

func (a mockAccessRequest) GetQuota() provisioning.Quota {
	if a.quota == nil {
		return nil
	}
	return a.quota
}

type mockQuota struct {
	interval provisioning.QuotaInterval
	limit    int64
	planName string
}

func (q *mockQuota) GetInterval() provisioning.QuotaInterval {
	return q.interval
}

func (q *mockQuota) GetIntervalString() string {
	return q.interval.String()
}

func (q *mockQuota) GetLimit() int64 {
	return q.limit
}

func (q *mockQuota) GetPlanName() string {
	return q.planName
}

func TestProvision(t *testing.T) {
	cases := map[string]struct {
		client  mockAccessClient
		request mockAccessRequest
		result  provisioning.Status
	}{
		"no app id configured": {
			result: provisioning.Error,
		},
		"no route id configured": {
			request: mockAccessRequest{
				values: map[string]string{
					common.AttrAppID: "appID",
				},
			},
			result: provisioning.Error,
		},
		"unsupported quota interval": {
			request: mockAccessRequest{
				values: map[string]string{
					common.AttrAppID: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteId: "routeID",
				},
				quota: &mockQuota{
					interval: provisioning.Weekly,
					limit:    4,
					planName: "planName",
				},
			},
			result: provisioning.Error,
		},
		"error adding managed app id to acl": {
			client: mockAccessClient{
				addManagedAppErr: true,
			},
			request: mockAccessRequest{
				values: map[string]string{
					common.AttrAppID: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteId: "routeID",
				},
				quota: &mockQuota{
					interval: provisioning.Daily,
					limit:    7,
					planName: "planName",
				},
			},
			result: provisioning.Error,
		},
		"error adding quota to plan": {
			client: mockAccessClient{
				addQuotaErr: true,
			},
			request: mockAccessRequest{
				values: map[string]string{
					common.AttrAppID: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteId: "routeID",
				},
				quota: &mockQuota{
					interval: provisioning.Daily,
					limit:    7,
					planName: "planName",
				},
			},
			result: provisioning.Error,
		},
		"success granting access to managed app": {
			client: mockAccessClient{},
			request: mockAccessRequest{
				values: map[string]string{
					common.AttrAppID: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteId: "routeID",
				},
				quota: &mockQuota{
					interval: provisioning.Daily,
					limit:    7,
					planName: "planName",
				},
			},
			result: provisioning.Success,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testName, name)

			result, _ := NewAccessProvisioner(ctx, tc.client, &tc.request).Provision()
			assert.Equal(t, tc.result, result.GetStatus())
		})
	}
}

func TestDeprovision(t *testing.T) {
	cases := map[string]struct {
		client  mockAccessClient
		request mockAccessRequest
		result  provisioning.Status
	}{
		"no app id configured": {
			result: provisioning.Error,
		},
		"no route id configured": {
			request: mockAccessRequest{
				values: map[string]string{
					common.AttrAppID: "appID",
				},
			},
			result: provisioning.Error,
		},
		"error revoking access for managed app": {
			client: mockAccessClient{
				removeManagedAppErr: true,
			},
			request: mockAccessRequest{
				values: map[string]string{
					common.AttrAppID: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteId: "routeID",
				},
				quota: &mockQuota{
					interval: provisioning.Daily,
					limit:    7,
					planName: "planName",
				},
			},
			result: provisioning.Error,
		},
		"success revoking access for managed app": {
			client: mockAccessClient{},
			request: mockAccessRequest{
				values: map[string]string{
					common.AttrAppID: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteId: "routeID",
				},
				quota: &mockQuota{
					interval: provisioning.Daily,
					limit:    7,
					planName: "planName",
				},
			},
			result: provisioning.Success,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testName, name)

			result := NewAccessProvisioner(ctx, tc.client, &tc.request).Deprovision()
			assert.Equal(t, tc.result, result.GetStatus())
		})
	}
}
