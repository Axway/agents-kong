package subscription

import (
	"context"

	klib "github.com/kong/go-kong/kong"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agents-kong/pkg/discovery/kong"
	"github.com/Axway/agents-kong/pkg/discovery/subscription/access"
	"github.com/Axway/agents-kong/pkg/discovery/subscription/application"
	"github.com/Axway/agents-kong/pkg/discovery/subscription/credential"
)

type ProvisionerOption func(*provisioner)

type kongClient interface {
	// Provisioning
	CreateConsumer(ctx context.Context, id, name string) (*klib.Consumer, error)
	AddConsumerACL(ctx context.Context, id string) error
	DeleteConsumer(ctx context.Context, id string) error
	// Credential
	DeleteOauth2(ctx context.Context, consumerID, clientID string) error
	DeleteHttpBasic(ctx context.Context, consumerID, username string) error
	DeleteAuthKey(ctx context.Context, consumerID, authKey string) error
	CreateHttpBasic(ctx context.Context, consumerID string, basicAuth *klib.BasicAuth) (*klib.BasicAuth, error)
	CreateOauth2(ctx context.Context, consumerID string, oauth2 *klib.Oauth2Credential) (*klib.Oauth2Credential, error)
	CreateAuthKey(ctx context.Context, consumerID string, keyAuth *klib.KeyAuth) (*klib.KeyAuth, error)
	// Access Request
	AddRouteACL(ctx context.Context, routeID, allowedID string) error
	RemoveRouteACL(ctx context.Context, routeID, revokedID string) error
	AddQuota(ctx context.Context, routeID, allowedID, quotaInterval string, quotaLimit int) error
	// Discovery
	ListServices(ctx context.Context) ([]*klib.Service, error)
	ListRoutesForService(ctx context.Context, serviceId string) ([]*klib.Route, error)
	GetSpecForService(ctx context.Context, service *klib.Service) ([]byte, bool, error)
	GetKongPlugins(ctx context.Context) *kong.Plugins
}

type provisioner struct {
	logger     log.FieldLogger
	client     kongClient
	aclDisable bool
	envName    string
	workspaces []string
}

// NewProvisioner creates a type to implement the SDK Provisioning methods for handling subscriptions
func NewProvisioner(client kongClient, envName string, workspaces []string, opts ...ProvisionerOption) {
	logger := log.NewFieldLogger().WithComponent("provision").WithPackage("subscription")
	logger.Info("Registering provisioning callbacks")
	provisioner := &provisioner{
		client:     client,
		logger:     logger,
		workspaces: workspaces,
		envName:    envName,
	}
	for _, o := range opts {
		o(provisioner)
	}
	agent.RegisterProvisioner(provisioner)
	for _, workspace := range workspaces {
		registerOauth2(workspace)
		registerBasicAuth(workspace)
		registerKeyAuth(workspace)
	}

}

func WithACLDisable() ProvisionerOption {
	return func(p *provisioner) {
		p.aclDisable = true
	}
}

func (p provisioner) ApplicationRequestProvision(request provisioning.ApplicationRequest) provisioning.RequestStatus {
	return application.NewApplicationProvisioner(context.Background(), p.client, request, p.workspaces).Provision()
}

func (p provisioner) ApplicationRequestDeprovision(request provisioning.ApplicationRequest) provisioning.RequestStatus {
	return application.NewApplicationProvisioner(context.Background(), p.client, request, p.workspaces).Deprovision()
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
	return access.NewAccessProvisioner(context.Background(), p.client, request, p.aclDisable, p.envName).Provision()
}

func (p provisioner) AccessRequestDeprovision(request provisioning.AccessRequest) provisioning.RequestStatus {
	return access.NewAccessProvisioner(context.Background(), p.client, request, p.aclDisable, p.envName).Deprovision()
}
