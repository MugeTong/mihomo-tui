package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"mihomo-tui/internal/xdg"
)

const appDirName = "mihomo-tui"

func Load() (Config, error) {
	cfg := Default()
	defaults := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	if cfg.ConfigPath == "" {
		cfg.ConfigPath = defaults.ConfigPath
	}
	if cfg.BinaryPath == "" {
		cfg.BinaryPath = defaults.BinaryPath
	}
	if cfg.HTTPPort == 0 {
		cfg.HTTPPort = defaults.HTTPPort
	}
	if cfg.SOCKSPort == 0 {
		cfg.SOCKSPort = defaults.SOCKSPort
	}
	if cfg.MixedPort == 0 {
		cfg.MixedPort = defaults.MixedPort
	}
	if len(cfg.Policies) == 0 {
		cfg.Policies = defaults.Policies
	}

	return cfg, nil
}

func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func Path() (string, error) {
	configDir, err := xdg.AppConfigDir(appDirName)
	if err != nil {
		return "", fmt.Errorf("locate user config directory: %w", err)
	}
	return filepath.Join(configDir, "config.json"), nil
}
