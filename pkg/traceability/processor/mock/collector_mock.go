package mock

import (
	"fmt"
	"sync"

	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
)

var collector *CollectorMock

func GetMockCollector() metric.Collector {
	return collector
}

func SetMockCollector(c *CollectorMock) {
	collector = c
}

type CollectorMock struct {
	sync.WaitGroup
	Details []metric.Detail
}

func (c *CollectorMock) AddMetric(apiDetails models.APIDetails, statusCode string, duration, bytes int64, appName string) {
}
func (c *CollectorMock) AddMetricDetail(metricDetail metric.Detail) {
	fmt.Printf("%v\n", metricDetail)
	c.Details = append(c.Details, metricDetail)
	c.Done()
}
func (c *CollectorMock) AddAPIMetric(apiMetric *metric.APIMetric) {}
func (c *CollectorMock) Publish()                                 {}
