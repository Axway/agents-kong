package gateway

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
)

var (
	kHttp  = kong.String("http")
	kHttps = kong.String("https")
)

func TestKongRoute(t *testing.T) {
	testCases := map[string]struct {
		cfgHost           string
		cfgHttpPort       int
		cfgHttpsPort      int
		cfgBasePath       string
		route             *kong.Route
		expectedEndpoints []apic.EndpointDefinition
	}{
		"http default route, no base path": {
			cfgHost:     "my.host.com",
			cfgHttpPort: 8080,
			route: &kong.Route{
				Hosts:     []*string{},
				Protocols: []*string{kHttp, kHttps},
				Paths:     []*string{kong.String("/path")},
			},
			expectedEndpoints: []apic.EndpointDefinition{
				{
					Host:     "my.host.com",
					Port:     8080,
					Protocol: "http",
					BasePath: "/path",
				},
			},
		},
		"https only route only has http": {
			cfgHost:      "my.host.com",
			cfgHttpsPort: 8443,
			route: &kong.Route{
				Hosts:     []*string{},
				Protocols: []*string{kHttp},
				Paths:     []*string{kong.String("/path")},
			},
			expectedEndpoints: []apic.EndpointDefinition{},
		},
		"http only route only has https": {
			cfgHost:     "my.host.com",
			cfgHttpPort: 8080,
			route: &kong.Route{
				Hosts:     []*string{},
				Protocols: []*string{kHttps},
				Paths:     []*string{kong.String("/path")},
			},
			expectedEndpoints: []apic.EndpointDefinition{},
		},
		"https default route, no base path": {
			cfgHost:      "my.host.com",
			cfgHttpsPort: 8443,
			route: &kong.Route{
				Hosts:     []*string{},
				Protocols: []*string{kHttp, kHttps},
				Paths:     []*string{kong.String("/path")},
			},
			expectedEndpoints: []apic.EndpointDefinition{
				{
					Host:     "my.host.com",
					Port:     8443,
					Protocol: "https",
					BasePath: "/path",
				},
			},
		},
		"http and https allowed, no base path, route only has http": {
			cfgHost:      "my.host.com",
			cfgHttpPort:  8080,
			cfgHttpsPort: 8443,
			route: &kong.Route{
				Hosts:     []*string{},
				Protocols: []*string{kHttp},
				Paths:     []*string{kong.String("/path")},
			},
			expectedEndpoints: []apic.EndpointDefinition{
				{
					Host:     "my.host.com",
					Port:     8080,
					Protocol: "http",
					BasePath: "/path",
				},
			},
		},
		"http and https allowed, no base path, route only has https": {
			cfgHost:      "my.host.com",
			cfgHttpPort:  8080,
			cfgHttpsPort: 8443,
			route: &kong.Route{
				Hosts:     []*string{},
				Protocols: []*string{kHttps},
				Paths:     []*string{kong.String("/path")},
			},
			expectedEndpoints: []apic.EndpointDefinition{
				{
					Host:     "my.host.com",
					Port:     8443,
					Protocol: "https",
					BasePath: "/path",
				},
			},
		},
		"http and https default routes, no base path": {
			cfgHost:      "my.host.com",
			cfgHttpPort:  8080,
			cfgHttpsPort: 8443,
			route: &kong.Route{
				Hosts:     []*string{},
				Protocols: []*string{kHttp, kHttps},
				Paths:     []*string{kong.String("/path")},
			},
			expectedEndpoints: []apic.EndpointDefinition{
				{
					Host:     "my.host.com",
					Port:     8080,
					Protocol: "http",
					BasePath: "/path",
				},
				{
					Host:     "my.host.com",
					Port:     8443,
					Protocol: "https",
					BasePath: "/path",
				},
			},
		},
		"http and https default routes, with base path": {
			cfgHost:      "my.host.com",
			cfgHttpPort:  8080,
			cfgHttpsPort: 8443,
			cfgBasePath:  "/base",
			route: &kong.Route{
				Hosts:     []*string{},
				Protocols: []*string{kHttp, kHttps},
				Paths:     []*string{kong.String("/path1"), kong.String("/path2")},
			},
			expectedEndpoints: []apic.EndpointDefinition{
				{
					Host:     "my.host.com",
					Port:     8080,
					Protocol: "http",
					BasePath: "/base/path1",
				},
				{
					Host:     "my.host.com",
					Port:     8443,
					Protocol: "https",
					BasePath: "/base/path1",
				},
				{
					Host:     "my.host.com",
					Port:     8080,
					Protocol: "http",
					BasePath: "/base/path2",
				},
				{
					Host:     "my.host.com",
					Port:     8443,
					Protocol: "https",
					BasePath: "/base/path2",
				},
			},
		},
		"http and https only configured routes": {
			cfgHost:      "my.host.com",
			cfgHttpPort:  8080,
			cfgHttpsPort: 8443,
			cfgBasePath:  "/base",
			route: &kong.Route{
				Hosts:     []*string{kong.String("kong.host.com")},
				Protocols: []*string{kHttp, kHttps},
				Paths:     []*string{kong.String("/path1"), kong.String("/path2")},
			},
			expectedEndpoints: []apic.EndpointDefinition{
				{
					Host:     "kong.host.com",
					Port:     8080,
					Protocol: "http",
					BasePath: "/path1",
				},
				{
					Host:     "kong.host.com",
					Port:     8443,
					Protocol: "https",
					BasePath: "/path1",
				},
				{
					Host:     "kong.host.com",
					Port:     8080,
					Protocol: "http",
					BasePath: "/path2",
				},
				{
					Host:     "kong.host.com",
					Port:     8443,
					Protocol: "https",
					BasePath: "/path2",
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			route := KongRoute{
				Route:       tc.route,
				defaultHost: tc.cfgHost,
				httpPort:    tc.cfgHttpPort,
				httpsPort:   tc.cfgHttpsPort,
				basePath:    tc.cfgBasePath,
			}

			endpoints := route.GetEndpoints()

			assert.Equal(t, len(endpoints), len(tc.expectedEndpoints))
			assert.ElementsMatch(t, endpoints, tc.expectedEndpoints)
		})
	}
}
