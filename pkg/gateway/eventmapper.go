package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/transaction"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// EventMapper -
type EventMapper struct {
}

const requestID = "kong-request-id"
const host = "host"

func (m *EventMapper) processMapping(kongTrafficLogEntry KongTrafficLogEntry) ([]*transaction.LogEvent, error) {
	centralCfg := agent.GetCentralConfig()
	transactionLegEvent, err := m.createTransactionEvent(kongTrafficLogEntry)
	if err != nil {
		log.Errorf("Error while building transaction leg event: %s", err)
		return nil, err
	}

	transSummaryLogEvent, err := m.createSummaryEvent(kongTrafficLogEntry, centralCfg.GetTeamID())
	if err != nil {
		log.Errorf("Error while building transaction summary event: %s", err)
		return nil, err
	}

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

func (m *EventMapper) createTransactionEvent(ktle KongTrafficLogEntry) (*transaction.LogEvent, error) {

	httpProtocolDetails, err := transaction.NewHTTPProtocolBuilder().
		SetURI(ktle.Request.URI).
		SetMethod(ktle.Request.Method).
		SetStatus(ktle.Response.Status, http.StatusText(ktle.Response.Status)).
		SetHost(ktle.Request.URL).
		SetHeaders(m.buildHeaders(ktle.Request.Headers), m.buildHeaders(ktle.Response.Headers)).
		SetByteLength(ktle.Request.Size, ktle.Response.Size).
		SetRemoteAddress("", ktle.Request.Headers[host], 80).
		SetLocalAddress(ktle.ClientIP, ktle.Service.Port).
		Build()

	if err != nil {
		log.Errorf("Error while filling protocol details for transaction event: %s", err)
		return nil, err
	}

	return transaction.NewTransactionEventBuilder().
		SetTimestamp(ktle.StartedAt).
		SetTransactionID(ktle.Request.Headers[requestID]).
		SetID("leg0").
		SetParentID("null").
		SetSource("client_ip").
		SetDestination("backend_api").
		SetDirection("outbound").
		SetStatus(m.getTransactionEventStatus(ktle.Response.Status)).
		SetProtocolDetail(httpProtocolDetails).
		Build()
}

func (m *EventMapper) createSummaryEvent(ktle KongTrafficLogEntry, teamID string) (*transaction.LogEvent, error) {

	return transaction.NewTransactionSummaryBuilder().
		SetTimestamp(ktle.StartedAt).
		SetTransactionID(ktle.Request.Headers[requestID]).
		SetStatus(m.getTransactionSummaryStatus(ktle.Response.Status),
			strconv.Itoa(ktle.Response.Status)).
		SetTeam(teamID).
		SetEntryPoint(ktle.Service.Protocol,
			ktle.Request.Method,
			ktle.Request.URI,
			ktle.Request.URL).
		SetDuration(ktle.Latencies.Request).
		SetProxy(transaction.FormatProxyID(ktle.Service.ID),
			ktle.Service.Name,
			0).
		Build()
}
