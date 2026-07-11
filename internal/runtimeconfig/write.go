package runtimeconfig

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Write atomically replaces the generated runtime configuration at path.
func Write(path string, data []byte) (string, error) {
	resolved, err := expandPath(path)
	if err != nil {
		return "", err
	}
	directory := filepath.Dir(resolved)
	if existing, readErr := os.ReadFile(resolved); readErr == nil && bytes.Equal(existing, data) {
		return resolved, nil
	} else if readErr != nil && !os.IsNotExist(readErr) {
		return "", fmt.Errorf("read existing runtime config: %w", readErr)
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("create runtime config directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".config-*.yaml")
	if err != nil {
		return "", fmt.Errorf("create temporary runtime config: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return "", fmt.Errorf("secure temporary runtime config: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return "", fmt.Errorf("write runtime config: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return "", fmt.Errorf("sync runtime config: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close runtime config: %w", err)
	}
	if err := os.Rename(temporaryPath, resolved); err != nil {
		return "", fmt.Errorf("replace runtime config: %w", err)
	}
	return resolved, nil
}

func expandPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("runtime config path is required")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locate home directory: %w", err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return filepath.Clean(path), nil
}
