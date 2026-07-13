package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"mihomo-tui/internal/runtimeconfig"
)

func TestXDGHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	if got, err := xdgHome("XDG_DATA_HOME", "/fallback"); err != nil || got != "/fallback" {
		t.Fatalf("xdgHome fallback = %q, %v", got, err)
	}
	t.Setenv("XDG_DATA_HOME", "relative")
	if _, err := xdgHome("XDG_DATA_HOME", "/fallback"); err == nil {
		t.Fatal("xdgHome accepted a relative directory")
	}
	t.Setenv("XDG_DATA_HOME", "/tmp/data/../data")
	if got, err := xdgHome("XDG_DATA_HOME", "/fallback"); err != nil || got != "/tmp/data" {
		t.Fatalf("xdgHome absolute = %q, %v", got, err)
	}
}

func TestEnsureShellSourceIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".bashrc")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	line := `. "$HOME/.local/share/mihomo-tui/env"`
	if err := ensureShellSource(path, line); err != nil {
		t.Fatal(err)
	}
	if err := ensureShellSource(path, line); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), line) != 1 {
		t.Fatalf("source line count != 1: %s", data)
	}
}

func TestShellEnvironmentAppliesToCurrentBash(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fakeBinary := "#!/bin/sh\ncase \"$1\" in\non) echo \"export http_proxy='http://127.0.0.1:7890'\" ;;\noff) echo 'unset http_proxy' ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(binDir, "mhmt"), []byte(fakeBinary), 0o755); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(home, "env")
	if err := os.WriteFile(envPath, []byte(shellEnvironment), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("bash", "--noprofile", "--norc", "-c", `. "$1"; mhmt on >/dev/null; printf '%s\n' "$http_proxy"; mhmt off >/dev/null; printf '%s\n' "${http_proxy-unset}"`, "bash", envPath)
	command.Env = append(os.Environ(), "HOME="+home)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("bash integration: %v: %s", err, output)
	}
	if got, want := string(output), "http://127.0.0.1:7890\nunset\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestWriteInitialRuntimeFiles(t *testing.T) {
	directory := t.TempDir()
	statePath := filepath.Join(directory, "state.json")
	configPath := filepath.Join(directory, "config.yaml")
	if err := writeInitialState(statePath); err != nil {
		t.Fatal(err)
	}
	if err := writeInitialRuntimeConfig(configPath); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{statePath, configPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("mode for %s = %o, want 600", path, info.Mode().Perm())
		}
	}
	groups, err := runtimeconfig.LoadProxyGroups(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || groups[0].Name != "Proxy" || groups[1].Name != "Final" {
		t.Fatalf("initial groups = %+v", groups)
	}
}

func TestInitialRuntimeFilesAreNotOverwritten(t *testing.T) {
	directory := t.TempDir()
	statePath := filepath.Join(directory, "state.json")
	configPath := filepath.Join(directory, "config.yaml")
	for _, path := range []string{statePath, configPath} {
		if err := os.WriteFile(path, []byte("user data\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeInitialState(statePath); err != nil {
		t.Fatal(err)
	}
	if err := writeInitialRuntimeConfig(configPath); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{statePath, configPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "user data\n" {
			t.Fatalf("%s was overwritten: %q", path, data)
		}
	}
}

func TestWriteInitialConfigDoesNotOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := writeInitialConfig(path, "/first/mihomo"); err != nil {
		t.Fatal(err)
	}
	if err := writeInitialConfig(path, "/second/mihomo"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "/first/mihomo") || strings.Contains(string(data), "/second/mihomo") {
		t.Fatalf("existing settings were overwritten: %s", data)
	}
}
