package subscription

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/kong/go-kong/kong"
	"github.com/sirupsen/logrus"
)

var constructors = []func(*kong.Client) Handler{}

func Register(constructor func(*kong.Client) Handler) {
	constructors = append(constructors, constructor)
}

// ConsumerInstanceGetter gets a consumer instance by id.
type ConsumerInstanceGetter interface {
	GetConsumerInstanceByID(id string) (*v1alpha1.ConsumerInstance, error)
}

type Info struct {
	APICPolicyName string
	SchemaName     string
}

var defaultInfo = Info{apic.Passthrough, ""}

type Handler interface {
	Schema() apic.SubscriptionSchema
	Name() string
	APICPolicy() string
	IsApplicable(map[string]*kong.Plugin) bool
	Subscribe(log logrus.FieldLogger, subs apic.Subscription)
	Unsubscribe(log logrus.FieldLogger, subs apic.Subscription)
}

// Manager handles the subscription aspects
type Manager struct {
	log      logrus.FieldLogger
	handlers map[string]Handler
	cig      ConsumerInstanceGetter
	//	plugins  *kutil.Plugins
}

func New(log logrus.FieldLogger, cig ConsumerInstanceGetter, kc *kong.Client) *Manager {
	handlers := make(map[string]Handler, len(constructors))

	for _, c := range constructors {
		h := c(kc)
		handlers[h.Name()] = h
	}

	return &Manager{
		// set supported subscription handlers
		handlers: handlers,
		cig:      cig,
		log:      log,
	}
}

func (sm *Manager) Schemas() []apic.SubscriptionSchema {
	res := make([]apic.SubscriptionSchema, 0, len(sm.handlers))
	for _, h := range sm.handlers {
		res = append(res, h.Schema())
	}

	return res
}

// GetSubscriptionInfo returns the appropriate Info for the given set of plugins
func (sm *Manager) GetSubscriptionInfo(plugins map[string]*kong.Plugin) Info {

	for _, h := range sm.handlers {
		if h.IsApplicable(plugins) {
			return Info{APICPolicyName: h.APICPolicy(), SchemaName: h.Name()}
		}
	}
	return defaultInfo
}

func (sm *Manager) ValidateSubscription(subscription apic.Subscription) bool {
	// TODO
	return true
}

func (sm *Manager) ProcessSubscribe(subscription apic.Subscription) {
	log := sm.log.
		WithField("subscriptionID", subscription.GetID()).
		WithField("catalogItemID", subscription.GetCatalogItemID()).
		WithField("remoteID", subscription.GetRemoteAPIID()).
		WithField("consumerInstanceID", subscription.GetApicID())

	log.Info("Processing subscription")

	ci, err := sm.cig.GetConsumerInstanceByID(subscription.GetApicID())
	if err != nil {
		log.WithError(err).Error("Failed to fetch consumer instance")
		return
	}

	if h, ok := sm.handlers[ci.Spec.Subscription.SubscriptionDefinition]; ok {
		h.Subscribe(log.WithField("handler", h.Name()), subscription)
	} else {
		log.Info("No known handler for type: ", ci.Spec.Subscription.SubscriptionDefinition)
	}
}

func (sm *Manager) ProcessUnsubscribe(subscription apic.Subscription) {
	log := sm.log.
		WithField("subscriptionID", subscription.GetID()).
		WithField("catalogItemID", subscription.GetCatalogItemID()).
		WithField("remoteID", subscription.GetRemoteAPIID()).
		WithField("consumerInstanceID", subscription.GetApicID())

	log.Info("Removing subscription")

	ci, err := sm.cig.GetConsumerInstanceByID(subscription.GetApicID())
	if err != nil {
		log.WithError(err).Error("Failed to fetch consumer instance")
		return
	}

	if h, ok := sm.handlers[ci.Spec.Subscription.SubscriptionDefinition]; ok {
		h.Unsubscribe(log.WithField("handler", h.Name()), subscription)

	} else {
		log.Info("No known handler for type: ", ci.Spec.Subscription.SubscriptionDefinition)
	}
}
