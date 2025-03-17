package access

import (
	"context"
	"errors"
	"fmt"
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"
	"github.com/google/uuid"
	klib "github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
)

const testName log.ContextField = "testName"

type mockAccessClient struct {
	addManagedAppErr    bool
	removeManagedAppErr bool
	addQuotaErr         bool
	createAppErr        bool
	addACLErr           bool
	consumer            *klib.Consumer
}

func (m mockAccessClient) CreateConsumer(ctx context.Context, id, name string) (*klib.Consumer, error) {
	if m.createAppErr {
		return nil, fmt.Errorf("error")
	}
	return m.consumer, nil
}

func (m mockAccessClient) AddConsumerACL(ctx context.Context, id string) error {
	if m.addACLErr {
		return fmt.Errorf("error")
	}
	return nil
}

func (c mockAccessClient) AddRouteACL(ctx context.Context, routeID, allowedID string) error {
	if c.addManagedAppErr {
		return fmt.Errorf("error")
	}
	return nil
}

func (c mockAccessClient) RemoveRouteACL(ctx context.Context, routeID, revokedID string) error {
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
	appName string
	values  map[string]string
	details map[string]interface{}
	quota   provisioning.Quota
}

func (a mockAccessRequest) GetApplicationName() string {
	return a.appName
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

type mockCentralClient struct {
	app          *management.ManagedApplication
	subResources map[string]interface{}
}

func (c *mockCentralClient) GetResource(url string) (*v1.ResourceInstance, error) {
	if c.app == nil {
		return nil, errors.New("app not found")
	}
	return c.app.AsInstance()
}

func (c *mockCentralClient) CreateSubResource(rm v1.ResourceMeta, subs map[string]interface{}) error {
	c.subResources = subs
	return nil
}

func TestProvision(t *testing.T) {
	appIDAttr := common.WksPrefixName("default", common.AttrAppID)
	cases := map[string]struct {
		client     mockAccessClient
		request    mockAccessRequest
		result     provisioning.Status
		aclDisable bool
		app        *management.ManagedApplication
	}{
		"no app id configured failure with create consumer": {
			client: mockAccessClient{
				createAppErr: true,
			},
			request: mockAccessRequest{
				appName: "test-app",
			},
			app:    management.NewManagedApplication("test-app", "test"),
			result: provisioning.Error,
		},
		"no app id configured success create app with acl failure": {
			client: mockAccessClient{
				addACLErr: true,
				consumer: &klib.Consumer{
					ID: klib.String(uuid.NewString()),
				},
			},
			request: mockAccessRequest{
				appName: "test-app",
				details: map[string]interface{}{
					common.AttrRouteID:       "routeID",
					common.AttrWorkspaceName: "default",
				},
				quota: &mockQuota{
					interval: provisioning.Daily,
					limit:    7,
					planName: "planName",
				},
			},
			app:    management.NewManagedApplication("test-app", "test"),
			result: provisioning.Success,
		},
		"no app id configured success create app": {
			client: mockAccessClient{
				consumer: &klib.Consumer{
					ID: klib.String(uuid.NewString()),
				},
			},
			request: mockAccessRequest{
				appName: "test-app",
				details: map[string]interface{}{
					common.AttrRouteID:       "routeID",
					common.AttrWorkspaceName: "default",
				},
				quota: &mockQuota{
					interval: provisioning.Daily,
					limit:    7,
					planName: "planName",
				},
			},
			app:    management.NewManagedApplication("test-app", "test"),
			result: provisioning.Success,
		},
		"no workspace configured": {
			request: mockAccessRequest{
				values: map[string]string{
					appIDAttr: "appID",
				},
				details: map[string]interface{}{
					common.AttrWorkspaceName: "default",
				},
			},
			result: provisioning.Error,
		},
		"no route id configured": {
			request: mockAccessRequest{
				values: map[string]string{
					appIDAttr: "appID",
				},
			},
			result: provisioning.Error,
		},
		"ACL disable is active": {
			request: mockAccessRequest{
				values: map[string]string{
					appIDAttr: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteID:       "routeID",
					common.AttrWorkspaceName: "default",
				},
			},
			result:     provisioning.Success,
			aclDisable: true,
		},
		"unsupported quota interval": {
			request: mockAccessRequest{
				values: map[string]string{
					appIDAttr: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteID:       "routeID",
					common.AttrWorkspaceName: "default",
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
					appIDAttr: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteID:       "routeID",
					common.AttrWorkspaceName: "default",
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
					appIDAttr: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteID:       "routeID",
					common.AttrWorkspaceName: "default",
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
					appIDAttr: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteID:       "routeID",
					common.AttrWorkspaceName: "default",
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
			prov := NewAccessProvisioner(ctx, tc.client, &tc.request, tc.aclDisable, "test")
			prov.centralClient = &mockCentralClient{
				app: tc.app,
			}
			result, _ := prov.Provision()
			assert.Equal(t, tc.result, result.GetStatus())
		})
	}
}

func TestDeprovision(t *testing.T) {
	appIDAttr := common.WksPrefixName("default", common.AttrAppID)
	cases := map[string]struct {
		client     mockAccessClient
		request    mockAccessRequest
		result     provisioning.Status
		aclDisable bool
	}{
		"no app id configured": {
			result: provisioning.Error,
		},
		"no route id configured": {
			request: mockAccessRequest{
				values: map[string]string{
					appIDAttr: "appID",
				},
			},
			result: provisioning.Error,
		},
		"ACL disable is active": {
			request: mockAccessRequest{
				values: map[string]string{
					appIDAttr: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteID:       "routeID",
					common.AttrWorkspaceName: "default",
				},
			},
			result:     provisioning.Success,
			aclDisable: true,
		},
		"error revoking access for managed app": {
			client: mockAccessClient{
				removeManagedAppErr: true,
			},
			request: mockAccessRequest{
				values: map[string]string{
					appIDAttr: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteID:       "routeID",
					common.AttrWorkspaceName: "default",
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
					appIDAttr: "appID",
				},
				details: map[string]interface{}{
					common.AttrRouteID:       "routeID",
					common.AttrWorkspaceName: "default",
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

			result := NewAccessProvisioner(ctx, tc.client, &tc.request, tc.aclDisable, "test").Deprovision()
			assert.Equal(t, tc.result, result.GetStatus())
		})
	}
}
