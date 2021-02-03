package subscription

import (
	"sync"

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

// SubscriptionGetter gets the all the subscription in any of the states for the catalog item with id
type SubscriptionGetter interface {
	GetSubscriptionsForCatalogItem(states []string, id string) ([]apic.CentralSubscription, error)
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
	sg       SubscriptionGetter
	dg       *duplicateGuard
	//	plugins  *kutil.Plugins
}

func New(log logrus.FieldLogger,
	cig ConsumerInstanceGetter,
	sg SubscriptionGetter,
	kc *kong.Client) *Manager {
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
		sg:       sg,
		dg: &duplicateGuard{
			cache: map[string]interface{}{},
			lock:  &sync.Mutex{},
		},
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
	if sm.dg.markActive(subscription.GetID()) {
		sm.log.Info("duplicate subscription event; already handling subscription")
		return false
	}

	return true
}

type duplicateGuard struct {
	cache map[string]interface{}
	lock  *sync.Mutex
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

func (sm *Manager) checkSubscriptionState(subscriptionID, catalogItemID, subscriptionState string) (bool, error) {

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

func (sm *Manager) ProcessSubscribe(subscription apic.Subscription) {
	defer sm.dg.markInactive(subscription.GetID())
	log := sm.log.
		WithField("subscriptionID", subscription.GetID()).
		WithField("catalogItemID", subscription.GetCatalogItemID()).
		WithField("remoteID", subscription.GetRemoteAPIID()).
		WithField("consumerInstanceID", subscription.GetApicID())

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

func (sm *Manager) ProcessUnsubscribe(subscription apic.Subscription) {
	defer sm.dg.markInactive(subscription.GetID())

	log := sm.log.
		WithField("subscriptionID", subscription.GetID()).
		WithField("catalogItemID", subscription.GetCatalogItemID()).
		WithField("remoteID", subscription.GetRemoteAPIID()).
		WithField("consumerInstanceID", subscription.GetApicID())

	if sm.dg.markActive(subscription.GetID()) {
		sm.log.Info("duplicate subscription event; already handling subscription")
	}
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
