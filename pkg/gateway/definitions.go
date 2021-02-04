package gateway

import (
	"github.com/Axway/agents-kong/pkg/kong"
	"github.com/Axway/agents-kong/pkg/subscription"

	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	corecfg "github.com/Axway/agent-sdk/pkg/config"

	config "github.com/Axway/agents-kong/pkg/config/discovery"
)

type InstanceEndpoint = v1alpha1.ApiServiceInstanceSpecEndpoint

type Client struct {
	centralCfg          corecfg.CentralConfig
	kongGatewayCfg      *config.KongGatewayConfig
	kongClient          kong.KongAPIClient
	apicClient          CentralClient
	subscriptionManager *subscription.Manager
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
	endpoints        []InstanceEndpoint
	subscriptionInfo subscription.Info
	nameToPush       string
}

type CachedService struct {
	kongServiceId   string
	kongServiceName string
	hash            string
	centralName     string
}
