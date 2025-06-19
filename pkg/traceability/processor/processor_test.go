package processor

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/agent"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/transaction"
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

var testErrorData = []byte(`[{
	"request": {"querystring": {"status": "available"},"size": 138,"uri": "/log","url": "http://localhost:8000/log","headers": {"host": "localhost:8000","accept-encoding": "gzip, deflate","user-agent": "HTTPie/2.4.0","accept": "*/*","connection": "keep-alive"},"method": "GET"},
	"response": {"headers": {"content-type": "application/json","date": "Thu, 25 Feb 2021 05:57:48 GMT","connection": "close","access-control-allow-credentials": "true","content-length": "503","server": "gunicorn/19.9.0","via": "kong/2.2.1.0-enterprise-edition","x-kong-proxy-latency": "57","x-kong-upstream-latency": "457","access-control-allow-origin": "*"},"status": 404,"size": 827},
	"tries": [{"balancer_latency": 0,"port": 80,"balancer_start": 1614232668399,"ip": "18.211.130.98"}],
	"client_ip": "192.168.144.1",
	"workspace": "54baa5a9-23d6-41e0-9c9a-02434b010b25",
	"workspace_name": "default",
	"upstream_uri": "/anything",
	"authenticated_entity": {"id": "c62c1455-9b1d-4f2d-8797-509ba83b8ae8"},
	"consumer": {"id": "ae974d6c-0f8a-4dc5-b701-fa0aa38592bd","created_at": 1674035962,"username_lower": "foo","username": "foo","type": 0},
	"started_at": 1614232668342
}]`)

var testNoRequestData = []byte(`[{
	"response": {"headers": {"content-type": "application/json","date": "Thu, 25 Feb 2021 05:57:48 GMT","connection": "close","access-control-allow-credentials": "true","content-length": "503","server": "gunicorn/19.9.0","via": "kong/2.2.1.0-enterprise-edition","x-kong-proxy-latency": "57","x-kong-upstream-latency": "457","access-control-allow-origin": "*"},"status": 404,"size": 827},
	"tries": [{"balancer_latency": 0,"port": 80,"balancer_start": 1614232668399,"ip": "18.211.130.98"}],
	"client_ip": "192.168.144.1",
	"workspace": "54baa5a9-23d6-41e0-9c9a-02434b010b25",
	"workspace_name": "default",
	"upstream_uri": "/anything",
	"authenticated_entity": {"id": "c62c1455-9b1d-4f2d-8797-509ba83b8ae8"},
	"consumer": {"id": "ae974d6c-0f8a-4dc5-b701-fa0aa38592bd","created_at": 1674035962,"username_lower": "foo","username": "foo","type": 0},
	"started_at": 1614232668342
}]`)

func TestNewHandler(t *testing.T) {
	sampling.SetupSampling(sampling.DefaultConfig(), false, "")
	sampling.GetGlobalSampling().EnableSampling(100, time.Now().Add(10*time.Second), map[string]management.TraceabilityAgentAgentstateSamplingEndpoints{})

	cases := map[string]struct {
		data                  []byte
		constructorErr        bool
		expectedEvents        int
		expectedMetricDetails int
		hasErrors             bool
	}{
		"expect error creating handler, when no data sent into handler": {
			data:           []byte{},
			constructorErr: true,
		},
		"expect no error when empty array data sent into handler": {
			data: []byte("[]"),
		},
		"handle data with no request info": {
			data: testNoRequestData,
		},
		"handle data with no latency, service, or route info": {
			data:                  testErrorData,
			expectedEvents:        2,
			expectedMetricDetails: 1,
			hasErrors:             true,
		},
		"handle data with sampling setup": {
			data:                  testData,
			expectedEvents:        4,
			expectedMetricDetails: 2,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			agent.InitializeForTest(&mock.Client{}, agent.TestWithAgentType(config.TraceabilityAgent))
			ctx := context.WithValue(context.Background(), "test", name)

			redaction.SetupGlobalRedaction(redaction.DefaultConfig())

			// create the handler
			h, err := NewEventsHandler(ctx, tc.data)
			if tc.constructorErr {
				assert.NotNil(t, err)
				assert.Nil(t, h)
				return
			}
			assert.Nil(t, err)
			assert.NotNil(t, h)

			// setup event generator
			h.eventGenerator = newEventGeneratorMock
			// execute the handler
			events := h.Handle()
			assert.Nil(t, err)
			assert.Len(t, events, tc.expectedEvents)
		})
	}
}

// eventGeneratorMock - mock event generator
type eventGeneratorMock struct{}

// newEventGeneratorMock - Create a new mock event generator
func newEventGeneratorMock() eventGenerator {
	return &eventGeneratorMock{}
}

// CreateFromEventReport - create from event report
func (c *eventGeneratorMock) CreateFromEventReport(eventReport transaction.EventReport) ([]beat.Event, error) {
	serializedSumEvent, _ := json.Marshal(eventReport.GetSummaryEvent())
	sumEventData := make(map[string]interface{})
	sumEventData["message"] = string(serializedSumEvent)
	events := []beat.Event{
		{
			Timestamp: eventReport.GetEventTime(),
			Meta:      eventReport.GetMetadata(),
			Private:   eventReport.GetPrivateData(),
			Fields:    sumEventData,
		},
	}

	for _, detailEvent := range eventReport.GetDetailEvents() {
		serializedEvent, _ := json.Marshal(detailEvent)
		eventData := make(map[string]interface{})
		eventData["message"] = string(serializedEvent)
		events = append(events, beat.Event{
			Timestamp: eventReport.GetEventTime(),
			Meta:      eventReport.GetMetadata(),
			Private:   eventReport.GetPrivateData(),
			Fields:    eventData,
		})
	}

	return events, nil
}
