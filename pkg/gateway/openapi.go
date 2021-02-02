package gateway

import (
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
