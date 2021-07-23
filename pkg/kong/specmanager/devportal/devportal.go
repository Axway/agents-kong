package devportal

import (
	"context"

	kutil "github.com/Axway/agents-kong/pkg/kong"
	"github.com/Axway/agents-kong/pkg/kong/specmanager"
	klib "github.com/kong/go-kong/kong"
)

type sourceConfig struct {
	name       string
	kongClient kutil.KongAPIClient
}

func NewSpecificationSource(kongClient kutil.KongAPIClient) specmanager.SpecificationSource {
	return sourceConfig{
		name:       "dev-portal",
		kongClient: kongClient,
	}
}

func (sc sourceConfig) Name() *string {
	return &sc.name
}

func (sc sourceConfig) GetSpecForService(ctx context.Context, service *klib.Service) (*specmanager.KongServiceSpec, error) {
	return sc.kongClient.GetSpecForService(ctx, *service.ID)
}
