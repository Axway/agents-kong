package apikey

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agents-kong/pkg/kong"
	klib "github.com/kong/go-kong/kong"
	"github.com/sirupsen/logrus"
)

type APIKey struct {
	kclient   *klib.Client
	kutil     *kong.Plugins
	kongURL   string
	kongToken string
}

const Name = "kong-apikey"

const (
	keyPluginName = "key-auth"
	propertyName  = "api-key"
)

func New(kc *klib.Client, kutil *kong.Plugins, kongURL, kongToken string) *APIKey {
	return &APIKey{kc, kutil, kongURL, kongToken}
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
func (*APIKey) IsApplicable(plugins map[string]*klib.Plugin) bool {
	_, ok := plugins[keyPluginName]
	return ok
}

func (ak *APIKey) Subscribe(log logrus.FieldLogger, subs apic.Subscription) {
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
	route, err := ak.kclient.Routes.Get(ctx, &routeID)
	if err != nil {
		log.WithError(err).Error("Failed to get route")
		subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("failed to get route %s: %s", routeID, err))
		return
	}
	log.Info("route: %v", route)
	consumerTags := []*string{&apicId, &agentTag}
	// Create Kong Consumer
	consumer := &klib.Consumer{
		Username: &subscriptionName,
		CustomID: &subscriptionId,
		Tags:     consumerTags,
	}

	consumerRes, err := ak.kclient.Consumers.Get(ctx, &subscriptionName)
	update := false
	if err != nil {
		log.Info("Conusmer is not exist on Kong gateway, hence creating a new consumer")
		consumerRes, err = ak.kclient.Consumers.Create(ctx, consumer)
		if err != nil {
			log.WithError(err).Error("Failed to create consumer")
			subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("failed to create Kong consumer %s: %s", routeID, err))
			return
		}
	} else {
		log.Info("Consuber %s exists on Kong gateway", consumerRes.Username)
		update = true
	}

	//ak.kutil.
	//plugins := ak.kutil.Plugins{ak.kclient.Plugins}
	ep, err := ak.kutil.GetEffectivePlugins(*route.ID, *route.Service.ID)
	log.Info("Plugins %v", ep)
	acl, ok := ep["acl"]
	if !ok {
		log.Error("acl Plugin is not configured on route / service")
		return
	}

	allowedGroup, ok := acl.Config["allow"].([]interface{})
	if !ok {
		log.Error("No restrictions on API anybody can call it")

	} else {
		for _, group := range allowedGroup {
			grouptStr := fmt.Sprintf("%v", group)
			if strings.HasPrefix(grouptStr, "apic") {
				acl := &klib.ACLGroup{
					Group: &grouptStr,
					Tags:  consumerTags,
				}
				aclRes, err := ak.kclient.ACLs.Create(ctx, consumerRes.ID, acl)
				if err != nil {
					log.WithError(err).Error("Group already exists / Failed to create ACL under consumer")
					subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to create ACL under consumer %s: %s", routeID, err))
					return
				}
				log.Info("acl: %v", aclRes)
				break
			}
		}
	}
	keyAuth := &klib.KeyAuth{
		Key:  &key,
		Tags: consumerTags,
	}
	if update {
		keyAuthId := searchKeyAuthbyTag(ak.kongURL, ak.kongToken, *consumerRes.ID, subscriptionId)
		if keyAuthId != "" {
			log.Info("Deleting existing api key")
			err = ak.kclient.KeyAuths.Delete(ctx, consumerRes.ID, &keyAuthId)
			if err != nil {
				log.WithError(err).Error("Failed to delete Consumer")
				subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to delete Consumer %s: %s", routeID, err))
				return
			}
		}
	}
	keyAuthRes, err := ak.kclient.KeyAuths.Create(ctx, consumerRes.ID, keyAuth)
	if err != nil {
		log.WithError(err).Error("Failed to create API Key")
		subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to create API Key %s: %s", routeID, err))
		return
	}
	log.Info("keyauth: %v", keyAuthRes)
	err = subs.UpdateStateWithProperties(apic.SubscriptionActive, "Toodles", map[string]interface{}{propertyName: keyAuthRes.Key})

	if err != nil {
		log.WithError(err).Error("failed to update subscription state")
	}
}

func (ak *APIKey) Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) {
	log.Info("Delete apikey subscription")
	subscriptionId := subs.GetID()
	routeID := subs.GetRemoteAPIID()
	subscriptionName := subs.GetName() + "_" + subscriptionId
	ctx := context.Background()

	err := ak.kclient.Consumers.Delete(ctx, &subscriptionName)
	if err != nil {
		log.WithError(err).Error("Failed to delete Consumer")
		subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to create API Key %s: %s", routeID, err))
		return
	}
	//subs.UpdateState(apic.SubscriptionUnsubscribed, "Toodles")
	err = subs.UpdateStateWithProperties(apic.SubscriptionUnsubscribed, "Toodles", map[string]interface{}{propertyName: ""})
	if err != nil {
		log.WithError(err).Error("failed to update subscription state")
	}
}

func searchKeyAuthbyTag(token, kongURL, consumerId, tag string) string {

	clientBase := &http.Client{}
	defaultTransport := http.DefaultTransport.(*http.Transport)
	clientBase.Transport = defaultTransport
	headers := make(http.Header)
	if token != "" {
		headers.Set("Kong-Admin-Token", token)
	}
	urlObj, _ := url.Parse(kongURL + "/consumers" + "/" + consumerId + "/key-auth?tags=" + tag)
	fmt.Println(urlObj)

	req := &http.Request{
		Method: "GET",
		URL:    urlObj,
		Header: headers,
	}
	res, _ := http.DefaultClient.Do(req)
	body, err := ioutil.ReadAll(res.Body)
	resKeyAuthId := ""
	if err != nil {
		type list struct {
			Data []json.RawMessage `json:"data"`
		}
		resList := list{}
		json.Unmarshal(body, &resList)
		for _, object := range resList.Data {
			b, err := object.MarshalJSON()
			fmt.Println(err)
			var keyAuth klib.KeyAuth
			err = json.Unmarshal(b, &keyAuth)
			return *keyAuth.ID
		}
	}
	return resKeyAuthId
}
