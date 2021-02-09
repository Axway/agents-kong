package kong

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Axway/agents-kong/pkg/kong/specmanager"

	config "github.com/Axway/agents-kong/pkg/config/discovery"

	klib "github.com/kong/go-kong/kong"
)

type KongAPIClient interface {
	ListServices(ctx context.Context) ([]*klib.Service, error)
	ListRoutesForService(ctx context.Context, serviceId string) ([]*klib.Route, error)
	GetSpecForService(ctx context.Context, serviceId string) (*specmanager.KongServiceSpec, error)
	GetKongPlugins() *Plugins
}

type Client struct {
	*klib.Client
	baseClient        DoRequest
	kongAdminEndpoint string
}

func NewKongClient(baseClient *http.Client, kongConfig *config.KongGatewayConfig) (*Client, error) {
	if kongConfig.Token != "" {
		defaultTransport := http.DefaultTransport.(*http.Transport)
		baseClient.Transport = defaultTransport

		headers := make(http.Header)
		headers.Set("Kong-Admin-Token", kongConfig.Token)
		client := klib.HTTPClientWithHeaders(baseClient, headers)
		baseClient = &client
	}

	baseKongClient, err := klib.NewClient(&kongConfig.AdminEndpoint, baseClient)
	if err != nil {
		return nil, err
	}
	return &Client{
		Client:            baseKongClient,
		baseClient:        baseClient,
		kongAdminEndpoint: kongConfig.AdminEndpoint,
	}, nil
}

func (k Client) ListServices(ctx context.Context) ([]*klib.Service, error) {
	return k.Services.ListAll(ctx)
}

func (k Client) ListRoutesForService(ctx context.Context, serviceId string) ([]*klib.Route, error) {
	routes, _, err := k.Routes.ListForService(ctx, &serviceId, nil)
	return routes, err
}

func (k Client) GetSpecForService(ctx context.Context, serviceId string) (*specmanager.KongServiceSpec, error) {
	endpoint := fmt.Sprintf("%s/services/%s/document_objects", k.kongAdminEndpoint, serviceId)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %s", err)
	}
	res, err := k.baseClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %s", err)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %s", err)
	}
	documents := &DocumentObjects{}
	err = json.Unmarshal(data, documents)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %s", err)
	}
	if len(documents.Data) < 1 {
		return nil, fmt.Errorf("no documents found")
	}
	return k.getSpec(ctx, documents.Data[0].Path)
}

func (k Client) getSpec(ctx context.Context, path string) (*specmanager.KongServiceSpec, error) {
	endpoint := fmt.Sprintf("%s/default/files/%s", k.kongAdminEndpoint, path)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %s", err)
	}

	res, err := k.baseClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %s", err)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %s", err)
	}

	kongServiceSpec := &specmanager.KongServiceSpec{}
	err = json.Unmarshal(data, kongServiceSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %s", err)
	}
	if len(kongServiceSpec.Contents) == 0 {
		return nil, fmt.Errorf("spec not found at '%s'", path)
	}
	return kongServiceSpec, nil
}

func (k Client) GetKongPlugins() *Plugins {
	return &Plugins{PluginLister: k.Plugins}
}

func (k Client) GetKongConsumers() *Consumers {
	return &Consumers{ConsumerService: k.Consumers}
}
