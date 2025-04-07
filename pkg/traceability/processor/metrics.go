package processor

import (
	"context"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
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
func (m *MetricsProcessor) process(entry TrafficLogEntry) (TrafficLogEntry, error) {
	details := sampling.TransactionDetails{}
	if entry.Response != nil {
		details.Status = sdkUtil.GetTransactionSummaryStatus(entry.Response.Status)
	}
	if entry.Service != nil {
		details.APIID = entry.Service.ID
		if entry.Route != nil {
			details.APIID = fmt.Sprintf("%s-%s", entry.Service.ID, entry.Route.ID)
		}
	}
	if entry.Consumer != nil {
		details.SubID = entry.Consumer.ID
	}

	sample, err := sampling.ShouldSampleTransaction(details)
	if err != nil {
		return TrafficLogEntry{}, err
	}
	entry.ShouldSample = sample
	return entry, nil
}
