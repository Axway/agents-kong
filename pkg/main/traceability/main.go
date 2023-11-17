package main

import (
	"fmt"
	"os"

	_ "github.com/Axway/agent-sdk/pkg/traceability"

	"github.com/Axway/agents-kong/pkg/cmd/traceability"
)

func main() {
	if err := traceability.TraceCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
