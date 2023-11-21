package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/google/uuid"

	"github.com/Axway/agent-sdk/pkg/agent"
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

func NewTransactionProcessor(ctx context.Context, entry TrafficLogEntry) *TransactionProcessor {
	p := &TransactionProcessor{
		ctx:            ctx,
		logger:         log.NewLoggerFromContext(ctx).WithComponent("eventMapper").WithPackage("processor"),
		eventGenerator: transaction.NewEventGenerator(),
		event:          entry,
	}

	p.eventGenerator.SetUseTrafficForAggregation(false)

	return p
}

func (m *TransactionProcessor) process() ([]beat.Event, error) {
	centralCfg := agent.GetCentralConfig()
	txnID := uuid.New().String()

	// leg 0
	transactionLogEvent, err := m.createTransactionEvent(m.event, txnID)
	if err != nil {
		m.logger.WithError(err).Error("building transaction leg event")
		return nil, err
	}
	legEvent, err := m.eventGenerator.CreateEvent(*transactionLogEvent, time.Unix(m.event.StartedAt, 0), nil, nil, nil)
	if err != nil {
		m.logger.WithError(err).Error("creating transaction leg event")
		return nil, err
	}

	// summary
	summaryLogEvent, err := m.createSummaryEvent(m.event, centralCfg.GetTeamID(), txnID)
	if err != nil {
		m.logger.WithError(err).Error("building transaction summary event")
		return nil, err
	}
	summaryEvent, err := m.eventGenerator.CreateEvent(*summaryLogEvent, time.Unix(m.event.StartedAt, 0), nil, nil, nil)
	if err != nil {
		m.logger.WithError(err).Error("creating transaction summary event")
		return nil, err
	}

	return []beat.Event{summaryEvent, legEvent}, nil
}

func (m *TransactionProcessor) getTransactionEventStatus(code int) transaction.TxEventStatus {
	if code >= 400 {
		return transaction.TxEventStatusFail
	}
	return transaction.TxEventStatusPass
}

func (m *TransactionProcessor) getTransactionSummaryStatus(statusCode int) transaction.TxSummaryStatus {
	transSummaryStatus := transaction.TxSummaryStatusUnknown
	if statusCode >= http.StatusOK && statusCode < http.StatusBadRequest {
		transSummaryStatus = transaction.TxSummaryStatusSuccess
	} else if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
		transSummaryStatus = transaction.TxSummaryStatusFailure
	} else if statusCode >= http.StatusInternalServerError && statusCode < http.StatusNetworkAuthenticationRequired {
		transSummaryStatus = transaction.TxSummaryStatusException
	}
	return transSummaryStatus
}

func (m *TransactionProcessor) buildHeaders(headers map[string]string) string {
	jsonHeader, err := json.Marshal(headers)
	if err != nil {
		log.Error(err.Error())
	}

	return string(jsonHeader)
}

func (m *TransactionProcessor) buildSSLInfoIfAvailable(ktle TrafficLogEntry) (string, string, string) {
	if ktle.Request.TLS != nil {
		return ktle.Request.TLS.Version,
			ktle.Request.URL,
			ktle.Request.URL // Using SSL server name as SSL subject name for now
	}
	return "", "", ""
}

func (m *TransactionProcessor) processQueryArgs(args map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range args {
		fmt.Fprintf(b, "%s=\"%s\",", key, value)
	}
	return b.String()
}

func (m *TransactionProcessor) createTransactionEvent(ktle TrafficLogEntry, txnid string) (*transaction.LogEvent, error) {

	httpProtocolDetails, err := transaction.NewHTTPProtocolBuilder().
		SetURI(ktle.Request.URI).
		SetMethod(ktle.Request.Method).
		SetArgs(m.processQueryArgs(ktle.Request.QueryString)).
		SetStatus(ktle.Response.Status, http.StatusText(ktle.Response.Status)).
		SetHost(ktle.Request.Headers[host]).
		SetHeaders(m.buildHeaders(ktle.Request.Headers), m.buildHeaders(ktle.Response.Headers)).
		SetByteLength(ktle.Request.Size, ktle.Response.Size).
		SetLocalAddress(ktle.ClientIP, 0). // Could not determine local port for now
		SetRemoteAddress("", "", ktle.Service.Port).
		SetSSLProperties(m.buildSSLInfoIfAvailable(ktle)).
		SetUserAgent(ktle.Request.Headers[userAgent]).
		Build()

	if err != nil {
		log.Errorf("Error while filling protocol details for transaction event: %s", err)
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
		SetStatus(m.getTransactionEventStatus(ktle.Response.Status)).
		SetProtocolDetail(httpProtocolDetails).
		Build()
}

func (m *TransactionProcessor) createSummaryEvent(ktle TrafficLogEntry, teamID string, txnid string) (*transaction.LogEvent, error) {

	builder := transaction.NewTransactionSummaryBuilder().
		SetTimestamp(ktle.StartedAt).
		SetTransactionID(txnid).
		SetStatus(m.getTransactionSummaryStatus(ktle.Response.Status), strconv.Itoa(ktle.Response.Status)).
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
