package kong

import (
	"context"

	klib "github.com/kong/go-kong/kong"
)

func (k KongClient) CreateConsumer(ctx context.Context, id, name string) (*klib.Consumer, error) {
	// validate that the consumer does not already exist
	log := k.logger.WithField("consumerID", id).WithField("consumerName", name)
	consumer, err := k.Consumers.Get(ctx, &id)
	if err == nil {
		log.Infof("found existing consumer")
		return consumer, err
	}

	log.Infof("creating new application")
	return k.Consumers.Create(ctx, &klib.Consumer{
		CustomID: &id,
		Username: &name,
	})
}
