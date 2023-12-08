package config

import (
	"fmt"
	"net/url"
	"strings"

	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type props interface {
	AddStringProperty(name string, defaultVal string, description string)
	AddStringSliceProperty(name string, defaultVal []string, description string)
	AddIntProperty(name string, defaultVal int, description string)
	AddBoolProperty(name string, defaultVal bool, description string)
	StringPropertyValue(name string) string
	StringSlicePropertyValue(name string) []string
	IntPropertyValue(name string) int
	BoolPropertyValue(name string) bool
}

const (
  cfgKongACLRequired            = "kong.acl.required"
	cfgKongAdminUrl               = "kong.admin.url"
	cfgKongAdminAPIKey            = "kong.admin.auth.apiKey.value"
	cfgKongAdminAPIKeyHeader      = "kong.admin.auth.apiKey.header"
	cfgKongAdminBasicUsername     = "kong.admin.auth.basicauth.username"
	cfgKongAdminBasicPassword     = "kong.admin.auth.basicauth.password"
	cfgKongProxyHost              = "kong.proxy.host"
	cfgKongProxyPortHttp          = "kong.proxy.ports.http"
	cfgKongProxyPortHttpDisabled  = "kong.proxy.ports.http.disabled"
	cfgKongProxyPortHttps         = "kong.proxy.ports.https"
	cfgKongProxyPortHttpsDisabled = "kong.proxy.ports.https.disabled"
	cfgKongProxyBasePath          = "kong.proxy.basePath"
	cfgKongSpecURLPaths           = "kong.spec.urlPaths"
	cfgKongSpecLocalPath          = "kong.spec.localPath"
	cfgKongSpecFilter             = "kong.spec.filter"
	cfgKongSpecDevPortal          = "kong.spec.devPortalEnabled"
)

func AddKongProperties(rootProps props) {
  rootProps.AddBoolProperty(cfgKongACLRequired, false, "Whether or not an ACL plugin on Kong is required. False by default.")
	rootProps.AddStringProperty(cfgKongAdminUrl, "", "The Admin API url")
	rootProps.AddStringProperty(cfgKongAdminAPIKey, "", "API Key value to authenticate with Kong Gateway")
	rootProps.AddStringProperty(cfgKongAdminAPIKeyHeader, "", "API Key header to authenticate with Kong Gateway")
	rootProps.AddStringProperty(cfgKongAdminBasicUsername, "", "Username for basic auth to authenticate with Kong Admin API")
	rootProps.AddStringProperty(cfgKongAdminBasicPassword, "", "Password for basic auth to authenticate with Kong Admin API")
	rootProps.AddStringProperty(cfgKongProxyHost, "", "The Kong proxy endpoint")
	rootProps.AddIntProperty(cfgKongProxyPortHttp, 80, "The Kong proxy http port")
	rootProps.AddBoolProperty(cfgKongProxyPortHttpDisabled, false, "Set to true to disable adding an http endpoint to discovered routes")
	rootProps.AddIntProperty(cfgKongProxyPortHttps, 443, "The Kong proxy https port")
	rootProps.AddBoolProperty(cfgKongProxyPortHttpsDisabled, false, "Set to true to disable adding an https endpoint to discovered routes")
	rootProps.AddStringProperty(cfgKongProxyBasePath, "", "The base path for the Kong proxy endpoint")
	rootProps.AddStringSliceProperty(cfgKongSpecURLPaths, []string{}, "URL paths that the agent will look in for spec files")
	rootProps.AddStringProperty(cfgKongSpecLocalPath, "", "Local paths where the agent will look for spec files")
	rootProps.AddStringProperty(cfgKongSpecFilter, "", "SDK Filter format. Empty means filters are ignored.")
	rootProps.AddBoolProperty(cfgKongSpecDevPortal, false, "Set to true to enable gathering specs from teh Kong's dev portal.")
}

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralCfg     corecfg.CentralConfig `config:"central"`
	KongGatewayCfg *KongGatewayConfig    `config:"kong"`
}

type KongAdminConfig struct {
	Url  string              `config:"url"`
	Auth KongAdminAuthConfig `config:"auth"`
}

type KongAdminAuthConfig struct {
	APIKey    KongAdminAuthAPIKeyConfig `config:"apiKey"`
	BasicAuth KongAdminBasicAuthConfig  `config:"basicAuth"`
}

type KongAdminBasicAuthConfig struct {
	Username string `config:"username"`
	Password string `config:"password"`
}

type KongAdminAuthAPIKeyConfig struct {
	Header string `config:"header"`
	Value  string `config:"value"`
}

type KongProxyConfig struct {
	Host     string         `config:"host"`
	Ports    KongPortConfig `config:"ports"`
	BasePath string         `config:"basePath"`
}

type KongPortConfig struct {
	HTTP  KongPortSettingsConfig `config:"http"`
	HTTPS KongPortSettingsConfig `config:"https"`
}

type KongPortSettingsConfig struct {
	Value   int  `config:"value"`
	Disable bool `config:"disabled"`
}

type KongSpecConfig struct {
	URLPaths         []string `config:"urlPaths"`
	LocalPath        string   `config:"localPath"`
	DevPortalEnabled bool     `config:"devPortalEnabled"`
	Filter           string   `config:"filter"`
}

type KongACLConfig struct {
	Required bool `config:"required"`
}

// KongGatewayConfig - represents the config for gateway
type KongGatewayConfig struct {
	corecfg.IConfigValidator
	Admin KongAdminConfig `config:"admin"`
	Proxy KongProxyConfig `config:"proxy"`
	Spec  KongSpecConfig  `config:"spec"`
	ACL   KongACLConfig   `config:"acl"`
}

const (
	hostErr           = "kong host must be provided"
	httpPortErr       = "a non-zero value is required for the http port number when it is enabled"
	httpsPortErr      = "a non-zero value is required for the https port number when it is enabled"
	basePathPrefixErr = "the base path must start with a '/' character"
	basePathSuffixErr = "the base path must not end with a '/' character"
	portErr           = "at least one port endpoint needs to be enabled"
	invalidUrlErr     = "invalid Admin API url provided. Must contain protocol + hostname + port." +
		"Examples: <http://kong.com:8001>, <https://kong.com:8444>"
	credentialConfigErr = "invalid authorization configuration provided. " +
		"If provided, (Username and Password) or (ClientID and ClientSecret) must be non-empty"
)

// ValidateCfg - Validates the gateway config
func (c *KongGatewayConfig) ValidateCfg() error {
	logger := log.NewFieldLogger().WithPackage("config").WithComponent("ValidateConfig")
	if c.Proxy.Host == "" {
		return fmt.Errorf(hostErr)
	}
	if !c.Proxy.Ports.HTTP.Disable && c.Proxy.Ports.HTTP.Value == 0 {
		return fmt.Errorf(httpPortErr)
	}
	if len(c.Proxy.BasePath) > 0 && !strings.HasPrefix(c.Proxy.BasePath, "/") {
		return fmt.Errorf(basePathPrefixErr)
	}
	if len(c.Proxy.BasePath) > 0 && strings.HasSuffix(c.Proxy.BasePath, "/") {
		return fmt.Errorf(basePathSuffixErr)
	}
	if !c.Proxy.Ports.HTTPS.Disable && c.Proxy.Ports.HTTPS.Value == 0 {
		return fmt.Errorf(httpsPortErr)
	}
	if c.Proxy.Ports.HTTP.Disable && c.Proxy.Ports.HTTPS.Disable {
		return fmt.Errorf(portErr)
	}
	if invalidAdminUrl(c.Admin.Url) {
		return fmt.Errorf(invalidUrlErr)
	}
	if noCredentialsProvided(c) {
		logger.Warn("No credentials provided. Assuming Kong Admin API requires no authorization.")
	}
	if invalidCredentialConfig(c) {
		return fmt.Errorf(credentialConfigErr)
	}
	return nil
}

func noCredentialsProvided(c *KongGatewayConfig) bool {
	apiKey := c.Admin.Auth.APIKey.Value
	user := c.Admin.Auth.BasicAuth.Username
	pass := c.Admin.Auth.BasicAuth.Password

	if apiKey == "" && user == "" && pass == "" {
		return true
	}
	return false
}

func invalidAdminUrl(u string) bool {
	parsedUrl, err := url.Parse(u)
	if err != nil {
		return true
	}
	if parsedUrl.Port() == "" ||
		strings.HasPrefix(parsedUrl.Host, "http://") || strings.HasPrefix(parsedUrl.Host, "https://") {
		return true
	}
	return false
}

func invalidCredentialConfig(c *KongGatewayConfig) bool {
	user := c.Admin.Auth.BasicAuth.Username
	pass := c.Admin.Auth.BasicAuth.Password

	if (user == "" && pass != "") ||
		(user != "" && pass == "") {
		return true
	}
	return false
}

func ParseProperties(rootProps props) *KongGatewayConfig {
	// Parse the config from bound properties and setup gateway config
	httpPortConf := KongPortSettingsConfig{
		Disable: rootProps.BoolPropertyValue(cfgKongProxyPortHttpDisabled),
		Value:   rootProps.IntPropertyValue(cfgKongProxyPortHttp),
	}
	if httpPortConf.Disable {
		httpPortConf.Value = 0
	}

	httpsPortConf := KongPortSettingsConfig{
		Disable: rootProps.BoolPropertyValue(cfgKongProxyPortHttpsDisabled),
		Value:   rootProps.IntPropertyValue(cfgKongProxyPortHttps),
	}
	if httpsPortConf.Disable {
		httpsPortConf.Value = 0
	}

	return &KongGatewayConfig{
		ACL: KongACLConfig{
			Required: rootProps.BoolPropertyValue(cfgKongACLRequired),
		},
		Admin: KongAdminConfig{
			Url: rootProps.StringPropertyValue(cfgKongAdminUrl),
			Auth: KongAdminAuthConfig{
				APIKey: KongAdminAuthAPIKeyConfig{
					Value:  rootProps.StringPropertyValue(cfgKongAdminAPIKey),
					Header: rootProps.StringPropertyValue(cfgKongAdminAPIKeyHeader),
				},
				BasicAuth: KongAdminBasicAuthConfig{
					Username: rootProps.StringPropertyValue(cfgKongAdminBasicUsername),
					Password: rootProps.StringPropertyValue(cfgKongAdminBasicPassword),
				},
			},
		},
		Proxy: KongProxyConfig{
			Host: rootProps.StringPropertyValue(cfgKongProxyHost),
			Ports: KongPortConfig{
				HTTP:  httpPortConf,
				HTTPS: httpsPortConf,
			},
			BasePath: rootProps.StringPropertyValue(cfgKongProxyBasePath),
		},
		Spec: KongSpecConfig{
			DevPortalEnabled: rootProps.BoolPropertyValue(cfgKongSpecDevPortal),
			URLPaths:         rootProps.StringSlicePropertyValue(cfgKongSpecURLPaths),
			LocalPath:        rootProps.StringPropertyValue(cfgKongSpecLocalPath),
			Filter:           rootProps.StringPropertyValue(cfgKongSpecFilter),
		},
	}
}
