package mock

import (
	"sync"

	"github.com/Axway/agent-sdk/pkg/transaction/metric"
)

var collector *CollectorMock

func GetMockCollector() *CollectorMock {
	return collector
}

func SetMockCollector(c *CollectorMock) {
	collector = c
}

type CollectorMock struct {
	sync.WaitGroup
	Details  []metric.Detail
	Expected int
}

func (c *CollectorMock) AddMetricDetail(metricDetail metric.Detail) {
	c.Details = append(c.Details, metricDetail)
	c.Done()
}
