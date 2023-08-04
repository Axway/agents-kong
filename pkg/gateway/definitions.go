package gateway

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agents-kong/pkg/kong"
	"github.com/Axway/agents-kong/pkg/subscription"

	corecfg "github.com/Axway/agent-sdk/pkg/config"
	config "github.com/Axway/agents-kong/pkg/config/discovery"
	kutil "github.com/Axway/agents-kong/pkg/kong"
)

type Client struct {
	centralCfg          corecfg.CentralConfig
	kongGatewayCfg      *config.KongGatewayConfig
	kongClient          kong.KongAPIClient
	apicClient          CentralClient
	subscriptionManager *subscription.Manager
	plugins             kutil.Plugins
}

type KongAPI struct {
	swaggerSpec      []byte
	id               string
	name             string
	description      string
	version          string
	url              string
	documentation    []byte
	resourceType     string
	endpoints        []apic.EndpointDefinition
	subscriptionInfo subscription.Info
	nameToPush       string
}

type CachedService struct {
	kongServiceId   string
	kongServiceName string
	hash            string
	centralName     string
}
