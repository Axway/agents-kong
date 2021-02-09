package apikey

import (
	"context"
	"fmt"
	"strings"

	"github.com/Axway/agent-sdk/pkg/apic"
	kutil "github.com/Axway/agents-kong/pkg/kong"
	"github.com/Axway/agents-kong/pkg/subscription"
	"github.com/kong/go-kong/kong"

	"github.com/sirupsen/logrus"
)

type apiKey struct {
	kc *kutil.Client
}

const Name = "kong-apikey"

const (
	keyPluginName = "key-auth"
	propertyName  = "api-key"
)

func init() {
	subscription.Register(func(kc *kutil.Client) subscription.Handler {
		return &apiKey{kc}
	})
}

func (*apiKey) Name() string {
	return Name
}

func (*apiKey) APICPolicy() string {
	return apic.Apikey
}

// Schema returns the schema
func (*apiKey) Schema() apic.SubscriptionSchema {
	schema := apic.NewSubscriptionSchema(Name)

	schema.AddProperty(propertyName,
		"string",
		"The api key. Leave empty for autogeneration",
		"",
		false,
		nil)

	return schema
}

// IsApplicable if this subscription method
// is applicable for a route with the given plugins.
func (*apiKey) IsApplicable(plugins map[string]*kong.Plugin) bool {
	_, ok := plugins[keyPluginName]
	return ok
}

func (ak *apiKey) Subscribe(log logrus.FieldLogger, subs apic.Subscription) {
	key, err := ak.doSubscribe(log, subs)

	if err != nil {
		log.WithError(err).Error("Failed to subscribe")
		err := subs.UpdateState(apic.SubscriptionFailedToSubscribe, err.Error())
		if err != nil {
			log.WithError(err).Error("Failed to update subscription state")
		}
		return
	}

	err = subs.UpdateStateWithProperties(apic.SubscriptionActive, "", map[string]interface{}{propertyName: key})
	if err != nil {
		log.WithError(err).Error("Failed to update subscription state")
	}
}

func (ak *apiKey) doSubscribe(log logrus.FieldLogger, subs apic.Subscription) (string, error) {
	key := subs.GetPropertyValue(propertyName)
	testapp := subs.GetPropertyValue("testprofile")
	log.Info("Got testapp: ", testapp)

	agentTag := "amplify-agent"
	ctx := context.Background()
	routeID := subs.GetRemoteAPIID()
	subscriptionID := subs.GetID()
	subscriptionName := subs.GetName() + "_" + subscriptionID
	apicID := subs.GetApicID()

	route, err := ak.kc.Routes.Get(ctx, &routeID)
	if err != nil {
		return "", fmt.Errorf("failed to get route: %w", err)
	}
	consumerTags := []*string{&apicID, &agentTag}

	ep, err := ak.kc.GetKongPlugins().GetEffectivePlugins(*route.ID, *route.Service.ID)
	if err != nil {
		return "", fmt.Errorf("failed to list route plugins: %w", err)
	}

	consumerRes, update, err := ak.kc.GetKongConsumers().CreateOrUpdateConsumer(subscriptionName, subscriptionID, consumerTags)
	if err != nil {
		return "", err
	}

	if update {
		ak.deleteAllKeys(*consumerRes.ID, subscriptionID)
	}

	keyAuth := &kong.KeyAuth{
		Tags: consumerTags,
	}
	// generate key if not provided
	if key != "" {
		keyAuth.Key = &key
	}

	keyAuthRes, err := ak.kc.KeyAuths.Create(ctx, consumerRes.ID, keyAuth)
	if err != nil {
		return "", fmt.Errorf("failed to create API Key: %w", err)
	}

	acl, ok := ep["acl"]
	if !ok {
		log.Warn("ACL Plugin is not configured on route / service")
		return *keyAuthRes.Key, nil
	}

	// add group
	if groups, ok := acl.Config["allow"].([]interface{}); ok {
		group := findACLGroup(groups)

		if group == "" {
			return "", fmt.Errorf("failed to find suitable acl group")
		}

		_, err = ak.kc.ACLs.Create(ctx, consumerRes.ID, &kong.ACLGroup{Group: &group, Tags: consumerTags})
		if err != nil {
			return "", fmt.Errorf("failed to add acl group on consumer: %w", err)
		}
	} else {
		log.Warn("No restrictions on API anybody can call it")
	}

	return *keyAuthRes.Key, nil
}

func (ak *apiKey) Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) {
	log.Info("Delete apikey subscription")
	subscriptionID := subs.GetID()
	routeID := subs.GetRemoteAPIID()
	subscriptionName := subs.GetName() + "_" + subscriptionID
	ctx := context.Background()

	err := ak.kc.Consumers.Delete(ctx, &subscriptionName)
	if err != nil {
		log.WithError(err).Error("Failed to delete Consumer")
		subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to delete consumer %s: %s", routeID, err))
		return
	}

	err = subs.UpdateState(apic.SubscriptionUnsubscribed, "Toodles")
	if err != nil {
		log.WithError(err).Error("failed to update subscription state")
	}
}

func findACLGroup(groups []interface{}) string {
	for _, group := range groups {
		if groupStr, ok := group.(string); ok && strings.HasPrefix(groupStr, "amplify.") {
			return groupStr
		}
	}
	return ""
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
			return fmt.Errorf("failed to delete consumer key: %w", err)
		}
	}

	return nil
}
