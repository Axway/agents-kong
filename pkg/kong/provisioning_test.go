package kong

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	klib "github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"

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
	cfg := &config.KongGatewayConfig{
		Admin: config.KongAdminConfig{
			URL: s.URL,
		},
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
