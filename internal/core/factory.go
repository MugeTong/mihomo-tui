package core

import (
	"path/filepath"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/xdg"
)

// NewConfiguredManager creates the managed Mihomo process shared by the TUI
// and non-interactive CLI commands.
func NewConfiguredManager(cfg config.Config) (*ProcessManager, error) {
	settingsPath, err := config.Path()
	if err != nil {
		return nil, err
	}
	dataDir, err := xdg.AppDataDir("mihomo-tui")
	if err != nil {
		return nil, err
	}
	appDir := filepath.Dir(settingsPath)
	return NewProcessManager(ProcessOptions{
		BinaryPath:        cfg.BinaryPath,
		ConfigPath:        cfg.ConfigPath,
		DataDir:           filepath.Join(dataDir, "mihomo"),
		PIDPath:           filepath.Join(appDir, "mihomo.pid"),
		LogPath:           filepath.Join(appDir, "mihomo.log"),
		ControllerAddress: config.ControllerAddress,
	}), nil
}
