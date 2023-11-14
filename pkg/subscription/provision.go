package subscription

import (
	"context"
	"errors"
	"fmt"

	klib "github.com/kong/go-kong/kong"
	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agents-kong/pkg/common"
	"github.com/Axway/agents-kong/pkg/kong"
	"github.com/Axway/agents-kong/pkg/subscription/access"
	"github.com/Axway/agents-kong/pkg/subscription/application"
)

var constructors []func(*klib.Client) Handler

func Add(constructor func(*klib.Client) Handler) {
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
	logger   log.FieldLogger
	client   kong.KongAPIClient
	kc       *klib.Client
	handlers map[string]Handler
}

// NewProvisioner creates a type to implement the SDK Provisioning methods for handling subscriptions
func NewProvisioner(client kong.KongAPIClient, logger log.FieldLogger) {
	logger.Info("Registering provisioning callbacks")
	logger.Infof("Handlers : %d", len(constructors))
	handlers := make(map[string]Handler, len(constructors))
	for _, c := range constructors {
		h := c(client.(*kong.KongClient).Client)
		h.Register()
		handlers[h.Name()] = h
	}
	provisioner := &provisioner{
		// set supported subscription handlers
		client:   client,
		handlers: handlers,
		logger:   logger,
	}
	agent.RegisterProvisioner(provisioner)
	for _, handler := range handlers {
		log.Infof("Registering authentication :%s", handler.Name())
		handler.Register()
	}
}

func (p provisioner) CredentialUpdate(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {
	p.logger.Info("provisioning credentials update")
	credentialType := request.GetCredentialType()
	if h, ok := p.handlers[credentialType]; ok {
		return h.UpdateCredential(request)
	}
	errorMsg := fmt.Sprintf("No known handler for type: %s", credentialType)
	logrus.Info(errorMsg)
	return Failed(provisioning.NewRequestStatusBuilder(), errors.New(errorMsg)), nil
}

func (p provisioner) CredentialProvision(request provisioning.CredentialRequest) (provisioning.RequestStatus, provisioning.Credential) {

	p.logger.Info("provisioning credentials")
	credentialType := request.GetCredentialType()
	if h, ok := p.handlers[credentialType]; ok {
		return h.CreateCredential(request)
	}
	errorMsg := fmt.Sprintf("No known handler for type: %s", credentialType)
	logrus.Info(errorMsg)
	return Failed(provisioning.NewRequestStatusBuilder(), errors.New(errorMsg)), nil
}
func (p provisioner) CredentialDeprovision(request provisioning.CredentialRequest) provisioning.RequestStatus {
	p.logger.Info("de_provisioning credentials")
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

func (p provisioner) ApplicationRequestProvision(request provisioning.ApplicationRequest) provisioning.RequestStatus {
	return application.NewApplicationProvisioner(context.Background(), p.client, request).Provision()
}

func (p provisioner) ApplicationRequestDeprovision(request provisioning.ApplicationRequest) provisioning.RequestStatus {
	return application.NewApplicationProvisioner(context.Background(), p.client, request).Deprovision()
}

func (p provisioner) AccessRequestProvision(request provisioning.AccessRequest) (provisioning.RequestStatus, provisioning.AccessData) {
	return access.NewAccessProvisioner(context.Background(), p.client, request).Provision()
}

func (p provisioner) AccessRequestDeprovision(request provisioning.AccessRequest) provisioning.RequestStatus {
	return access.NewAccessProvisioner(context.Background(), p.client, request).Deprovision()
}
