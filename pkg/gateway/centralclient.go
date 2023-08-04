package gateway

import (
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

type CentralAPIClient interface {
	ExecuteAPI(method, endpoint string, queryParam map[string]string, buffer []byte) ([]byte, error)
}

type CentralClient struct {
	client        CentralAPIClient
	envName       string
	apiServerHost string
}

func NewCentralClient(client CentralAPIClient, config corecfg.CentralConfig) CentralClient {
	return CentralClient{
		client:        client,
		envName:       config.GetEnvironmentName(),
		apiServerHost: config.GetAPIServerURL(),
	}
}

func (cc *CentralClient) execute(method, endpoint string, queryParam map[string]string, buffer []byte) ([]byte, error) {
	host := cc.apiServerHost + cc.envName + endpoint
	return cc.client.ExecuteAPI(method, host, queryParam, buffer)
}
