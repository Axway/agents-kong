package gateway

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tidwall/sjson"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/tidwall/gjson"
)

type Openapi struct {
	spec string
}

func (oas *Openapi) ResourceType() string {
	oas2 := gjson.Get(oas.spec, "swagger").Str
	oas3 := gjson.Get(oas.spec, "openapi").Str
	if len(oas2) > 0 {
		return apic.Oas2
	}
	if len(oas3) > 0 {
		return apic.Oas3
	}
	log.Error("not a valid spec")
	return ""
}

func (oas *Openapi) Description() string {
	return gjson.Get(oas.spec, "info.description").Str
}

func (oas *Openapi) Version() string {
	return gjson.Get(oas.spec, "info.version").Str
}

func (oas *Openapi) SetOas3Servers(servers openapi3.Servers) {
	if oas.ResourceType() == apic.Oas3 {
		openAPI, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData([]byte(oas.spec))
		if err != nil {
			log.Errorf("failed to load OAS3 spec: %s", err)
			return
		}
		openAPI.Servers = servers
		swaggerBytes, err := openAPI.MarshalJSON()
		if err != nil {
			log.Errorf("failed to unmarshal to OAS3 spec: %s", err)
			return
		}
		oas.spec = string(swaggerBytes)
	}
}
func (oas *Openapi) SetOas2Host(defaultHost string, defaultBasePath string, schemes []*string) {
	if oas.ResourceType() == apic.Oas2 {
		spec, _ := sjson.Set(oas.spec, "host", defaultHost)
		spec, _ = sjson.Set(spec, "schemes", schemes)
		spec, _ = sjson.Set(spec, "basePath", defaultBasePath)
		oas.spec = spec
	}
}
