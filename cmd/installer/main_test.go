package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
