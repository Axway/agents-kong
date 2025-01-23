package mock

import (
	"encoding/json"
	"time"

	"github.com/Axway/agent-sdk/pkg/transaction"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

// EventGeneratorMock - mock event generator
type EventGeneratorMock struct {
	shouldUseTrafficForAggregation bool
}

// NewEventGeneratorMock - Create a new mock event generator
func NewEventGeneratorMock() transaction.EventGenerator {
	return &EventGeneratorMock{}
}

// CreateEvent - Creates a new mocked event for tests
func (c *EventGeneratorMock) CreateEvent(logEvent transaction.LogEvent, eventTime time.Time, metaData, eventFields common.MapStr, privateData interface{}) (event beat.Event, err error) {
	serializedLogEvent, _ := json.Marshal(logEvent)
	eventData := make(map[string]interface{})
	eventData["message"] = string(serializedLogEvent)
	event = beat.Event{
		Timestamp: eventTime,
		Meta:      metaData,
		Private:   privateData,
		Fields:    eventData,
	}
	return
}

// CreateEvents - Creates a new mocked event for tests
func (c *EventGeneratorMock) CreateEvents(summaryEvent transaction.LogEvent, detailEvents []transaction.LogEvent, eventTime time.Time, metaData, eventFields common.MapStr, privateData interface{}) ([]beat.Event, error) {
	serializedSumEvent, _ := json.Marshal(summaryEvent)
	sumEventData := make(map[string]interface{})
	sumEventData["message"] = string(serializedSumEvent)
	events := []beat.Event{
		{
			Timestamp: eventTime,
			Meta:      metaData,
			Private:   privateData,
			Fields:    sumEventData,
		},
	}

	for _, detailEvent := range detailEvents {
		serializedEvent, _ := json.Marshal(detailEvent)
		eventData := make(map[string]interface{})
		eventData["message"] = string(serializedEvent)
		events = append(events, beat.Event{
			Timestamp: eventTime,
			Meta:      metaData,
			Private:   privateData,
			Fields:    eventData,
		})
	}

	return events, nil
}

// SetUseTrafficForAggregation - set the flag to use traffic events for aggregation.
func (c *EventGeneratorMock) SetUseTrafficForAggregation(useTrafficForAggregation bool) {
	c.shouldUseTrafficForAggregation = useTrafficForAggregation
}

// CreateFromEventReport - create from event report
func (c *EventGeneratorMock) CreateFromEventReport(eventReport transaction.EventReport) ([]beat.Event, error) {
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
