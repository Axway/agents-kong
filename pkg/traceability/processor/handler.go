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
