package xdg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigHome follows the XDG Base Directory specification on every platform.
func ConfigHome() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); configured != "" {
		if !filepath.IsAbs(configured) {
			return "", fmt.Errorf("XDG_CONFIG_HOME must be an absolute path")
		}
		return filepath.Clean(configured), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".config"), nil
}

func AppConfigDir(app string) (string, error) {
	home, err := ConfigHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, app), nil
}
