package local

import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed licenses/*
var licenses embed.FS

func installLicenses(layout Layout) error {
	licensesDir := filepath.Join(layout.DataDir, "licenses")
	if err := os.MkdirAll(licensesDir, 0755); err != nil {
		return err
	}

	// Read the embedded license files
	entries, err := licenses.ReadDir("licenses")
	if err != nil {
		return err
	}

	// Write each license file to the licenses directory
	for _, entry := range entries {
		if !entry.IsDir() {
			data, err := licenses.ReadFile(filepath.Join("licenses", entry.Name()))
			if err != nil {
				return err
			}
			destPath := filepath.Join(licensesDir, entry.Name())
			if err := os.WriteFile(destPath, data, 0644); err != nil {
				return err
			}
		}
	}

	// Mihomo TUI, Mihomo, and meta-rules-dat use the same GPL-3.0 license
	// text. Keep separately named copies so every bundled component is clear.
	gpl, err := licenses.ReadFile("licenses/mihomo-GPL-3.0.txt")
	if err != nil {
		return err
	}
	for _, name := range []string{"LICENSE", "meta-rules-dat-GPL-3.0.txt"} {
		if err := os.WriteFile(filepath.Join(licensesDir, name), gpl, 0o644); err != nil {
			return err
		}
	}

	return nil
}
