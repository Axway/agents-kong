package kong

import (
	"context"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agents-kong/pkg/common"
	kCred "github.com/Axway/agents-kong/pkg/subscription/credential"
	klib "github.com/kong/go-kong/kong"
)

const Name = provisioning.APIKeyARD

const (
	reqApiKey       = "kong-api-key"
	reqUsername     = "kong-username"
	reqClientID     = "client_id"
	reqClientSecret = "client_secret"
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

func (k KongClient) DeleteCredential(ctx context.Context, req kCred.CredRequest) provisioning.RequestStatus {
	k.logger.WithField("handler", "DeleteCredential").WithField("application", req.GetApplicationName())
	k.logger.Info("Deleting credential")

	consumerID := req.GetApplicationDetailsValue(common.AttrAppID)
	rs := provisioning.NewRequestStatusBuilder()

	credentialType := req.GetCredentialType()
	switch credentialType {
	case provisioning.APIKeyARD:
		{
			authKey := req.GetCredentialDetailsValue(reqApiKey)
			err := k.KeyAuths.Delete(ctx, &consumerID, &authKey)
			if err != nil {
				k.logger.Errorf("failed to delete API Key: %s for consumerID %s. Reason: %w", authKey, consumerID, err)
				return rs.Failed()
			}

			return rs.SetMessage("API Key successfully deleted.").Success()
		}
	case provisioning.BasicAuthARD:
		{
			username := req.GetCredentialDetailsValue(reqUsername)
			if err := k.KeyAuths.Delete(ctx, &consumerID, &username); err != nil {
				k.logger.Errorf("failed to delete http-basic credential for user: %s for consumerID %s. Reason: %w", username, consumerID, err)
				return rs.Failed()
			}
			return rs.SetMessage("Http-basic credential successfully deleted.").Success()
		}
	case "oauth2":
		{
			clientID := req.GetApplicationDetailsValue(reqClientID)
			if err := k.Oauth2Credentials.Delete(ctx, &consumerID, &clientID); err != nil {
				k.logger.Errorf("failed to delete oauth2 credential for consumerID %s. Reason: %w", consumerID, err)
				return rs.Failed()
			}
			return rs.SetMessage("OAuth2 credential successfully deleted.").Success()
		}
	}
	k.logger.Error("failed to identify credential type")
	return rs.SetMessage("Failed to identify credential type").Failed()
}

func (k KongClient) UpdateCredential(ctx context.Context, req kCred.CredRequest) (provisioning.RequestStatus, provisioning.Credential) {
	k.logger.WithField("handler", "CredentialUpdate").WithField("application", req.GetApplicationName())
	k.logger.Info("Updating credential")

	consumerID := req.GetApplicationDetailsValue(common.AttrAppID)
	agentTag := "amplify-agent"
	consumerTags := []*string{&agentTag}
	kongBuilder := kCred.NewKongCredentialBuilder().
		WithConsumerTags(consumerTags)

	rs := provisioning.NewRequestStatusBuilder()
	credentialType := req.GetCredentialType()

	switch credentialType {
	case provisioning.APIKeyARD:
		{
			authKey := req.GetCredentialDetailsValue(reqApiKey)
			err := k.KeyAuths.Delete(ctx, &consumerID, &authKey)
			if err != nil {
				k.logger.Errorf("failed to delete API Key: %s for consumerID %s. Reason: %w", authKey, consumerID, err)
				return rs.Failed(), nil
			}
			keyAuth := kongBuilder.WithAuthKey(authKey).
				ToKeyAuth()
			resp, err := k.KeyAuths.Create(ctx, &consumerID, keyAuth)
			if err != nil {
				k.logger.Errorf("failed to create API Key for consumerID %s. Reason: %w", authKey, consumerID, err)
				return rs.Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			return rs.Success(), provisioning.NewCredentialBuilder().SetAPIKey(*resp.Key)
		}
	case provisioning.BasicAuthARD:
		{
			username := req.GetCredentialDetailsValue(reqUsername)
			if err := k.BasicAuths.Delete(ctx, &consumerID, &username); err != nil {
				k.logger.Errorf("failed to delete http-basic credential by username: %s from consumer: %s. Reason: %w", username, consumerID, err)
				return rs.Failed(), nil
			}

			basicAuth := kongBuilder.WithUsername(username).
				WithPassword("").
				ToBasicAuth()
			resp, err := k.BasicAuths.Create(ctx, &consumerID, basicAuth)
			if err != nil {
				k.logger.Errorf("failed to create http-basic credential for consumerID %s. Reason: %w", consumerID, err)
				return rs.Failed(), nil
			}
			return rs.Success(), provisioning.NewCredentialBuilder().SetHTTPBasic(*resp.Username, *resp.Password)
		}
	case "oauth2":
		{
			clientID := req.GetCredentialDetailsValue(reqClientID)
			if err := k.Oauth2Credentials.Delete(ctx, &consumerID, &clientID); err != nil {
				k.logger.Errorf("failed to delete oauth2 credential by clientID: %s from consumer: %s. Reason: %w", clientID, consumerID, err)
				return rs.Failed(), nil
			}
			oauth2 := kongBuilder.WithClientID(reqClientID).
				ToOauth2()
			resp, err := k.Oauth2Credentials.Create(ctx, &consumerID, oauth2)
			if err != nil {
				k.logger.Errorf("failed to create oauth2 credential for consumerID %s. Reason: %w", consumerID, err)
				return rs.Failed(), nil
			}
			return rs.Success(), provisioning.NewCredentialBuilder().SetOAuthIDAndSecret(*resp.ID, *resp.ClientSecret)
		}
	}
	k.logger.Error("failed to identify credential type")
	return rs.Failed(), nil
}

func (k KongClient) CreateCredential(ctx context.Context, req kCred.CredRequest) (provisioning.RequestStatus, provisioning.Credential) {
	k.logger.WithField("handler", "CreateCredential").WithField("application", req.GetApplicationName())
	k.logger.Info("Creating credential")

	consumerID := req.GetApplicationDetailsValue(common.AttrAppID)
	agentTag := "amplify-agent"
	consumerTags := []*string{&agentTag}
	kongBuilder := kCred.NewKongCredentialBuilder().
		WithConsumerTags(consumerTags)

	rs := provisioning.NewRequestStatusBuilder()
	credentialType := req.GetCredentialType()

	switch credentialType {
	case provisioning.APIKeyARD:
		{
			keyAuth := kongBuilder.WithAuthKey("").
				ToKeyAuth()
			resp, err := k.KeyAuths.Create(ctx, &consumerID, keyAuth)
			if err != nil {
				k.logger.Errorf("failed to create api-key credential for consumerID %s. Reason: %w", consumerID, err)
				return rs.Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			return rs.Success(), provisioning.NewCredentialBuilder().SetAPIKey(*resp.Key)
		}
	case provisioning.BasicAuthARD:
		{
			basicAuth := kongBuilder.WithUsername("").
				WithPassword("").
				ToBasicAuth()
			resp, err := k.BasicAuths.Create(ctx, &consumerID, basicAuth)
			if err != nil {
				k.logger.Errorf("failed to create http-basic credential for consumerID %s. Reason: %w", consumerID, err)
				rs.SetMessage(err.Error())
				return rs.Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			return rs.Success(), provisioning.NewCredentialBuilder().SetHTTPBasic(*resp.Username, *resp.Password)
		}
	case "oauth2":
		{
			oauth2 := kongBuilder.WithClientID("").
				WithClientSecret("").
				ToOauth2()
			resp, err := k.Oauth2Credentials.Create(ctx, &consumerID, oauth2)
			if err != nil {
				k.logger.Errorf("failed to create oauth2 credential for consumerID %s. Reason: %w", consumerID, err)
				return rs.Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			return rs.Success(), provisioning.NewCredentialBuilder().SetOAuthIDAndSecret(*resp.ID, *resp.ClientSecret)
		}
	}
	k.logger.Errorf("failed to identify credential type")
	return rs.Failed(), nil
}
