package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func createOffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "off",
		Short: "Disable proxy variables in the integrated shell",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("unset http_proxy https_proxy all_proxy")
			fmt.Println("unset HTTP_PROXY HTTPS_PROXY ALL_PROXY")
			return nil
		},
	}
}
