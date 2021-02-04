package specmanager

import (
	"context"
	"fmt"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/kong/go-kong/kong"
)

type KongServiceSpec struct {
	Contents  string `json:"contents"`
	CreatedAt int    `json:"created_at"`
	ID        string `json:"id"`
	Path      string `json:"path"`
	Checksum  string `json:"checksum"`
}

type SpecificationSource interface {
	Name() *string
	GetSpecForService(ctx context.Context, service *kong.Service) (*KongServiceSpec, error)
}

var specificationManager struct {
	sources []SpecificationSource
}

func AddSource(source SpecificationSource) {
	specificationManager.sources = append(specificationManager.sources, source)
	log.Infof("specification source added: %s", *source.Name())
}

func GetSpecification(ctx context.Context, service *kong.Service) (*KongServiceSpec, error) {
	for _, source := range specificationManager.sources {
		spec, err := source.GetSpecForService(ctx, service)
		if err == nil {
			if spec != nil {
				return spec, nil
			}
		}
	}
	return nil, fmt.Errorf("no specification found for service %s (%s)", *service.Name, *service.ID)
}
