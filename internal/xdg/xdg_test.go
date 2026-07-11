package xdg

import (
	"path/filepath"
	"testing"
)

func TestConfigHomeHonorsXDGConfigHome(t *testing.T) {
	want := filepath.Join(t.TempDir(), "config")
	t.Setenv("XDG_CONFIG_HOME", want)
	got, err := ConfigHome()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("config home = %q, want %q", got, want)
	}
}

func TestConfigHomeRejectsRelativeOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "relative/config")
	if _, err := ConfigHome(); err == nil {
		t.Fatal("relative XDG_CONFIG_HOME unexpectedly accepted")
	}
}

func TestDataHomeHonorsXDGDataHome(t *testing.T) {
	want := filepath.Join(t.TempDir(), "data")
	t.Setenv("XDG_DATA_HOME", want)
	got, err := DataHome()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("data home = %q, want %q", got, want)
	}
}

func TestDataHomeRejectsRelativeOverride(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "relative/data")
	if _, err := DataHome(); err == nil {
		t.Fatal("relative XDG_DATA_HOME unexpectedly accepted")
	}
}
