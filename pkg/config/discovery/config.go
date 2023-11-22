package config

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/sirupsen/logrus"
)

const (
	cfgKongAdminURL          = "kong.admin.url"
	cfgKongAdminRoutePath    = "kong.admin.routePath"
	cfgKongAdminAPIKey       = "kong.admin.auth.apikey.value"
	cfgKongAdminAPIKeyHeader = "kong.admin.auth.apikey.header"
	cfgKongAdminUsername     = "kong.admin.auth.basicauth.username"
	cfgKongAdminPassword     = "kong.admin.auth.basicauth.password"
	cfgKongAdminClientID     = "kong.admin.auth.oauth.clientID"
	cfgKongAdminClientSecret = "kong.admin.auth.oauth.clientSecret"
	cfgKongProxyHost         = "kong.proxy.host"
	cfgKongProxyPortHttp     = "kong.proxy.port.http"
	cfgKongProxyPortHttps    = "kong.proxy.port.https"
	cfgKongSpecURLPaths      = "kong.spec.urlPaths"
	cfgKongSpecLocalPath     = "kong.spec.localPath"
)

func AddKongProperties(rootProps properties.Properties) {
	rootProps.AddStringProperty(cfgKongAdminURL, "", "The Kong admin endpoint")
	rootProps.AddStringProperty(cfgKongAdminRoutePath, "", "The Kong route path for the secured admin API")
	rootProps.AddStringProperty(cfgKongAdminAPIKey, "", "API Key value to authenticate with Kong Gateway")
	rootProps.AddStringProperty(cfgKongAdminAPIKeyHeader, "", "API Key header to authenticate with Kong Gateway")
	rootProps.AddStringProperty(cfgKongAdminUsername, "", "Username for basic auth to authenticate with Kong Admin API")
	rootProps.AddStringProperty(cfgKongAdminPassword, "", "Password for basic auth to authenticate with Kong Admin API")
	rootProps.AddStringProperty(cfgKongAdminClientID, "", "ClientID for oauth2 to authenticate with Kong Admin API")
	rootProps.AddStringProperty(cfgKongAdminClientSecret, "", "ClientID for oauth2 to authenticate with Kong Admin API")
	rootProps.AddStringProperty(cfgKongProxyHost, "", "The Kong proxy endpoint")
	rootProps.AddIntProperty(cfgKongProxyPortHttp, 80, "The Kong proxy http port")
	rootProps.AddIntProperty(cfgKongProxyPortHttps, 443, "The Kong proxy https port")
	rootProps.AddStringSliceProperty(cfgKongSpecURLPaths, []string{}, "URL paths that the agent will look in for spec files")
	rootProps.AddStringProperty(cfgKongSpecLocalPath, "", "Local paths where the agent will look for spec files")
}

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralCfg     corecfg.CentralConfig `config:"central"`
	KongGatewayCfg *KongGatewayConfig    `config:"kong"`
}

type KongAdminConfig struct {
	URL       string              `config:"url"`
	Auth      KongAdminAuthConfig `config:"auth"`
	RoutePath string              `config:"routePath"`
}

type KongAdminAuthConfig struct {
	APIKey    KongAdminAuthAPIKeyConfig `config:"apiKey"`
	BasicAuth KongAdminBasicAuthConfig  `config:"basicAuth"`
	OAuth     KongAdminOauthConfig      `config:"oauth"`
}

type KongAdminBasicAuthConfig struct {
	Username string `config:"username"`
	Password string `config:"password"`
}

type KongAdminOauthConfig struct {
	ClientID     string `config:"clientID"`
	ClientSecret string `config:"clientSecret"`
}

type KongAdminAuthAPIKeyConfig struct {
	Header string `config:"header"`
	Value  string `config:"value"`
}

type KongProxyConfig struct {
	Host string              `config:"host"`
	Port KongProxyPortConfig `config:"port"`
}

type KongProxyPortConfig struct {
	HTTP  int `config:"http"`
	HTTPS int `config:"https"`
}

type KongSpecConfig struct {
	URLPaths  []string `config:"urlPaths"`
	LocalPath string   `config:"localPaths"`
}

// KongGatewayConfig - represents the config for gateway
type KongGatewayConfig struct {
	corecfg.IConfigValidator
	Admin KongAdminConfig `config:"admin"`
	Proxy KongProxyConfig `config:"proxy"`
	Spec  KongSpecConfig  `config:"spec"`
}

// ValidateCfg - Validates the gateway config
func (c *KongGatewayConfig) ValidateCfg() (err error) {
	if c.Admin.URL == "" && c.Admin.RoutePath == "" {
		return fmt.Errorf("error: admin url or the route path for the proxy is required")
	}
	if c.Admin.RoutePath != "" && c.Proxy.Port.HTTPS == 0 {
		return fmt.Errorf("error: secured admin API is only usable with HTTPS")
	}
	if c.Proxy.Host == "" {
		return fmt.Errorf("error: proxy host is required")
	}
	if c.Proxy.Port.HTTP == 0 && c.Proxy.Port.HTTPS == 0 {
		return fmt.Errorf("error: at least one proxy port value of either http or https is required")
	}
	if noCredentialsProvided(c) {
		logrus.Warn("No credentials provided. Assuming Kong Admin API requires no authorization")
	}
	if invalidCredentialConfig(c) {
		return fmt.Errorf("error: Invalid authorization configuration provided. " +
			"If provided, (Username and Password) or (ClientID and ClientSecret) must be non-empty")
	}
	return
}

func noCredentialsProvided(c *KongGatewayConfig) bool {
	apiKey := c.Admin.Auth.APIKey.Value
	user := c.Admin.Auth.BasicAuth.Username
	pass := c.Admin.Auth.BasicAuth.Password
	clientID := c.Admin.Auth.OAuth.ClientID
	secret := c.Admin.Auth.OAuth.ClientSecret

	if apiKey == "" && user == "" && pass == "" && clientID == "" && secret == "" {
		return true
	}
	return false
}

func invalidCredentialConfig(c *KongGatewayConfig) bool {
	user := c.Admin.Auth.BasicAuth.Username
	pass := c.Admin.Auth.BasicAuth.Password
	clientID := c.Admin.Auth.OAuth.ClientID
	secret := c.Admin.Auth.OAuth.ClientSecret

	if (user == "" && pass != "") ||
		(user != "" && pass == "") {
		return true
	}
	if (clientID == "" && secret != "") ||
		(clientID != "" && secret == "") {
		return true
	}
	return false
}

func ParseProperties(rootProps properties.Properties) *KongGatewayConfig {
	// Parse the config from bound properties and setup gateway config
	return &KongGatewayConfig{
		Admin: KongAdminConfig{
			URL:       rootProps.StringPropertyValue(cfgKongAdminURL),
			RoutePath: rootProps.StringPropertyValue(cfgKongAdminRoutePath),
			Auth: KongAdminAuthConfig{
				APIKey: KongAdminAuthAPIKeyConfig{
					Value:  rootProps.StringPropertyValue(cfgKongAdminAPIKey),
					Header: rootProps.StringPropertyValue(cfgKongAdminAPIKeyHeader),
				},
				BasicAuth: KongAdminBasicAuthConfig{
					Username: rootProps.StringPropertyValue(cfgKongAdminUsername),
					Password: rootProps.StringPropertyValue(cfgKongAdminPassword),
				},
				OAuth: KongAdminOauthConfig{
					ClientID:     rootProps.StringPropertyValue(cfgKongAdminClientID),
					ClientSecret: rootProps.StringPropertyValue(cfgKongAdminClientSecret),
				},
			},
		},
		Proxy: KongProxyConfig{
			Host: rootProps.StringPropertyValue(cfgKongProxyHost),
			Port: KongProxyPortConfig{
				HTTP:  rootProps.IntPropertyValue(cfgKongProxyPortHttp),
				HTTPS: rootProps.IntPropertyValue(cfgKongProxyPortHttps),
			},
		},
		Spec: KongSpecConfig{
			URLPaths:  rootProps.StringSlicePropertyValue(cfgKongSpecURLPaths),
			LocalPath: rootProps.StringPropertyValue(cfgKongSpecLocalPath),
		},
	}
}
