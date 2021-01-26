package cmd

import (
	"os"

	"github.com/Axway/agents-kong/pkg/cmd/discovery"
	"github.com/Axway/agents-kong/pkg/cmd/traceability"

	"github.com/spf13/cobra"
)

// RootCmd is the root
var RootCmd = &cobra.Command{
	Use:   "kong-agent",
	Short: "Kong Discovery & Traceability Agent",
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	RootCmd.AddCommand(discovery.DiscoveryCmd.RootCmd())
	RootCmd.AddCommand(traceability.TraceCmd.RootCmd())
}
