package oauth2

import (
	"context"
	"errors"
	"fmt"
	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agents-kong/pkg/common"
	"github.com/Axway/agents-kong/pkg/subscription"
	"github.com/kong/go-kong/kong"
	"github.com/sirupsen/logrus"
)

type oauth2 struct {
	kc *kong.Client
}

const Name = "oauth2"

const (
	clientId     = "client_id"
	clientSecret = "client_secret"
)

func init() {
	subscription.Add(func(kc *kong.Client) subscription.Handler {
		return &oauth2{kc}
	})
}

func (*oauth2) Name() string {
	return Name
}

func (*oauth2) Register() {
	corsProp := subscription.GetCorsSchemaPropertyBuilder()
	//apiKeyProp := provisioning.NewSchemaPropertyBuilder().
	//	SetName(Name).
	//	SetLabel(propertyName).
	//	SetDescription("The api key. Leave empty for auto generation").
	//	IsString()
	_, err := agent.NewAccessRequestBuilder().SetName(Name).Register()
	if err != nil {
		logrus.Errorf("Error registering Oauth2  Access Request %v", err)
	}
	_, err = agent.NewOAuthCredentialRequestBuilder(agent.WithCRDRequestSchemaProperty(corsProp)).SetName(Name).IsRenewable().Register()
	if err != nil {
		logrus.Errorf("Error registering Oauth2  credential Request %v", err)

	}
}

func (ak *oauth2) UpdateCredential(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {
	logrus.Info("provisioning credential update")
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()
	agentTag := "amplify-agent"
	consumerTags := []*string{&agentTag}
	clientId := request.GetCredentialDetailsValue(clientId)
	clientSecret := request.GetCredentialDetailsValue(clientSecret)

	consumerId := request.GetApplicationDetailsValue(common.AttrAppID)
	oauth2Credential := &kong.Oauth2Credential{
		Tags: consumerTags,
	}
	// generate key if not provided
	if clientId != "" && clientSecret == "" {
		oauth2Credential.ClientID = &clientId
		oauth2Credential.ClientSecret = &clientSecret
	}
	oauth2Res, err := ak.kc.Oauth2Credentials.Create(ctx, &consumerId, oauth2Credential)
	if err != nil {
		return subscription.Failed(rs, fmt.Errorf("failed to create API Key: %w", err)), nil
	}
	credential := provisioning.NewCredentialBuilder().SetOAuthIDAndSecret(*oauth2Res.ClientID, *oauth2Res.ClientSecret)
	return rs.Success(), credential

}

func (ak *oauth2) CreateCredential(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {
	logrus.Info("provisioning credentials")
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()
	agentTag := "amplify-agent"
	consumerTags := []*string{&agentTag}
	clientId := request.GetCredentialDetailsValue(clientId)
	clientSecret := request.GetCredentialDetailsValue(clientSecret)

	consumerId := request.GetApplicationDetailsValue(common.AttrAppID)
	oauth2Credential := &kong.Oauth2Credential{
		Tags: consumerTags,
	}
	// generate key if not provided
	if clientId != "" && clientSecret == "" {
		oauth2Credential.ClientID = &clientId
		oauth2Credential.ClientSecret = &clientSecret
	}
	oauth2Res, err := ak.kc.Oauth2Credentials.Create(ctx, &consumerId, oauth2Credential)
	if err != nil {
		return subscription.Failed(rs, fmt.Errorf("failed to create API Key: %w", err)), nil
	}
	credential := provisioning.NewCredentialBuilder().SetOAuthIDAndSecret(*oauth2Res.ClientID, *oauth2Res.ClientSecret)
	return rs.Success(), credential
}

func (ak *oauth2) DeleteCredential(request provisioning.CredentialRequest) provisioning.RequestStatus {
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()
	consumerId := request.GetCredentialDetailsValue(common.AttrAppID)
	apiKeyId := request.GetCredentialDetailsValue(common.AttrAppID)
	logrus.Infof("consumerId : %s", consumerId)
	if consumerId == "" {
		return subscription.Failed(rs, errors.New("unable to delete Credential as consumerId is empty"))
	}
	err := ak.kc.Oauth2Credentials.Delete(ctx, &consumerId, &apiKeyId)
	if err != nil {
		logrus.WithError(err).Error("Failed to delete Consumer")
		return subscription.Failed(rs, errors.New(fmt.Sprintf("Failed to create API Key %s: %s", consumerId, err)))
	}
	return rs.Success()
}
