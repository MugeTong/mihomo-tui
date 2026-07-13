package main

import (
	"fmt"

	"mihomo-tui/internal/config"

	"github.com/spf13/cobra"
)

func createOnCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "on",
		Short: "Enable proxy variables in the integrated shell",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			fmt.Printf("export http_proxy='http://127.0.0.1:%d'\n", cfg.HTTPPort)
			fmt.Printf("export https_proxy='http://127.0.0.1:%d'\n", cfg.HTTPPort)
			fmt.Printf("export all_proxy='socks5://127.0.0.1:%d'\n", cfg.SOCKSPort)
			fmt.Println("export HTTP_PROXY=\"$http_proxy\"")
			fmt.Println("export HTTPS_PROXY=\"$https_proxy\"")
			fmt.Println("export ALL_PROXY=\"$all_proxy\"")
			fmt.Printf("printf '%%s\\n' 'Shell proxy enabled'\n")
			return nil
		},
	}
}
