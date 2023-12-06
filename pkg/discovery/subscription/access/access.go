package access

import (
	"context"
	"strconv"

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
	AddRouteACL(ctx context.Context, routeID, allowedID string) error
	RemoveRouteACL(ctx context.Context, routeID, revokedID string) error
	AddQuota(ctx context.Context, routeID, allowedID, quotaInterval string, quotaLimit int) error
}

type accessRequest interface {
	GetApplicationDetailsValue(key string) string
	GetInstanceDetails() map[string]interface{}
	GetQuota() provisioning.Quota
}

type AccessProvisioner struct {
	ctx     context.Context
	logger  log.FieldLogger
	client  accessClient
	quota   provisioning.Quota
	routeID string
	appID   string
	hasACL  bool
}

func NewAccessProvisioner(ctx context.Context, client accessClient, request accessRequest) AccessProvisioner {
	instDetails := request.GetInstanceDetails()
	routeID := sdkUtil.ToString(instDetails[common.AttrRouteID])
	logger := log.NewFieldLogger().
		WithComponent("AccessProvisioner").
		WithPackage("access")
	hasACL, err := strconv.ParseBool(sdkUtil.ToString(instDetails[common.AttrHasACL]))
	if err != nil {
		logger.WithError(err).Error("Could not retrieve information for ACL from the request. Assuming ACL is disabled.")
	}

	a := AccessProvisioner{
		ctx:     context.Background(),
		logger:  logger,
		client:  client,
		quota:   request.GetQuota(),
		routeID: routeID,
		appID:   request.GetApplicationDetailsValue(common.AttrAppID),
		hasACL:  hasACL,
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

	if a.routeID == "" {
		a.logger.Error("could not find the route ID on the resource")
		return rs.SetMessage("route ID not found").Failed(), nil
	}

	if !a.hasACL {
		a.logger.Info("ACL plugin is disabled or not existing for current spec. Skipping access request provisioning")
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

	if a.appID == "" {
		a.logger.Error("could not find the managed application ID on the resource")
		return rs.SetMessage("managed application ID not found").Failed()
	}

	if a.routeID == "" {
		a.logger.Error("could not find the route ID on the resource")
		return rs.SetMessage("route ID not found").Failed()
	}

	if !a.hasACL {
		a.logger.Info("ACL plugin is disabled or not existing for current spec. Skipping access request deprovisioning")
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
