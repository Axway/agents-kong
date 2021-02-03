package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type CentralAPIClient interface {
	ExecuteAPI(method, endpoint string, queryParam map[string]string, buffer []byte) ([]byte, error)
}

type CentralClient struct {
	client        CentralAPIClient
	envName       string
	apiServerHost string
}

func (cc *CentralClient) execute(method, endpoint string, queryParam map[string]string, buffer []byte) ([]byte, error) {
	host := cc.apiServerHost + cc.envName + endpoint
	log.Infof("sending %s request: %s", method, host)
	return cc.client.ExecuteAPI(method, host, queryParam, buffer)
}

func (cc *CentralClient) fetchCentralAPIServices(queryParam map[string]string) ([]*v1alpha1.APIService, error) {
	data, err := cc.execute(http.MethodGet, "/apiservices", queryParam, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get apiservices: %s", err)
	}

	var centralAPIServices []*v1alpha1.APIService
	err = json.Unmarshal(data, &centralAPIServices)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal apiservices: %s", err)
	}
	return centralAPIServices, nil
}

func (cc *CentralClient) deleteCentralAPIService(cachedService CachedService) error {
	// TODO: ExecuteAPI only returns a success when status code is 200
	cc.execute(http.MethodGet, "/apiservices"+cachedService.centralName, nil, nil)

	log.Infof("service removed: %s", cachedService.kongServiceName)
	return nil
}
