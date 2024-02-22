package agent

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/apic"
	klib "github.com/kong/go-kong/kong"
)

const (
	httpScheme  = "http"
	httpsScheme = "https"
)

type KongRoute struct {
	*klib.Route
	defaultHost string
	basePath    string
	httpPort    int
	httpsPort   int
}

func (r *KongRoute) GetEndpoints() []apic.EndpointDefinition {
	endpoints := r.handleHosts()
	if len(endpoints) == 0 {
		return r.handlePaths(r.defaultHost, r.basePath)
	}
	return endpoints
}

func (r *KongRoute) handleHosts() []apic.EndpointDefinition {
	endpoints := make([]apic.EndpointDefinition, 0)
	for _, host := range r.Hosts {
		endpoints = append(endpoints, r.handlePaths(*host, "")...)
	}
	return endpoints
}

func (r *KongRoute) handlePaths(host, basePath string) []apic.EndpointDefinition {
	endpoints := make([]apic.EndpointDefinition, 0)
	for _, path := range r.Paths {
		fullPath := *path
		if basePath != "" {
			// prepend the base path to the path
			fullPath = fmt.Sprintf("%s%s", basePath, fullPath)
		}
		endpoints = append(endpoints, r.handleProtocols(host, fullPath)...)
	}
	return endpoints
}

func (r *KongRoute) handleProtocols(host, path string) []apic.EndpointDefinition {
	endpoints := make([]apic.EndpointDefinition, 0)
	for _, protocol := range r.Protocols {
		if *protocol == httpScheme && r.httpPort != 0 {
			endpoints = append(endpoints, apic.EndpointDefinition{
				Host:     host,
				Port:     int32(r.httpPort),
				Protocol: httpScheme,
				BasePath: path,
			})
		}
		if *protocol == httpsScheme && r.httpsPort != 0 {
			endpoints = append(endpoints, apic.EndpointDefinition{
				Host:     host,
				Port:     int32(r.httpsPort),
				Protocol: httpsScheme,
				BasePath: path,
			})
		}
	}
	return endpoints
}
