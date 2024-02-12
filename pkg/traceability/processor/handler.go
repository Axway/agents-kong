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

		// skip any entry that does not have request or response info
		if entry.Request == nil || entry.Response == nil {
			continue
		}

		updatedEntry := p.validateTransaction(ctx, entry)
		sample, err := p.metrics.process(updatedEntry)
		if err != nil {
			p.logger.WithError(err).Error("handling event for metric")
			continue
		}
		if !sample {
			continue
		}

		// Map the log entry to log event structure expected by AMPLIFY Central Observer
		events = append(events, p.handleTransaction(ctx, updatedEntry)...)
	}

	return events
}

func (p *EventsHandler) validateTransaction(ctx context.Context, entry TrafficLogEntry) TrafficLogEntry {
	logger := log.UpdateLoggerWithContext(ctx, p.logger)

	logger.Trace("checking if any entry objects are nil")

	if entry.Service == nil {
		// service into is nil, lets add service data so the transaction will be processed still
		logger.Debug("entry service details were nil, adding ErrorService info")
		entry.Service = &Service{
			Name: "ErrorService",
			ID:   "ErrorServiceID",
		}
	}

	if entry.Route == nil {
		logger.Debug("entry route details were nil, adding ErrorRoute info")
		entry.Route = &Route{
			Name: "ErrorRoute",
			ID:   "ErrorRouteID",
		}
	}

	if entry.Latencies == nil {
		logger.Debug("entry latencies details were nil, adding empty latency info")
		entry.Latencies = &Latencies{}
	}

	return entry
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
