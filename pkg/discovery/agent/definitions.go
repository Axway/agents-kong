package agent

import (
	"github.com/Axway/agent-sdk/pkg/apic"
)

type KongAPI struct {
	spec              []byte
	id                string
	name              string
	description       string
	version           string
	url               string
	documentation     []byte
	resourceType      string
	endpoints         []apic.EndpointDefinition
	image             string
	imageContentType  string
	crds              []string
	apiUpdateSeverity string
	agentDetails      map[string]string
	tags              []string
	stage             string
	stageName         string
	ard               string
}
