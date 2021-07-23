package main

import (
	"fmt"
	"os"

	"github.com/Axway/agents-kong/pkg/cmd/discovery"
)

func main() {
	if err := discovery.DiscoveryCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
