package core

import (
	"path/filepath"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/local"
)

// NewConfiguredManager creates the managed Mihomo process shared by the TUI
// and non-interactive CLI commands.
func NewConfiguredManager(cfg config.Config) (*ProcessManager, error) {
	dataDir, err := local.DataDir()
	if err != nil {
		return nil, err
	}
	stateDir, err := local.StateDir()
	if err != nil {
		return nil, err
	}
	return NewProcessManager(ProcessOptions{
		BinaryPath:        cfg.BinaryPath,
		ConfigPath:        cfg.ConfigPath,
		DataDir:           filepath.Join(dataDir, "mihomo"),
		PIDPath:           filepath.Join(stateDir, "mihomo.pid"),
		LogPath:           filepath.Join(stateDir, "mihomo.log"),
		ControllerAddress: config.ControllerAddress,
	}), nil
}
