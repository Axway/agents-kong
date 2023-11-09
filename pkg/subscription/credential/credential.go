package credential

import (
	"context"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"
	_ "github.com/kong/go-kong/kong"
)

type credentialProvisioner struct {
	ctx            context.Context
	client         CredentialClient
	appID          string
	managedAppName string
	logger         log.FieldLogger
	request        CredRequest
}

type CredentialClient interface {
	CreateCredential(ctx context.Context, req CredRequest) (provisioning.RequestStatus, provisioning.Credential)
	UpdateCredential(ctx context.Context, req CredRequest) (provisioning.RequestStatus, provisioning.Credential)
	DeleteCredential(ctx context.Context, req CredRequest) provisioning.RequestStatus
}

type CredRequest interface {
	GetApplicationDetailsValue(key string) string
	GetApplicationName() string
	GetCredentialDetailsValue(key string) string
	GetCredentialData() map[string]interface{}
	GetCredentialType() string
}

func NewCredentialProvisioner(ctx context.Context, client CredentialClient, req CredRequest) credentialProvisioner {
	a := credentialProvisioner{
		ctx: context.Background(),
		logger: log.NewFieldLogger().
			WithComponent("credentialProvisioner").
			WithPackage("credential"),
		client:  client,
		request: req,
	}
	return a
}

func (p credentialProvisioner) Deprovision() prov.RequestStatus {
	return p.client.DeleteCredential(p.ctx, p.request)
}

func (p credentialProvisioner) Provision() (prov.RequestStatus, prov.Credential) {
	return p.client.CreateCredential(p.ctx, p.request)

}

func (p credentialProvisioner) Update() (prov.RequestStatus, prov.Credential) {
	return p.client.UpdateCredential(p.ctx, p.request)
}
