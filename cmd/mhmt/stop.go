package main

import (
	"fmt"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/core"

	"github.com/spf13/cobra"
)

func createStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the managed Mihomo core",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			manager, err := core.NewConfiguredManager(cfg)
			if err != nil {
				return err
			}
			if err := manager.Stop(); err != nil {
				return err
			}
			fmt.Println("Mihomo stopped")
			return nil
		},
	}
}
