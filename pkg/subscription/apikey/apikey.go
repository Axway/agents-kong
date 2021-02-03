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
	kc *kong.Client
}

const Name = "kong-apikey"

const (
	keyPluginName = "key-auth"
	propertyName  = "api-key"
)

func init() {
	subscription.Register(func(kc *kong.Client) subscription.Handler {
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
	key := subs.GetPropertyValue(propertyName)
	if key != "" {
		log.Info("got subscription with key: ", key)
	} else {
		log.Info("will generate key")
	}

	agentTag := "amplify-agent"
	ctx := context.Background()
	routeID := subs.GetRemoteAPIID()
	subscriptionId := subs.GetID()
	subscriptionName := subs.GetName() + "_" + subscriptionId
	apicId := subs.GetApicID()
	amplifyUserid := subs.GetCreatedUserID()
	route, err := ak.kc.Routes.Get(ctx, &routeID)
	if err != nil {
		log.WithError(err).Error("Failed to get route")
		subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("failed to get route %s: %s", routeID, err))
		return
	}
	log.Info("route: %v", route)
	consumerTags := []*string{&apicId, &agentTag}
	// Create Kong Consumer
	consumer := &kong.Consumer{
		Username: &subscriptionName,
		CustomID: &amplifyUserid,
		Tags:     consumerTags,
	}
	consumerRes, err := ak.kc.Consumers.Create(ctx, consumer)
	if err != nil {
		log.WithError(err).Error("Failed to create consumer")
		subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("failed to create Kong consumer %s: %s", routeID, err))

		return
	}

	//ak.kutil.
	plugins := kutil.Plugins{ak.kc.Plugins}
	ep, err := plugins.GetEffectivePlugins(*route.ID, *route.Service.ID)
	log.Info("Plugins %v", ep)
	acl, ok := ep["acl"]
	if !ok {
		log.Error("acl Plugin is not configured on route / service")
		return
	}

	allowedGroup, ok := acl.Config["allow"].([]interface{})
	if !ok {
		log.Error("No restrictions anybody can call API")

	} else {

		for _, group := range allowedGroup {
			grouptStr := fmt.Sprintf("%v", group)
			if strings.HasPrefix(grouptStr, "apic") {
				acl := &kong.ACLGroup{
					Group: &grouptStr,
					Tags:  consumerTags,
				}
				aclRes, err := ak.kc.ACLs.Create(ctx, consumerRes.ID, acl)
				if err != nil {
					log.WithError(err).Error("Failed to create ACL under consumer")
					subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to create ACL under consumer %s: %s", routeID, err))
					return
				}
				log.Info("acl: %v", aclRes)
				break
			}
		}
	}
	// create consumer and tag
	// create apikey

	keyAuth := &kong.KeyAuth{
		Key:  &key,
		Tags: consumerTags,
	}
	keyAuthRes, err := ak.kc.KeyAuths.Create(ctx, consumerRes.ID, keyAuth)

	if err != nil {
		log.WithError(err).Error("Failed to create API Key")
		subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to create API Key %s: %s", routeID, err))
		return
	}
	log.Info("keyauth: %v", keyAuthRes)
	err = subs.UpdateStateWithProperties(apic.SubscriptionActive, "Toodles", map[string]interface{}{propertyName: "this is your key now"})
	if err != nil {
		log.WithError(err).Error("failed to update subscription state")
	}
}

func (ak *apiKey) Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) {
	log.Info("Delete apikey subscription")
	subscriptionId := subs.GetID()
	routeID := subs.GetRemoteAPIID()
	subscriptionName := subs.GetName() + "_" + subscriptionId
	ctx := context.Background()

	err := ak.kc.Consumers.Delete(ctx, &subscriptionName)
	if err != nil {
		log.WithError(err).Error("Failed to delete Consumer")
		subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to create API Key %s: %s", routeID, err))
		return
	}
	//subs.UpdateState(apic.SubscriptionUnsubscribed, "Toodles")
	err = subs.UpdateStateWithProperties(apic.SubscriptionUnsubscribed, "Toodles", map[string]interface{}{propertyName: "this is your key now"})
	if err != nil {
		log.WithError(err).Error("failed to update subscription state")
	}
}
