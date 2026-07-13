package local

import (
	"path/filepath"
	"testing"
)

func TestDirectoriesUseFixedHomeLayout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		name string
		get  func() (string, error)
		want string
	}{
		{"config", ConfigDir, filepath.Join(home, ".config", appName)},
		{"data", DataDir, filepath.Join(home, ".local", "share", appName)},
		{"state", StateDir, filepath.Join(home, ".local", "state", appName)},
		{"bin", BinDir, filepath.Join(home, ".local", "bin")},
	}
	for _, test := range tests {
		got, err := test.get()
		if err != nil {
			t.Fatalf("%s directory: %v", test.name, err)
		}
		if got != test.want {
			t.Fatalf("%s directory = %q, want %q", test.name, got, test.want)
		}
	}
}
