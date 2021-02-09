package kong

import (
	"context"
	"fmt"

	"github.com/kong/go-kong/kong"
)

type Consumers struct {
	*kong.ConsumerService
}

func (c Consumers) CreateOrUpdateConsumer(name string, customID string, tags []*string) (*kong.Consumer, bool, error) {
	ctx := context.TODO()

	consumerRes, err := c.Get(ctx, &name)
	if err == nil {
		return consumerRes, false, nil
	}

	consumerRes, err = c.Create(ctx, &kong.Consumer{
		CustomID: &customID,
		Username: &name,
		Tags:     tags,
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to create consumer: %w", err)
	}

	return consumerRes, true, nil
}
