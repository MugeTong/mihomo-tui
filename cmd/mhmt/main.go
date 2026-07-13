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
		Use:           "mhmt",
		Version:       version,
		Short:         "A TUI for Mihomo",
		Long:          "A TUI for Mihomo, a tool to manage your nodes",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          func(_ *cobra.Command, _ []string) error { return app.StartTUI(version) },
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(createVersionCmd(), createStartCmd(), createStopCmd(), createOnCmd(), createOffCmd())
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
