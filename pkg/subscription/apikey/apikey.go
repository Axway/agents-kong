package apikey

import (
	"context"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/apic"
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

	routeID := subs.GetRemoteAPIID()
	route, err := ak.kclient.Routes.Get(context.Background(), &routeID)
	if err != nil {
		log.WithError(err).Error("Failed to get route")
		subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("failed to get route %s: %s", routeID, err))

		return
	}

	log.Info("route: %v", route)

	// plugins := &kutil.Plugins{ak.kclient.Plugins}

	// ep, err := plugins.GetEffectivePlugins(*route.ID, *route.Service.ID)
	// acl, ok := ep["acl"]
	// if !ok {
	// 	// log warning
	// }
	// acl.Config

	// create consumer and tag
	// create apikey

	// once is done

	subs.UpdateState(apic.SubscriptionActive, "Toodles")
}

func (*apiKey) Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) {
	// TODO
	subs.UpdateState(apic.SubscriptionUnsubscribed, "Toodles")
}
