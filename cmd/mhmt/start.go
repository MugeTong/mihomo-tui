package main

import (
	"context"
	"fmt"
	"time"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/core"

	"github.com/spf13/cobra"
)

func createStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the managed Mihomo core",
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
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := manager.Start(ctx); err != nil {
				return err
			}
			fmt.Println("Mihomo started")
			return nil
		},
	}
}
