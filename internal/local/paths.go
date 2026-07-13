package local

import (
	"fmt"
	"os"
	"path/filepath"
)

const appName = "mihomo-tui"

// Layout is the single source of truth for every per-user directory.
type Layout struct {
	ConfigDir string
	DataDir   string
	StateDir  string
	BinDir    string
}

// ResolveLayout derives the fixed application layout from the user's home.
func ResolveLayout() (Layout, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Layout{}, fmt.Errorf("locate user home directory: %w", err)
	}
	return Layout{
		ConfigDir: filepath.Join(home, ".config", appName),
		DataDir:   filepath.Join(home, ".local", "share", appName),
		StateDir:  filepath.Join(home, ".local", "state", appName),
		BinDir:    filepath.Join(home, ".local", "bin"),
	}, nil
}

func ConfigDir() (string, error) {
	layout, err := ResolveLayout()
	return layout.ConfigDir, err
}

func DataDir() (string, error) {
	layout, err := ResolveLayout()
	return layout.DataDir, err
}

func StateDir() (string, error) {
	layout, err := ResolveLayout()
	return layout.StateDir, err
}

func BinDir() (string, error) {
	layout, err := ResolveLayout()
	return layout.BinDir, err
}

func initializeDirs(layout Layout) error {
	for _, directory := range []string{layout.ConfigDir, layout.DataDir, layout.StateDir, layout.BinDir} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", directory, err)
		}
	}
	return nil
}
