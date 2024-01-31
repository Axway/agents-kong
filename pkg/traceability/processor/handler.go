package processor

import (
	"context"
	"encoding/json"

	"github.com/Axway/agent-sdk/pkg/transaction"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/google/uuid"
)

// EventsHandler -
type EventsHandler struct {
	ctx             context.Context
	logger          log.FieldLogger
	metrics         MetricsProcessor
	logEntries      []TrafficLogEntry
	requestID       string
	eventGenerator  func() transaction.EventGenerator
	collectorGetter func() metricCollector
}

// NewEventsHandler - return a new EventProcessor
func NewEventsHandler(ctx context.Context, logData []byte) (*EventsHandler, error) {
	requestID := uuid.NewString()

	p := &EventsHandler{
		ctx:             ctx,
		logger:          log.NewLoggerFromContext(ctx).WithComponent("eventsHandler").WithPackage("processor").WithField(string(ctxRequestID), requestID),
		requestID:       requestID,
		metrics:         NewMetricsProcessor(ctx),
		eventGenerator:  transaction.NewEventGenerator,
		collectorGetter: getMetricCollector,
	}

	p.logger.WithField("inputData", string(logData)).Debug("data sent from kong")
	err := json.Unmarshal(logData, &p.logEntries)
	if err != nil {
		p.logger.WithError(err).Error("could not read log data")
		return nil, err
	}

	return p, nil
}

// Handle - processes the batch of events from the http request
func (p *EventsHandler) Handle() []beat.Event {
	events := make([]beat.Event, 0)
	p.logger.WithField("numEvents", len(p.logEntries)).Info("handling events in request")

	p.metrics.setCollector(p.collectorGetter())
	for i, entry := range p.logEntries {
		ctx := context.WithValue(p.ctx, ctxEntryIndex, i)

		if entry.Service == nil {
			// service into is nil, lets add service data so the transaction will be processed still
			entry.Service = &Service{
				Name:     "ErrorService",
				ID:       "ErrorServiceID",
				Port:     0,
				Protocol: "",
			}
		}
		if entry.Route == nil {
			entry.Route = &Route{
				Name: "ErrorRoute",
				ID:   "ErrorRouteID",
			}
		}

		sample, err := p.metrics.process(entry)
		if err != nil {
			p.logger.WithError(err).Error("handling event for metric")
			continue
		}
		if !sample {
			continue
		}

		// Map the log entry to log event structure expected by AMPLIFY Central Observer
		events = append(events, p.handleTransaction(ctx, entry)...)
	}

	return events
}

func (p *EventsHandler) handleTransaction(ctx context.Context, entry TrafficLogEntry) []beat.Event {
	log := p.logger.WithField(string(ctxEntryIndex), ctx.Value(ctxEntryIndex))

	newEvents, err := NewTransactionProcessor(ctx).setEventGenerator(p.eventGenerator()).setEntry(entry).process()
	if err != nil {
		log.WithError(err).Error("executing transaction processor")
		return []beat.Event{}
	}
	return newEvents
}
