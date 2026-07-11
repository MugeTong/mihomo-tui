package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func createStartCmd() *cobra.Command {
	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start Mihomo proxy silently",
		Long:  "Start Mihomo proxy in background without opening TUI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Starting Mihomo proxy...")
			// TODO: start mihomo proxy in background
			fmt.Println("Mihomo proxy started.")
		},
	}

	return startCmd
}
