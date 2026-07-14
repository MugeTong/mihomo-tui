package local

import (
	"os"
	"os/exec"
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

func TestShellEnvironmentImportsProxyForRunningCore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	layout, err := ResolveLayout()
	if err != nil {
		t.Fatal(err)
	}
	if err := initializeDirs(layout); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layout.BinDir, "mhmt"), []byte("#!/bin/sh\n[ \"$1\" = on ] || exit 1\necho \"export http_proxy=http://127.0.0.1:7890\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := installShellIntegration(layout); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("bash", "-c", `. "$HOME/.local/share/mihomo-tui/env"; printf %s "$http_proxy"`)
	command.Env = append(os.Environ(), "HOME="+home)
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := string(output); got != "http://127.0.0.1:7890" {
		t.Fatalf("http_proxy = %q", got)
	}
}

func TestShellEnvironmentDoesNotClearProxyWhenCoreIsStopped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	layout, err := ResolveLayout()
	if err != nil {
		t.Fatal(err)
	}
	if err := initializeDirs(layout); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layout.BinDir, "mhmt"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := installShellIntegration(layout); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("bash", "-c", `. "$HOME/.local/share/mihomo-tui/env"; printf %s "$http_proxy"`)
	command.Env = append(os.Environ(), "HOME="+home, "http_proxy=keep-me")
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := string(output); got != "keep-me" {
		t.Fatalf("http_proxy = %q", got)
	}
}
