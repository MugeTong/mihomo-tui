package runtimeconfig

import (
	"fmt"
	"os"

	"mihomo-tui/internal/config"

	"go.yaml.in/yaml/v3"
)

// ApplySettings updates app-owned top-level settings without rebuilding the
// subscription-derived proxies, groups, and rules in the runtime config.
func ApplySettings(sourcePath string, cfg config.Config) error {
	resolved, err := expandPath(sourcePath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Errorf("read runtime config for settings update: %w", err)
	}

	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("parse runtime config for settings update: %w", err)
	}
	setOptionalPort(document, "port", cfg.HTTPPort)
	setOptionalPort(document, "socks-port", cfg.SOCKSPort)
	setOptionalPort(document, "mixed-port", cfg.MixedPort)
	document["external-controller"] = config.ControllerAddress
	document["secret"] = cfg.Secret

	updated, err := yaml.Marshal(document)
	if err != nil {
		return fmt.Errorf("encode runtime config settings: %w", err)
	}
	if _, err := Write(cfg.ConfigPath, updated); err != nil {
		return err
	}
	return nil
}

func setOptionalPort(document map[string]any, key string, port int) {
	if port > 0 {
		document[key] = port
		return
	}
	delete(document, key)
}
