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
