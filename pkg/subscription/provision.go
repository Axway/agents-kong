package subscription

import (
	"context"
	"errors"
	"fmt"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"
	kutil "github.com/Axway/agents-kong/pkg/kong"
	"github.com/kong/go-kong/kong"
	"github.com/sirupsen/logrus"
)

const aclGroup = "amplify.group"

type Info struct {
	APICPolicyName string
	SchemaName     string
}

var constructors []func(*kong.Client) Handler

func Register(constructor func(*kong.Client) Handler) {
	constructors = append(constructors, constructor)
}

type Handler interface {
	Schema() apic.SubscriptionSchema
	Name() string
	APICPolicy() string
	CreateCredential(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential)
	DeleteCredential(request provisioning.CredentialRequest) provisioning.RequestStatus
	UpdateCredential(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential)
}

type provisioner struct {
	kc       *kong.Client
	log      logrus.FieldLogger
	handlers map[string]Handler
}

func (p provisioner) CredentialUpdate(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {
	p.log.Info("provisioning credentials update")
	credentialType := request.GetCredentialType()
	if h, ok := p.handlers[credentialType]; ok {
		return h.UpdateCredential(request)
	}
	errorMsg := fmt.Sprintf("No known handler for type: %s", credentialType)
	logrus.Info(errorMsg)
	return Failed(provisioning.NewRequestStatusBuilder(), errors.New(errorMsg)), nil
}

// NewProvisioner creates a type to implement the SDK Provisioning methods for handling subscriptions
func (p provisioner) NewProvisioner(kc *kong.Client, log logrus.FieldLogger) (provisioning.Provisioning, error) {
	ctx := context.Background()
	handlers := make(map[string]Handler, len(constructors))
	for _, c := range constructors {
		h := c(kc)
		handlers[h.Name()] = h
	}

	plugins, err := kc.Plugins.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	for _, plugin := range plugins {
		if *plugin.Name == "acl" {
			if groups, ok := plugin.Config["allow"].([]interface{}); ok {
				allowedGroup := findACLGroup(groups)
				logrus.Infof("Allowed ACL group %s", allowedGroup)
				if allowedGroup == "" {
					return nil, fmt.Errorf("failed to find  acl with group value amplify.group")
				} else {
					return &provisioner{
						// set supported subscription handlers
						kc:       kc,
						handlers: handlers,
						log:      log,
					}, nil
				}

			}
		}
	}
	return nil, fmt.Errorf("failed to find  acl with group value amplify.group")

}

func (p provisioner) ApplicationRequestProvision(request provisioning.ApplicationRequest) provisioning.RequestStatus {
	p.log.Info("provisioning application")
	ctx := context.Background()
	rs := provisioning.NewRequestStatusBuilder()
	appName := request.GetManagedApplicationName()
	if appName == "" {
		return Failed(rs, notFound("managed application name"))
	}

	id := request.GetID()
	consumer := kong.Consumer{
		CustomID: &id,
		Username: &appName,
	}
	consumerResponse, err := p.kc.Consumers.Create(ctx, &consumer)
	if err != nil {
		return Failed(rs, errors.New("error creating consumer"))
	}
	// process application create
	rs.AddProperty(common.AttrAppID, *consumerResponse.ID)
	p.log.
		WithField("appName", request.GetManagedApplicationName()).
		Info("created application")

	return rs.Success()
}

func (p provisioner) ApplicationRequestDeprovision(request provisioning.ApplicationRequest) provisioning.RequestStatus {
	p.log.Info("de-provisioning application")
	ctx := context.Background()
	rs := provisioning.NewRequestStatusBuilder()
	appID := request.GetApplicationDetailsValue(common.AttrAppID)
	if appID == "" {
		return Failed(rs, notFound(common.AttrAppID))
	}
	consumerResponse, err := p.kc.Consumers.Get(ctx, &appID)
	if err != nil {
		return Failed(rs, errors.New("error getting consumer details"))
	}
	if consumerResponse == nil {
		log.Warnf("Application with id %s is already deleted", appID)
		return rs.Success()
	}
	err = p.kc.Consumers.Delete(ctx, &appID)
	if err != nil {
		return Failed(rs, errors.New("error deleting kong consumer"))
	}
	log.Infof("Application with Id %s deleted successfully on Kong", appID)
	p.log.
		WithField("appName", request.GetManagedApplicationName()).
		WithField("appID", appID).
		Info("removed application")
	return rs.Success()
}
func (p provisioner) AccessRequestProvision(request provisioning.AccessRequest) (provisioning.RequestStatus, provisioning.AccessData) {
	p.log.Info("provisioning access request")
	agentTag := "amplify-agent"
	ctx := context.Background()
	rs := provisioning.NewRequestStatusBuilder()
	instDetails := request.GetInstanceDetails()
	serviceId := util.ToString(instDetails[common.AttrServiceId])
	routeId := util.ToString(instDetails[common.AttrRouteId])
	if serviceId == "" {
		return Failed(rs, notFound(common.AttrServiceId)), nil
	}
	if routeId == "" {
		return Failed(rs, notFound(common.AttrRouteId)), nil
	}
	kongApplicationId := request.GetApplicationDetailsValue(common.AttrAppID)
	plugins := kutil.Plugins{PluginLister: p.kc.Plugins}
	ep, err := plugins.GetEffectivePlugins(routeId, serviceId)
	if err != nil {
		return Failed(rs, fmt.Errorf("failed to list route plugins: %w", err)), nil
	}
	plugin, ok := ep["acl"]
	if !ok {
		log.Infof("ACL Plugin is not configured on route / service %s", serviceId)
		_, err := p.kc.Plugins.CreateForService(ctx, &serviceId, plugin)
		if err != nil {
			return Failed(rs, fmt.Errorf("failed to add ACL pluing to service: %w", err)), nil
		}
	}
	group := fmt.Sprintf("group=%s", aclGroup)
	consumerTags := []*string{&agentTag}

	_, err = p.kc.ACLs.Create(ctx, &kongApplicationId, &kong.ACLGroup{Group: &group, Tags: consumerTags})
	if err != nil {
		return Failed(rs, fmt.Errorf("failed to add acl group on consumer: %w", err)), nil
	}

	// process access request create
	rs.AddProperty(common.AttrAppID, kongApplicationId)
	p.log.
		WithField("api", serviceId).
		WithField("app", request.GetApplicationName()).
		Info("granted access")
	return rs.Success(), nil
}
func (p provisioner) AccessRequestDeprovision(request provisioning.AccessRequest) provisioning.RequestStatus {
	p.log.Info("deprovisioning access request")
	rs := provisioning.NewRequestStatusBuilder()
	instDetails := request.GetInstanceDetails()

	serviceId := util.ToString(instDetails[common.AttrServiceId])
	routeId := util.ToString(instDetails[common.AttrRouteId])

	if serviceId == "" {
		return Failed(rs, notFound(common.AttrServiceId))
	}

	if routeId == "" {
		return Failed(rs, notFound(common.AttrRouteId))
	}

	// process access request delete
	webmethodsApplicationId := request.GetAccessRequestDetailsValue(common.AttrAppID)
	//GetApplicationDetailsValue(common.AttrAppID)
	if webmethodsApplicationId == "" {
		return Failed(rs, notFound(common.AttrAppID))
	}
	//err := p.client.UnsubscribeApplication(webmethodsApplicationId, apiID)
	//if err != nil {
	//	return Failed(rs, errors.New("Error removing API from Webmethods Application"))
	//}

	p.log.
		WithField("api", serviceId).
		WithField("app", request.GetApplicationName()).
		Info("removed access")
	return rs.Success()
}
func (p provisioner) CredentialProvision(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {

	p.log.Info("provisioning credentials")
	credentialType := request.GetCredentialType()
	if h, ok := p.handlers[credentialType]; ok {
		return h.CreateCredential(request)
	}
	errorMsg := fmt.Sprintf("No known handler for type: %s", credentialType)
	logrus.Info(errorMsg)
	return Failed(provisioning.NewRequestStatusBuilder(), errors.New(errorMsg)), nil
}
func (p provisioner) CredentialDeprovision(request provisioning.CredentialRequest) provisioning.RequestStatus {
	p.log.Info("de_provisioning credentials")

	credentialType := request.GetCredentialType()
	if h, ok := p.handlers[credentialType]; ok {
		return h.DeleteCredential(request)
	}
	errorMsg := fmt.Sprintf("No known handler for type: %s", credentialType)
	logrus.Info(errorMsg)
	return Failed(provisioning.NewRequestStatusBuilder(), errors.New(errorMsg))
}

func Failed(rs provisioning.RequestStatusBuilder, err error) provisioning.RequestStatus {
	logrus.Info("handle failed event")
	rs.SetMessage(err.Error())
	logrus.Error(err)
	return rs.Failed()
}

func notFound(msg string) error {
	return fmt.Errorf("%s not found", msg)
}

func findACLGroup(groups []interface{}) string {
	for _, group := range groups {
		if groupStr, ok := group.(string); ok && groupStr == aclGroup {
			return groupStr
		}
	}
	return ""
}