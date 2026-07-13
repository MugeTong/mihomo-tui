package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mihomo-tui/internal/runtimeconfig"
)

func TestInitializeConfigFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".config", "mihomo-tui")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	corePath := filepath.Join(home, "mihomo")
	if err := initializeConfigFiles(corePath); err != nil {
		t.Fatal(err)
	}

	settings, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(settings), corePath) {
		t.Fatalf("settings do not contain core path: %s", settings)
	}
	groups, err := runtimeconfig.LoadProxyGroups(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || groups[0].Name != "Proxy" || groups[1].Name != "Final" {
		t.Fatalf("initial proxy groups = %+v", groups)
	}
}

func TestInitializeConfigFilesDoesNotOverwrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".config", "mihomo-tui")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(path, []byte("user data\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := initializeConfigFiles("/some/mihomo"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "user data\n" {
		t.Fatalf("existing settings were overwritten: %q", data)
	}
}

func TestValidatePayloadRejectsMissingResources(t *testing.T) {
	if err := validatePayload(Payload{MihomoVersion: "1.0.0"}); err == nil {
		t.Fatal("validatePayload accepted an empty payload")
	}
}
