package kong

import (
	"context"
	"fmt"
	"strings"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agents-kong/pkg/common"
	klib "github.com/kong/go-kong/kong"
	"github.com/mitchellh/mapstructure"
)

type ACLConfig struct {
	AllowedGroups    []string `json:"allow,omitempty" yaml:"allow,omitempty"`
	DeniedGroups     []string `json:"deny,omitempty" yaml:"deny,omitempty"`
	HideGroupsHeader bool     `json:"hide_groups_header" yaml:"hide_groups_header"`
}

func (k KongClient) CreateConsumer(ctx context.Context, id, name string) (*klib.Consumer, error) {
	// validate that the consumer does not already exist
	log := k.logger.WithField("consumerID", id).WithField("consumerName", name)
	consumer, err := k.Consumers.Get(ctx, klib.String(id))
	if err == nil {
		log.Debug("found existing consumer")
		return consumer, err
	}

	log.Debug("creating new consumer")
	consumer, err = k.Consumers.Create(ctx, &klib.Consumer{
		CustomID: klib.String(id),
		Username: klib.String(name),
	})
	if err != nil {
		log.WithError(err).Error("creating consumer")
		return nil, err
	}

	return consumer, nil
}

func (k KongClient) AddConsumerACL(ctx context.Context, id string) error {
	log := k.logger.WithField("consumerID", id)
	consumer, err := k.Consumers.Get(ctx, klib.String(id))
	if err != nil {
		log.Debug("could not find consumer")
		return err
	}

	log.Debug("adding consumer acl")
	_, err = k.ACLs.Create(ctx, consumer.ID, &klib.ACLGroup{
		Consumer: consumer,
		Group:    klib.String(id),
	})

	if err != nil {
		log.WithError(err).Error("adding acl to consumer")
		return err
	}
	return nil
}

func (k KongClient) DeleteConsumer(ctx context.Context, id string) error {
	// validate that the consumer has not already been removed
	log := k.logger.WithField("consumerID", id)
	_, err := k.Consumers.Get(ctx, klib.String(id))
	if err != nil {
		log.Debug("could not find consumer")
		return nil
	}

	log.Debug("deleting consumer")
	return k.Consumers.Delete(ctx, klib.String(id))
}

func (k KongClient) DeleteOauth2(ctx context.Context, consumerID, clientID string) error {
	if err := k.Oauth2Credentials.Delete(ctx, &consumerID, &clientID); err != nil {
		k.logger.Errorf("failed to delete oauth2 credential with clientID: %s for consumerID: %s. Reason: %w", clientID, consumerID, err)
		return err
	}
	return nil
}

func (k KongClient) DeleteHttpBasic(ctx context.Context, consumerID, username string) error {
	if err := k.BasicAuths.Delete(ctx, &consumerID, &username); err != nil {
		k.logger.Errorf("failed to delete http-basic credential for user: %s for consumerID %s. Reason: %w", username, consumerID, err)
		return err
	}
	return nil
}

func (k KongClient) DeleteAuthKey(ctx context.Context, consumerID, authKey string) error {
	if err := k.KeyAuths.Delete(ctx, &consumerID, &authKey); err != nil {
		k.logger.Errorf("failed to delete API Key: %s for consumerID %s. Reason: %w", authKey, consumerID, err)
		return err
	}
	return nil
}

func (k KongClient) CreateHttpBasic(ctx context.Context, consumerID string, basicAuth *klib.BasicAuth) (*klib.BasicAuth, error) {
	basicAuth, err := k.BasicAuths.Create(ctx, &consumerID, basicAuth)
	if err != nil {
		k.logger.Errorf("failed to create http-basic credential for consumerID %s. Reason: %w", consumerID, err)
		return nil, err
	}
	return basicAuth, nil
}

func (k KongClient) CreateOauth2(ctx context.Context, consumerID string, oauth2 *klib.Oauth2Credential) (*klib.Oauth2Credential, error) {
	oauth2, err := k.Oauth2Credentials.Create(ctx, &consumerID, oauth2)
	if err != nil {
		k.logger.Errorf("failed to create oauth2 credential for consumerID %s. Reason: %w", consumerID, err)
		return nil, err
	}
	return oauth2, nil
}

func (k KongClient) CreateAuthKey(ctx context.Context, consumerID string, keyAuth *klib.KeyAuth) (*klib.KeyAuth, error) {
	keyAuth, err := k.KeyAuths.Create(ctx, &consumerID, keyAuth)
	if err != nil {
		k.logger.Errorf("failed to create oauth2 credential for consumerID %s. Reason: %w", consumerID, err)
		return nil, err
	}
	return keyAuth, nil
}

func (k KongClient) AddRouteACL(ctx context.Context, routeID, allowedID string) error {
	log := k.logger.WithField("consumerID", allowedID).WithField("routeID", routeID)
	plugins, err := k.Plugins.ListAll(ctx)
	if err != nil {
		log.WithError(err).Error("failed to get plugins")
		return err
	}

	log = log.WithField("plugin", common.AclPlugin)
	aclPlugin, err := getSpecificPlugin(plugins, "", routeID, "", common.AclPlugin)
	if err != nil {
		log.WithError(err).Debug("no acl for route")
		aclConfig := ACLConfig{
			AllowedGroups: []string{allowedID},
		}
		err = k.createACL(ctx, aclConfig, routeID)
		if err != nil {
			log.WithError(err).Error("failed to create acl")
			return err
		}

		log.Info("acl created, access granted")
		return nil
	}

	// verify if access is granted
	var aclCfg ACLConfig
	_, hasAccess := aclCfg.checkAccess(aclPlugin, allowedID)
	if hasAccess {
		log.Info("access is already granted")
		return nil
	}

	// provide access to managed application
	aclCfg.AllowedGroups = append(aclCfg.AllowedGroups, allowedID)
	err = k.updateOrDeleteACL(ctx, aclPlugin, aclCfg, routeID)
	if err != nil {
		log.WithError(err).Error("failed to grant access")
		return err
	}

	log.Info("granted access")
	return nil
}

func (k KongClient) RemoveRouteACL(ctx context.Context, routeID, revokedID string) error {
	log := k.logger.WithField("consumerID", revokedID).WithField("routeID", routeID)
	plugins, err := k.Plugins.ListAll(ctx)
	if err != nil {
		log.WithError(err).Error("failed to get plugins")
		return err
	}

	log = log.WithField("plugin", common.AclPlugin)
	aclPlugin, err := getSpecificPlugin(plugins, "", routeID, "", common.AclPlugin)
	if err != nil {
		log.WithError(err).Error("failed to get plugin")
		return err
	}

	// verify if access is granted
	var aclCfg ACLConfig
	i, hasAccess := aclCfg.checkAccess(aclPlugin, revokedID)
	if !hasAccess {
		log.Info("access is already denied")
		return nil
	}

	aclCfg.AllowedGroups = append(aclCfg.AllowedGroups[:i], aclCfg.AllowedGroups[i+1:]...)
	err = k.updateOrDeleteACL(ctx, aclPlugin, aclCfg, routeID)
	if err != nil {
		log.WithError(err).Error("failed to deny access")
		return err
	}

	// disable rate limiting plugin
	log = log.WithField("plugin", common.RateLimitingPlugin)
	rateLimitingPlugin, err := getSpecificPlugin(plugins, "", routeID, revokedID, common.RateLimitingPlugin)
	if err != nil {
		log.WithError(err).Debug("no plugin to disable")
		return nil
	}

	rateLimitingPlugin.Enabled = klib.Bool(false)
	_, err = k.Plugins.UpdateForRoute(ctx, &routeID, rateLimitingPlugin)
	if err != nil {
		log.WithError(err).Error("failed to disable plugin")
		return err
	}

	return nil
}

func (k KongClient) AddQuota(ctx context.Context, routeID, managedAppID, quotaInterval string, quotaLimit int) error {
	log := k.logger.WithField("consumerID", managedAppID).WithField("routeID", routeID)
	plugins, err := k.Plugins.ListAll(ctx)
	if err != nil {
		log.WithError(err).Error("failed to get plugins")
		return err
	}

	// enable the rate limiting plugin if it already exists
	log = log.WithField("plugin", common.RateLimitingPlugin)
	rateLimitPlugin, err := getSpecificPlugin(plugins, "", routeID, managedAppID, common.RateLimitingPlugin)
	if err == nil {
		// plugin was found
		if *rateLimitPlugin.Enabled {
			log.Info("plugin already enabled")
			return nil
		} else {
			rateLimitPlugin.Enabled = klib.Bool(true)
			_, err := k.Plugins.UpdateForRoute(ctx, &routeID, rateLimitPlugin)
			if err != nil {
				log.WithError(err).Error("failed to update plugin")
				return err
			}
			return nil
		}
	}

	// create plugin
	config := setQuota(quotaInterval, quotaLimit)
	err = k.addRateLimitingPlugin(ctx, config, routeID, managedAppID)
	if err != nil {
		log.WithError(err).Error("failed to add quota")
		return err
	}

	return nil
}

// checkAccess verifies if managedApp is allowed on ACL plugin
func (acl *ACLConfig) checkAccess(aclPlugin *klib.Plugin, managedAppID string) (int, bool) {
	decodeCfg := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   acl,
		TagName:  "json",
	}
	decoder, _ := mapstructure.NewDecoder(decodeCfg)
	decoder.Decode(aclPlugin.Config)

	for i, allowed := range acl.AllowedGroups {
		if allowed == managedAppID {
			return i, true
		}
	}

	return -1, false
}

func (k KongClient) createACL(ctx context.Context, aclConfig ACLConfig, routeID string) error {
	pluginName := common.AclPlugin
	aclPlugin := &klib.Plugin{
		Name: &pluginName,
		Config: klib.Configuration{
			"allow":              aclConfig.AllowedGroups,
			"deny":               aclConfig.DeniedGroups,
			"hide_groups_header": aclConfig.HideGroupsHeader,
		},
	}

	_, err := k.Plugins.CreateForRoute(ctx, &routeID, aclPlugin)
	if err != nil {
		return err
	}

	return nil
}

func (k KongClient) updateOrDeleteACL(ctx context.Context, aclPlugin *klib.Plugin, aclConfig ACLConfig, routeID string) error {
	// delete acl if there's no allowed group
	if len(aclConfig.AllowedGroups) == 0 {
		err := k.Plugins.DeleteForRoute(ctx, &routeID, aclPlugin.ID)
		if err != nil {
			return err
		}
		return nil
	}

	aclPlugin.Config = klib.Configuration{
		"allow":              aclConfig.AllowedGroups,
		"deny":               aclConfig.DeniedGroups,
		"hide_groups_header": aclConfig.HideGroupsHeader,
	}

	// enable the plugin in case it is disabled
	aclPlugin.Enabled = klib.Bool(true)
	_, err := k.Plugins.UpdateForRoute(ctx, &routeID, aclPlugin)
	if err != nil {
		return err
	}

	return nil
}

func (k KongClient) addRateLimitingPlugin(ctx context.Context, config map[string]interface{}, routeID, managedAppID string) error {
	rateLimitPlugin := klib.Plugin{
		Name:   klib.String(common.RateLimitingPlugin),
		Config: config,
		Consumer: &klib.Consumer{
			ID: &managedAppID,
		},
	}

	_, err := k.Plugins.CreateForRoute(ctx, &routeID, &rateLimitPlugin)
	if err != nil {
		return err
	}

	return nil
}

func setQuota(quotaInterval string, quotaLimit int) klib.Configuration {
	config := klib.Configuration{
		"policy": "local",
	}

	switch strings.ToLower(quotaInterval) {
	case provisioning.Daily.String():
		config["day"] = quotaLimit
	case provisioning.Monthly.String():
		config["month"] = quotaLimit
	case provisioning.Annually.String():
		config["year"] = quotaLimit
	}

	return config
}

func getSpecificPlugin(plugins []*klib.Plugin, serviceID, routeID, consumerID, pluginName string) (*klib.Plugin, error) {
	serviceMatch, routeMatch, consumerMatch := false, false, false
	for _, plugin := range plugins {
		if *plugin.Name != pluginName {
			continue
		}

		if consumerID == "" || plugin.Consumer == nil || (plugin.Consumer != nil && *plugin.Consumer.ID == consumerID) {
			consumerMatch = true
		}

		if routeID == "" || plugin.Route == nil || (plugin.Route != nil && *plugin.Route.ID == routeID) {
			routeMatch = true
		}

		if serviceID == "" || plugin.Service == nil || (plugin.Service != nil && *plugin.Service.ID == serviceID) {
			serviceMatch = true
		}

		if serviceMatch && routeMatch && consumerMatch {
			return plugin, nil
		}
	}

	return nil, fmt.Errorf("no %s plugin found", pluginName)
}
