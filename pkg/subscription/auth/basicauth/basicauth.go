package basicauth

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

type basicAuth struct {
	kc *kong.Client
}

const Name = provisioning.BasicAuthARD

const (
	propertyName = "basic-auth"
)

func init() {
	subscription.Add(func(kc *kong.Client) subscription.Handler {
		return &basicAuth{kc}
	})
}

func (*basicAuth) Name() string {
	return Name
}

func (*basicAuth) Register() {
	//"The api key. Leave empty for autogeneration"
	corsProp := subscription.GetCorsSchemaPropertyBuilder()
	agent.NewBasicAuthAccessRequestBuilder().SetName(Name).Register()
	agent.NewBasicAuthCredentialRequestBuilder(agent.WithCRDRequestSchemaProperty(corsProp)).IsRenewable().Register()
}

func (auth *basicAuth) deleteAllKeys(consumerID, subscriptionID string) error {
	ctx := context.Background()
	keys, _, err := auth.kc.KeyAuths.ListForConsumer(ctx, &consumerID, &kong.ListOpt{Tags: []*string{&subscriptionID}})
	if err != nil {
		return fmt.Errorf("failed to list all consumers: %w", err)
	}
	for _, k := range keys {
		err := auth.kc.KeyAuths.Delete(ctx, &consumerID, k.ID)
		if err != nil {
			return fmt.Errorf("failed to delete consumer key: ")
		}
	}
	return nil
}

func (auth *basicAuth) UpdateCredential(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {
	logrus.Info("provisioning credential update")
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()
	agentTag := "amplify-agent"
	consumerTags := []*string{&agentTag}
	username := request.GetCredentialDetailsValue(propertyName)
	password := request.GetCredentialDetailsValue(propertyName)
	consumerId := request.GetApplicationDetailsValue(common.AttrAppID)

	kongBasicAuth := kong.BasicAuth{
		Username: &username,
		Password: &password,
		Tags:     consumerTags,
	}
	basicAuthResponse, err := auth.kc.BasicAuths.Create(ctx, &consumerId, &kongBasicAuth)
	if err != nil {
		return subscription.Failed(rs, fmt.Errorf("failed to create API Key: %w", err)), nil
	}
	credential := provisioning.NewCredentialBuilder().SetHTTPBasic(*basicAuthResponse.Username, *basicAuthResponse.Password)
	return rs.Success(), credential

}

func (auth *basicAuth) CreateCredential(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {
	logrus.Info("provisioning credentials")
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()
	agentTag := "amplify-agent"
	consumerTags := []*string{&agentTag}
	username := request.GetCredentialDetailsValue(propertyName)
	password := request.GetCredentialDetailsValue(propertyName)

	consumerId := request.GetApplicationDetailsValue(common.AttrAppID)

	kongBasicAuth := kong.BasicAuth{
		Username: &username,
		Password: &password,
		Tags:     consumerTags,
	}
	basicAuthResponse, err := auth.kc.BasicAuths.Create(ctx, &consumerId, &kongBasicAuth)
	if err != nil {
		return subscription.Failed(rs, fmt.Errorf("failed to create API Key: %w", err)), nil
	}
	credential := provisioning.NewCredentialBuilder().SetHTTPBasic(*basicAuthResponse.Username, *basicAuthResponse.Password)
	return rs.Success(), credential

}

func (auth *basicAuth) DeleteCredential(request provisioning.CredentialRequest) provisioning.RequestStatus {
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()
	consumerId := request.GetCredentialDetailsValue(common.AttrAppID)
	apiKeyId := request.GetCredentialDetailsValue(common.AttrAppID)
	logrus.Infof("consumerId : %s", consumerId)
	if consumerId == "" {
		return subscription.Failed(rs, errors.New("unable to delete Credential as consumerId is empty"))
	}
	err := auth.kc.BasicAuths.Delete(ctx, &consumerId, &apiKeyId)
	if err != nil {
		logrus.WithError(err).Error("Failed to delete Consumer")
		return subscription.Failed(rs, errors.New(fmt.Sprintf("Failed to create API Key %s: %s", consumerId, err)))
	}
	return rs.Success()
}
