package subscription

import (
	"context"
	"errors"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agents-kong/pkg/common"
)

const (
	logFieldAppID   = "appID"
	logFieldAppName = "appName"
)

func (p provisioner) ApplicationRequestProvision(request provisioning.ApplicationRequest) provisioning.RequestStatus {
	log := p.logger
	log.Info("provisioning application")

	ctx := context.Background()
	rs := provisioning.NewRequestStatusBuilder()

	appName := request.GetManagedApplicationName()
	if appName == "" {
		log.Error("could not find the managed application name on the resource")
		return Failed(rs, notFound("managed application name"))
	}
	appID := request.GetID()
	log = log.WithField(logFieldAppID, appID).WithField(logFieldAppName, appName)

	consumer, err := p.client.CreateConsumer(ctx, appID, appName)
	if err != nil {
		log.WithError(err).Error("error creating kong consumer")
		return Failed(rs, errors.New("could not create a new consumer in kong"))
	}

	// process application create
	rs.AddProperty(common.AttrAppID, *consumer.ID)
	log.Info("created application")

	return rs.Success()
}

func (p provisioner) ApplicationRequestDeprovision(request provisioning.ApplicationRequest) provisioning.RequestStatus {
	log := p.logger
	log.Info("deprovisioning application")

	ctx := context.Background()
	rs := provisioning.NewRequestStatusBuilder()

	appID := request.GetApplicationDetailsValue(common.AttrAppID)
	if appID == "" {
		log.Error("could not find the consumer id on the managed application resource")
		return Failed(rs, notFound(common.AttrAppID))
	}
	log = log.WithField(logFieldAppID, appID).WithField(logFieldAppName, request.GetManagedApplicationName())

	err := p.client.DeleteConsumer(ctx, appID)
	if err != nil {
		log.WithError(err).Error("error deleting kong consumer")
		return Failed(rs, errors.New("could not remove consumer in kong"))
	}
	p.logger.Info("removed application")

	return rs.Success()
}
