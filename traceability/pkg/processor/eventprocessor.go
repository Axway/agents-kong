package processor

import (
	"encoding/json"
	"time"

	"github.com/Axway/agent-sdk/pkg/transaction"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/beat"
)

// EventProcessor - represents the processor for received event to generate event(s) for AMPLIFY Central
// The event processing can be done either when the beat input receives the log entry or before the beat transport
// publishes the event to transport.
// When processing the received log entry on input, the log entry is mapped to structure expected for AMPLIFY Central Observer
// and then beat.Event is published to beat output that produces the event over the configured transport.
// When processing the log entry on output, the log entry is published to output as beat.Event. The output transport invokes
// the Process(events []publisher.Event) method which is set as output event processor. The Process() method processes the received
// log entry and performs the mapping to structure expected for AMPLIFY Central Observer. The method returns the converted Events to
// transport publisher which then produces the events over the transport.
type EventProcessor struct {
	eventGenerator transaction.EventGenerator
	eventMapper    *EventMapper
}

// NewEventProcessor - return a new EventProcessor
func NewEventProcessor() *EventProcessor {
	ep := &EventProcessor{
		eventGenerator: transaction.NewEventGenerator(),
		eventMapper:    &EventMapper{},
	}
	return ep
}

// ProcessRaw - process the received log entry and returns the event to be published to AMPLIFY ingestion service
func (p *EventProcessor) ProcessRaw(rawEventData []byte) []beat.Event {
	var kongTrafficLogEntry KongTrafficLogEntry
	err := json.Unmarshal(rawEventData, &kongTrafficLogEntry)
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	// Map the log entry to log event structure expected by AMPLIFY Central Observer
	logEvents, err := p.eventMapper.processMapping(kongTrafficLogEntry)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	events := make([]beat.Event, 0)
	for _, logEvent := range logEvents {
		// Generates the beat.Event with attributes by AMPLIFY ingestion service
		event, err := p.eventGenerator.CreateEvent(*logEvent, time.Now(), nil, nil, nil)
		if err != nil {
			log.Error(err.Error())
		} else {
			events = append(events, event)
		}
	}
	return events
}
