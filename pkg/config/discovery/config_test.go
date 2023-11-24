package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKongGateCfg(t *testing.T) {
	cfg := &KongGatewayConfig{}

	err := cfg.ValidateCfg()
	assert.Equal(t, hostErr, err.Error())

	cfg.Proxy.Host = "localhost"
	err = cfg.ValidateCfg()
	assert.Equal(t, proxyPortErr, err.Error())

	cfg.Proxy.Ports.HTTP = 8000
	cfg.Proxy.Ports.HTTPS = 8443
	err = cfg.ValidateCfg()
	assert.Equal(t, adminUrlOrRoutePath, err.Error())

	cfg.Admin.RoutePath = "sa"
	err = cfg.ValidateCfg()
	assert.Equal(t, noLeadingSlashRoutePathErr, err.Error())

	cfg.Admin.RoutePath = "/sa"
	err = cfg.ValidateCfg()
	assert.Equal(t, err, nil)

	cfg.Admin.Auth.BasicAuth.Username = "test"
	err = cfg.ValidateCfg()
	assert.Equal(t, credentialConfigErr, err.Error())

	cfg.Admin.Auth.BasicAuth.Username = ""
	cfg.Admin.Auth.BasicAuth.Password = "sas"
	err = cfg.ValidateCfg()
	assert.Equal(t, credentialConfigErr, err.Error())

	cfg.Admin.Auth.BasicAuth.Password = ""
	cfg.Admin.Url = "http:/sld:"
	err = cfg.ValidateCfg()
	assert.Equal(t, invalidUrlErr, err.Error())

	cfg.Admin.Url = "sdl.com:8000"
	err = cfg.ValidateCfg()
	assert.Equal(t, invalidUrlErr, err.Error())

	cfg.Admin.Url = "http://sdl.com"
	err = cfg.ValidateCfg()
	assert.Equal(t, invalidUrlErr, err.Error())

	cfg.Admin.Url = "https://sds.com:8000"
	err = cfg.ValidateCfg()
	assert.Equal(t, nil, err)

}
