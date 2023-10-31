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
		log.Debug("found existing consumer")
		return consumer, err
	}

	log.Debug("creating new consumer")
	return k.Consumers.Create(ctx, &klib.Consumer{
		CustomID: &id,
		Username: &name,
	})
}

func (k KongClient) DeleteConsumer(ctx context.Context, id string) error {
	// validate that the consumer does not already exist
	log := k.logger.WithField("consumerID", id)
	consumer, err := k.Consumers.Get(ctx, &id)
	if err != nil {
		return err
	}
	if consumer == nil {
		log.Debug("consumer does not exist")
		return nil
	}

	log.Debug("deleting consumer")
	return k.Consumers.Delete(ctx, &id)
}
