package jwt

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agents-kong/pkg/clientreg"
	kutil "github.com/Axway/agents-kong/pkg/kong"
	"github.com/Axway/agents-kong/pkg/subscription"
	"github.com/kong/go-kong/kong"
	"github.com/sirupsen/logrus"
)

type jwt struct {
	kc *kutil.Client
	cr *clientreg.Client
}

const Name = "kong-jwt"

const (
	keyPluginName = "jwt"
)

func init() {
	subscription.Register(func(kc *kutil.Client) subscription.Handler {
		conf := agent.GetCentralConfig()
		u, err := url.Parse(conf.GetURL())
		if err != nil {
			panic(err)
		}

		aConf := agent.GetCentralConfig().GetAuthConfig()

		aa := auth.NewWithFlow(
			conf.GetTenantID(),
			aConf.GetPrivateKey(),
			aConf.GetPublicKey(),
			aConf.GetKeyPassword(),
			aConf.GetTokenURL(),
			aConf.GetAudience(),
			aConf.GetClientID(),
			aConf.GetTimeout())

		return &jwt{
			kc,
			clientreg.NewClient(
				u.Host,
				u.RawPath+"/api/v1",
				u.Scheme,
				&http.Client{},
				aa)}
	})
}

func (*jwt) Name() string {
	return Name
}

func (*jwt) APICPolicy() string {
	return apic.Apikey
}

// Schema returns the schema used by this type of subscription
func (*jwt) Schema() apic.SubscriptionSchema {
	schema := apic.NewSubscriptionSchema(Name)

	schema.AddProperty("application",
		"string",
		"The application to use.",
		"APIC_APPLICATION_ID",
		true,
		nil)

	schema.AddProperty("profile",
		"string",
		"Must name an existing identity profile of type jwk under the application.",
		"",
		true,
		nil)

	return schema
}

func (*jwt) IsApplicable(plugins map[string]*kong.Plugin) bool {
	_, ok := plugins[keyPluginName]
	return ok
}

func (j *jwt) doSubscribe(log logrus.FieldLogger, subs apic.Subscription) error {
	appID := subs.GetPropertyValue("application")
	profileName := subs.GetPropertyValue("profile")

	agentTag := "amplify-agent"
	ctx := context.Background()
	routeID := subs.GetRemoteAPIID()

	subscriptionID := subs.GetID()
	subscriptionName := subs.GetName() + "_" + subscriptionID
	apicID := subs.GetApicID()

	profile, err := j.cr.GetAppProfile(appID, profileName)
	if err != nil {
		return err
	}

	if profile == nil {
		return fmt.Errorf("no jwt profile named %s for application %s", profileName, appID)
	}

	consumerTags := []*string{&apicID, &agentTag}
	consumerRes, update, err := j.kc.GetKongConsumers().CreateOrUpdateConsumer(subscriptionName, subscriptionID, consumerTags)
	if err != nil {
		return fmt.Errorf("failed to create or update consumuer: %w", err)
	}

	if update {
		j.deleteAllJWTs(*consumerRes.ID, subscriptionID)
	}

	algo := "RS256"

	_, err = j.kc.JWTAuths.Create(context.Background(), consumerRes.ID, &kong.JWTAuth{
		Consumer:     consumerRes,
		Algorithm:    &algo,
		Key:          &profile.KeyID,
		RSAPublicKey: profile.PemEncodedPublicKey,
		Tags:         consumerTags,
	})

	if err != nil {
		return fmt.Errorf("failed to create jwt config: %w", err)
	}

	route, err := j.kc.Routes.Get(ctx, &routeID)
	if err != nil {
		return fmt.Errorf("failed to get route: %w", err)
	}

	ep, err := j.kc.GetKongPlugins().GetEffectivePlugins(*route.ID, *route.Service.ID)
	if err != nil {
		return fmt.Errorf("failed to list route plugins: %w", err)
	}

	acl, ok := ep["acl"]
	if !ok {
		log.Warn("ACL Plugin is not configured on route / service")
		return nil
	}

	// add group
	if groups, ok := acl.Config["allow"].([]interface{}); ok {
		group := findACLGroup(groups)

		if group == "" {
			return fmt.Errorf("failed to find suitable acl group")
		}

		_, err = j.kc.ACLs.Create(ctx, consumerRes.ID, &kong.ACLGroup{Group: &group, Tags: consumerTags})
		if err != nil {
			return fmt.Errorf("failed to add acl group on consumer: %w", err)
		}
	} else {
		log.Warn("No restrictions on API anybody can call it")
	}

	log.Infof("got profile %+v", *profile)
	return nil
}

func (j *jwt) Subscribe(log logrus.FieldLogger, subs apic.Subscription) {
	err := j.doSubscribe(log, subs)

	if err != nil {
		log.WithError(err).Error("")
		if err := subs.UpdateState(apic.SubscriptionFailedToSubscribe, err.Error()); err != nil {
			log.WithError(err).Error("Failed to update subscription state")
		}
		return
	}

	if err := subs.UpdateState(apic.SubscriptionActive, ""); err != nil {
		log.WithError(err).Error("Failed to update subscription state")
	}
}

func (j *jwt) Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) {
	subscriptionID := subs.GetID()
	routeID := subs.GetRemoteAPIID()
	subscriptionName := subs.GetName() + "_" + subscriptionID
	ctx := context.Background()

	err := j.kc.Consumers.Delete(ctx, &subscriptionName)
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

func (j *jwt) deleteAllJWTs(consumerID, subscriptionID string) error {
	ctx := context.Background()
	keys, _, err := j.kc.JWTAuths.ListForConsumer(ctx, &consumerID, &kong.ListOpt{Tags: []*string{&subscriptionID}})
	if err != nil {
		return fmt.Errorf("failed to list all consumer jwts: %w", err)
	}

	for _, k := range keys {
		err := j.kc.JWTAuths.Delete(ctx, &consumerID, k.ID)
		if err != nil {
			return fmt.Errorf("failed to delete consumer jwt: %w", err)
		}
	}

	return nil
}

func findACLGroup(groups []interface{}) string {
	for _, group := range groups {
		if groupStr, ok := group.(string); ok && strings.HasPrefix(groupStr, "amplify.") {
			return groupStr
		}
	}
	return ""
}
