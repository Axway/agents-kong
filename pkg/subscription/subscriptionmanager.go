package subscription

import (
	"strings"

	kutil "github.com/Axway/agents-kong/pkg/kong"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/subscription/apikey"
	"github.com/kong/go-kong/kong"
	"github.com/sirupsen/logrus"
)

type SubscriptionSchemaRegistry interface {
	RegisterSubscriptionSchema(schema apic.SubscriptionSchema) error
}

type SubscriptionHandlerRegistry interface {
	RegisterValidator()
}

// ConsumerInstanceGetter gets a consumer instance by id.
type ConsumerInstanceGetter interface {
	GetConsumerInstanceByID(id string) (*v1alpha1.ConsumerInstance, error)
}

type SubscriptionHandler interface {
	Name() string
	APICPolicy() string
	Schema() apic.SubscriptionSchema
	IsApplicable(map[string]*kong.Plugin) bool
	Subscribe(log logrus.FieldLogger, subs apic.Subscription)
	Unsubscribe(log logrus.FieldLogger, subs apic.Subscription)
}

// SubscriptionManager handles the subscription aspects
type SubscriptionManager struct {
	log      logrus.FieldLogger
	handlers map[string]SubscriptionHandler
	cig      ConsumerInstanceGetter
	//	plugins  *kutil.Plugins
}

func New(log logrus.FieldLogger, cig ConsumerInstanceGetter) *SubscriptionManager {
	return &SubscriptionManager{
		// set supported subscription handlers
		handlers: map[string]SubscriptionHandler{
			apikey.Name: apikey.New(),
		},
		cig: cig,
		log: log,
	}
}

func (sm *SubscriptionManager) Schemas() []apic.SubscriptionSchema {
	res := make([]apic.SubscriptionSchema, 0, len(sm.handlers))
	for _, h := range sm.handlers {
		res = append(res, h.Schema())
	}

	return res
}

func (sm *SubscriptionManager) GetEffectiveSubscriptionHandler(routeID *string, serviceID *string, plugins *kutil.Plugins) (SubscriptionHandler, error) {
	ep, err := plugins.GetEffectivePlugins(*routeID, *serviceID)
	if err != nil {
		log.Errorf("error on determine effective plugins: %s", err)
		return nil, err
	}

	builder := strings.Builder{}
	for _, p := range ep {
		builder.WriteString(*p.Name)
		builder.WriteString(", ")
	}

	for _, h := range sm.handlers {
		if h.IsApplicable(ep) {
			log.Info("Using subscription handler: ", h.Name())
			return h, nil
		}
	}
	return nil, nil
}

func (sm *SubscriptionManager) ValidateSubscription(subscription apic.Subscription) bool {
	// TODO
	return true
}

func (sm *SubscriptionManager) ProcessSubscribe(subscription apic.Subscription) {
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

func (sm *SubscriptionManager) ProcessUnsubscribe(subscription apic.Subscription) {
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
