package access

import (
	"context"
	"errors"

	"github.com/Axway/agent-sdk/pkg/agent"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	sdkUtil "github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"
	klib "github.com/kong/go-kong/kong"
)

const (
	logFieldAppID     = "appID"
	logFieldAppName   = "appName"
	logFieldServiceID = "serviceID"
	logFieldRouteID   = "routeID"
)

type accessClient interface {
	CreateConsumer(ctx context.Context, id, name string) (*klib.Consumer, error)
	AddConsumerACL(ctx context.Context, id string) error

	AddRouteACL(ctx context.Context, routeID, allowedID string) error
	RemoveRouteACL(ctx context.Context, routeID, revokedID string) error
	AddQuota(ctx context.Context, routeID, allowedID, quotaInterval string, quotaLimit int) error
}

type accessRequest interface {
	GetApplicationName() string
	GetApplicationDetailsValue(key string) string
	GetInstanceDetails() map[string]interface{}
	GetQuota() provisioning.Quota
}

type AccessProvisioner struct {
	ctx        context.Context
	logger     log.FieldLogger
	client     accessClient
	quota      provisioning.Quota
	workspace  string
	routeID    string
	appID      string
	appName    string
	aclDisable bool
}

func NewAccessProvisioner(ctx context.Context, client accessClient, request accessRequest, aclDisable bool) AccessProvisioner {
	instDetails := request.GetInstanceDetails()
	workspace := sdkUtil.ToString(instDetails[common.AttrWorkspaceName])
	routeID := sdkUtil.ToString(instDetails[common.AttrRouteID])
	logger := log.NewFieldLogger().
		WithComponent("AccessProvisioner").
		WithPackage("access")

	a := AccessProvisioner{
		ctx:        context.WithValue(context.Background(), common.ContextWorkspace, workspace),
		logger:     logger,
		client:     client,
		quota:      request.GetQuota(),
		workspace:  workspace,
		routeID:    routeID,
		appID:      request.GetApplicationDetailsValue(common.WksPrefixName(workspace, common.AttrAppID)),
		appName:    request.GetApplicationName(),
		aclDisable: aclDisable,
	}

	if a.routeID != "" {
		a.logger = a.logger.WithField(logFieldRouteID, a.routeID)
	}
	if a.appID != "" {
		a.logger = a.logger.WithField(logFieldAppID, a.appID)
	}
	return a
}

func (a AccessProvisioner) provisionApp() (string, error) {
	a.logger.Info("provisioning application")

	app := management.NewManagedApplication(a.appName, agent.GetAgentResource().Metadata.Scope.Name)
	ri, err := agent.GetCentralClient().GetResource(app.GetSelfLink())
	if err != nil {
		a.logger.Error("could not find the managed application resource")
		return "", errors.New("managed application not found")
	}
	app.FromInstance(ri)

	consumer, err := a.client.CreateConsumer(a.ctx, app.Metadata.ID, a.appName)
	if err != nil {
		a.logger.WithError(err).Error("error creating kong consumer")
		return "", errors.New("could not create a new consumer in kong")
	}

	err = a.client.AddConsumerACL(a.ctx, *consumer.ID)
	if err != nil {
		a.logger.WithError(err).Error("could not add acl to kong consumer")
	}

	agentDetails := sdkUtil.GetAgentDetails(app)
	if agentDetails == nil {
		agentDetails = make(map[string]interface{})
	}
	agentDetails[common.WksPrefixName(a.workspace, common.AttrAppID)] = *consumer.ID
	agent.GetCentralClient().CreateSubResource(app.ResourceMeta, map[string]interface{}{definitions.XAgentDetails: agentDetails})

	a.logger.Info("provisioned application")
	return *consumer.ID, nil
}

func (a AccessProvisioner) Provision() (provisioning.RequestStatus, provisioning.AccessData) {
	a.logger.Info("provisioning access")
	rs := provisioning.NewRequestStatusBuilder()

	if a.workspace == "" {
		a.logger.Error("could not identify the workspace for the resource")
		return rs.SetMessage("workspace not found").Failed(), nil
	}
	if a.routeID == "" {
		a.logger.Error("could not find the route ID on the resource")
		return rs.SetMessage("route ID not found").Failed(), nil
	}

	if a.appID == "" {
		appID, err := a.provisionApp()
		if err != nil {
			return rs.SetMessage(err.Error()).Failed(), nil
		}
		a.appID = appID
	}

	if a.appID == "" {
		a.logger.Error("could not find the managed application ID on the resource")
		return rs.SetMessage("managed application ID not found").Failed(), nil
	}

	if a.aclDisable {
		a.logger.Debug("ACL plugin check is disabled or not existing for current spec. Skipping access request provisioning")
		return rs.Success(), nil
	}

	if a.quota != nil && a.quota.GetInterval().String() == provisioning.Weekly.String() {
		a.logger.Debug("weekly quota interval is not supported")
		return rs.SetMessage("weekly quota is not supported by kong").Failed(), nil
	}

	err := a.client.AddRouteACL(a.ctx, a.routeID, a.appID)
	if err != nil {
		a.logger.WithError(err).Error("failed to provide access to managed application")
		return rs.SetMessage("could not provide access to consumer in kong").Failed(), nil
	}

	if a.quota == nil {
		a.logger.Info("provisioned access")
		return rs.Success(), nil
	}

	quotaInterval := a.quota.GetIntervalString()
	quotaLimit := int(a.quota.GetLimit())
	err = a.client.AddQuota(a.ctx, a.routeID, a.appID, quotaInterval, quotaLimit)
	if err != nil {
		a.logger.WithError(err).Error("failed to create quota for consumer")
		return rs.SetMessage("could not create limits for consumer in kong").Failed(), nil
	}

	a.logger.Info("provisioned access")
	return rs.Success(), nil
}

func (a AccessProvisioner) Deprovision() provisioning.RequestStatus {
	a.logger.Info("deprovisioning access")
	rs := provisioning.NewRequestStatusBuilder()

	if a.workspace == "" {
		a.logger.Error("could not identify the workspace for the resource")
		return rs.SetMessage("workspace not found").Failed()
	}

	if a.appID == "" {
		a.logger.Error("could not find the managed application ID on the resource")
		return rs.SetMessage("managed application ID not found").Failed()
	}

	if a.routeID == "" {
		a.logger.Error("could not find the route ID on the resource")
		return rs.SetMessage("route ID not found").Failed()
	}

	if a.aclDisable {
		a.logger.Debug("ACL plugin check is disabled or not existing for current spec. Skipping access request deprovisioning")
		return rs.Success()
	}

	err := a.client.RemoveRouteACL(a.ctx, a.routeID, a.appID)
	if err != nil {
		a.logger.WithError(err).Error("failed to remove managed app from ACL")
		return rs.SetMessage("could not remove consumer from ACL").Failed()
	}

	a.logger.Info("deprovisioned access")
	return rs.Success()
}
