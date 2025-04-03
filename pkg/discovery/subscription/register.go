package subscription

import (
	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agents-kong/pkg/common"
	"github.com/sirupsen/logrus"
)

const Oauth2Name = provisioning.OAuthSecretCRD
const HttpBasicName = provisioning.BasicAuthARD
const ApiKeyName = provisioning.APIKeyARD

func getCredTypes() []string {
	return []string{"confidential", "public"}
}

func registerOauth2(workspace string) {
	oAuthRedirects := getAuthRedirectSchemaPropertyBuilder()
	corsProp := getCorsSchemaPropertyBuilder()
	oAuthTypeProp := provisioning.NewSchemaPropertyBuilder().
		SetName(common.ApplicationTypeField).
		SetRequired().
		SetLabel("Application Type").
		IsString().
		SetEnumValues(getCredTypes())

	_, err := agent.NewAccessRequestBuilder().SetName(Oauth2Name).Register()
	if err != nil {
		logrus.Errorf("Error registering Oauth2 Access Request %v", err)
	}

	_, err = agent.NewOAuthCredentialRequestBuilder(
		agent.WithCRDOAuthSecret(),
		agent.WithCRDRequestSchemaProperty(corsProp),
		agent.WithCRDRequestSchemaProperty(oAuthTypeProp),
		agent.WithCRDRequestSchemaProperty(oAuthRedirects),
		agent.WithCRDIsRenewable(),
		agent.WithCRDIsSuspendable(),
	).
		SetName(common.WksPrefixName(workspace, Oauth2Name)).
		Register()
	if err != nil {
		logrus.Errorf("Error registering Oauth2 credential Request %v", err)
	}
}

func registerBasicAuth(workspace string) {
	corsProp := getCorsSchemaPropertyBuilder()
	_, err := agent.NewBasicAuthAccessRequestBuilder().SetName(HttpBasicName).Register()
	if err != nil {
		logrus.Error("Failed to register Basic Auth Access request")
	}
	_, err = agent.NewBasicAuthCredentialRequestBuilder(
		agent.WithCRDRequestSchemaProperty(corsProp),
	).
		IsRenewable().
		SetName(common.WksPrefixName(workspace, HttpBasicName)).
		Register()
	if err != nil {
		logrus.Error("Failed to register Basic Auth Credential request")
	}
}

func registerKeyAuth(workspace string) {
	//"The api key. Leave empty for autogeneration"
	corsProp := getCorsSchemaPropertyBuilder()
	_, err := agent.NewAPIKeyAccessRequestBuilder().SetName(ApiKeyName).Register()
	if err != nil {
		logrus.Error("Error registering API key Access Request")
	}
	_, err = agent.NewAPIKeyCredentialRequestBuilder(
		agent.WithCRDRequestSchemaProperty(corsProp),
	).
		IsRenewable().
		SetName(common.WksPrefixName(workspace, ApiKeyName)).
		Register()
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
