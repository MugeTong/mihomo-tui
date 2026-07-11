package main

import (
	"fmt"
	"os"

	"mihomo-tui/internal/app"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	var rootCmd = &cobra.Command{
		Use:     "mhmt",
		Version: version,
		Short:   "A TUI for Mihomo",
		Long:    "A TUI for Mihomo, a tool to manage your nods",
		Run: func(cmd *cobra.Command, args []string) {
			if err := app.StartTUI(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(createVersionCmd())
	rootCmd.AddCommand(createStartCmd())
	rootCmd.AddCommand(createStopCmd())
	rootCmd.Execute()
}
