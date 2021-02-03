package apikey

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/kong/go-kong/kong"
	"github.com/sirupsen/logrus"
)

type APIKey struct {
	log logrus.FieldLogger
}

const Name = "kong-apikey"

const (
	keyPluginName = "key-auth"
	propertyName  = "api-key"
)

func New() *APIKey {
	return &APIKey{}
}

func (*APIKey) Name() string {
	return Name
}

func (*APIKey) APICPolicy() string {
	return apic.Apikey
}

// Schema returns the schema
func (*APIKey) Schema() apic.SubscriptionSchema {
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
func (*APIKey) IsApplicable(plugins map[string]*kong.Plugin) bool {
	_, ok := plugins[keyPluginName]
	return ok
}

func (*APIKey) Subscribe(log logrus.FieldLogger, subs apic.Subscription) {
	key := subs.GetPropertyValue(propertyName)
	if key != "" {
		log.Info("got subscription with key: ", key)
	} else {
		log.Info("will generate key")
	}

	subs.UpdateState(apic.SubscriptionActive, "Toodles")
}

func (*APIKey) Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) {
	// TODO
	subs.UpdateState(apic.SubscriptionUnsubscribed, "Toodles")
}
