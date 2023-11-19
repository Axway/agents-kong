package processor

import (
	"context"
	"encoding/json"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/beat"
)

// EventsHandler -
type EventsHandler struct {
	ctx        context.Context
	logger     log.FieldLogger
	logEntries []TrafficLogEntry
}

// NewEventsHandler - return a new EventProcessor
func NewEventsHandler(ctx context.Context, logData []byte) (*EventsHandler, error) {
	p := &EventsHandler{
		ctx:    ctx,
		logger: log.NewLoggerFromContext(ctx).WithComponent("eventsHandler").WithPackage("processor"),
	}

	err := json.Unmarshal(logData, &p.logEntries)
	if err != nil {
		p.logger.WithError(err).Error("could not read log data")
		return nil, err
	}

	return p, nil
}

// {\"client_ip\":\"10.129.216.201\",\"started_at\":1700260849296,\"upstream_uri\":\"/api/v3/store/inventory\",\"latencies\":{\"request\":247,\"kong\":9,\"proxy\":238},\"request\":{\"querystring\":{},\"size\":100,\"uri\":\"/petstore3/store/inventory\",\"url\":\"https://sl3rdapp090303.pcloud.axway.int:8443/petstore3/store/inventory\",\"headers\":{\"accept\":\"*/*\",\"authorization\":\"REDACTED\",\"host\":\"sl3rdapp090303.pcloud.axway.int:8443\",\"user-agent\":\"curl/8.1.2\",\"x-consumer-custom-id\":\"8ac9978e8bdbe022018bdd2279f6039a\",\"x-consumer-groups\":\"0aeebe2d-93ae-42c2-a685-2f514eb2363b\",\"x-consumer-id\":\"0aeebe2d-93ae-42c2-a685-2f514eb2363b\",\"x-consumer-username\":\"kong-demo\",\"x-credential-identifier\":\"vC8H5w2scg8LS5qqBGUvhLVlCQGp4nfL\"},\"method\":\"GET\",\"tls\":{\"version\":\"TLSv1.3\",\"cipher\":\"TLS_AES_256_GCM_SHA384\",\"supported_client_ciphers\":\"\",\"client_verify\":\"NONE\"}},\"response\":{\"headers\":{\"access-control-allow-headers\":\"Content-Type, api_key, Authorization\",\"access-control-allow-methods\":\"GET, POST, DELETE, PUT\",\"access-control-allow-origin\":\"*\",\"access-control-expose-headers\":\"Content-Disposition\",\"connection\":\"close\",\"content-length\":\"110\",\"content-type\":\"application/json\",\"date\":\"Fri, 17 Nov 2023 22:41:03 GMT\",\"server\":\"Jetty(9.4.9.v20180320)\",\"via\":\"kong/3.4.1.0-enterprise-edition\",\"x-kong-proxy-latency\":\"9\",\"x-kong-upstream-latency\":\"238\"},\"status\":500,\"size\":422},\"route\":{\"id\":\"c63d3f96-e178-454f-8b70-4efc9bfef5e9\",\"updated_at\":1698677184,\"protocols\":[\"http\",\"https\"],\"strip_path\":true,\"created_at\":1698333843,\"ws_id\":\"973db277-ffd7-4db0-a837-d35edc3690cc\",\"service\":{\"id\":\"069c7945-07ae-4197-a6cb-a5c1ae968d73\"},\"name\":\"PetStoreRoot\",\"hosts\":null,\"preserve_host\":false,\"regex_priority\":0,\"paths\":[\"/petstore3\"],\"response_buffering\":true,\"https_redirect_status_code\":426,\"path_handling\":\"v0\",\"request_buffering\":true},\"service\":{\"host\":\"petstore3.swagger.io\",\"created_at\":1698333790,\"connect_timeout\":60000,\"id\":\"069c7945-07ae-4197-a6cb-a5c1ae968d73\",\"protocol\":\"https\",\"name\":\"PetStore\",\"read_timeout\":60000,\"port\":443,\"path\":\"/api/v3\",\"updated_at\":1698399310,\"write_timeout\":60000,\"retries\":5,\"ws_id\":\"973db277-ffd7-4db0-a837-d35edc3690cc\"},\"consumer\":{\"custom_id\":\"8ac9978e8bdbe022018bdd2279f6039a\",\"created_at\":1700222094,\"id\":\"0aeebe2d-93ae-42c2-a685-2f514eb2363b\",\"tags\":null,\"username\":\"kong-demo\"}}

// Handle - processes the batch of events from the http request
func (p *EventsHandler) Handle() ([]beat.Event, error) {
	events := make([]beat.Event, 0)
	p.logger.WithField("numEvents", len(p.logEntries)).Info("handling events in request")

	for i, entry := range p.logEntries {
		log := p.logger.WithField(string(ctxEntryIndex), i)
		processor, _ := NewTransactionProcessor(context.WithValue(p.ctx, ctxEntryIndex, i), entry)

		// Map the log entry to log event structure expected by AMPLIFY Central Observer
		newEvents, err := processor.process()
		if err != nil {
			log.WithError(err).Error("creating event")
			continue
		}
		events = append(events, newEvents...)
	}
	return events, nil
}
