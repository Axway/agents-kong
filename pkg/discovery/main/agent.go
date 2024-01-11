package main

import (
	"fmt"
	"os"

	discovery "github.com/Axway/agents-kong/pkg/discovery/cmd"
)

func main() {
	os.Setenv("AGENTFEATURES_VERSIONCHECKER", "false")

	// update to set the default pattern for kong discovery
	pattern := os.Getenv("CENTRAL_APISERVICEREVISIONPATTERN")
	if pattern == "" {
		os.Setenv("CENTRAL_APISERVICEREVISIONPATTERN", "{{.APIServiceName}} - {{.Date:YYYY/MM/DD}} - r {{.Revision}}")
	}
	if err := discovery.DiscoveryCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
