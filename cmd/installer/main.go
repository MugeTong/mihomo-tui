package main

import (
	"compress/gzip"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/runtimeconfig"
	"mihomo-tui/internal/subscription"
)

// The release build stages platform-specific files in payload before compiling
// this installer. The repository placeholder keeps normal `go test ./...`
// builds valid without checking large third-party binaries into Git.
//
//go:embed payload/*
var payload embed.FS

var (
	version     = "dev"
	coreVersion = "dev"
)

func main() {
	if err := install(); err != nil {
		fmt.Fprintf(os.Stderr, "Installation failed: %v\n", err)
		os.Exit(1)
	}
}

func install() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("locate home directory: %w", err)
	}
	binDir := filepath.Join(home, ".local", "bin")
	dataHome, err := xdgHome("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	if err != nil {
		return err
	}
	configHome, err := xdgHome("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	if err != nil {
		return err
	}
	appDataDir := filepath.Join(dataHome, "mihomo-tui")
	coreDir := filepath.Join(appDataDir, "bin")
	mihomoDataDir := filepath.Join(appDataDir, "mihomo")
	licenseDir := filepath.Join(appDataDir, "licenses")
	configDir := filepath.Join(configHome, "mihomo-tui")
	corePath := filepath.Join(coreDir, "mihomo-v"+coreVersion)

	for _, dir := range []string{binDir, coreDir, mihomoDataDir, licenseDir, configDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	if err := installFile("payload/mhmt", filepath.Join(binDir, "mhmt"), 0o755); err != nil {
		return err
	}
	if err := installGzip("payload/mihomo.gz", corePath); err != nil {
		return err
	}
	if err := installFile("payload/geoip.metadb", filepath.Join(mihomoDataDir, "geoip.metadb"), 0o644); err != nil {
		return err
	}
	for _, name := range []string{
		"LICENSE",
		"THIRD_PARTY_NOTICES.md",
		"mihomo-GPL-3.0.txt",
		"bubbletea-MIT.txt",
		"meta-rules-dat-GPL-3.0.txt",
		"MIHOMO_SOURCE.txt",
		"META_RULES_SOURCE.txt",
	} {
		if err := installFile("payload/"+name, filepath.Join(licenseDir, name), 0o644); err != nil {
			return err
		}
	}
	if err := writeInitialConfig(filepath.Join(configDir, "config.json"), corePath); err != nil {
		return err
	}
	if err := writeInitialState(filepath.Join(configDir, "state.json")); err != nil {
		return err
	}
	if err := writeInitialRuntimeConfig(filepath.Join(configDir, "config.yaml")); err != nil {
		return err
	}

	fmt.Printf("Mihomo TUI %s installed successfully.\n", version)
	fmt.Printf("  TUI:    %s\n", filepath.Join(binDir, "mhmt"))
	fmt.Printf("  Mihomo: %s\n", corePath)
	if !pathContains(binDir) {
		fmt.Printf("  Note: add %s to PATH.\n", binDir)
	}
	return nil
}

func installFile(source, destination string, mode fs.FileMode) error {
	data, err := payload.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read embedded %s: %w", source, err)
	}
	return atomicWrite(destination, mode, func(writer io.Writer) error {
		_, err := writer.Write(data)
		return err
	})
}

func installGzip(source, destination string) error {
	file, err := payload.Open(source)
	if err != nil {
		return fmt.Errorf("open embedded Mihomo archive: %w", err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open embedded Mihomo gzip stream: %w", err)
	}
	defer reader.Close()
	return atomicWrite(destination, 0o755, func(writer io.Writer) error {
		_, err := io.Copy(writer, reader)
		return err
	})
}

func atomicWrite(destination string, mode fs.FileMode, write func(io.Writer) error) error {
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".install-*")
	if err != nil {
		return fmt.Errorf("create temporary install file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return fmt.Errorf("set install permissions: %w", err)
	}
	if err := write(temporary); err != nil {
		temporary.Close()
		return fmt.Errorf("write %s: %w", destination, err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync %s: %w", destination, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close %s: %w", destination, err)
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return fmt.Errorf("install %s: %w", destination, err)
	}
	return nil
}

func writeInitialConfig(path, corePath string) error {
	data, err := json.MarshalIndent(map[string]string{"binary_path": corePath}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode initial settings: %w", err)
	}
	data = append(data, '\n')
	return writeMissing(path, data, 0o600)
}

func writeInitialState(path string) error {
	data, err := json.MarshalIndent(subscription.NewState(), "", "  ")
	if err != nil {
		return fmt.Errorf("encode initial subscription state: %w", err)
	}
	return writeMissing(path, append(data, '\n'), 0o600)
}

func writeInitialRuntimeConfig(path string) error {
	cfg := config.Default()
	cfg.ConfigPath = path
	data, err := runtimeconfig.Generate(cfg, subscription.NewState())
	if err != nil {
		return fmt.Errorf("generate initial runtime config: %w", err)
	}
	return writeMissing(path, data, 0o600)
}

func writeMissing(path string, data []byte, mode fs.FileMode) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect existing file %s: %w", path, err)
	}
	return atomicWrite(path, mode, func(writer io.Writer) error {
		_, err := writer.Write(data)
		return err
	})
}

func xdgHome(environment, fallback string) (string, error) {
	value := strings.TrimSpace(os.Getenv(environment))
	if value == "" {
		return fallback, nil
	}
	if !filepath.IsAbs(value) {
		return "", fmt.Errorf("%s must be an absolute path", environment)
	}
	return filepath.Clean(value), nil
}

func pathContains(directory string) bool {
	for _, candidate := range filepath.SplitList(os.Getenv("PATH")) {
		if filepath.Clean(candidate) == filepath.Clean(directory) {
			return true
		}
	}
	return false
}
