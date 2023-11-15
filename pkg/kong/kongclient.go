package kong

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agent-sdk/pkg/apic"
	config "github.com/Axway/agents-kong/pkg/config/discovery"

	klib "github.com/kong/go-kong/kong"
)

const tagPrefix = "spec_local_"

type KongAPIClient interface {
	// Provisioning
	CreateConsumer(ctx context.Context, id, name string) (*klib.Consumer, error)
	AddConsumerACL(ctx context.Context, id string) error
	DeleteConsumer(ctx context.Context, id string) error
	// Credential
	DeleteOauth2(ctx context.Context, consumerID, clientID string) error
	DeleteHttpBasic(ctx context.Context, consumerID, username string) error
	DeleteAuthKey(ctx context.Context, consumerID, authKey string) error
	CreateHttpBasic(ctx context.Context, consumerID string, basicAuth *klib.BasicAuth) (*klib.BasicAuth, error)
	CreateOauth2(ctx context.Context, consumerID string, oauth2 *klib.Oauth2Credential) (*klib.Oauth2Credential, error)
	CreateAuthKey(ctx context.Context, consumerID string, keyAuth *klib.KeyAuth) (*klib.KeyAuth, error)
	// Access Request
	AddRouteACL(ctx context.Context, routeID, allowedID string) error
	RemoveRouteACL(ctx context.Context, routeID, revokedID string) error
	AddQuota(ctx context.Context, routeID, allowedID, quotaInterval string, quotaLimit int) error

	ListServices(ctx context.Context) ([]*klib.Service, error)
	ListRoutesForService(ctx context.Context, serviceId string) ([]*klib.Route, error)
	GetSpecForService(ctx context.Context, service *klib.Service) ([]byte, error)
	GetKongPlugins() *Plugins
}

type KongClient struct {
	*klib.Client
	logger            log.FieldLogger
	baseClient        DoRequest
	kongAdminEndpoint string
	specURLPaths      []string
	specLocalPath     string
	clientTimeout     time.Duration
}

func NewKongClient(baseClient *http.Client, kongConfig *config.KongGatewayConfig) (*KongClient, error) {
	if kongConfig.Admin.Auth.APIKey.Value != "" {
		defaultTransport := http.DefaultTransport.(*http.Transport)
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		baseClient.Transport = defaultTransport

		headers := make(http.Header)
		headers.Set(kongConfig.Admin.Auth.APIKey.Header, kongConfig.Admin.Auth.APIKey.Value)
		client := klib.HTTPClientWithHeaders(baseClient, headers)
		baseClient = client
	}

	logger := log.NewFieldLogger().WithComponent("client").WithPackage("kong")

	baseKongClient, err := klib.NewClient(&kongConfig.Admin.URL, baseClient)
	if err != nil {
		logger.WithError(err).Error("failed to create kong client")
		return nil, err
	}
	return &KongClient{
		Client:            baseKongClient,
		logger:            log.NewFieldLogger().WithComponent("KongClient").WithPackage("kong"),
		baseClient:        baseClient,
		kongAdminEndpoint: kongConfig.Admin.URL,
		specURLPaths:      kongConfig.Spec.URLPaths,
		specLocalPath:     kongConfig.Spec.LocalPath,
		clientTimeout:     60 * time.Second,
	}, nil
}

func (k KongClient) ListServices(ctx context.Context) ([]*klib.Service, error) {
	return k.Services.ListAll(ctx)
}

func (k KongClient) ListRoutesForService(ctx context.Context, serviceId string) ([]*klib.Route, error) {
	routes, _, err := k.Routes.ListForService(ctx, &serviceId, nil)
	return routes, err
}

func (k KongClient) GetSpecForService(ctx context.Context, service *klib.Service) ([]byte, error) {
	log := k.logger.WithField("serviceID", service.ID).WithField("serviceName", service.Name)

	if k.specLocalPath != "" {
		return k.getSpecFromLocal(ctx, service)
	}

	// all three fields are needed to form the backend URL used in discovery process
	if service.Protocol == nil && service.Host == nil {
		err := fmt.Errorf("fields for backend URL are not set")
		log.WithError(err).Error("failed to create backend URL")
		return nil, err
	}
	backendURL := *service.Protocol + "://" + *service.Host
	if service.Path != nil {
		backendURL = backendURL + *service.Path
	}

	return k.getSpecFromBackend(ctx, backendURL)
}

func (k KongClient) getSpecFromLocal(ctx context.Context, service *klib.Service) ([]byte, error) {
	log := k.logger.WithField("serviceID", service.ID).WithField("serviceName", service.Name)

	specTag := ""
	for _, tag := range service.Tags {
		if strings.HasPrefix(*tag, tagPrefix) {
			specTag = *tag
			break
		}
	}

	if specTag == "" {
		log.Info("no specification tag found")
		return nil, nil
	}

	filename := specTag[len(tagPrefix):]
	specFilePath := path.Join(k.specLocalPath, filename)
	specContent, err := k.loadSpecFile(specFilePath)
	if err != nil {
		log.WithError(err).Error("failed to get spec from file")
		return nil, err
	}

	return specContent, nil
}

func (k KongClient) loadSpecFile(specFilePath string) ([]byte, error) {
	log := k.logger.WithField("specFilePath", specFilePath)

	if _, err := os.Stat(specFilePath); os.IsNotExist(err) {
		log.Debug("spec file not found")
		return nil, nil
	}

	data, err := os.ReadFile(specFilePath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (k KongClient) getSpecFromBackend(ctx context.Context, backendURL string) ([]byte, error) {
	if len(k.specURLPaths) == 0 {
		k.logger.Info("no spec paths configured")
		return nil, nil
	}

	for _, specPath := range k.specURLPaths {
		endpoint := fmt.Sprintf("%s/%s", backendURL, strings.TrimPrefix(specPath, "/"))

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
	return nil, nil
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
