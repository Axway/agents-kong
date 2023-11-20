package credential

import (
	"context"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"
	"github.com/google/uuid"
	klib "github.com/kong/go-kong/kong"
)

type credentialProvisioner struct {
	ctx     context.Context
	client  credentialClient
	logger  log.FieldLogger
	request credRequest
}

type credentialClient interface {
	DeleteOauth2(ctx context.Context, consumerID, clientID string) error
	DeleteHttpBasic(ctx context.Context, consumerID, username string) error
	DeleteAuthKey(ctx context.Context, consumerID, authKey string) error
	CreateHttpBasic(ctx context.Context, consumerID string, basicAuth *klib.BasicAuth) (*klib.BasicAuth, error)
	CreateOauth2(ctx context.Context, consumerID string, oauth2 *klib.Oauth2Credential) (*klib.Oauth2Credential, error)
	CreateAuthKey(ctx context.Context, consumerID string, keyAuth *klib.KeyAuth) (*klib.KeyAuth, error)
}

type credRequest interface {
	GetApplicationDetailsValue(key string) string
	GetApplicationName() string
	GetCredentialDetailsValue(key string) string
	GetCredentialData() map[string]interface{}
	GetCredentialType() string
}

func NewCredentialProvisioner(ctx context.Context, client credentialClient, req credRequest) credentialProvisioner {
	a := credentialProvisioner{
		ctx: context.Background(),
		logger: log.NewFieldLogger().
			WithComponent("credentialProvisioner").
			WithPackage("credential"),
		client:  client,
		request: req,
	}
	return a
}

func (p credentialProvisioner) Deprovision() provisioning.RequestStatus {
	consumerID := p.request.GetApplicationDetailsValue(common.AttrAppID)
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()

	credentialType := p.request.GetCredentialType()
	credentialID := p.request.GetCredentialDetailsValue(common.AttrCredentialID)
	if credentialID == "" {
		return rs.SetMessage("CredentialID cannot be empty").Failed()
	}
	p.logger.WithField("credentialID", credentialID).
		WithField("consumerID", consumerID)
	p.logger.Info("Started credential de-provisioning")

	switch credentialType {
	case provisioning.APIKeyARD:
		{
			if err := p.client.DeleteAuthKey(ctx, consumerID, credentialID); err != nil {
				return rs.SetMessage("API Key credential does not exist or it has already been deleted").Success()
			}
			p.logger.Info("API Key successful de-provision")
			return rs.SetMessage("API Key successfully deleted.").Success()
		}
	case provisioning.BasicAuthARD:
		{
			if err := p.client.DeleteHttpBasic(ctx, consumerID, credentialID); err != nil {
				return rs.SetMessage("Basic auth credential does not exist or it has already been deleted").Success()
			}
			p.logger.Info("Basic Auth successful de-provision")
			return rs.SetMessage("Basic auth credential successfully deleted.").Success()
		}
	case provisioning.OAuthSecretCRD:
		{
			if err := p.client.DeleteOauth2(ctx, consumerID, credentialID); err != nil {
				return rs.SetMessage("OAuth2 credential does not exist or it has already been deleted").Success()
			}
			p.logger.Info("OAuth2 successful de-provision")
			return rs.SetMessage("OAuth2 credential successfully deleted.").Success()
		}
	}
	return rs.SetMessage("Failed to identify credential type").Failed()
}

func (p credentialProvisioner) Provision() (provisioning.RequestStatus, provisioning.Credential) {
	consumerID := p.request.GetApplicationDetailsValue(common.AttrAppID)
	agentTag := "amplify-agent"
	consumerTags := []*string{&agentTag}
	kongBuilder := NewKongCredentialBuilder().
		WithConsumerTags(consumerTags)

	ctx := context.Background()
	rs := provisioning.NewRequestStatusBuilder()
	credentialType := p.request.GetCredentialType()
	p.logger.WithField("consumerID", consumerID)
	p.logger.Debug("Started credential provisioning")

	switch credentialType {
	case provisioning.APIKeyARD:
		{
			keyAuth := kongBuilder.WithAuthKey("").
				ToKeyAuth()
			resp, err := p.client.CreateAuthKey(ctx, consumerID, keyAuth)
			if err != nil {
				p.logger.Info("API key unsuccessful provisioning")
				return rs.SetMessage("Failed to create api-key credential").Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			p.logger.Info("API key successful provisioning")
			return rs.Success(), provisioning.NewCredentialBuilder().SetAPIKey(*resp.Key)
		}
	case provisioning.BasicAuthARD:
		{
			user := uuid.NewString()
			pass := uuid.NewString()
			basicAuth := kongBuilder.WithUsername(user).
				WithPassword(pass).
				ToBasicAuth()
			resp, err := p.client.CreateHttpBasic(ctx, consumerID, basicAuth)
			if err != nil {
				p.logger.Info("Basic auth unsuccessful provisioning")
				return rs.SetMessage("Failed to create basic auth credential").Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			rs.AddProperty(common.AttrCredUpdater, *resp.Username)
			p.logger.Info("Basic auth successful provisioning")
			return rs.Success(), provisioning.NewCredentialBuilder().SetHTTPBasic(user, pass)
		}
	case provisioning.OAuthSecretCRD:
		{
			oauth2 := kongBuilder.WithClientID("").
				WithClientSecret("").
				WithName("").
				ToOauth2()
			resp, err := p.client.CreateOauth2(ctx, consumerID, oauth2)
			if err != nil {
				p.logger.Info("Oauth2 unsuccessful provisioning")
				return rs.SetMessage("Failed to create oauth2 credential").Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			rs.AddProperty(common.AttrCredUpdater, *resp.ClientID)
			p.logger.Info("OAuth2 successful provisioning")
			return rs.Success(), provisioning.NewCredentialBuilder().SetOAuthIDAndSecret(*resp.ClientID, *resp.ClientSecret)
		}
	}
	return rs.Failed(), nil
}

func (p credentialProvisioner) Update() (provisioning.RequestStatus, provisioning.Credential) {
	consumerID := p.request.GetApplicationDetailsValue(common.AttrAppID)
	agentTag := "amplify-agent"
	consumerTags := []*string{&agentTag}
	kongBuilder := NewKongCredentialBuilder().
		WithConsumerTags(consumerTags)

	ctx := context.Background()
	rs := provisioning.NewRequestStatusBuilder()
	credentialType := p.request.GetCredentialType()
	credentialID := p.request.GetCredentialDetailsValue(common.AttrCredentialID)
	key := p.request.GetCredentialDetailsValue(common.AttrCredUpdater)
	if credentialID == "" {
		return rs.SetMessage("kongCredentialId cannot be empty").Failed(), nil
	}

	p.logger.WithField("credentialID", credentialID).
		WithField("consumerID", consumerID)
	p.logger.Info("Started credential update")

	switch credentialType {
	case provisioning.APIKeyARD:
		{
			if err := p.client.DeleteAuthKey(ctx, consumerID, credentialID); err != nil {
				p.logger.WithError(err).Error("Could not delete api-key credential")
				return rs.SetMessage(fmt.Sprintf("Could not delete credential %s for consumer %s", consumerID, credentialID)).Failed(), nil
			}
			keyAuth := kongBuilder.WithAuthKey("").
				ToKeyAuth()
			resp, err := p.client.CreateAuthKey(ctx, consumerID, keyAuth)
			if err != nil {
				p.logger.WithError(err).Error("Could not create api-key credential")
				return rs.SetMessage("Failed to create api-key credential").Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			p.logger.Info("API Key successful update")
			return rs.Success(), provisioning.NewCredentialBuilder().SetAPIKey(*resp.Key)
		}
	case provisioning.BasicAuthARD:
		{
			if err := p.client.DeleteHttpBasic(ctx, consumerID, credentialID); err != nil {
				p.logger.WithError(err).Error("Could not delete basic auth credential")
				return rs.SetMessage(fmt.Sprintf("Could not delete credential %s for consumer %s", consumerID, credentialID)).Failed(), nil
			}
			basicAuth := kongBuilder.WithUsername(key).
				WithPassword("").
				ToBasicAuth()
			resp, err := p.client.CreateHttpBasic(ctx, consumerID, basicAuth)
			if err != nil {
				p.logger.WithError(err).Error("Could not create basic auth credential")
				return rs.SetMessage("Failed to create basic auth credential").Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			rs.AddProperty(common.AttrCredUpdater, *resp.Username)
			p.logger.Info("Basic Auth successful update")
			return rs.Success(), provisioning.NewCredentialBuilder().SetHTTPBasic(*resp.Username, *resp.Password)
		}
	case provisioning.OAuthSecretCRD:
		{
			if err := p.client.DeleteOauth2(ctx, consumerID, credentialID); err != nil {
				p.logger.WithError(err).Error("Could not delete oauth2 credential")
				return rs.SetMessage(fmt.Sprintf("Could not delete credential %s for consumer %s", consumerID, credentialID)).Failed(), nil
			}
			oauth2 := kongBuilder.WithClientID(key).
				WithClientSecret("").
				WithName("").
				ToOauth2()
			resp, err := p.client.CreateOauth2(ctx, consumerID, oauth2)
			if err != nil {
				p.logger.WithError(err).Error("Could not create oauth2 credential")
				return rs.SetMessage("Failed to create oauth2 credential").Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			rs.AddProperty(common.AttrCredUpdater, *resp.ClientID)
			p.logger.Info("Oauth2 successful update")
			return rs.Success(), provisioning.NewCredentialBuilder().SetOAuthIDAndSecret(*resp.ClientID, *resp.ClientSecret)
		}
	}
	return rs.SetMessage("Failed to identify credential type").Failed(), nil
}
