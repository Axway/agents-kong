package main

import (
	"fmt"
	"os"

	discovery "github.com/Axway/agents-kong/pkg/discovery/cmd"
)

func main() {
	os.Setenv("AGENTFEATURES_VERSIONCHECKER", "false")
	if err := discovery.DiscoveryCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
