package apikey

import (
	"context"
	"errors"
	"fmt"
	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agents-kong/pkg/common"
	"github.com/Axway/agents-kong/pkg/gateway"
	"github.com/Axway/agents-kong/pkg/subscription"
	"github.com/kong/go-kong/kong"
	"github.com/sirupsen/logrus"
)

type apiKey struct {
	kc *kong.Client
}

const Name = "kong-apikey"

const (
	propertyName = "api-key"
)

func init() {
	subscription.Add(func(kc *kong.Client) subscription.Handler {
		return &apiKey{kc}
	})
}

func (*apiKey) Name() string {
	return Name
}

func (*apiKey) Register() {
	//"The api key. Leave empty for autogeneration"
	corsProp := gateway.GetCorsSchemaPropertyBuilder()
	apiKeyProp := provisioning.NewSchemaPropertyBuilder().
		SetName(Name).
		SetLabel(propertyName).
		IsString()
	agent.NewAPIKeyAccessRequestBuilder().Register()
	agent.NewAPIKeyCredentialRequestBuilder(agent.WithCRDProvisionSchemaProperty(apiKeyProp), agent.WithCRDRequestSchemaProperty(corsProp)).IsRenewable().Register()
}

func (ak *apiKey) deleteAllKeys(consumerID, subscriptionID string) error {
	ctx := context.Background()
	keys, _, err := ak.kc.KeyAuths.ListForConsumer(ctx, &consumerID, &kong.ListOpt{Tags: []*string{&subscriptionID}})
	if err != nil {
		return fmt.Errorf("failed to list all consumers: %w", err)
	}
	for _, k := range keys {
		err := ak.kc.KeyAuths.Delete(ctx, &consumerID, k.ID)
		if err != nil {
			return fmt.Errorf("failed to delete consumer key: ")
		}
	}
	return nil
}

func (ak *apiKey) UpdateCredential(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {
	logrus.Info("provisioning credential update")
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()
	agentTag := "amplify-agent"
	consumerTags := []*string{&agentTag}
	key := request.GetCredentialDetailsValue(propertyName)
	consumerId := request.GetApplicationDetailsValue(common.AttrAppID)

	keyAuth := &kong.KeyAuth{
		Tags: consumerTags,
	}
	// generate key if not provided
	if key != "" {
		keyAuth.Key = &key
	}
	keyAuthRes, err := ak.kc.KeyAuths.Update(ctx, &consumerId, keyAuth)
	if err != nil {
		return subscription.Failed(rs, fmt.Errorf("failed to create API Key: %w", err)), nil
	}
	credential := provisioning.NewCredentialBuilder().SetAPIKey(*keyAuthRes.Key)
	return rs.Success(), credential
}

func (ak *apiKey) CreateCredential(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {
	logrus.Info("provisioning credentials")
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()
	agentTag := "amplify-agent"
	consumerTags := []*string{&agentTag}
	key := request.GetCredentialDetailsValue(propertyName)
	consumerId := request.GetApplicationDetailsValue(common.AttrAppID)

	keyAuth := &kong.KeyAuth{
		Tags: consumerTags,
	}
	// generate key if not provided
	if key != "" {
		keyAuth.Key = &key
	}
	keyAuthRes, err := ak.kc.KeyAuths.Create(ctx, &consumerId, keyAuth)
	if err != nil {
		return subscription.Failed(rs, fmt.Errorf("failed to create API Key: %w", err)), nil
	}
	credential := provisioning.NewCredentialBuilder().SetAPIKey(*keyAuthRes.Key)
	return rs.Success(), credential

}

func (ak *apiKey) DeleteCredential(request provisioning.CredentialRequest) provisioning.RequestStatus {
	rs := provisioning.NewRequestStatusBuilder()
	ctx := context.Background()
	consumerId := request.GetCredentialDetailsValue(common.AttrAppID)
	apiKeyId := request.GetCredentialDetailsValue(common.AttrAppID)
	logrus.Infof("consumerId : %s", consumerId)
	if consumerId == "" {
		return subscription.Failed(rs, errors.New("unable to delete Credential as consumerId is empty"))
	}
	err := ak.kc.KeyAuths.Delete(ctx, &consumerId, &apiKeyId)
	if err != nil {
		logrus.WithError(err).Error("Failed to delete Consumer")
		return subscription.Failed(rs, errors.New(fmt.Sprintf("Failed to create API Key %s: %s", consumerId, err)))
	}
	return rs.Success()
}
