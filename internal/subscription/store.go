package subscription

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	Path string
}

func DefaultStatePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config directory: %w", err)
	}
	return filepath.Join(configDir, "mihomo-tui", "state.json"), nil
}

func (s Store) Load() (State, ReconcileReport, error) {
	state := NewState()
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state, ReconcileReport{}, nil
		}
		return state, ReconcileReport{}, fmt.Errorf("read subscription state: %w", err)
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return NewState(), ReconcileReport{}, fmt.Errorf("parse subscription state: %w", err)
	}
	report := state.Reconcile()
	return state, report, nil
}

func (s Store) Save(state State) error {
	report := state.Reconcile()
	if len(report.Issues) > 0 {
		return fmt.Errorf("subscription state is inconsistent: %s", report.Issues[0])
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return fmt.Errorf("create subscription state directory: %w", err)
	}

	temporary, err := os.CreateTemp(filepath.Dir(s.Path), ".state-*.json")
	if err != nil {
		return fmt.Errorf("create temporary subscription state: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure temporary subscription state: %w", err)
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(state); err != nil {
		temporary.Close()
		return fmt.Errorf("encode subscription state: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync subscription state: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close subscription state: %w", err)
	}
	if err := os.Rename(temporaryPath, s.Path); err != nil {
		return fmt.Errorf("replace subscription state: %w", err)
	}
	return nil
}
