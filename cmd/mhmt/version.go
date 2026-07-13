package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func createVersionCmd() *cobra.Command {
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Show version information for Mihomo TUI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("mhmt version %s\n", version)
		},
	}

	return versionCmd
}
