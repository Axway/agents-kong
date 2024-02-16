package main

import (
	"fmt"
	"os"

	_ "github.com/Axway/agent-sdk/pkg/traceability"

	traceability "github.com/Axway/agents-kong/pkg/traceability/cmd"
)

func main() {
	os.Setenv("AGENTFEATURES_VERSIONCHECKER", "false")

	// use the pod name as the agent name
	podName := os.Getenv("POD_NAME")
	if podName != "" {
		os.Setenv("CENTRAL_AGENTNAME", podName)
	}

	if err := traceability.TraceCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
