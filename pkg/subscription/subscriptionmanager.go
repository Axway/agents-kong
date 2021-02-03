package subscription

import (
	"strings"
	"sync"

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

// SubscriptionGetter gets the all the subscription in any of the states for the catalog item with id
type SubscriptionGetter interface {
	GetSubscriptionsForCatalogItem(states []string, id string) ([]apic.CentralSubscription, error)
}

type SubscriptionHandler interface {
	Name() string
	APICPolicy() string
	Schema() apic.SubscriptionSchema
	IsApplicable(map[string]*kong.Plugin) bool
	Subscribe(log logrus.FieldLogger, subs apic.Subscription)
	Unsubscribe(log logrus.FieldLogger, subs apic.Subscription)
}

type duplicateGuard struct {
	cache map[string]interface{}
	lock  *sync.Mutex
}

// SubscriptionManager handles the subscription aspects
type SubscriptionManager struct {
	log      logrus.FieldLogger
	handlers map[string]SubscriptionHandler
	cig      ConsumerInstanceGetter
	sg       SubscriptionGetter
	plugins  *kutil.Plugins
	dupGuard *duplicateGuard
}

func New(
	log logrus.FieldLogger,
	cig ConsumerInstanceGetter,
	sg SubscriptionGetter,
	kc *kong.Client) *SubscriptionManager {
	return &SubscriptionManager{
		handlers: map[string]SubscriptionHandler{
			apikey.Name: apikey.New(kc, &kutil.Plugins{PluginLister: kc.Plugins}),
		},
		cig: cig,
		sg:  sg,
		log: log,
		// TODO don't need this inside SubscriptionManager
		plugins: &kutil.Plugins{PluginLister: kc.Plugins},
		dupGuard: &duplicateGuard{
			cache: map[string]interface{}{},
			lock:  &sync.Mutex{},
		},
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

// markActive returns
func (dg *duplicateGuard) markActive(id string) bool {
	dg.lock.Lock()
	defer dg.lock.Unlock()
	if _, ok := dg.cache[id]; ok {
		return true
	}

	dg.cache[id] = true

	return false
}

// markActive returns
func (dg *duplicateGuard) markInactive(id string) bool {
	dg.lock.Lock()
	defer dg.lock.Unlock()

	delete(dg.cache, id)
	return false
}

func (sm *SubscriptionManager) checkSubscriptionState(subscriptionID, catalogItemID, subscriptionState string) (bool, error) {

	subs, err := sm.sg.GetSubscriptionsForCatalogItem([]string{string(subscriptionState)}, catalogItemID)
	if err != nil {
		return false, err
	}

	for _, sub := range subs {
		if sub.GetID() == subscriptionID {
			return true, nil
		}
	}

	return false, nil
}

func (sm *SubscriptionManager) ProcessSubscribe(subscription apic.Subscription) {
	log := sm.log.
		WithField("subscriptionID", subscription.GetID()).
		WithField("catalogItemID", subscription.GetCatalogItemID()).
		WithField("remoteID", subscription.GetRemoteAPIID()).
		WithField("consumerInstanceID", subscription.GetApicID())

	if sm.dupGuard.markActive(subscription.GetID()) {
		sm.log.Info("duplicate subscription event; already handling subscription")
		return
	}
	defer sm.dupGuard.markInactive(subscription.GetID())
	isApproved, err := sm.checkSubscriptionState(subscription.GetID(), subscription.GetCatalogItemID(), string(apic.SubscriptionApproved))
	if err != nil {
		log.WithError(err).Error("Failed to verify subscription state")
		return
	}

	if !isApproved {
		log.Info("Subscription not in approved state. Nothing to do")
		return
	}

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

	if sm.dupGuard.markActive(subscription.GetID()) {
		sm.log.Info("duplicate subscription event; already handling subscription")
	}
	defer sm.dupGuard.markInactive(subscription.GetID())
	isUnsubscribeInitiated, err := sm.checkSubscriptionState(subscription.GetID(), subscription.GetCatalogItemID(), string(apic.SubscriptionUnsubscribeInitiated))
	if err != nil {
		log.WithError(err).Error("Failed to verify subscription state")
		return
	}

	if !isUnsubscribeInitiated {
		log.Info("Subscription not in unsubscribe initiated state. Nothing to do")
		return
	}

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
