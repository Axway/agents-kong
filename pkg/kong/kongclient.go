package kong

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/util/log"
	config "github.com/Axway/agents-kong/pkg/config/discovery"

	klib "github.com/kong/go-kong/kong"
)

type KongAPIClient interface {
	// Provisioning
	CreateConsumer(ctx context.Context, id, name string) (*klib.Consumer, error)
	AddConsumerACL(ctx context.Context, id string) error
	DeleteConsumer(ctx context.Context, id string) error

	ListServices(ctx context.Context) ([]*klib.Service, error)
	ListRoutesForService(ctx context.Context, serviceId string) ([]*klib.Route, error)
	GetSpecForService(ctx context.Context, backendURL string) ([]byte, error)
	GetKongPlugins() *Plugins
}

type KongClient struct {
	*klib.Client
	ctx               context.Context
	logger            log.FieldLogger
	baseClient        DoRequest
	kongAdminEndpoint string
	specPaths         []string
	clientTimeout     time.Duration
}

func NewKongClient(baseClient *http.Client, kongConfig *config.KongGatewayConfig) (*KongClient, error) {
	if kongConfig.Token != "" {
		defaultTransport := http.DefaultTransport.(*http.Transport)
		baseClient.Transport = defaultTransport

		headers := make(http.Header)
		headers.Set("Kong-Admin-Token", kongConfig.Token)
		client := klib.HTTPClientWithHeaders(baseClient, headers)
		baseClient = client
	}

	logger := log.NewFieldLogger().WithComponent("client").WithPackage("kong")

	baseKongClient, err := klib.NewClient(&kongConfig.AdminEndpoint, baseClient)
	if err != nil {
		logger.WithError(err).Error("failed to create kong client")
		return nil, err
	}
	return &KongClient{
		Client:            baseKongClient,
		logger:            log.NewFieldLogger().WithComponent("KongClient").WithPackage("kong"),
		baseClient:        baseClient,
		kongAdminEndpoint: kongConfig.AdminEndpoint,
		specPaths:         kongConfig.SpecDownloadPaths,
		clientTimeout:     10 * time.Second,
	}, nil
}

func (k KongClient) ListServices(ctx context.Context) ([]*klib.Service, error) {
	return k.Services.ListAll(ctx)
}

func (k KongClient) ListRoutesForService(ctx context.Context, serviceId string) ([]*klib.Route, error) {
	routes, _, err := k.Routes.ListForService(ctx, &serviceId, nil)
	return routes, err
}

func (k KongClient) GetSpecForService(ctx context.Context, backendURL string) ([]byte, error) {
	if len(k.specPaths) == 0 {
		k.logger.Info("no spec paths configured")
		return nil, nil
	}

	for _, specPath := range k.specPaths {
		endpoint := fmt.Sprintf("%s/%s", backendURL, specPath)

		spec, err := k.getSpec(ctx, endpoint)
		if err != nil {
			return nil, err
		}
		if spec == nil {
			continue
		}
		return spec, nil
	}

	k.logger.Info("no spec found")
	return []byte{}, nil
}

func (k KongClient) getSpec(ctx context.Context, endpoint string) ([]byte, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, k.clientTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxTimeout, "GET", endpoint, nil)
	if err != nil {
		k.logger.WithError(err).Error("failed to create request")
		return nil, err
	}
	res, err := k.baseClient.Do(req)
	if err != nil {
		k.logger.WithError(err).Error("failed to execute request")
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, nil
	}

	specContent, err := io.ReadAll(res.Body)
	if err != nil {
		k.logger.WithError(err).Error("failed to read body")
		return nil, err
	}

	specParser := apic.NewSpecResourceParser(specContent, "")
	err = specParser.Parse()
	if err != nil {
		k.logger.Debug("invalid api spec")
		return nil, nil
	}

	return specContent, nil
}

func (k KongClient) GetKongPlugins() *Plugins {
	return &Plugins{PluginLister: k.Plugins}
}
