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
	consumer, err = k.Consumers.Create(ctx, &klib.Consumer{
		CustomID: klib.String(id),
		Username: klib.String(name),
	})
	if err != nil {
		log.WithError(err).Error("creating consumer")
		return nil, err
	}

	return consumer, nil
}

func (k KongClient) AddConsumerACL(ctx context.Context, id string) error {
	log := k.logger.WithField("consumerID", id)
	consumer, err := k.Consumers.Get(ctx, klib.String(id))
	if err != nil {
		log.Debug("could not find consumer")
		return err
	}

	log.Debug("adding consumer acl")
	_, err = k.ACLs.Create(ctx, consumer.ID, &klib.ACLGroup{
		Consumer: consumer,
		Group:    klib.String(id),
	})

	if err != nil {
		log.WithError(err).Error("adding acl to consumer")
		return err
	}
	return nil
}

func (k KongClient) DeleteConsumer(ctx context.Context, id string) error {
	// validate that the consumer has not already been removed
	log := k.logger.WithField("consumerID", id)
	_, err := k.Consumers.Get(ctx, klib.String(id))
	if err != nil {
		log.Debug("could not find consumer")
		return nil
	}

	log.Debug("deleting consumer")
	return k.Consumers.Delete(ctx, klib.String(id))
}
