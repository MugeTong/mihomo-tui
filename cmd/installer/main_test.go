package main

import (
	"os"
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
