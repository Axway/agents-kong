package access

import (
	"context"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	sdkUtil "github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"
)

const (
	logFieldAppID     = "appID"
	logFieldAppName   = "appName"
	logFieldServiceID = "serviceID"
	logFieldRouteID   = "routeID"
)

type accessClient interface {
	AddManagedAppACL(ctx context.Context, managedAppID, routeID string) error
	RemoveManagedAppACL(ctx context.Context, serviceID, routeID, managedAppID string) error
	AddQuota(ctx context.Context, serviceID, managedAppID, quotaInterval string, quotaLimit int) error
}

type accessRequest interface {
	GetApplicationDetailsValue(key string) string
	GetInstanceDetails() map[string]interface{}
	GetQuota() provisioning.Quota
}

type AccessProvisioner struct {
	ctx       context.Context
	logger    log.FieldLogger
	client    accessClient
	quota     provisioning.Quota
	routeID   string
	serviceID string
	appID     string
}

func NewAccessProvisioner(ctx context.Context, client accessClient, request accessRequest) AccessProvisioner {
	instDetails := request.GetInstanceDetails()
	serviceID := sdkUtil.ToString(instDetails[common.AttrServiceId])
	routeID := sdkUtil.ToString(instDetails[common.AttrRouteId])

	a := AccessProvisioner{
		ctx: context.Background(),
		logger: log.NewFieldLogger().
			WithComponent("AccessProvisioner").
			WithPackage("access"),
		client:    client,
		quota:     request.GetQuota(),
		routeID:   routeID,
		serviceID: serviceID,
		appID:     request.GetApplicationDetailsValue(common.AttrAppID),
	}

	if a.serviceID != "" {
		a.logger = a.logger.WithField(logFieldServiceID, a.serviceID)
	}
	if a.routeID != "" {
		a.logger = a.logger.WithField(logFieldRouteID, a.routeID)
	}
	if a.appID != "" {
		a.logger = a.logger.WithField(logFieldAppID, a.appID)
	}
	return a
}

func (a AccessProvisioner) Provision() (provisioning.RequestStatus, provisioning.AccessData) {
	a.logger.Info("provisioning access")

	rs := provisioning.NewRequestStatusBuilder()
	if a.appID == "" {
		a.logger.Error("could not find the managed application ID on the resource")
		return rs.SetMessage("managed application ID not found").Failed(), nil
	}

	if a.quota.GetInterval().String() == provisioning.Weekly.String() {
		a.logger.Debug("weekly quota interval is not supported")
		return rs.SetMessage("weekly quota is not supported by kong").Failed(), nil
	}

	err := a.client.AddManagedAppACL(a.ctx, a.appID, a.routeID)
	if err != nil {
		a.logger.WithError(err).Error("failed to provide access to managed application")
		return rs.SetMessage("could not provide access to consumer in kong").Failed(), nil
	}

	quotaInterval := a.quota.GetIntervalString()
	quotaLimit := int(a.quota.GetLimit())
	err = a.client.AddQuota(a.ctx, a.serviceID, a.appID, quotaInterval, quotaLimit)
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
	if a.appID == "" {
		a.logger.Error("could not find the managed application ID on the resource")
		return rs.SetMessage("managed application ID not found").Failed()
	}

	err := a.client.RemoveManagedAppACL(a.ctx, a.serviceID, a.routeID, a.appID)
	if err != nil {
		a.logger.WithError(err).Error("failed to remove managed app from ACL")
		return rs.SetMessage("could not remove consumer from ACL").Failed()
	}

	a.logger.Info("deprovisioned access")
	return rs.Success()
}
