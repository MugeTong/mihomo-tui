package local

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallShellIntegrationIsIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	layout, err := ResolveLayout()
	if err != nil {
		t.Fatal(err)
	}
	if err := initializeDirs(layout); err != nil {
		t.Fatal(err)
	}
	rcPath := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(rcPath, []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installShellIntegration(layout); err != nil {
		t.Fatal(err)
	}
	if err := installShellIntegration(layout); err != nil {
		t.Fatal(err)
	}

	envPath := filepath.Join(layout.DataDir, "env")
	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatal(err)
	}
	if sourceLine := `. "` + envPath + `"`; strings.Count(string(data), sourceLine) != 1 {
		t.Fatalf("shell source line is not idempotent: %s", data)
	}
	if _, err := os.Stat(envPath); err != nil {
		t.Fatalf("shell environment was not installed: %v", err)
	}
}
