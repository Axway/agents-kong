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

// GetCredTypes -
func GetCredTypes() []string {
	return []string{"confidential", "public"}
}

func (a *oauth2) getAuthRedirectSchemaPropertyBuilder() provisioning.PropertyBuilder {
	return provisioning.NewSchemaPropertyBuilder().
		SetName(common.RedirectURLsField).
		SetLabel("Redirect URLs").
		IsArray().
		AddItem(
			provisioning.NewSchemaPropertyBuilder().
				SetName("URL").
				IsString())
}

func (o *oauth2) Register() {
	oAuthRedirects := o.getAuthRedirectSchemaPropertyBuilder()
	corsProp := subscription.GetCorsSchemaPropertyBuilder()
	provisionKey := subscription.GetProvisionKeyPropertyBuilder()
	oAuthTypeProp := provisioning.NewSchemaPropertyBuilder().
		SetName(common.ApplicationTypeField).
		SetRequired().
		SetLabel("Application Type").
		IsString().
		SetEnumValues(GetCredTypes())

	_, err := agent.NewAccessRequestBuilder().SetName(Name).Register()
	if err != nil {
		logrus.Errorf("Error registering Oauth2  Access Request %v", err)
	}

	_, err = agent.NewOAuthCredentialRequestBuilder(
		agent.WithCRDOAuthSecret(),
		agent.WithCRDRequestSchemaProperty(corsProp),
		agent.WithCRDRequestSchemaProperty(oAuthTypeProp),
		agent.WithCRDRequestSchemaProperty(oAuthRedirects),
		agent.WithCRDIsRenewable(),
		agent.WithCRDIsSuspendable(),
	).Register()
	if err != nil {
		logrus.Errorf("Error registering Oauth2  credential Request %v", err)
	}

	_, err = agent.NewOAuthCredentialRequestBuilder(agent.WithCRDRequestSchemaProperty(corsProp), agent.WithCRDProvisionSchemaProperty(provisionKey)).SetName(Name).IsRenewable().Register()
	if err != nil {
		logrus.Errorf("Error registering Oauth2  credential Request %v", err)

	}
}

func (o *oauth2) UpdateCredential(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {
	logrus.Info("provisioning credential update")
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()
	consumerId := request.GetApplicationDetailsValue(common.AttrAppID)
	oauth2Credential := o.createOauthCredentialStruct(request)
	oauth2Res, err := o.kc.Oauth2Credentials.Create(ctx, &consumerId, oauth2Credential)
	if err != nil {
		return subscription.Failed(rs, fmt.Errorf("failed to create API Key: %w", err)), nil
	}
	credential := provisioning.NewCredentialBuilder().SetOAuthIDAndSecret(*oauth2Res.ClientID, *oauth2Res.ClientSecret)
	return rs.Success(), credential
}

func (o *oauth2) createOauthCredentialStruct(request provisioning.CredentialRequest) *kong.Oauth2Credential {
	agentTag := "amplify-agent"
	consumerTags := []*string{&agentTag}
	clientId := request.GetCredentialDetailsValue(clientId)
	clientSecret := request.GetCredentialDetailsValue(clientSecret)
	provData := o.getCredProvData(request.GetCredentialData())

	oauth2Credential := &kong.Oauth2Credential{
		Tags: consumerTags,
	}
	// generate key if not provided
	if clientId != "" {
		oauth2Credential.ClientID = &clientId
	}

	if clientSecret == "" {
		oauth2Credential.ClientSecret = &clientSecret
	}
	if len(provData.cors) > 0 {
		var redirectUris []*string
		for i := range provData.cors {
			redirectUris = append(redirectUris, &provData.cors[i])
		}
		oauth2Credential.RedirectURIs = redirectUris
	}
	oauth2Credential.ClientType = &provData.appType
	return oauth2Credential
}

func (o *oauth2) CreateCredential(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {
	logrus.Info("provisioning credentials")
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()
	consumerId := request.GetApplicationDetailsValue(common.AttrAppID)
	oauth2Credential := o.createOauthCredentialStruct(request)
	oauth2Res, err := o.kc.Oauth2Credentials.Create(ctx, &consumerId, oauth2Credential)
	if err != nil {
		return subscription.Failed(rs, fmt.Errorf("failed to create API Key: %w", err)), nil
	}
	credential := provisioning.NewCredentialBuilder().SetOAuthIDAndSecret(*oauth2Res.ClientID, *oauth2Res.ClientSecret)
	return rs.Success(), credential
}

func (o *oauth2) DeleteCredential(request provisioning.CredentialRequest) provisioning.RequestStatus {
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()
	consumerId := request.GetCredentialDetailsValue(common.AttrAppID)
	apiKeyId := request.GetCredentialDetailsValue(common.AttrAppID)
	logrus.Infof("consumerId : %s", consumerId)
	if consumerId == "" {
		return subscription.Failed(rs, errors.New("unable to delete Credential as consumerId is empty"))
	}
	err := o.kc.Oauth2Credentials.Delete(ctx, &consumerId, &apiKeyId)
	if err != nil {
		logrus.WithError(err).Error("Failed to delete Oauth2 Credential")
		return subscription.Failed(rs, errors.New(fmt.Sprintf("Failed to create API Key %s: %s", consumerId, err)))
	}
	return rs.Success()
}

type credentialMetaData struct {
	cors            []string
	redirectURLs    []string
	oauthServerName string
	appType         string
	audience        string
}

func (o *oauth2) getCredProvData(credData map[string]interface{}) credentialMetaData {
	// defaults
	credMetaData := credentialMetaData{
		cors:         []string{"*"},
		redirectURLs: []string{},
		appType:      "Confidential",
		audience:     "",
	}

	// get cors from credential request
	if data, ok := credData[common.CorsField]; ok && data != nil {
		credMetaData.cors = []string{}
		for _, c := range data.([]interface{}) {
			credMetaData.cors = append(credMetaData.cors, c.(string))
		}
	}
	// get redirectURLs
	if data, ok := credData[common.RedirectURLsField]; ok && data != nil {
		credMetaData.redirectURLs = []string{}
		for _, u := range data.([]interface{}) {
			credMetaData.redirectURLs = append(credMetaData.redirectURLs, u.(string))
		}
	}
	// Oauth Server  field
	if data, ok := credData[common.OauthServerField]; ok && data != nil {
		credMetaData.oauthServerName = data.(string)
	}
	// credential type field
	if data, ok := credData[common.ApplicationTypeField]; ok && data != nil {
		credMetaData.appType = data.(string)
	}

	return credMetaData
}
