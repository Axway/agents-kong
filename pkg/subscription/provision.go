package subscription

import (
	"context"
	"errors"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"
	kutil "github.com/Axway/agents-kong/pkg/kong"
	"github.com/kong/go-kong/kong"
	"github.com/sirupsen/logrus"
)

var constructors []func(*kong.Client) Handler

func Add(constructor func(*kong.Client) Handler) {
	constructors = append(constructors, constructor)
}

type Handler interface {
	Register()
	Name() string
	CreateCredential(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential)
	DeleteCredential(request provisioning.CredentialRequest) provisioning.RequestStatus
	UpdateCredential(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential)
}

type provisioner struct {
	log      logrus.FieldLogger
	client   *kong.Client
	handlers map[string]Handler
}

// NewProvisioner creates a type to implement the SDK Provisioning methods for handling subscriptions
func NewProvisioner(client *kong.Client, log logrus.FieldLogger) {
	log.Info("Registering provisioning callbacks")
	log.Infof("Handlers : %d", len(constructors))
	handlers := make(map[string]Handler, len(constructors))
	for _, c := range constructors {
		h := c(client)
		h.Register()
		handlers[h.Name()] = h
	}
	provisioner := &provisioner{
		// set supported subscription handlers
		client:   client,
		handlers: handlers,
		log:      log,
	}
	agent.RegisterProvisioner(provisioner)
	for _, handler := range handlers {
		log.Infof("Registering authentication :%s", handler.Name())
		handler.Register()
	}
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
	if kongApplicationId == "" {
		// Using the existing application
		appName := request.GetApplicationName()
		consumer := kong.Consumer{
			CustomID: &kongApplicationId,
			Username: &appName,
		}
		var err error
		consumerResponse, err := createConsumer(p.client, consumer, ctx)
		if err != nil {
			return Failed(rs, errors.New("error creating kong consumer")), nil
		}
		kongApplicationId = *consumerResponse.ID
	}
	plugins := kutil.Plugins{PluginLister: p.client.Plugins}
	ep, err := plugins.GetEffectivePlugins(routeId, serviceId)
	if err != nil {
		return Failed(rs, fmt.Errorf("failed to list route plugins: %w", err)), nil
	}
	plugin, ok := ep["acl"]
	if !ok {
		log.Infof("ACL Plugin is not configured on route / service %s", serviceId)
		_, err := p.client.Plugins.CreateForService(ctx, &serviceId, plugin)
		if err != nil {
			return Failed(rs, fmt.Errorf("failed to add ACL pluing to service: %w", err)), nil
		}
	}
	group := common.AclGroup
	consumerTags := []*string{&agentTag}
	_, err = p.client.ACLs.Create(ctx, &kongApplicationId, &kong.ACLGroup{Group: &group, Tags: consumerTags})
	if err != nil {
		return Failed(rs, fmt.Errorf("failed to add acl group on consumer: %w", err)), nil
	}
	// process access request create
	rs.AddProperty(common.AttrAppID, kongApplicationId)
	amplifyQuota := request.GetQuota()
	if amplifyQuota != nil {
		planName := amplifyQuota.GetPlanName()
		planDesc := amplifyQuota.GetPlanName()
		quotaLimit := int(amplifyQuota.GetLimit())
		p.log.Info(" Plan name :%s, Plan Description :%s Quota Limit: %s", planName, planDesc, quotaLimit)
		config := kong.Configuration{
			"limit":  []interface{}{quotaLimit},
			"policy": "local",
		}
		p.log.Info("%v", config)
		//err := addRateLimit(p.kc, ctx, config, "")
		//if err != nil {
		//	return nil, nil
		//}

		//amplifyQuota.GetInterval().
	}
	p.log.
		WithField("api", serviceId).
		WithField("app", request.GetApplicationName()).
		Info("granted access")
	return rs.Success(), nil
}
func (p provisioner) AccessRequestDeprovision(request provisioning.AccessRequest) provisioning.RequestStatus {
	p.log.Info("deprovisioning access request")
	ctx := context.Background()
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
	kongConsumerId := request.GetAccessRequestDetailsValue(common.AttrAppID)
	//GetApplicationDetailsValue(common.AttrAppID)
	if kongConsumerId == "" {
		return Failed(rs, notFound(common.AttrAppID))
	}
	group := common.AclGroup
	err := p.client.ACLs.Delete(ctx, &kongConsumerId, &group)
	if err != nil {
		return Failed(rs, fmt.Errorf("failed to remove acl group on consumer: %w", err))
	}
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

func createConsumer(kc *kong.Client, consumer kong.Consumer, ctx context.Context) (*kong.Consumer, error) {
	consumerResponse, err := kc.Consumers.Get(ctx, consumer.CustomID)
	if err != nil {
		log.Infof("Unable to find consumer application, Response from kong %s", err.Error())
		log.Infof("Creating new application with name %s", consumer.Username)
		consumerResponse, err = kc.Consumers.Create(ctx, &consumer)
		if err != nil {
			return nil, err
		}
	} else {
		log.Infof("Using the existing consumer %s", consumer.Username)
	}
	return consumerResponse, nil
}

func GetCorsSchemaPropertyBuilder() provisioning.PropertyBuilder {
	return provisioning.NewSchemaPropertyBuilder().
		SetName(common.CorsField).
		SetLabel("Javascript Origins").
		IsArray().
		AddItem(
			provisioning.NewSchemaPropertyBuilder().
				SetName("Origins").
				IsString())
}

func GetProvisionKeyPropertyBuilder() provisioning.PropertyBuilder {
	return provisioning.NewSchemaPropertyBuilder().
		SetName(common.ProvisionKey).
		SetLabel("Provision key").
		SetRequired().
		IsString()
}

//func addRateLimit(kc *kong.Client, ctx context.Context, config map[string]interface{}, serviceId string) error {
//	pluginName := "rate-limiting"
//	rateLimitPlugin := kong.Plugin{
//		Name:   &pluginName,
//		Config: config,
//	}
//	kc.Do(ctx)
//	//_, err := kc.Consumers.C(ctx, &serviceId, &rateLimitPlugin)
//	if err != nil {
//		return err
//	}
//	return nil
//}
