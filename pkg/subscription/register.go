package subscription

import (
	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agents-kong/pkg/common"
	"github.com/sirupsen/logrus"
)

const Oauth2Name = "oauth2"
const HttpBasicName = provisioning.BasicAuthARD
const ApiKeyName = provisioning.APIKeyARD
const propertyName = "kong-api-key"

func getCredTypes() []string {
	return []string{"confidential", "public"}
}

type Register struct{}

func (Register) RegisterOauth2() {
	oAuthRedirects := getAuthRedirectSchemaPropertyBuilder()
	corsProp := getCorsSchemaPropertyBuilder()
	provisionKey := getProvisionKeyPropertyBuilder()
	oAuthTypeProp := provisioning.NewSchemaPropertyBuilder().
		SetName(common.ApplicationTypeField).
		SetRequired().
		SetLabel("Application Type").
		IsString().
		SetEnumValues(getCredTypes())

	_, err := agent.NewAccessRequestBuilder().SetName(Oauth2Name).Register()
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

	_, err = agent.NewOAuthCredentialRequestBuilder(agent.WithCRDRequestSchemaProperty(corsProp),
		agent.WithCRDProvisionSchemaProperty(provisionKey)).SetName(Oauth2Name).IsRenewable().Register()
	if err != nil {
		logrus.Errorf("Error registering Oauth2  credential Request %v", err)

	}
}

func (Register) RegisterBasicAuth() {
	corsProp := getCorsSchemaPropertyBuilder()
	_, err := agent.NewBasicAuthAccessRequestBuilder().SetName(HttpBasicName).Register()
	if err != nil {
		logrus.Error("Failed to register Basic Auth Access request")
	}
	_, err = agent.NewBasicAuthCredentialRequestBuilder(agent.WithCRDRequestSchemaProperty(corsProp)).IsRenewable().Register()
	if err != nil {
		logrus.Error("Failed to register Basic Auth Credential request")
	}
}

func (Register) RegisterKeyAuth() {
	//"The api key. Leave empty for autogeneration"
	corsProp := getCorsSchemaPropertyBuilder()
	apiKeyProp := provisioning.NewSchemaPropertyBuilder().
		SetName(ApiKeyName).
		SetLabel(propertyName).
		SetDescription("The api key. Leave empty for auto generation").
		IsString()
	_, err := agent.NewAPIKeyAccessRequestBuilder().SetName(ApiKeyName).Register()
	if err != nil {
		logrus.Error("Error registering API key Access Request")
	}
	_, err = agent.NewAPIKeyCredentialRequestBuilder(agent.WithCRDProvisionSchemaProperty(apiKeyProp), agent.WithCRDRequestSchemaProperty(corsProp)).IsRenewable().Register()
	if err != nil {
		logrus.Error("Error registering API Credential Access Request")
	}
}

func getAuthRedirectSchemaPropertyBuilder() provisioning.PropertyBuilder {
	return provisioning.NewSchemaPropertyBuilder().
		SetName(common.RedirectURLsField).
		SetLabel("Redirect URLs").
		IsArray().
		AddItem(
			provisioning.NewSchemaPropertyBuilder().
				SetName("URL").
				IsString())
}

func getCorsSchemaPropertyBuilder() provisioning.PropertyBuilder {
	return provisioning.NewSchemaPropertyBuilder().
		SetName(common.CorsField).
		SetLabel("Javascript Origins").
		IsArray().
		AddItem(
			provisioning.NewSchemaPropertyBuilder().
				SetName("Origins").
				IsString())
}

func getProvisionKeyPropertyBuilder() provisioning.PropertyBuilder {
	return provisioning.NewSchemaPropertyBuilder().
		SetName(common.ProvisionKey).
		SetLabel("Provision key").
		SetRequired().
		IsString()
}
