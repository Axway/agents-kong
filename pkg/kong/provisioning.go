package kong

import (
	"context"

	klib "github.com/kong/go-kong/kong"
)

func (k KongClient) CreateConsumer(ctx context.Context, id, name string) (*klib.Consumer, error) {
	// validate that the consumer does not already exist
	log := k.logger.WithField("consumerID", id).WithField("consumerName", name)
	consumer, err := k.Consumers.Get(ctx, klib.String(id))
	if err == nil {
		log.Debug("found existing consumer")
		return consumer, err
	}

	log.Debug("creating new consumer")
	return k.Consumers.Create(ctx, &klib.Consumer{
		CustomID: klib.String(id),
		Username: klib.String(name),
	})
}

func (k KongClient) DeleteConsumer(ctx context.Context, id string) error {
	// validate that the consumer has not already been removed
	log := k.logger.WithField("consumerID", id)
	_, err := k.Consumers.Get(ctx, klib.String(id))
	if err != nil {
		log.Debug("could not get consumer")
		return nil
	}

	log.Debug("deleting consumer")
	return k.Consumers.Delete(ctx, klib.String(id))
}
