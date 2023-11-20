package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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

// EventMapper -
type EventMapper struct {
	logger log.FieldLogger
}

func NewEventMapper() *EventMapper {
	return &EventMapper{
		logger: log.NewFieldLogger().WithComponent("eventMapper").WithPackage("processor"),
	}
}

func (m *EventMapper) processMapping(ctx context.Context, kongTrafficLogEntry KongTrafficLogEntry) ([]*transaction.LogEvent, error) {
	log := log.UpdateLoggerWithContext(ctx, m.logger)
	centralCfg := agent.GetCentralConfig()
	txnID := uuid.New().String()

	transactionLegEvent, err := m.createTransactionEvent(kongTrafficLogEntry, txnID)
	if err != nil {
		log.WithError(err).Error("building transaction leg event")
		return nil, err
	}

	jTransactionLegEvent, err := json.Marshal(transactionLegEvent)
	if err != nil {
		log.WithError(err).Error("serialize transaction leg event")
	}

	log.WithField("leg", string(jTransactionLegEvent)).Debug("generated transaction leg event")

	transSummaryLogEvent, err := m.createSummaryEvent(kongTrafficLogEntry, centralCfg.GetTeamID(), txnID)
	if err != nil {
		log.WithError(err).Error("building transaction summary event")
		return nil, err
	}

	jTransactionSummary, err := json.Marshal(transSummaryLogEvent)
	if err != nil {
		log.WithError(err).Error("serialize transaction summary event")
	}
	log.WithField("summary", string(jTransactionSummary)).Debug("generated transaction summary event")

	return []*transaction.LogEvent{
		transSummaryLogEvent,
		transactionLegEvent,
	}, nil
}

func (m *EventMapper) getTransactionEventStatus(code int) transaction.TxEventStatus {
	if code >= 400 {
		return transaction.TxEventStatusFail
	}
	return transaction.TxEventStatusPass
}

func (m *EventMapper) getTransactionSummaryStatus(statusCode int) transaction.TxSummaryStatus {
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

func (m *EventMapper) buildHeaders(headers map[string]string) string {
	jsonHeader, err := json.Marshal(headers)
	if err != nil {
		log.Error(err.Error())
	}

	return string(jsonHeader)
}

func (m *EventMapper) buildSSLInfoIfAvailable(ktle KongTrafficLogEntry) (string, string, string) {
	if ktle.Request.TLS != nil {
		return ktle.Request.TLS.Version,
			ktle.Request.URL,
			ktle.Request.URL // Using SSL server name as SSL subject name for now
	}
	return "", "", ""
}

func (m *EventMapper) processQueryArgs(args map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range args {
		fmt.Fprintf(b, "%s=\"%s\",", key, value)
	}
	return b.String()
}

func (m *EventMapper) createTransactionEvent(ktle KongTrafficLogEntry, txnid string) (*transaction.LogEvent, error) {

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

func (m *EventMapper) createSummaryEvent(ktle KongTrafficLogEntry, teamID string, txnid string) (*transaction.LogEvent, error) {

	builder := transaction.NewTransactionSummaryBuilder().
		SetTimestamp(ktle.StartedAt).
		SetTransactionID(txnid).
		SetStatus(m.getTransactionSummaryStatus(ktle.Response.Status),
			strconv.Itoa(ktle.Response.Status)).
		SetTeam(teamID).
		SetEntryPoint(ktle.Service.Protocol,
			ktle.Request.Method,
			ktle.Request.URI,
			ktle.Request.URL).
		SetDuration(ktle.Latencies.Request).
		SetProxy(sdkUtil.FormatProxyID(ktle.Route.ID),
			ktle.Service.Name,
			1)

	if ktle.Consumer != nil {
		builder.SetApplication(sdkUtil.FormatApplicationID(ktle.Consumer.ID), ktle.Consumer.Username)
	}

	return builder.Build()
}
