package config

import (
	"fmt"
	"strings"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/sirupsen/logrus"
)

const (
	cfgKongHost              = "kong.host"
	cfgKongAdminPortHttp     = "kong.admin.ports.http"
	cfgKongAdminPortHttps    = "kong.admin.ports.https"
	cfgKongAdminRoutePath    = "kong.admin.routePath"
	cfgKongAdminAPIKey       = "kong.admin.auth.apiKey.value"
	cfgKongAdminAPIKeyHeader = "kong.admin.auth.apiKey.header"
	cfgKongAdminUsername     = "kong.admin.auth.basicauth.username"
	cfgKongAdminPassword     = "kong.admin.auth.basicauth.password"
	cfgKongAdminClientID     = "kong.admin.auth.oauth.clientID"
	cfgKongAdminClientSecret = "kong.admin.auth.oauth.clientSecret"
	cfgKongProxyPortHttp     = "kong.proxy.ports.http"
	cfgKongProxyPortHttps    = "kong.proxy.ports.https"
	cfgKongSpecURLPaths      = "kong.spec.urlPaths"
	cfgKongSpecLocalPath     = "kong.spec.localPath"
)

func AddKongProperties(rootProps properties.Properties) {
	rootProps.AddStringProperty(cfgKongHost, "", "The Kong host")
	rootProps.AddIntProperty(cfgKongAdminPortHttp, 8001, "The Kong admin http port")
	rootProps.AddIntProperty(cfgKongAdminPortHttps, 8444, "The Kong admin https port")
	rootProps.AddStringProperty(cfgKongAdminRoutePath, "", "The Kong route path for the secured admin API")
	rootProps.AddStringProperty(cfgKongAdminAPIKey, "", "API Key value to authenticate with Kong Gateway")
	rootProps.AddStringProperty(cfgKongAdminAPIKeyHeader, "", "API Key header to authenticate with Kong Gateway")
	rootProps.AddStringProperty(cfgKongAdminUsername, "", "Username for basic auth to authenticate with Kong Admin API")
	rootProps.AddStringProperty(cfgKongAdminPassword, "", "Password for basic auth to authenticate with Kong Admin API")
	rootProps.AddStringProperty(cfgKongAdminClientID, "", "ClientID for oauth2 to authenticate with Kong Admin API")
	rootProps.AddStringProperty(cfgKongAdminClientSecret, "", "ClientSecret for oauth2 to authenticate with Kong Admin API")
	rootProps.AddIntProperty(cfgKongProxyPortHttp, 8000, "The Kong proxy http port")
	rootProps.AddIntProperty(cfgKongProxyPortHttps, 4443, "The Kong proxy https port")
	rootProps.AddStringSliceProperty(cfgKongSpecURLPaths, []string{}, "URL paths that the agent will look in for spec files")
	rootProps.AddStringProperty(cfgKongSpecLocalPath, "", "Local paths where the agent will look for spec files")
}

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralCfg     corecfg.CentralConfig `config:"central"`
	KongGatewayCfg *KongGatewayConfig    `config:"kong"`
}

type KongAdminConfig struct {
	Ports     KongPortConfig      `config:"ports"`
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
	Ports KongPortConfig `config:"ports"`
}

type KongPortConfig struct {
	HTTP  int `config:"http"`
	HTTPS int `config:"https"`
}

type KongSpecConfig struct {
	URLPaths         []string `config:"urlPaths"`
	LocalPath        string   `config:"localPath"`
	DevPortalEnabled bool     `config:"devPortalEnabled"`
}

// KongGatewayConfig - represents the config for gateway
type KongGatewayConfig struct {
	corecfg.IConfigValidator
	Host  string          `config:"host"`
	Admin KongAdminConfig `config:"admin"`
	Proxy KongProxyConfig `config:"proxy"`
	Spec  KongSpecConfig  `config:"spec"`
}

const (
	hostErr                     = "Kong Host must be provided."
	proxyPortErr                = "At least one proxy port value of either http or https is required"
	routePathOrAdminHttpPortErr = "Admin API HTTP port or the route path for the secured admin API is required"
	routePathWithoutHttpsErr    = "Secured Admin API is only usable with HTTPS. Please provide the Proxy https port"
	noLeadingSlashRoutePathErr  = "non-empty route path must have a leading slash. Example: '/route-name'"
	credentialConfigErr         = "error: Invalid authorization configuration provided. " +
		"If provided, (Username and Password) or (ClientID and ClientSecret) must be non-empty"
)

// ValidateCfg - Validates the gateway config
func (c *KongGatewayConfig) ValidateCfg() (err error) {
	if c.Host == "" {
		return fmt.Errorf(hostErr)
	}
	if c.Proxy.Ports.HTTP == 0 && c.Proxy.Ports.HTTPS == 0 {
		return fmt.Errorf(proxyPortErr)
	}
	if c.Admin.Ports.HTTP == 0 && c.Admin.RoutePath == "" {
		return fmt.Errorf(routePathOrAdminHttpPortErr)
	}
	if c.Admin.RoutePath != "" && c.Proxy.Ports.HTTPS == 0 {
		return fmt.Errorf(routePathWithoutHttpsErr)
	}
	if c.Admin.RoutePath != "" && !strings.HasPrefix(c.Admin.RoutePath, "/") {
		return fmt.Errorf(noLeadingSlashRoutePathErr)
	}
	if c.Admin.RoutePath != "" && noCredentialsProvided(c) {
		logrus.Warn("No credentials provided. Assuming Kong Admin API requires no authorization.")
	}
	if invalidCredentialConfig(c) {
		return fmt.Errorf(credentialConfigErr)
	}
	return
}

func (c *KongGatewayConfig) IsSecured() bool {
	if c.Admin.RoutePath != "" {
		return true
	}
	return false
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
		Host: rootProps.StringPropertyValue(cfgKongHost),
		Admin: KongAdminConfig{
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
			Ports: KongPortConfig{
				HTTP:  rootProps.IntPropertyValue(cfgKongAdminPortHttp),
				HTTPS: rootProps.IntPropertyValue(cfgKongAdminPortHttps),
			},
		},
		Proxy: KongProxyConfig{
			Ports: KongPortConfig{
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
