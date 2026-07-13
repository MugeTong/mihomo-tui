package main

import (
	_ "embed"
	"fmt"
	"os"

	"mihomo-tui/internal/install"
)

var (
	version     = "dev"
	coreVersion = "dev"
)

// Makefile replaces these placeholders in each platform staging directory.
//
//go:embed mhmt
var mhmtBin []byte

//go:embed mihomo.gz
var mihomoArchive []byte

//go:embed geoip.metadb
var geoIP []byte

func main() {
	result, err := install.Run(install.Payload{
		Client:        mhmtBin,
		MihomoArchive: mihomoArchive,
		GeoIP:         geoIP,
		MihomoVersion: coreVersion,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Installation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Mihomo TUI %s installed successfully.\n", version)
	fmt.Printf("  TUI:    %s\n", result.ClientPath)
	fmt.Printf("  Mihomo: %s\n", result.CorePath)
	fmt.Printf("  Data:   %s\n", result.DataDir)
}
