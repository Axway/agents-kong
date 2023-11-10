package credential

import (
	"context"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"
	_ "github.com/kong/go-kong/kong"
	klib "github.com/kong/go-kong/kong"
)

const (
	reqApiKey       = "apiKey"
	reqUsername     = "username"
	reqClientID     = "clientId"
	reqClientSecret = "clientSecret"
)

type credentialProvisioner struct {
	ctx            context.Context
	client         credentialClient
	appID          string
	managedAppName string
	logger         log.FieldLogger
	request        credRequest
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

func (p credentialProvisioner) Deprovision() prov.RequestStatus {
	consumerID := p.request.GetApplicationDetailsValue(common.AttrAppID)
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()

	credentialType := p.request.GetCredentialType()
	switch credentialType {
	case provisioning.APIKeyARD:
		{
			authKey := p.request.GetCredentialDetailsValue(reqApiKey)
			if err := p.client.DeleteAuthKey(ctx, consumerID, authKey); err != nil {
				return rs.SetMessage("Could not delete auth key credential").Failed()
			}
			return rs.SetMessage("API Key successfully deleted.").Success()
		}
	case provisioning.BasicAuthARD:
		{
			username := p.request.GetCredentialDetailsValue(common.AttrCredentialID)
			if err := p.client.DeleteHttpBasic(ctx, consumerID, username); err != nil {
				return rs.SetMessage("Could not delete basic auth credential").Failed()
			}
			return rs.SetMessage("Basic auth credential successfully deleted.").Success()
		}
	case provisioning.OAuthSecretCRD:
		{
			clientID := p.request.GetApplicationDetailsValue(reqClientID)
			if err := p.client.DeleteOauth2(ctx, consumerID, clientID); err != nil {
				return rs.SetMessage("Could not delete oauth2 credential").Failed()
			}
			return rs.SetMessage("OAuth2 credential successfully deleted.").Success()
		}
	}
	return rs.SetMessage("Failed to identify credential type").Failed()
}

func (p credentialProvisioner) Provision() (prov.RequestStatus, prov.Credential) {
	consumerID := p.request.GetApplicationDetailsValue(common.AttrAppID)
	agentTag := "amplify-agent"
	consumerTags := []*string{&agentTag}
	kongBuilder := NewKongCredentialBuilder().
		WithConsumerTags(consumerTags)

	ctx := context.Background()
	rs := provisioning.NewRequestStatusBuilder()
	credentialType := p.request.GetCredentialType()

	switch credentialType {
	case provisioning.APIKeyARD:
		{
			keyAuth := kongBuilder.WithAuthKey("").
				ToKeyAuth()
			resp, err := p.client.CreateAuthKey(ctx, consumerID, keyAuth)
			if err != nil {
				return rs.SetMessage("Failed to create api-key credential").Failed(), nil
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
			resp, err := p.client.CreateHttpBasic(ctx, consumerID, basicAuth)
			if err != nil {
				return rs.SetMessage("Failed to create basic auth credential").Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			return rs.Success(), provisioning.NewCredentialBuilder().SetHTTPBasic(*resp.Username, *resp.Password)
		}
	case provisioning.OAuthSecretCRD:
		{
			oauth2 := kongBuilder.WithClientID("").
				WithClientSecret("").
				WithName("").
				ToOauth2()
			resp, err := p.client.CreateOauth2(ctx, consumerID, oauth2)
			if err != nil {
				return rs.SetMessage("Failed to create basic auth credential").Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			return rs.Success(), provisioning.NewCredentialBuilder().SetOAuthIDAndSecret(*resp.ClientID, *resp.ClientSecret)
		}
	}
	return rs.Failed(), nil
}

func (p credentialProvisioner) Update() (prov.RequestStatus, prov.Credential) {
	consumerID := p.request.GetApplicationDetailsValue(common.AttrAppID)
	agentTag := "amplify-agent"
	consumerTags := []*string{&agentTag}
	kongBuilder := NewKongCredentialBuilder().
		WithConsumerTags(consumerTags)

	ctx := context.Background()
	rs := provisioning.NewRequestStatusBuilder()
	credentialType := p.request.GetCredentialType()

	switch credentialType {
	case provisioning.APIKeyARD:
		{
			authKey := p.request.GetCredentialDetailsValue(reqApiKey)
			if authKey == "" {
				return rs.SetMessage("Empty api-key passed").Failed(), nil
			}
			if err := p.client.DeleteAuthKey(ctx, consumerID, authKey); err != nil {
				return rs.SetMessage("Could not delete api-key credential").Failed(), nil
			}
			keyAuth := kongBuilder.WithAuthKey(authKey).
				ToKeyAuth()
			resp, err := p.client.CreateAuthKey(ctx, consumerID, keyAuth)
			if err != nil {
				return rs.SetMessage("Failed to create api-key credential").Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			return rs.Success(), provisioning.NewCredentialBuilder().SetAPIKey(*resp.Key)
		}
	case provisioning.BasicAuthARD:
		{
			username := p.request.GetCredentialDetailsValue(reqUsername)
			if username == "" {
				return rs.SetMessage("Empty username passed").Failed(), nil
			}
			if err := p.client.DeleteHttpBasic(ctx, consumerID, username); err != nil {
				return rs.SetMessage("Failed to delete basic auth credential").Failed(), nil
			}

			basicAuth := kongBuilder.WithUsername(username).
				WithPassword("").
				ToBasicAuth()
			resp, err := p.client.CreateHttpBasic(ctx, consumerID, basicAuth)
			if err != nil {
				return rs.SetMessage("Failed to create basic auth credential").Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			return rs.Success(), provisioning.NewCredentialBuilder().SetHTTPBasic(*resp.Username, *resp.Password)
		}
	case provisioning.OAuthSecretCRD:
		{
			clientID := p.request.GetCredentialDetailsValue(reqClientID)
			if clientID == "" {
				return rs.SetMessage("Empty client ID passed").Failed(), nil
			}
			if err := p.client.DeleteOauth2(ctx, consumerID, clientID); err != nil {
				return rs.SetMessage("Failed to delete oauth2 credential").Failed(), nil
			}

			oauth2 := kongBuilder.WithClientID(reqClientID).
				WithClientSecret("").
				WithName("").
				ToOauth2()
			resp, err := p.client.CreateOauth2(ctx, consumerID, oauth2)
			if err != nil {
				return rs.SetMessage("Failed to create oauth2 credential").Failed(), nil
			}
			rs.AddProperty(common.AttrAppID, *resp.Consumer.ID)
			rs.AddProperty(common.AttrCredentialID, *resp.ID)
			return rs.Success(), provisioning.NewCredentialBuilder().SetOAuthIDAndSecret(*resp.ClientID, *resp.ClientSecret)
		}
	}
	return rs.SetMessage("Failed to identify credential type").Failed(), nil
}
