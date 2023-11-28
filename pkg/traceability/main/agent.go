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
	pod_name := os.Getenv("POD_NAME")
	if pod_name != "" {
		os.Setenv("CENTRAL_AGENTNAME", pod_name)
	}

	if err := traceability.TraceCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
