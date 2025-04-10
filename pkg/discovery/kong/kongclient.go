package kong

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	klib "github.com/kong/go-kong/kong"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agents-kong/pkg/common"
	config "github.com/Axway/agents-kong/pkg/discovery/config"
)

const tagPrefix = "spec_local_"
const unstructuredSpec = "Unstructured API Example"

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
	GetSpecForService(ctx context.Context, service *klib.Service) ([]byte, bool, error)
	GetKongPlugins(ctx context.Context) *Plugins
}

type KongServiceSpec struct {
	Contents  string `json:"contents"`
	CreatedAt int    `json:"created_at"`
	ID        string `json:"id"`
	Path      string `json:"path"`
	Checksum  string `json:"checksum"`
}

type KongClient struct {
	workspaceClients      map[string]*klib.Client
	logger                log.FieldLogger
	baseClient            DoRequest
	kongAdminEndpoint     string
	specURLPaths          []string
	specLocalPath         string
	devPortalEnabled      bool
	createUnstructuredAPI bool
	clientTimeout         time.Duration
}

func NewKongClient(kongConfig *config.KongGatewayConfig) (*KongClient, error) {
	headers := make(http.Header)
	var kongEndpoint string
	kongTransport := http.DefaultTransport.(*http.Transport)
	kongTransport.TLSClientConfig = kongConfig.Admin.TLS.BuildTLSConfig()
	baseClient := &http.Client{
		Transport: kongTransport,
	}
	kongEndpoint = kongConfig.Admin.Url

	if kongConfig.Admin.Auth.APIKey.Value != "" {
		headers.Set(kongConfig.Admin.Auth.APIKey.Header, kongConfig.Admin.Auth.APIKey.Value)
	}
	if kongConfig.Admin.Auth.BasicAuth.Username != "" {
		headers.Set("Authorization", "Basic "+basicAuth(kongConfig.Admin.Auth.BasicAuth.Username, kongConfig.Admin.Auth.BasicAuth.Password))
	}
	headers.Set("Host", kongConfig.Proxy.Host)
	baseClient = klib.HTTPClientWithHeaders(baseClient, headers)

	logger := log.NewFieldLogger().WithComponent("client").WithPackage("kong")

	workspaceClients, err := createWorkspaceClients(baseClient, kongEndpoint, kongConfig.Workspaces)
	if err != nil {
		logger.WithError(err).Error("failed to create kong client")
		return nil, err
	}

	return &KongClient{
		workspaceClients:      workspaceClients,
		logger:                log.NewFieldLogger().WithComponent("KongClient").WithPackage("kong"),
		baseClient:            baseClient,
		kongAdminEndpoint:     kongEndpoint,
		specURLPaths:          kongConfig.Spec.URLPaths,
		specLocalPath:         kongConfig.Spec.LocalPath,
		devPortalEnabled:      kongConfig.Spec.DevPortalEnabled,
		createUnstructuredAPI: kongConfig.Spec.CreateUnstructuredAPI,
		clientTimeout:         10 * time.Second,
	}, nil
}

func createWorkspaceClients(baseClient *http.Client, kongEndpoint string, workspaces []string) (map[string]*klib.Client, error) {
	clients := make(map[string]*klib.Client)
	for _, workspace := range workspaces {
		client, err := createWorkspaceClient(baseClient, kongEndpoint, workspace)
		if err != nil {
			return nil, err
		}
		clients[workspace] = client
	}
	if len(clients) == 0 {
		client, err := createWorkspaceClient(baseClient, kongEndpoint, "")
		if err != nil {
			return nil, err
		}
		clients[common.DefaultWorkspace] = client
	}
	return clients, nil
}

func createWorkspaceClient(baseClient *http.Client, kongEndpoint string, workspace string) (*klib.Client, error) {
	kongClient, err := klib.NewClient(&kongEndpoint, baseClient)
	if err != nil {
		return nil, err
	}
	if workspace != "" {
		kongClient.SetWorkspace(workspace)
	}
	return kongClient, nil
}

func (k KongClient) getWorkspaceClient(ctx context.Context) *klib.Client {
	workspace := common.GetStringValueFromCtx(ctx, common.ContextWorkspace)
	if workspace == "" {
		workspace = common.DefaultWorkspace
	}
	return k.workspaceClients[workspace]
}

func (k KongClient) ListServices(ctx context.Context) ([]*klib.Service, error) {
	return k.getWorkspaceClient(ctx).Services.ListAll(ctx)
}

func (k KongClient) ListRoutesForService(ctx context.Context, serviceId string) ([]*klib.Route, error) {
	routes, _, err := k.getWorkspaceClient(ctx).Routes.ListForService(ctx, &serviceId, nil)
	return routes, err
}

func (k KongClient) GetSpecForService(ctx context.Context, service *klib.Service) ([]byte, bool, error) {
	log := k.logger.WithField(common.AttrServiceName, *service.Name)

	if k.specLocalPath != "" {
		spec, err := k.getSpecFromLocal(ctx, service)
		return spec, false, err
	}

	if k.devPortalEnabled {
		spec, err := k.getSpecFromDevPortal(ctx, *service.ID)
		return spec, false, err
	}

	// all three fields are needed to form the backend URL used in discovery process
	if service.Protocol == nil && service.Host == nil {
		err := fmt.Errorf("fields for backend URL are not set")
		log.WithError(err).Error("failed to create backend URL")
		return nil, false, err
	}
	backendURL := *service.Protocol + "://" + *service.Host
	if service.Path != nil {
		backendURL = backendURL + *service.Path
	}

	spec, err := k.getSpecFromBackend(ctx, backendURL)
	if spec == nil && err == nil && k.createUnstructuredAPI {
		return k.getUnstructuredSpec(ctx), true, nil
	}
	return spec, false, err
}

func (k KongClient) getUnstructuredSpec(ctx context.Context) []byte {
	k.logger.Info("Adding unstructured API to services which had no spec associated")
	return []byte(unstructuredSpec)
}

func (k KongClient) getSpecFromLocal(ctx context.Context, service *klib.Service) ([]byte, error) {
	log := k.logger.WithField(common.AttrServiceName, *service.Name)

	specTag := ""
	for _, tag := range service.Tags {
		if strings.HasPrefix(*tag, tagPrefix) {
			specTag = *tag
			break
		}
	}

	if specTag == "" {
		log.Error("in order to map local specs to the desired services, a tag with format 'spec_local_fileName.extension' must be present")
		return nil, fmt.Errorf("no specification tag found")
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

func (k KongClient) getSpecFromDevPortal(ctx context.Context, serviceID string) ([]byte, error) {
	log := k.logger.WithField(common.AttrServiceID, serviceID)
	log.Info("getting spec file from dev portal")
	workspace := common.GetStringValueFromCtx(ctx, common.ContextWorkspace)

	endpoint := fmt.Sprintf("%s/services/%s/document_objects", k.kongAdminEndpoint, serviceID)
	if workspace != "" {
		endpoint = fmt.Sprintf("%s/%s/services/%s/document_objects", k.kongAdminEndpoint, workspace, serviceID)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		log.WithError(err).Error("failed to create request")
		return nil, err
	}
	res, err := k.baseClient.Do(req)
	if err != nil {
		log.WithError(err).Error("failed to execute request")
		return nil, err
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.WithError(err).Error("failed to read body")
		return nil, err
	}
	documents := &DocumentObjects{}
	err = json.Unmarshal(data, documents)
	if err != nil {
		log.WithError(err).Error("failed to unmarshal")
		return nil, err
	}
	if len(documents.Data) < 1 {
		log.Debug("no documents found")
		return nil, nil
	}
	if workspace == "" {
		workspace = common.DefaultWorkspace
	}
	endpoint = fmt.Sprintf("%s/%s/files/%s", k.kongAdminEndpoint, workspace, documents.Data[0].Path)
	return k.getSpec(ctx, endpoint, true)
}

func (k KongClient) getSpecFromBackend(ctx context.Context, backendURL string) ([]byte, error) {
	k.logger.Info("trying to get spec file from service backend")

	if len(k.specURLPaths) == 0 {
		k.logger.Info("no spec paths configured")
		return nil, nil
	}

	for _, specPath := range k.specURLPaths {
		endpoint := fmt.Sprintf("%s/%s", strings.TrimSuffix(backendURL, "/"), strings.TrimPrefix(specPath, "/"))

		spec, err := k.getSpec(ctx, endpoint, false)
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

func (k KongClient) getSpec(ctx context.Context, endpoint string, fromDevPortal bool) ([]byte, error) {
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

	data, err := io.ReadAll(res.Body)
	if err != nil {
		k.logger.WithError(err).Error("failed to read body")
		return nil, err
	}

	var specContent []byte
	if fromDevPortal {
		kongServiceSpec := &KongServiceSpec{}
		err = json.Unmarshal(data, kongServiceSpec)
		if err != nil {
			k.logger.WithError(err).Error("failed to unmarshal")
			return nil, err
		}
		specContent = []byte(kongServiceSpec.Contents)
	} else {
		specContent = data
	}

	specParser := apic.NewSpecResourceParser(specContent, "")
	err = specParser.Parse()
	if err != nil {
		k.logger.Debug("invalid api spec")
		return nil, nil
	}

	return specContent, nil
}

func (k KongClient) GetKongPlugins(ctx context.Context) *Plugins {
	return &Plugins{PluginLister: k.getWorkspaceClient(ctx).Plugins}
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
