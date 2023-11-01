package gateway

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cache"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agents-kong/pkg/kong"

	config "github.com/Axway/agents-kong/pkg/config/discovery"
	kutil "github.com/Axway/agents-kong/pkg/kong"
)

type Client struct {
	centralCfg     corecfg.CentralConfig
	kongGatewayCfg *config.KongGatewayConfig
	kongClient     kong.KongAPIClient
	apicClient     CentralClient
	//subscriptionManager *subscription.Manager
	plugins kutil.Plugins
	cache   cache.Cache
	mode    string
	ard     string
	crds    []string
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
	serviceAttributes map[string]string
	agentDetails      map[string]string
	subscriptionName  string
	tags              []string
	stage             string
	state             string
	status            string
	ard               string
}

type CachedService struct {
	kongServiceId   string
	kongServiceName string
	hash            string
	centralName     string
}
