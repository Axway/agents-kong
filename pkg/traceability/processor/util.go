package processor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/transaction"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func getTransactionEventStatus(code int) transaction.TxEventStatus {
	if code >= 400 {
		return transaction.TxEventStatusFail
	}
	return transaction.TxEventStatusPass
}

func getTransactionSummaryStatus(statusCode int) transaction.TxSummaryStatus {
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

func buildHeaders(headers map[string]interface{}) string {
	newHeaders := make(map[string]string)
	for key, val := range headers {
		newHeaders[key] = fmt.Sprintf("%v", val)
	}

	jsonHeader, err := json.Marshal(newHeaders)
	if err != nil {
		log.Error(err.Error())
	}
	return string(jsonHeader)
}

func buildSSLInfoIfAvailable(ktle TrafficLogEntry) (string, string, string) {
	if ktle.Request.TLS != nil {
		return ktle.Request.TLS.Version,
			ktle.Request.URL,
			ktle.Request.URL // Using SSL server name as SSL subject name for now
	}
	return "", "", ""
}

func processQueryArgs(args map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range args {
		fmt.Fprintf(b, "%s=\"%s\",", key, value)
	}
	return b.String()
}

func getMetricCollector() metricCollector {
	return metric.GetMetricCollector()
}
