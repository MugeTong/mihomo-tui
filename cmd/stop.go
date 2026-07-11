package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func createStopCmd() *cobra.Command {
	var stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop Mihomo proxy silently",
		Long:  "Stop Mihomo proxy without opening TUI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Stopping Mihomo proxy...")
			// TODO: stop mihomo proxy
			fmt.Println("Mihomo proxy stopped.")
		},
	}

	return stopCmd
}
