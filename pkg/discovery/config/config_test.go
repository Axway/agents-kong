package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKongGatewayCfg(t *testing.T) {
	cfg := &KongGatewayConfig{}

	err := cfg.ValidateCfg()
	assert.Equal(t, hostErr, err.Error())

	cfg.Proxy.Host = "localhost"
	err = cfg.ValidateCfg()
	assert.Equal(t, httpPortErr, err.Error())

	cfg.Proxy.Ports.HTTP.Value = 8000
	err = cfg.ValidateCfg()
	assert.Equal(t, httpsPortErr, err.Error())

	cfg.Proxy.Ports.HTTPS.Value = 8443
	cfg.Proxy.Ports.HTTP.Disable = true
	cfg.Proxy.Ports.HTTPS.Disable = true
	err = cfg.ValidateCfg()
	assert.Equal(t, portErr, err.Error())

	cfg.Proxy.Ports.HTTP.Disable = false
	cfg.Proxy.BasePath = "base"
	err = cfg.ValidateCfg()
	assert.Equal(t, basePathPrefixErr, err.Error())

	cfg.Proxy.BasePath = "/base/"
	err = cfg.ValidateCfg()
	assert.Equal(t, basePathSuffixErr, err.Error())

	cfg.Proxy.BasePath = "/base"
	cfg.Admin.Url = "sdl.com:8000"
	err = cfg.ValidateCfg()
	assert.Equal(t, invalidUrlErr, err.Error())

	cfg.Admin.Url = "http://sdl.com"
	err = cfg.ValidateCfg()
	assert.Equal(t, invalidUrlErr, err.Error())

	cfg.Admin.Url = "https://sds.com:8000"
	cfg.Admin.Auth.BasicAuth.Username = "test"
	err = cfg.ValidateCfg()
	assert.Equal(t, credentialConfigErr, err.Error())

	cfg.Admin.Auth.BasicAuth.Username = ""
	cfg.Admin.Auth.BasicAuth.Password = "sas"
	err = cfg.ValidateCfg()
	assert.Equal(t, credentialConfigErr, err.Error())

	cfg.Admin.Auth.BasicAuth.Password = ""

	err = cfg.ValidateCfg()
	assert.Equal(t, nil, err)

}

type propData struct {
	pType string
	desc  string
	val   interface{}
}

type fakeProps struct {
	props map[string]propData
}

func (f *fakeProps) AddStringProperty(name string, defaultVal string, description string) {
	f.props[name] = propData{"string", description, defaultVal}
}

func (f *fakeProps) AddStringSliceProperty(name string, defaultVal []string, description string) {
	f.props[name] = propData{"string", description, defaultVal}
}

func (f *fakeProps) AddIntProperty(name string, defaultVal int, description string) {
	f.props[name] = propData{"int", description, defaultVal}
}

func (f *fakeProps) AddBoolProperty(name string, defaultVal bool, description string) {
	f.props[name] = propData{"bool", description, defaultVal}
}

func (f *fakeProps) StringPropertyValue(name string) string {
	if prop, ok := f.props[name]; ok {
		return prop.val.(string)
	}
	return ""
}

func (f *fakeProps) StringSlicePropertyValue(name string) []string {
	if prop, ok := f.props[name]; ok {
		return prop.val.([]string)
	}
	return []string{}
}

func (f *fakeProps) IntPropertyValue(name string) int {
	if prop, ok := f.props[name]; ok {
		return prop.val.(int)
	}
	return 0
}

func (f *fakeProps) BoolPropertyValue(name string) bool {
	if prop, ok := f.props[name]; ok {
		return prop.val.(bool)
	}
	return false
}

func TestKongProperties(t *testing.T) {
	newProps := &fakeProps{props: map[string]propData{}}

	// validate add props
	AddKongProperties(newProps)
	assert.Contains(t, newProps.props, cfgKongACLDisabled)
	assert.Contains(t, newProps.props, cfgKongAdminUrl)
	assert.Contains(t, newProps.props, cfgKongAdminAPIKey)
	assert.Contains(t, newProps.props, cfgKongAdminAPIKeyHeader)
	assert.Contains(t, newProps.props, cfgKongAdminBasicUsername)
	assert.Contains(t, newProps.props, cfgKongAdminBasicPassword)
	assert.Contains(t, newProps.props, cfgKongProxyHost)
	assert.Contains(t, newProps.props, cfgKongProxyPortHttp)
	assert.Contains(t, newProps.props, cfgKongProxyPortHttpDisabled)
	assert.Contains(t, newProps.props, cfgKongProxyPortHttps)
	assert.Contains(t, newProps.props, cfgKongProxyPortHttpsDisabled)
	assert.Contains(t, newProps.props, cfgKongProxyBasePath)
	assert.Contains(t, newProps.props, cfgKongSpecURLPaths)
	assert.Contains(t, newProps.props, cfgKongSpecLocalPath)
	assert.Contains(t, newProps.props, cfgKongSpecFilter)
	assert.Contains(t, newProps.props, cfgKongSpecDevPortal)

	// validate defaults
	cfg := ParseProperties(newProps)
	assert.Equal(t, false, cfg.ACL.Disabled)
	assert.Equal(t, "", cfg.Admin.Url)
	assert.Equal(t, "", cfg.Admin.Auth.APIKey.Value)
	assert.Equal(t, "", cfg.Admin.Auth.APIKey.Header)
	assert.Equal(t, "", cfg.Admin.Auth.BasicAuth.Username)
	assert.Equal(t, "", cfg.Admin.Auth.BasicAuth.Password)
	assert.Equal(t, "", cfg.Proxy.Host)
	assert.Equal(t, 80, cfg.Proxy.Ports.HTTP.Value)
	assert.Equal(t, 443, cfg.Proxy.Ports.HTTPS.Value)
	assert.Equal(t, false, cfg.Proxy.Ports.HTTP.Disable)
	assert.Equal(t, false, cfg.Proxy.Ports.HTTPS.Disable)
	assert.Equal(t, "", cfg.Proxy.BasePath)
	assert.Equal(t, []string{}, cfg.Spec.URLPaths)
	assert.Equal(t, "", cfg.Spec.LocalPath)
	assert.Equal(t, "", cfg.Spec.Filter)
	assert.Equal(t, false, cfg.Spec.DevPortalEnabled)

	// validate changed values
	newProps.props[cfgKongACLDisabled] = propData{"bool", "", true}
	newProps.props[cfgKongAdminUrl] = propData{"string", "", "http://host:port/path"}
	newProps.props[cfgKongAdminAPIKey] = propData{"string", "", "apikey"}
	newProps.props[cfgKongAdminAPIKeyHeader] = propData{"string", "", "header"}
	newProps.props[cfgKongAdminBasicUsername] = propData{"string", "", "username"}
	newProps.props[cfgKongAdminBasicPassword] = propData{"string", "", "password"}
	newProps.props[cfgKongProxyHost] = propData{"string", "", "proxyhost"}
	newProps.props[cfgKongProxyPortHttp] = propData{"int", "", 8080}
	newProps.props[cfgKongProxyPortHttps] = propData{"int", "", 8443}
	newProps.props[cfgKongProxyHost] = propData{"string", "", "proxyhost"}
	newProps.props[cfgKongSpecURLPaths] = propData{"string", "", []string{"path1", "path2"}}
	newProps.props[cfgKongSpecLocalPath] = propData{"string", "", "/path/to/specs"}
	newProps.props[cfgKongSpecFilter] = propData{"string", "", "tag_filter"}
	newProps.props[cfgKongSpecDevPortal] = propData{"bool", "", true}
	cfg = ParseProperties(newProps)
	assert.Equal(t, true, cfg.ACL.Disabled)
	assert.Equal(t, "http://host:port/path", cfg.Admin.Url)
	assert.Equal(t, "apikey", cfg.Admin.Auth.APIKey.Value)
	assert.Equal(t, "header", cfg.Admin.Auth.APIKey.Header)
	assert.Equal(t, "username", cfg.Admin.Auth.BasicAuth.Username)
	assert.Equal(t, "password", cfg.Admin.Auth.BasicAuth.Password)
	assert.Equal(t, "proxyhost", cfg.Proxy.Host)
	assert.Equal(t, 8080, cfg.Proxy.Ports.HTTP.Value)
	assert.Equal(t, 8443, cfg.Proxy.Ports.HTTPS.Value)
	assert.Equal(t, false, cfg.Proxy.Ports.HTTP.Disable)
	assert.Equal(t, false, cfg.Proxy.Ports.HTTPS.Disable)
	assert.Equal(t, "", cfg.Proxy.BasePath)
	assert.Equal(t, []string{"path1", "path2"}, cfg.Spec.URLPaths)
	assert.Equal(t, "/path/to/specs", cfg.Spec.LocalPath)
	assert.Equal(t, "tag_filter", cfg.Spec.Filter)
	assert.Equal(t, true, cfg.Spec.DevPortalEnabled)

	// validate no port configured when port type disabled
	newProps.props[cfgKongProxyPortHttpDisabled] = propData{"bool", "", true}
	newProps.props[cfgKongProxyPortHttpsDisabled] = propData{"bool", "", true}
	cfg = ParseProperties(newProps)
	assert.Equal(t, 0, cfg.Proxy.Ports.HTTP.Value)
	assert.Equal(t, 0, cfg.Proxy.Ports.HTTPS.Value)
	assert.Equal(t, true, cfg.Proxy.Ports.HTTP.Disable)
	assert.Equal(t, true, cfg.Proxy.Ports.HTTPS.Disable)
}
