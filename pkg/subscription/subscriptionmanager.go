package subscription

import (
	"strings"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	kutil "github.com/Axway/agents-kong/pkg/kong"
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
	plugins  *kutil.Plugins
}

func New(
	log logrus.FieldLogger,
	cig ConsumerInstanceGetter,
	pl kutil.PluginLister) *SubscriptionManager {
	return &SubscriptionManager{
		handlers: map[string]SubscriptionHandler{
			apikey.Name: apikey.New(),
		},
		cig: cig,
		log: log,
		// TODO don't need this inside SubscriptionManager
		plugins: &kutil.Plugins{PluginLister: pl},
	}
}

func (sm *SubscriptionManager) Schemas() []apic.SubscriptionSchema {
	res := make([]apic.SubscriptionSchema, 0, len(sm.handlers))
	for _, h := range sm.handlers {
		res = append(res, h.Schema())
	}

	return res
}

// PopulateSubscriptionParameters indentifies the subscription type to use and populates sb with
// the appropriate parameters
func (sm *SubscriptionManager) PopulateSubscriptionParameters(routeID, serviceID string, sb *apic.ServiceBody) error {
	log := sm.log.WithField("routeID", routeID).
		WithField("serviceID", serviceID)

	log.Info("Populating subscription parameters")
	ep, err := sm.plugins.GetEffectivePlugins(routeID, serviceID)
	builder := strings.Builder{}
	for k := range ep {
		builder.WriteString(k)
		builder.WriteString(", ")
	}

	log.Info("Got plugins: ", builder.String())
	if err != nil {
		return err
	}

	for _, h := range sm.handlers {
		if h.IsApplicable(ep) {
			log.Info("Using subscription handler: ", h.Name())
			sb.AuthPolicy = h.APICPolicy()
			sb.SubscriptionName = h.Name()
			return nil
		}
	}

	log.Info("No subscription handler")

	return nil
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
