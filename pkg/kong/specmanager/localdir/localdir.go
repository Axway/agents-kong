package localdir

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-kong/pkg/kong/specmanager"
	klib "github.com/kong/go-kong/kong"
)

const tagPrefix = "spec_local_"

type sourceConfig struct {
	name     string
	rootPath string
}

func NewSpecificationSource(rootPath string) specmanager.SpecificationSource {
	return sourceConfig{
		name:     "local",
		rootPath: rootPath,
	}
}

func (sc sourceConfig) Name() *string {
	return &sc.name
}

func (sc sourceConfig) GetSpecForService(ctx context.Context, service *klib.Service) (*specmanager.KongServiceSpec, error) {
	specTag := ""
	for _, tag := range service.Tags {
		if strings.HasPrefix(*tag, tagPrefix) {
			specTag = *tag
		}
	}

	if len(specTag) > 0 {
		name := specTag[len(tagPrefix):]

		return sc.loadSpecFile(name)
	}
	log.Infof("no specification tag found for service %s (%s)", *service.Name, *service.ID)
	return nil, nil
}

func (sc sourceConfig) loadSpecFile(name string) (*specmanager.KongServiceSpec, error) {
	specFilePath := path.Join(sc.rootPath, name)

	if _, err := os.Stat(specFilePath); os.IsNotExist(err) {
		log.Warnf("specification file not found %s", specFilePath)
		return nil, nil
	}

	data, err := ioutil.ReadFile(specFilePath)
	if err != nil {
		return nil, fmt.Errorf("error on reading spec file %s: %s", specFilePath, err)
	}

	kongServiceSpec := &specmanager.KongServiceSpec{
		Contents:  string(data),
		CreatedAt: 0,
		ID:        "",
		Path:      specFilePath,
		Checksum:  "",
	}

	return kongServiceSpec, nil
}
