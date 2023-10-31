package processor

import (
	"bytes"
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

// EventMapper -
type EventMapper struct {
}

const (
	host          = "host"
	userAgent     = "user-agent"
	redactedValue = "****************************"
)

func (m *EventMapper) processMapping(kongTrafficLogEntry KongTrafficLogEntry) ([]*transaction.LogEvent, error) {
	centralCfg := agent.GetCentralConfig()
	txnID := uuid.New().String()

	transactionLegEvent, err := m.createTransactionEvent(kongTrafficLogEntry, txnID)
	if err != nil {
		log.Errorf("Error while building transaction leg event: %s", err)
		return nil, err
	}

	jTransactionLegEvent, err := json.Marshal(transactionLegEvent)
	if err != nil {
		log.Errorf("Failed to serialize transaction leg event as json: %s", err)
	}

	log.Debug("Generated Transaction leg event: ", string(jTransactionLegEvent))

	transSummaryLogEvent, err := m.createSummaryEvent(kongTrafficLogEntry, centralCfg.GetTeamID(), txnID)
	if err != nil {
		log.Errorf("Error while building transaction summary event: %s", err)
		return nil, err
	}

	jTransactionSummary, err := json.Marshal(transSummaryLogEvent)
	if err != nil {
		log.Errorf("Failed to serialize transaction summary as json: %s", err)
	}

	log.Debug("Generated Transaction summary event: ", string(jTransactionSummary))

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
	if headers["apikey"] != "" {
		headers["apikey"] = redactedValue
	}

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
		SetID("leg0").
		SetParentID("").
		SetSource(ktle.ClientIP).
		SetDestination(ktle.Request.Headers[host]).
		SetDuration(ktle.Latencies.Request).
		SetDirection("inbound").
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
