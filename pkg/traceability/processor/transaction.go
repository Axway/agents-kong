package processor

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/google/uuid"

	"github.com/Axway/agent-sdk/pkg/transaction"
	sdkUtil "github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	host      = "host"
	userAgent = "user-agent"
	leg0      = "leg0"
	inbound   = "inbound"
)

// TransactionProcessor -
type TransactionProcessor struct {
	ctx            context.Context
	logger         log.FieldLogger
	eventGenerator transaction.EventGenerator
	event          TrafficLogEntry
}

func NewTransactionProcessor(ctx context.Context) *TransactionProcessor {
	p := &TransactionProcessor{
		ctx:    ctx,
		logger: log.NewLoggerFromContext(ctx).WithComponent("eventMapper").WithPackage("processor"),
	}
	return p
}

func (p *TransactionProcessor) setEntry(entry TrafficLogEntry) *TransactionProcessor {
	p.event = entry
	return p
}

func (p *TransactionProcessor) setEventGenerator(eventGenerator transaction.EventGenerator) *TransactionProcessor {
	p.eventGenerator = eventGenerator
	p.eventGenerator.SetUseTrafficForAggregation(false)
	return p
}

func (p *TransactionProcessor) process() ([]beat.Event, error) {
	if p.eventGenerator == nil {
		return nil, fmt.Errorf("an event generator is required")
	}
	txnID := uuid.New().String()

	// leg 0
	transactionLogEvent, err := createTransactionEvent(p.event, txnID)
	if err != nil {
		p.logger.WithError(err).Error("building transaction leg event")
		return nil, err
	}
	legEvent, err := p.eventGenerator.CreateEvent(*transactionLogEvent, time.Unix(p.event.StartedAt, 0), nil, nil, nil)
	if err != nil {
		p.logger.WithError(err).Error("creating transaction leg event")
		return nil, err
	}

	// summary
	summaryLogEvent, err := createSummaryEvent(p.event, "id", txnID)
	if err != nil {
		p.logger.WithError(err).Error("building transaction summary event")
		return nil, err
	}
	summaryEvent, err := p.eventGenerator.CreateEvent(*summaryLogEvent, time.Unix(p.event.StartedAt, 0), nil, nil, nil)
	if err != nil {
		p.logger.WithError(err).Error("creating transaction summary event")
		return nil, err
	}

	return []beat.Event{summaryEvent, legEvent}, nil
}

func createTransactionEvent(ktle TrafficLogEntry, txnid string) (*transaction.LogEvent, error) {
	httpProtocolDetails, err := transaction.NewHTTPProtocolBuilder().
		SetURI(ktle.Request.URI).
		SetMethod(ktle.Request.Method).
		SetArgs(processQueryArgs(ktle.Request.QueryString)).
		SetStatus(ktle.Response.Status, http.StatusText(ktle.Response.Status)).
		SetHost(ktle.Request.Headers[host]).
		SetHeaders(buildHeaders(ktle.Request.Headers), buildHeaders(ktle.Response.Headers)).
		SetByteLength(ktle.Request.Size, ktle.Response.Size).
		SetLocalAddress(ktle.ClientIP, 0). // Could not determine local port for now
		SetRemoteAddress("", "", ktle.Service.Port).
		SetSSLProperties(buildSSLInfoIfAvailable(ktle)).
		SetUserAgent(ktle.Request.Headers[userAgent]).
		Build()

	if err != nil {
		return nil, err
	}

	return transaction.NewTransactionEventBuilder().
		SetTimestamp(ktle.StartedAt).
		SetTransactionID(txnid).
		SetID(leg0).
		SetSource(ktle.ClientIP).
		SetDestination(ktle.Request.Headers[host]).
		SetDuration(ktle.Latencies.Request).
		SetDirection(inbound).
		SetStatus(getTransactionEventStatus(ktle.Response.Status)).
		SetProtocolDetail(httpProtocolDetails).
		Build()
}

func createSummaryEvent(ktle TrafficLogEntry, teamID string, txnid string) (*transaction.LogEvent, error) {
	builder := transaction.NewTransactionSummaryBuilder().
		SetTimestamp(ktle.StartedAt).
		SetTransactionID(txnid).
		SetStatus(getTransactionSummaryStatus(ktle.Response.Status), strconv.Itoa(ktle.Response.Status)).
		SetTeam(teamID).
		SetEntryPoint(ktle.Service.Protocol, ktle.Request.Method, ktle.Request.URI, ktle.Request.URL).
		SetDuration(ktle.Latencies.Request).
		// TODO: APIGOV-26720 - service ID should be the API ID and Route should be the Stage
		SetProxy(sdkUtil.FormatProxyID(ktle.Route.ID), ktle.Service.Name, 1)

	if ktle.Consumer != nil {
		builder.SetApplication(sdkUtil.FormatApplicationID(ktle.Consumer.ID), ktle.Consumer.Username)
	}

	return builder.Build()
}
