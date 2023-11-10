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

func (k KongClient) DeleteOauth2(ctx context.Context, consumerID, clientID string) error {
	if err := k.Oauth2Credentials.Delete(ctx, &consumerID, &clientID); err != nil {
		k.logger.Errorf("failed to delete oauth2 credential with clientID: %s for consumerID: %s. Reason: %w", clientID, consumerID, err)
		return err
	}
	return nil
}

func (k KongClient) DeleteHttpBasic(ctx context.Context, consumerID, username string) error {
	if err := k.BasicAuths.Delete(ctx, &consumerID, &username); err != nil {
		k.logger.Errorf("failed to delete http-basic credential for user: %s for consumerID %s. Reason: %w", username, consumerID, err)
		return err
	}
	return nil
}

func (k KongClient) DeleteAuthKey(ctx context.Context, consumerID, authKey string) error {
	if err := k.KeyAuths.Delete(ctx, &consumerID, &authKey); err != nil {
		k.logger.Errorf("failed to delete API Key: %s for consumerID %s. Reason: %w", authKey, consumerID, err)
		return err
	}
	return nil
}

func (k KongClient) CreateHttpBasic(ctx context.Context, consumerID string, basicAuth *klib.BasicAuth) (*klib.BasicAuth, error) {
	basicAuth, err := k.BasicAuths.Create(ctx, &consumerID, basicAuth)
	if err != nil {
		k.logger.Errorf("failed to create http-basic credential for consumerID %s. Reason: %w", consumerID, err)
		return nil, err
	}
	return basicAuth, nil
}

func (k KongClient) CreateOauth2(ctx context.Context, consumerID string, oauth2 *klib.Oauth2Credential) (*klib.Oauth2Credential, error) {
	oauth2, err := k.Oauth2Credentials.Create(ctx, &consumerID, oauth2)
	if err != nil {
		k.logger.Errorf("failed to create oauth2 credential for consumerID %s. Reason: %w", consumerID, err)
		return nil, err
	}
	return oauth2, nil
}

func (k KongClient) CreateAuthKey(ctx context.Context, consumerID string, keyAuth *klib.KeyAuth) (*klib.KeyAuth, error) {
	keyAuth, err := k.KeyAuths.Create(ctx, &consumerID, keyAuth)
	if err != nil {
		k.logger.Errorf("failed to create oauth2 credential for consumerID %s. Reason: %w", consumerID, err)
		return nil, err
	}
	return keyAuth, nil
}
