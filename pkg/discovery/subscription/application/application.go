package application

import (
	"context"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"
)

const (
	logFieldAppID      = "appID"
	logFieldAppName    = "appName"
	logFieldConsumerID = "consumerID"
)

type appClient interface {
	DeleteConsumer(ctx context.Context, id string) error
}

type appRequest interface {
	GetApplicationDetailsValue(key string) string
	// GetManagedApplicationName returns the name of the managed application for this credential
	GetManagedApplicationName() string
	// GetID returns the ID of the resource for the request
	GetID() string
}

type AppProvisioner struct {
	ctx         context.Context
	logger      log.FieldLogger
	client      appClient
	appName     string
	appID       string
	consumerIDs map[string]string
}

func NewApplicationProvisioner(ctx context.Context, client appClient, request appRequest, workspaces []string) AppProvisioner {
	a := AppProvisioner{
		ctx: context.Background(),
		logger: log.NewFieldLogger().
			WithComponent("AppProvisioner").
			WithPackage("application"),
		client:  client,
		appName: request.GetManagedApplicationName(),
		appID:   request.GetID(),
	}
	if a.appName != "" {
		a.logger = a.logger.WithField(logFieldAppName, a.appName)
	}
	if a.appID != "" {
		a.logger = a.logger.WithField(logFieldAppID, a.appID)
	}

	consumerIDs := make(map[string]string)
	for _, workspace := range workspaces {
		consumerID := request.GetApplicationDetailsValue(common.WksPrefixName(workspace, common.AttrAppID))
		if consumerID != "" {
			consumerIDs[workspace] = consumerID
		}
	}
	a.consumerIDs = consumerIDs
	return a
}

func (a AppProvisioner) Provision() provisioning.RequestStatus {
	// No op for app provisioning, consumer to represent application will
	// be created while provisioning access request
	return provisioning.NewRequestStatusBuilder().Success()
}

func (a AppProvisioner) Deprovision() provisioning.RequestStatus {
	a.logger.Info("deprovisioning application")

	rs := provisioning.NewRequestStatusBuilder()
	if len(a.consumerIDs) == 0 {
		a.logger.Error("could not identify the workspace consumer IDs for the resource")
		return rs.SetMessage("workspace not found").Failed()
	}

	for workspace, consumerID := range a.consumerIDs {
		ctx := context.WithValue(a.ctx, common.ContextWorkspace, workspace)
		err := a.client.DeleteConsumer(ctx, consumerID)
		if err != nil {
			a.logger.WithError(err).Error("error deleting kong consumer")
			return rs.SetMessage("could not remove consumer in kong").Failed()
		}
	}
	a.logger.Info("deprovisioned application")

	return rs.Success()
}
