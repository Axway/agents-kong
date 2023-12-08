package subscription

import (
	"context"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agents-kong/pkg/discovery/kong"
	"github.com/Axway/agents-kong/pkg/discovery/subscription/access"
	"github.com/Axway/agents-kong/pkg/discovery/subscription/application"
	"github.com/Axway/agents-kong/pkg/discovery/subscription/credential"
)

type ProvisionerOption func(*provisioner)

type provisioner struct {
	logger     log.FieldLogger
	client     kong.KongAPIClient
	aclDisable bool
}

// NewProvisioner creates a type to implement the SDK Provisioning methods for handling subscriptions
func NewProvisioner(client kong.KongAPIClient, logger log.FieldLogger, opts ...ProvisionerOption) {
	logger.Info("Registering provisioning callbacks")
	provisioner := &provisioner{
		client: client,
		logger: logger,
	}
	for _, o := range opts {
		o(provisioner)
	}
	agent.RegisterProvisioner(provisioner)
	registerOauth2()
	registerBasicAuth()
	registerKeyAuth()
}

func WithACLDisable() ProvisionerOption {
	return func(p *provisioner) {
		p.aclDisable = true
	}
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
	return access.NewAccessProvisioner(context.Background(), p.client, request, p.aclDisable).Provision()
}

func (p provisioner) AccessRequestDeprovision(request provisioning.AccessRequest) provisioning.RequestStatus {
	return access.NewAccessProvisioner(context.Background(), p.client, request, p.aclDisable).Deprovision()
}
