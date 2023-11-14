package subscription

import (
	"context"

	klib "github.com/kong/go-kong/kong"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agents-kong/pkg/kong"
	"github.com/Axway/agents-kong/pkg/subscription/access"
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

func (p provisioner) AccessRequestProvision(request provisioning.AccessRequest) (provisioning.RequestStatus, provisioning.AccessData) {
	return access.NewAccessProvisioner(context.Background(), p.client, request).Provision()
}

func (p provisioner) AccessRequestDeprovision(request provisioning.AccessRequest) provisioning.RequestStatus {
	return access.NewAccessProvisioner(context.Background(), p.client, request).Deprovision()
}
