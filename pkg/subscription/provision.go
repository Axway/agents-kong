package subscription

import (
	"context"
	"fmt"

	klib "github.com/kong/go-kong/kong"
	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	sdkUtil "github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agents-kong/pkg/common"
	"github.com/Axway/agents-kong/pkg/kong"
	"github.com/Axway/agents-kong/pkg/subscription/application"
	"github.com/Axway/agents-kong/pkg/subscription/credential"
)

type provisioner struct {
	logger log.FieldLogger
	client kong.KongAPIClient
	kc     *klib.Client
}

// NewProvisioner creates a type to implement the SDK Provisioning methods for handling subscriptions
func NewProvisioner(client kong.KongAPIClient, logger log.FieldLogger) {
	logger.Info("Registering provisioning callbacks")
	provisioner := &provisioner{
		client: client,
		logger: logger,
	}
	agent.RegisterProvisioner(provisioner)
	registerOauth2()
	registerBasicAuth()
	registerKeyAuth()
}

func (p provisioner) AccessRequestProvision(request provisioning.AccessRequest) (provisioning.RequestStatus, provisioning.AccessData) {
	p.logger.Info("provisioning access request")
	agentTag := "amplify-agent"
	ctx := context.Background()
	rs := provisioning.NewRequestStatusBuilder()
	instDetails := request.GetInstanceDetails()
	serviceId := sdkUtil.ToString(instDetails[common.AttrServiceId])
	routeId := sdkUtil.ToString(instDetails[common.AttrRouteId])
	if serviceId == "" {
		return Failed(rs, notFound(common.AttrServiceId)), nil
	}
	if routeId == "" {
		return Failed(rs, notFound(common.AttrRouteId)), nil
	}
	kongApplicationId := request.GetApplicationDetailsValue(common.AttrAppID)
	if kongApplicationId == "" {
		return Failed(rs, fmt.Errorf("kong application id not set")), nil
	}
	plugins := kong.Plugins{PluginLister: p.kc.Plugins}
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
	group := common.AclGroup
	consumerTags := []*string{&agentTag}
	_, err = p.kc.ACLs.Create(ctx, &kongApplicationId, &klib.ACLGroup{Group: &group, Tags: consumerTags})
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
		p.logger.Info(" Plan name :%s, Plan Description :%s Quota Limit: %s", planName, planDesc, quotaLimit)
		config := klib.Configuration{
			"limit":  []interface{}{quotaLimit},
			"policy": "local",
		}
		p.logger.Info("%v", config)
		//err := addRateLimit(p.kc, ctx, config, "")
		//if err != nil {
		//	return nil, nil
		//}

		//amplifyQuota.GetInterval().
	}
	p.logger.
		WithField("api", serviceId).
		WithField("app", request.GetApplicationName()).
		Info("granted access")
	return rs.Success(), nil
}
func (p provisioner) AccessRequestDeprovision(request provisioning.AccessRequest) provisioning.RequestStatus {
	p.logger.Info("deprovisioning access request")
	ctx := context.Background()
	rs := provisioning.NewRequestStatusBuilder()
	instDetails := request.GetInstanceDetails()
	serviceId := sdkUtil.ToString(instDetails[common.AttrServiceId])
	routeId := sdkUtil.ToString(instDetails[common.AttrRouteId])
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
	err := p.kc.ACLs.Delete(ctx, &kongConsumerId, &group)
	if err != nil {
		return Failed(rs, fmt.Errorf("failed to remove acl group on consumer: %w", err))
	}
	p.logger.
		WithField("api", serviceId).
		WithField("app", request.GetApplicationName()).
		Info("removed access")
	return rs.Success()
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

//func addRateLimit(kc *klib.Client, ctx context.Context, config map[string]interface{}, serviceId string) error {
//	pluginName := "rate-limiting"
//	rateLimitPlugin := klib.Plugin{
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

func (p provisioner) ApplicationRequestProvision(request provisioning.ApplicationRequest) provisioning.RequestStatus {
	return application.NewApplicationProvisioner(context.Background(), p.client, request).Provision()
}

func (p provisioner) ApplicationRequestDeprovision(request provisioning.ApplicationRequest) provisioning.RequestStatus {
	return application.NewApplicationProvisioner(context.Background(), p.client, request).Deprovision()
}

func (p provisioner) CredentialProvision(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {
	return credential.NewCredentialProvisioner(context.Background(), p.client, request).Provision()
}

func (p provisioner) CredentialDeprovision(request provisioning.CredentialRequest) provisioning.RequestStatus {
	return credential.NewCredentialProvisioner(context.Background(), p.client, request).Deprovision()
}

func (p provisioner) CredentialUpdate(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {
	return credential.NewCredentialProvisioner(context.Background(), p.client, request).Update()
}
