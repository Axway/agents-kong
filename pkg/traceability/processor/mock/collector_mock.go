package mock

import (
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
)

var collector *CollectorMock

func init() {
	collector = &CollectorMock{Details: make([]metric.Detail, 0)}
}

func GetMockCollector() metric.Collector {
	return collector
}

type CollectorMock struct {
	Details []metric.Detail
}

func (c *CollectorMock) AddMetric(apiDetails models.APIDetails, statusCode string, duration, bytes int64, appName string) {
}
func (c *CollectorMock) AddMetricDetail(metricDetail metric.Detail) {
	c.Details = append(c.Details, metricDetail)
}
func (c *CollectorMock) AddAPIMetric(apiMetric *metric.APIMetric) {}
func (c *CollectorMock) Publish()                                 {}
