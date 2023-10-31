package kong

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	klib "github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"

	config "github.com/Axway/agents-kong/pkg/config/discovery"
)

type response struct {
	code      int
	dataIface interface{}
	data      []byte
}

func createClient(responses map[string]map[string]response) KongAPIClient {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if pathRes, foundPath := responses[req.URL.Path]; foundPath {
			if res, found := pathRes[req.Method]; found {
				resp.WriteHeader(res.code)
				if res.dataIface != nil {
					data, _ := json.Marshal(res.dataIface)
					resp.Write(data)
				} else {
					resp.Write(res.data)
				}
				return
			}
		}
	}))
	cfg := &config.KongGatewayConfig{
		AdminEndpoint: s.URL,
	}
	client, _ := NewKongClient(&http.Client{}, cfg)
	return client
}

func TestCreateConsumer(t *testing.T) {
	testCases := map[string]struct {
		expectErr bool
		id        string
		name      string
		responses map[string]map[string]response
	}{
		"find existing consumer": {
			expectErr: false,
			id:        "existingID",
			name:      "existingName",
			responses: map[string]map[string]response{
				"/consumers/" + "existingID": {
					http.MethodGet: {
						code: http.StatusOK,
						dataIface: &klib.Consumer{
							ID:       klib.String("existingID"),
							Username: klib.String("existingName"),
						},
					},
				},
			},
		},
		"create new consumer": {
			expectErr: false,
			id:        "nameID",
			name:      "newName",
			responses: map[string]map[string]response{
				"/consumers/" + "nameID": {
					http.MethodGet: {
						code: http.StatusNotFound,
					},
				},
				"/consumers": {
					http.MethodPost: {
						code: http.StatusCreated,
						dataIface: &klib.Consumer{
							ID:       klib.String("nameID"),
							Username: klib.String("newName"),
						},
					},
				},
			},
		},
		"create new consumer error": {
			expectErr: true,
			id:        "nameID",
			name:      "newName",
			responses: map[string]map[string]response{
				"/consumers/" + "nameID": {
					http.MethodGet: {
						code: http.StatusNotFound,
					},
				},
				"/consumers": {
					http.MethodPost: {
						code: http.StatusBadRequest,
					},
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

func TestDeleteConsumer(t *testing.T) {
	testCases := map[string]struct {
		expectErr bool
		responses map[string]map[string]response
	}{
		"consumer does not exist": {
			expectErr: false,
			responses: map[string]map[string]response{
				"/consumers/" + "id": {
					http.MethodGet: {
						code: http.StatusNotFound,
					},
				},
			},
		},
		"delete consumer": {
			expectErr: false,
			responses: map[string]map[string]response{
				"/consumers/" + "id": {
					http.MethodGet: {
						code: http.StatusOK,
						dataIface: &klib.Consumer{
							ID:       klib.String("id"),
							Username: klib.String("name"),
						},
					},
					http.MethodDelete: {
						code: http.StatusAccepted,
					},
				},
			},
		},
		"delete consumer error": {
			expectErr: true,
			responses: map[string]map[string]response{
				"/consumers/" + "id": {
					http.MethodGet: {
						code: http.StatusOK,
						dataIface: &klib.Consumer{
							ID:       klib.String("id"),
							Username: klib.String("name"),
						},
					},
					http.MethodDelete: {
						code: http.StatusBadRequest,
					},
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
