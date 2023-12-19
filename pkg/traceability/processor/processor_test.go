package processor

import (
	"context"
	"sync"
	"testing"

	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/Axway/agents-kong/pkg/traceability/processor/mock"
	"github.com/stretchr/testify/assert"
)

var testData = []byte(`[{
	"service": {"host": "httpbin.org","created_at": 1614232642,"connect_timeout": 60000,"id": "167290ee-c682-4ebf-bdea-e49a3ac5e260","protocol": "http","read_timeout": 60000,"port": 80,"path": "/anything","updated_at": 1614232642,"write_timeout": 60000,"retries": 5,"ws_id": "54baa5a9-23d6-41e0-9c9a-02434b010b25"},
	"route": {"id": "78f79740-c410-4fd9-a998-d0a60a99dc9b","paths": ["/log"],"protocols": ["http"],"strip_path": true,"created_at": 1614232648,"ws_id": "54baa5a9-23d6-41e0-9c9a-02434b010b25","request_buffering": true,"updated_at": 1614232648,"preserve_host": false,"regex_priority": 0,"response_buffering": true,"https_redirect_status_code": 426,"path_handling": "v0","service": {"id": "167290ee-c682-4ebf-bdea-e49a3ac5e260"}},
	"request": {"querystring": {"status": "available"},"size": 138,"uri": "/log","url": "http://localhost:8000/log","headers": {"host": "localhost:8000","accept-encoding": "gzip, deflate","user-agent": "HTTPie/2.4.0","accept": "*/*","connection": "keep-alive"},"method": "GET"},
	"response": {"headers": {"content-type": "application/json","date": "Thu, 25 Feb 2021 05:57:48 GMT","connection": "close","access-control-allow-credentials": "true","content-length": "503","server": "gunicorn/19.9.0","via": "kong/2.2.1.0-enterprise-edition","x-kong-proxy-latency": "57","x-kong-upstream-latency": "457","access-control-allow-origin": "*"},"status": 200,"size": 827},
	"latencies": {"request": 515,"kong": 58,"proxy": 457},
	"tries": [{"balancer_latency": 0,"port": 80,"balancer_start": 1614232668399,"ip": "18.211.130.98"}],
	"client_ip": "192.168.144.1",
	"workspace": "54baa5a9-23d6-41e0-9c9a-02434b010b25",
	"workspace_name": "default",
	"upstream_uri": "/anything",
	"authenticated_entity": {"id": "c62c1455-9b1d-4f2d-8797-509ba83b8ae8"},
	"consumer": {"id": "ae974d6c-0f8a-4dc5-b701-fa0aa38592bd","created_at": 1674035962,"username_lower": "foo","username": "foo","type": 0},
	"started_at": 1614232668342
},{
	"service": {"host": "httpbin.org","created_at": 1614232642,"connect_timeout": 60000,"id": "167290ee-c682-4ebf-bdea-e49a3ac5e260","protocol": "http","read_timeout": 60000,"port": 80,"path": "/anything","updated_at": 1614232642,"write_timeout": 60000,"retries": 5,"ws_id": "54baa5a9-23d6-41e0-9c9a-02434b010b25"},
	"route": {"id": "78f79740-c410-4fd9-a998-d0a60a99dc9b","paths": ["/log"],"protocols": ["http"],"strip_path": true,"created_at": 1614232648,"ws_id": "54baa5a9-23d6-41e0-9c9a-02434b010b25","request_buffering": true,"updated_at": 1614232648,"preserve_host": false,"regex_priority": 0,"response_buffering": true,"https_redirect_status_code": 426,"path_handling": "v0","service": {"id": "167290ee-c682-4ebf-bdea-e49a3ac5e260"}},
	"request": {"querystring": {"status": "available"},"size": 138,"uri": "/log","url": "http://localhost:8000/log","headers": {"host": "localhost:8000","accept-encoding": "gzip, deflate","user-agent": "HTTPie/2.4.0","accept": "*/*","connection": "keep-alive"},"method": "GET"},
	"response": {"headers": {"content-type": "application/json","date": "Thu, 25 Feb 2021 05:57:48 GMT","connection": "close","access-control-allow-credentials": "true","content-length": "503","server": "gunicorn/19.9.0","via": "kong/2.2.1.0-enterprise-edition","x-kong-proxy-latency": "57","x-kong-upstream-latency": "457","access-control-allow-origin": "*"},"status": 200,"size": 827},
	"latencies": {"request": 515,"kong": 58,"proxy": 457},
	"tries": [{"balancer_latency": 0,"port": 80,"balancer_start": 1614232668399,"ip": "18.211.130.98"}],
	"client_ip": "192.168.144.1",
	"workspace_name": "default",
	"upstream_uri": "/anything",
	"authenticated_entity": {"id": "c62c1455-9b1d-4f2d-8797-509ba83b8ae8"},
	"consumer": {"id": "ae974d6c-0f8a-4dc5-b701-fa0aa38592bd","created_at": 1674035962,"username_lower": "foo","username": "foo","type": 0},
	"started_at": 1614232668342
}]`)

func TestNewHandler(t *testing.T) {
	testLock := sync.Mutex{}
	cases := map[string]struct {
		data                  []byte
		constructorErr        bool
		setupSampling         bool
		expectedEvents        int
		expectedMetricDetails int
	}{
		"expect error creating handler, when no data sent into handler": {
			data:           []byte{},
			constructorErr: true,
		},
		"expect no error when empty array data sent into handler": {
			data: []byte("[]"),
		},
		"handle data without sampling setup": {
			data: testData,
		},
		"handle data with sampling setup": {
			data:                  testData,
			setupSampling:         true,
			expectedEvents:        4,
			expectedMetricDetails: 2,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			testLock.Lock()
			defer testLock.Unlock()

			ctx := context.WithValue(context.Background(), "test", name)

			redaction.SetupGlobalRedaction(redaction.DefaultConfig())
			if tc.setupSampling {
				sampling.SetupSampling(sampling.DefaultConfig(), false)
			}

			// create the handler
			h, err := NewEventsHandler(ctx, tc.data)
			if tc.constructorErr {
				assert.NotNil(t, err)
				assert.Nil(t, h)
				return
			}
			assert.Nil(t, err)
			assert.NotNil(t, h)

			// setup collector
			collector := &mock.CollectorMock{Details: make([]metric.Detail, 0)}
			mock.SetMockCollector(collector)
			h.collectorGetter = func() metricCollector {
				return mock.GetMockCollector()
			}

			// setup event generator
			h.eventGenerator = mock.NewEventGeneratorMock

			// if metric details are expected
			if tc.expectedMetricDetails > 1 {
				collector.Add(tc.expectedMetricDetails)
			}

			// execute the handler
			events := h.Handle()
			collector.Wait()
			assert.Nil(t, err)
			assert.Len(t, events, tc.expectedEvents)
			assert.Equal(t, tc.expectedMetricDetails, len(mock.GetMockCollector().Details))
		})
	}
}
