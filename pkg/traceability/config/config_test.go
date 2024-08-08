package config

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/stretchr/testify/assert"
)

func TestValidateConfig(t *testing.T) {
	cfg := &KongGatewayConfig{}

	err := cfg.ValidateCfg()
	assert.Equal(t, pathErr, err.Error())

	cfg.Logs.HTTP.Path = "/"
	err = cfg.ValidateCfg()
	assert.Equal(t, portErr, err.Error())

	cfg.Logs.HTTP.Port = 9000
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

func (f *fakeProps) AddIntProperty(name string, defaultVal int, description string, options ...properties.IntOpt) {
	f.props[name] = propData{"int", description, defaultVal}
}

func (f *fakeProps) StringPropertyValue(name string) string {
	if prop, ok := f.props[name]; ok {
		return prop.val.(string)
	}
	return ""
}

func (f *fakeProps) IntPropertyValue(name string) int {
	if prop, ok := f.props[name]; ok {
		return prop.val.(int)
	}
	return 0
}

func TestKongProperties(t *testing.T) {
	newProps := &fakeProps{props: map[string]propData{}}

	// validate add props
	AddKongProperties(newProps)
	assert.Contains(t, newProps.props, cfgKongHTTPLogsPath)
	assert.Contains(t, newProps.props, cfgKongHTTPLogsPort)

	// validate defaults
	cfg := ParseProperties(newProps)
	assert.Equal(t, "/requestlogs", cfg.Logs.HTTP.Path)
	assert.Equal(t, 9000, cfg.Logs.HTTP.Port)

	// validate changed values
	newProps.props[cfgKongHTTPLogsPath] = propData{"string", "", "another/path"}
	newProps.props[cfgKongHTTPLogsPort] = propData{"int", "", 30123}
	cfg = ParseProperties(newProps)
	assert.Equal(t, "/another/path", cfg.Logs.HTTP.Path)
	assert.Equal(t, 30123, cfg.Logs.HTTP.Port)
}
