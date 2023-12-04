package processor

import (
	"context"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/Axway/agent-sdk/pkg/transaction/util"
	sdkUtil "github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// MetricsProcessor -
type MetricsProcessor struct {
	ctx       context.Context
	logger    log.FieldLogger
	collector metricCollector
}

func NewMetricsProcessor(ctx context.Context) MetricsProcessor {
	return MetricsProcessor{
		ctx:    ctx,
		logger: log.NewLoggerFromContext(ctx).WithComponent("eventMapper").WithPackage("processor"),
	}
}

func (m *MetricsProcessor) setCollector(collector metricCollector) {
	m.collector = collector
}

// process - receives the log event and returns if the transaction should be sampled
func (m *MetricsProcessor) process(entry TrafficLogEntry) (bool, error) {
	details := sampling.TransactionDetails{}
	if entry.Response != nil {
		details.Status = util.GetTransactionSummaryStatus(entry.Response.Status)
	}
	if entry.Service != nil {
		details.APIID = entry.Route.ID
	}
	if entry.Consumer != nil {
		details.SubID = entry.Consumer.ID
	}

	sample, err := sampling.ShouldSampleTransaction(details)
	if err != nil {
		return false, err
	}
	m.updateMetric(entry)

	return sample, nil
}

func (m *MetricsProcessor) updateMetric(entry TrafficLogEntry) {
	apiDetails := models.APIDetails{
		ID:    entry.Service.Name,
		Name:  entry.Service.Name,
		Stage: entry.Route.Name,
	}

	statusCode := entry.Response.Status
	duration := entry.Latencies.Request
	appDetails := models.AppDetails{}
	if entry.Consumer != nil {
		appDetails.Name = entry.Consumer.Username
		appDetails.ID = sdkUtil.FormatApplicationID(entry.Consumer.ID)
	}

	if m.collector != nil {
		metricDetail := metric.Detail{
			APIDetails: apiDetails,
			StatusCode: fmt.Sprint(statusCode),
			Duration:   int64(duration),
			Bytes:      int64(entry.Request.Size),
			AppDetails: appDetails,
		}
		m.collector.AddMetricDetail(metricDetail)
	}
}
