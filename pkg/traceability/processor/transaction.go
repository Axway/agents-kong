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
	eventGenerator eventGenerator
	event          TrafficLogEntry
}

func NewTransactionProcessor(ctx context.Context) *TransactionProcessor {
	p := &TransactionProcessor{
		ctx: ctx,
		logger: log.NewLoggerFromContext(ctx).WithComponent("eventMapper").WithPackage("processor").
			WithField(string(ctxEntryIndex), ctx.Value(ctxEntryIndex)).
			WithField(string(ctxRequestID), ctx.Value(ctxRequestID)),
	}
	return p
}

func (p *TransactionProcessor) setEntry(entry TrafficLogEntry) *TransactionProcessor {
	p.event = entry
	return p
}

func (p *TransactionProcessor) setEventGenerator(eventGenerator eventGenerator) *TransactionProcessor {
	p.eventGenerator = eventGenerator
	return p
}

func (p *TransactionProcessor) process() ([]beat.Event, error) {
	if p.eventGenerator == nil {
		return nil, fmt.Errorf("an event generator is required")
	}
	txnID := uuid.New().String()

	builder := transaction.NewEventReportBuilder()

	transactionLogEvent, err := p.createTransactionEvent(txnID)
	if err != nil {
		p.logger.WithError(err).Error("building transaction leg event")
		return nil, err
	}

	// summary
	summaryLogEvent, err := p.createSummaryEvent("id", txnID)
	if err != nil {
		p.logger.WithError(err).Error("building transaction summary event")
		return nil, err
	}

	report, err := builder.SetSummaryEvent(*summaryLogEvent).
		SetDetailEvents([]transaction.LogEvent{*transactionLogEvent}).
		SetEventTime(time.Unix(p.event.StartedAt, 0)).
		SetSkipSampleHandling().
		SetOnlyTrackMetrics(!p.event.ShouldSample).
		Build()

	if err != nil {
		return nil, err
	}

	beatEvents, err := p.eventGenerator.CreateFromEventReport(report)
	if err != nil {
		return nil, err
	}

	return beatEvents, nil
}

func (p *TransactionProcessor) createTransactionEvent(txnid string) (*transaction.LogEvent, error) {
	if !p.event.ShouldSample {
		return nil, nil
	}

	requestHost := ""
	if value, found := p.event.Request.Headers[host]; found {
		requestHost = fmt.Sprintf("%v", value)
	}

	userAgentVal := ""
	if value, found := p.event.Request.Headers[userAgent]; found {
		userAgentVal = fmt.Sprintf("%v", value)
	}

	httpProtocolDetails, err := transaction.NewHTTPProtocolBuilder().
		SetURI(p.event.Request.URI).
		SetMethod(p.event.Request.Method).
		SetArgsMap(processQueryArgs(p.event.Request.QueryString)).
		SetStatus(p.event.Response.Status, http.StatusText(p.event.Response.Status)).
		SetHost(requestHost).
		SetHeaders(buildHeaders(p.event.Request.Headers), buildHeaders(p.event.Response.Headers)).
		SetByteLength(p.event.Request.Size, p.event.Response.Size).
		SetLocalAddress(p.event.ClientIP, 0). // Could not determine local port for now
		SetRemoteAddress("", "", p.event.Service.Port).
		SetSSLProperties(buildSSLInfoIfAvailable(p.event)).
		SetUserAgent(userAgentVal).
		Build()

	if err != nil {
		return nil, err
	}

	return transaction.NewTransactionEventBuilder().
		SetTimestamp(p.event.StartedAt).
		SetTransactionID(txnid).
		SetID(leg0).
		SetSource(p.event.ClientIP).
		SetDestination(requestHost).
		SetDuration(p.event.Latencies.Request).
		SetDirection(inbound).
		SetStatus(getTransactionEventStatus(p.event.Response.Status)).
		SetProtocolDetail(httpProtocolDetails).
		Build()
}

func (p *TransactionProcessor) createSummaryEvent(teamID string, txnid string) (*transaction.LogEvent, error) {
	builder := transaction.NewTransactionSummaryBuilder().
		SetTimestamp(p.event.StartedAt).
		SetTransactionID(txnid).
		SetStatus(getTransactionSummaryStatus(p.event.Response.Status), strconv.Itoa(p.event.Response.Status)).
		SetTeam(teamID).
		SetEntryPoint(p.event.Service.Protocol, p.event.Request.Method, p.event.Request.URI, p.event.Request.URL).
		SetDuration(p.event.Latencies.Request).
		SetProxyWithStage(sdkUtil.FormatProxyID(p.event.Service.ID), p.event.Service.Name, p.event.Route.ID, 1)

	if p.event.Consumer != nil {
		builder.SetApplication(sdkUtil.FormatApplicationID(p.event.Consumer.ID), p.event.Consumer.Username)
	}

	return builder.Build()
}
