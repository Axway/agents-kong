package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKongGateCfg(t *testing.T) {
	cfg := &KongGatewayConfig{}

	err := cfg.ValidateCfg()
	assert.Equal(t, hostErr, err.Error())

	cfg.Host = "localhost"
	err = cfg.ValidateCfg()
	assert.Equal(t, proxyPortErr, err.Error())

	cfg.Proxy.Port.HTTP = 8000
	err = cfg.ValidateCfg()
	assert.Equal(t, routePathOrAdminHttpPortErr, err.Error())

	cfg.Admin.Port.HTTP = 8001
	cfg.Admin.RoutePath = "sa"
	err = cfg.ValidateCfg()
	assert.Equal(t, routePathWithoutHttpsErr, err.Error())

	cfg.Proxy.Port.HTTPS = 8443
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

	cfg.Admin.Auth.OAuth.ClientID = "test"
	cfg.Admin.Auth.BasicAuth.Password = ""
	err = cfg.ValidateCfg()
	assert.Equal(t, credentialConfigErr, err.Error())

	cfg.Admin.Auth.OAuth.ClientID = ""
	cfg.Admin.Auth.OAuth.ClientSecret = "test"
	err = cfg.ValidateCfg()
	assert.Equal(t, credentialConfigErr, err.Error())

	cfg.Admin.Auth.OAuth.ClientID = "test"
	err = cfg.ValidateCfg()
	assert.Equal(t, nil, err)

}
