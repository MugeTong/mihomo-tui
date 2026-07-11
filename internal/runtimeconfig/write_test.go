package runtimeconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomicallyCreatesPrivateRuntimeConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	resolved, err := Write(path, []byte("mode: rule\n"))
	if err != nil {
		t.Fatal(err)
	}
	if resolved != path {
		t.Fatalf("resolved path = %q, want %q", resolved, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "mode: rule\n" {
		t.Fatalf("content = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestWriteRejectsEmptyPath(t *testing.T) {
	if _, err := Write(" ", []byte("mode: rule\n")); err == nil {
		t.Fatal("empty path unexpectedly accepted")
	}
}

func TestWriteDoesNotReplaceIdenticalConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte("mode: rule\n")
	if _, err := Write(path, data); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write(path, data); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("identical runtime config was replaced")
	}
}
