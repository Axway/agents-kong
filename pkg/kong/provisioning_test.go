package kong

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	klib "github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agents-kong/pkg/common"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
)

func formatRequestKey(method, path string) string {
	return fmt.Sprintf("%s-%s", method, path)
}

type response struct {
	code      int
	dataIface interface{}
	data      []byte
}

type mockCredentialRequest struct {
	credType string
	appName  string
	details  string
}

func (m mockCredentialRequest) GetApplicationDetailsValue(key string) string {
	return m.details
}

func (m mockCredentialRequest) GetApplicationName() string {
	return m.appName
}
func (mockCredentialRequest) GetCredentialDetailsValue(key string) string {
	return key
}
func (mockCredentialRequest) GetCredentialData() map[string]interface{} {
	return nil
}
func (m mockCredentialRequest) GetCredentialType() string {
	return m.credType
}

type mockKeyAuthService struct{}

func (mockKeyAuthService) Create(ctx context.Context, consumerUsernameOrID *string, keyAuth *klib.KeyAuth) (*klib.KeyAuth, error) {
	return &klib.KeyAuth{}, nil
}
func (mockKeyAuthService) Get(ctx context.Context, consumerUsernameOrID, keyOrID *string) (*klib.KeyAuth, error) {
	return &klib.KeyAuth{}, nil
}
func (mockKeyAuthService) Update(ctx context.Context, consumerUsernameOrID *string, keyAuth *klib.KeyAuth) (*klib.KeyAuth, error) {
	return &klib.KeyAuth{}, nil
}
func (mockKeyAuthService) Delete(ctx context.Context, consumerUsernameOrID, keyOrID *string) error {
	return nil
}
func (mockKeyAuthService) List(ctx context.Context, opt *klib.ListOpt) ([]*klib.KeyAuth, *klib.ListOpt, error) {
	return []*klib.KeyAuth{}, &klib.ListOpt{}, nil
}
func (mockKeyAuthService) ListAll(ctx context.Context) ([]*klib.KeyAuth, error) {
	return []*klib.KeyAuth{}, nil
}
func (mockKeyAuthService) ListForConsumer(ctx context.Context, consumerUsernameOrID *string, opt *klib.ListOpt) ([]*klib.KeyAuth, *klib.ListOpt, error) {
	return []*klib.KeyAuth{}, &klib.ListOpt{}, nil
}

func createClient(responses map[string]response) KongAPIClient {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if res, found := responses[formatRequestKey(req.Method, req.URL.Path)]; found {
			resp.WriteHeader(res.code)
			if res.dataIface != nil {
				data, _ := json.Marshal(res.dataIface)
				resp.Write(data)
			} else {
				resp.Write(res.data)
			}
			return
		}
	}))
	u, _ := url.Parse(s.URL)
	port, _ := strconv.Atoi(u.Port())
	cfg := &config.KongGatewayConfig{
		Host: u.Hostname(),
		Proxy: config.KongProxyConfig{
			Ports: config.KongPortConfig{
				HTTP:  port,
				HTTPS: port,
			},
		},
		Admin: config.KongAdminConfig{
			Url: s.URL,
		},
	}
	if err := cfg.ValidateCfg(); err != nil {
		panic(err)
	}
	client, _ := NewKongClient(&http.Client{}, cfg)
	return client
}

func TestCreateConsumer(t *testing.T) {
	testCases := map[string]struct {
		expectErr bool
		id        string
		name      string
		responses map[string]response
	}{
		"find existing consumer": {
			expectErr: false,
			id:        "existingID",
			name:      "existingName",
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/consumers/existingID"): {
					code: http.StatusOK,
					dataIface: &klib.Consumer{
						ID:       klib.String("existingID"),
						Username: klib.String("existingName"),
					},
				},
			},
		},
		"create new consumer": {
			expectErr: false,
			id:        "nameID",
			name:      "newName",
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/consumers/nameID"): {
					code: http.StatusNotFound,
				},
				formatRequestKey(http.MethodPost, "/consumers"): {
					code: http.StatusCreated,
					dataIface: &klib.Consumer{
						ID:       klib.String("nameID"),
						Username: klib.String("newName"),
					},
				},
			},
		},
		"create new consumer error": {
			expectErr: true,
			id:        "nameID",
			name:      "newName",
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/consumers/nameID"): {
					code: http.StatusNotFound,
				},
				formatRequestKey(http.MethodPost, "/consumers"): {
					code: http.StatusBadRequest,
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client := createClient(tc.responses)
			c, err := client.CreateConsumer(context.TODO(), tc.id, tc.name)
			if tc.expectErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
			assert.Equal(t, tc.id, *c.ID)
			assert.Equal(t, tc.name, *c.Username)
		})
	}
}

func TestCreateCredentials(t *testing.T) {
	testCases := map[string]struct {
		expectErr bool
		req       mockCredentialRequest
		responses map[string]response
	}{
		"find existing consumer": {
			expectErr: false,
			req: mockCredentialRequest{
				credType: "api-key",
			},
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/consumers/existingID"): {
					code: http.StatusOK,
					dataIface: &klib.Consumer{
						ID:       klib.String("existingID"),
						Username: klib.String("existingName"),
					},
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client := createClient(tc.responses)
			c := client.(*KongClient)
			c.KeyAuths = mockKeyAuthService{}
			var err error
			if tc.expectErr {
				assert.NotNil(t, err)
				return
			}
		})
	}
}

func TestAddConsumerACL(t *testing.T) {
	testCases := map[string]struct {
		expectErr bool
		responses map[string]response
	}{
		"consumer does not exist": {
			expectErr: true,
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/consumers/id"): {
					code: http.StatusNotFound,
				},
			},
		},
		"add consumer acl": {
			expectErr: false,
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/consumers/id"): {
					code: http.StatusOK,
					dataIface: &klib.Consumer{
						ID:       klib.String("id"),
						Username: klib.String("name"),
					},
				},
				formatRequestKey(http.MethodPost, "/consumers/id/acls"): {
					code:      http.StatusOK,
					dataIface: &klib.ACLGroup{},
				},
			},
		},
		"add consumer acl error": {
			expectErr: true,
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/consumers/id"): {
					code: http.StatusOK,
					dataIface: &klib.Consumer{
						ID:       klib.String("id"),
						Username: klib.String("name"),
					},
				},
				formatRequestKey(http.MethodPost, "/consumers/id/acls"): {
					code: http.StatusBadRequest,
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client := createClient(tc.responses)
			err := client.AddConsumerACL(context.TODO(), "id")
			if tc.expectErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
		})
	}
}

func TestDeleteConsumer(t *testing.T) {
	testCases := map[string]struct {
		expectErr bool
		responses map[string]response
	}{
		"consumer does not exist": {
			expectErr: false,
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/consumers/id"): {
					code: http.StatusNotFound,
				},
			},
		},
		"delete consumer": {
			expectErr: false,
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/consumers/id"): {
					code: http.StatusOK,
					dataIface: &klib.Consumer{
						ID:       klib.String("id"),
						Username: klib.String("name"),
					},
				},
				formatRequestKey(http.MethodDelete, "/consumers/id"): {
					code: http.StatusAccepted,
				},
			},
		},
		"delete consumer error": {
			expectErr: true,
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/consumers/id"): {
					code: http.StatusOK,
					dataIface: &klib.Consumer{
						ID:       klib.String("id"),
						Username: klib.String("name"),
					},
				},
				formatRequestKey(http.MethodDelete, "/consumers/id"): {
					code: http.StatusBadRequest,
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client := createClient(tc.responses)
			err := client.DeleteConsumer(context.TODO(), "id")
			if tc.expectErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
		})
	}
}

func TestAddRouteACL(t *testing.T) {
	testCases := map[string]struct {
		expectErr  bool
		consumerID string
		routeID    string
		responses  map[string]response
	}{
		"access already granted": {
			expectErr:  false,
			consumerID: "consumerID",
			routeID:    "routeID",
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/plugins"): {
					code: http.StatusOK,
					dataIface: map[string]interface{}{
						"data": []*klib.Plugin{
							{
								ID:   klib.String("aclPluginID"),
								Name: klib.String(common.AclPlugin),
								Route: &klib.Route{
									ID: klib.String("routeID"),
								},
								Config: klib.Configuration{
									"allow": []string{"consumerID"},
								},
							},
						},
						"next": "null",
					},
				},
			},
		},
		"grant access, acl doesn't exist": {
			expectErr:  false,
			consumerID: "consumerID",
			routeID:    "routeID",
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/plugins"): {
					code: http.StatusOK,
					dataIface: map[string]interface{}{
						"data": []*klib.Plugin{},
						"next": "null",
					},
				},
				formatRequestKey(http.MethodPost, "/routes/routeID/plugins"): {
					code: http.StatusOK,
					dataIface: &klib.Plugin{
						ID:   klib.String("aclPluginID"),
						Name: klib.String(common.AclPlugin),
						Route: &klib.Route{
							ID: klib.String("routeID"),
						},
						Config: klib.Configuration{
							"allow": []string{"consumerID"},
						},
					},
				},
			},
		},
		"grant access, acl exists": {
			expectErr:  false,
			consumerID: "consumerID",
			routeID:    "routeID",
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/plugins"): {
					code: http.StatusOK,
					dataIface: map[string]interface{}{
						"data": []*klib.Plugin{
							{
								ID:   klib.String("aclPluginID"),
								Name: klib.String(common.AclPlugin),
								Route: &klib.Route{
									ID: klib.String("routeID"),
								},
								Config: klib.Configuration{
									"allow": []string{},
								},
							},
						},
						"next": "null",
					},
				},
				formatRequestKey(http.MethodPatch, "/routes/routeID/plugins/aclPluginID"): {
					code: http.StatusOK,
					dataIface: &klib.Plugin{
						ID:   klib.String("aclPluginID"),
						Name: klib.String(common.AclPlugin),
						Route: &klib.Route{
							ID: klib.String("routeID"),
						},
						Config: klib.Configuration{
							"allow": []string{"consumerID"},
						},
					},
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client := createClient(tc.responses)
			err := client.AddRouteACL(context.TODO(), tc.routeID, tc.consumerID)
			if tc.expectErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
		})
	}
}

func TestRemoveRouteACL(t *testing.T) {
	testCases := map[string]struct {
		expectErr  bool
		consumerID string
		routeID    string
		responses  map[string]response
	}{
		"access already revoked": {
			expectErr:  false,
			consumerID: "consumerID",
			routeID:    "routeID",
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/plugins"): {
					code: http.StatusOK,
					dataIface: map[string]interface{}{
						"data": []*klib.Plugin{
							{
								ID:   klib.String("aclPluginID"),
								Name: klib.String(common.AclPlugin),
								Route: &klib.Route{
									ID: klib.String("routeID"),
								},
								Config: klib.Configuration{
									"allow": []string{},
								},
							},
						},
						"next": "null",
					},
				},
			},
		},
		"access revoked, no rate limiting plugin": {
			expectErr:  false,
			consumerID: "consumerID",
			routeID:    "routeID",
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/plugins"): {
					code: http.StatusOK,
					dataIface: map[string]interface{}{
						"data": []*klib.Plugin{
							{
								ID:   klib.String("aclPluginID"),
								Name: klib.String(common.AclPlugin),
								Route: &klib.Route{
									ID: klib.String("routeID"),
								},
								Config: klib.Configuration{
									"allow": []string{"consumerID"},
								},
							},
						},
						"next": "null",
					},
				},
				formatRequestKey(http.MethodDelete, "/routes/routeID/plugins/aclPluginID"): {
					code: http.StatusNoContent,
				},
			},
		},
		"access revoked, rate limiting plugin disabled": {
			expectErr:  false,
			consumerID: "consumerID",
			routeID:    "routeID",
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/plugins"): {
					code: http.StatusOK,
					dataIface: map[string]interface{}{
						"data": []*klib.Plugin{
							{
								ID:   klib.String("aclPluginID"),
								Name: klib.String(common.AclPlugin),
								Route: &klib.Route{
									ID: klib.String("routeID"),
								},
								Config: klib.Configuration{
									"allow": []string{"consumerID"},
								},
							},
							{
								ID:   klib.String("rateLimitingID"),
								Name: klib.String(common.RateLimitingPlugin),
								Route: &klib.Route{
									ID: klib.String("routeID"),
								},
								Enabled: klib.Bool(true),
							},
						},
						"next": "null",
					},
				},
				formatRequestKey(http.MethodDelete, "/routes/routeID/plugins/aclPluginID"): {
					code: http.StatusNoContent,
				},
				formatRequestKey(http.MethodPatch, "/routes/routeID/plugins/rateLimitingID"): {
					code: http.StatusOK,
					dataIface: &klib.Plugin{
						ID:   klib.String("rateLimitingID"),
						Name: klib.String(common.RateLimitingPlugin),
						Route: &klib.Route{
							ID: klib.String("routeID"),
						},
						Enabled: klib.Bool(false),
					},
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client := createClient(tc.responses)
			err := client.RemoveRouteACL(context.TODO(), tc.routeID, tc.consumerID)
			if tc.expectErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
		})
	}
}

func TestAddQuota(t *testing.T) {
	testCases := map[string]struct {
		expectErr     bool
		consumerID    string
		routeID       string
		quotaInterval string
		quotaLimit    int
		responses     map[string]response
	}{
		"rate limiting already enabled": {
			expectErr:  false,
			consumerID: "consumerID",
			routeID:    "routeID",
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/plugins"): {
					code: http.StatusOK,
					dataIface: map[string]interface{}{
						"data": []*klib.Plugin{
							{
								ID:   klib.String("rateLimitingID"),
								Name: klib.String(common.RateLimitingPlugin),
								Route: &klib.Route{
									ID: klib.String("routeID"),
								},
								Enabled: klib.Bool(true),
							},
						},
						"next": "null",
					},
				},
			},
		},
		"enable rate limiting": {
			expectErr:  false,
			consumerID: "consumerID",
			routeID:    "routeID",
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/plugins"): {
					code: http.StatusOK,
					dataIface: map[string]interface{}{
						"data": []*klib.Plugin{
							{
								ID:   klib.String("rateLimitingID"),
								Name: klib.String(common.RateLimitingPlugin),
								Route: &klib.Route{
									ID: klib.String("routeID"),
								},
								Enabled: klib.Bool(false),
							},
						},
						"next": "null",
					},
				},
				formatRequestKey(http.MethodPatch, "/routes/routeID/plugins/rateLimitingID"): {
					code: http.StatusOK,
					dataIface: &klib.Plugin{
						ID:   klib.String("rateLimitingID"),
						Name: klib.String(common.RateLimitingPlugin),
						Route: &klib.Route{
							ID: klib.String("routeID"),
						},
						Enabled: klib.Bool(true),
					},
				},
			},
		},
		"add rate limiting plugin": {
			expectErr:     false,
			consumerID:    "consumerID",
			routeID:       "routeID",
			quotaInterval: provisioning.Daily.String(),
			quotaLimit:    7,
			responses: map[string]response{
				formatRequestKey(http.MethodGet, "/plugins"): {
					code: http.StatusOK,
					dataIface: map[string]interface{}{
						"data": []*klib.Plugin{},
						"next": "null",
					},
				},
				formatRequestKey(http.MethodPost, "/routes/routeID/plugins"): {
					code: http.StatusOK,
					dataIface: &klib.Plugin{
						ID:   klib.String("rateLimitingID"),
						Name: klib.String(common.RateLimitingPlugin),
						Route: &klib.Route{
							ID: klib.String("routeID"),
						},
						Config: klib.Configuration{
							"policy": "local",
							"day":    7,
						},
					},
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client := createClient(tc.responses)
			err := client.AddQuota(context.TODO(), tc.routeID, tc.consumerID, tc.quotaInterval, tc.quotaLimit)
			if tc.expectErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
		})
	}
}
