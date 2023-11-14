package application

import (
	"context"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"

	klib "github.com/kong/go-kong/kong"
)

const (
	logFieldAppID      = "appID"
	logFieldAppName    = "appName"
	logFieldConsumerID = "consumerID"
)

type appClient interface {
	CreateConsumer(ctx context.Context, id, name string) (*klib.Consumer, error)
	AddConsumerACL(ctx context.Context, id string) error
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
	ctx        context.Context
	logger     log.FieldLogger
	client     appClient
	appName    string
	appID      string
	consumerID string
}

func NewApplicationProvisioner(ctx context.Context, client appClient, request appRequest) AppProvisioner {
	a := AppProvisioner{
		ctx: context.Background(),
		logger: log.NewFieldLogger().
			WithComponent("AppProvisioner").
			WithPackage("application"),
		client:     client,
		appName:    request.GetManagedApplicationName(),
		appID:      request.GetID(),
		consumerID: request.GetApplicationDetailsValue(common.AttrAppID),
	}
	if a.appName != "" {
		a.logger = a.logger.WithField(logFieldAppName, a.appName)
	}
	if a.appID != "" {
		a.logger = a.logger.WithField(logFieldAppID, a.appID)
	}
	if a.consumerID != "" {
		a.logger = a.logger.WithField(logFieldConsumerID, a.consumerID)
	}
	return a
}

func (a AppProvisioner) Provision() provisioning.RequestStatus {
	a.logger.Info("provisioning application")

	rs := provisioning.NewRequestStatusBuilder()
	if a.appName == "" {
		a.logger.Error("could not find the managed application name on the resource")
		return rs.SetMessage("managed application name not found").Failed()
	}

	consumer, err := a.client.CreateConsumer(a.ctx, a.appID, a.appName)
	if err != nil {
		a.logger.WithError(err).Error("error creating kong consumer")
		return rs.SetMessage("could not create a new consumer in kong").Failed()
	}

	err = a.client.AddConsumerACL(a.ctx, *consumer.ID)
	if err != nil {
		a.logger.WithError(err).Error("could not add acl to kong consumer")
	}

	rs.AddProperty(common.AttrAppID, *consumer.ID)
	a.logger.Info("provisioned application")

	return rs.Success()
}

func (a AppProvisioner) Deprovision() provisioning.RequestStatus {
	a.logger.Info("deprovisioning application")

	rs := provisioning.NewRequestStatusBuilder()

	if a.consumerID == "" {
		a.logger.Error("could not find the consumer id on the managed application resource")
		return rs.SetMessage(fmt.Sprintf("%s not found", common.AttrAppID)).Failed()
	}

	err := a.client.DeleteConsumer(a.ctx, a.consumerID)
	if err != nil {
		a.logger.WithError(err).Error("error deleting kong consumer")
		return rs.SetMessage("could not remove consumer in kong").Failed()
	}
	a.logger.Info("deprovisioned application")

	return rs.Success()
}
