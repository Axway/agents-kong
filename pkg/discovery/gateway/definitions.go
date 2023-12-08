package gateway

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cache"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/filter"
	"github.com/Axway/agent-sdk/pkg/util/log"

	config "github.com/Axway/agents-kong/pkg/discovery/config"
	"github.com/Axway/agents-kong/pkg/discovery/kong"
)

type Client struct {
	logger         log.FieldLogger
	centralCfg     corecfg.CentralConfig
	kongGatewayCfg *config.KongGatewayConfig
	kongClient     kong.KongAPIClient
	plugins        kong.Plugins
	cache          cache.Cache
	mode           string
	filter         filter.Filter
	aclDisabled    string
}

type KongAPI struct {
	swaggerSpec       []byte
	id                string
	name              string
	description       string
	version           string
	url               string
	documentation     []byte
	resourceType      string
	endpoints         []apic.EndpointDefinition
	image             string
	imageContentType  string
	crds              []string
	apiUpdateSeverity string
	agentDetails      map[string]string
	tags              []string
	stage             string
	ard               string
}
