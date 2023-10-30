package subscription

import (
	"context"
	"errors"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/common"
)

func (p provisioner) ApplicationRequestProvision(request provisioning.ApplicationRequest) provisioning.RequestStatus {
	p.logger.Info("provisioning application")
	ctx := context.Background()
	rs := provisioning.NewRequestStatusBuilder()
	appName := request.GetManagedApplicationName()
	if appName == "" {
		return Failed(rs, notFound("managed application name"))
	}
	consumer, err := p.client.CreateConsumer(ctx, request.GetID(), appName)
	if err != nil {
		return Failed(rs, errors.New("error creating consumer "+err.Error()))
	}

	// process application create
	rs.AddProperty(common.AttrAppID, *consumer.ID)
	p.logger.
		WithField("appName", request.GetManagedApplicationName()).
		Info("created application")

	return rs.Success()
}

func (p provisioner) ApplicationRequestDeprovision(request provisioning.ApplicationRequest) provisioning.RequestStatus {
	p.logger.Info("de-provisioning application")
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
	p.logger.
		WithField("appName", request.GetManagedApplicationName()).
		WithField("appID", appID).
		Info("removed application")
	return rs.Success()
}
