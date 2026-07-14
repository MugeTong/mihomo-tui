package runtimeconfig

import (
	"os"
	"path/filepath"
	"testing"

	"mihomo-tui/internal/config"

	"go.yaml.in/yaml/v3"
)

func TestApplySettingsPreservesRuntimeContent(t *testing.T) {
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "old.yaml")
	destinationPath := filepath.Join(directory, "new.yaml")
	original := []byte("port: 7890\nsocks-port: 7891\nmixed-port: 7892\nproxies:\n  - {name: Tokyo, type: trojan, server: example.test, port: 443}\nproxy-groups:\n  - {name: Proxy, type: select, proxies: [Tokyo]}\nrules:\n  - MATCH,Proxy\n")
	if err := os.WriteFile(sourcePath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.HTTPPort = 8890
	cfg.SOCKSPort = 8891
	cfg.MixedPort = 8892
	cfg.ConfigPath = destinationPath
	if err := ApplySettings(sourcePath, cfg); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(destinationPath)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		HTTPPort  int              `yaml:"port"`
		SOCKSPort int              `yaml:"socks-port"`
		MixedPort int              `yaml:"mixed-port"`
		Proxies   []map[string]any `yaml:"proxies"`
		Groups    []map[string]any `yaml:"proxy-groups"`
		Rules     []string         `yaml:"rules"`
	}
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document.HTTPPort != 8890 || document.SOCKSPort != 8891 || document.MixedPort != 8892 {
		t.Fatalf("ports = %d/%d/%d", document.HTTPPort, document.SOCKSPort, document.MixedPort)
	}
	if len(document.Proxies) != 1 || document.Proxies[0]["name"] != "Tokyo" || len(document.Groups) != 1 {
		t.Fatalf("runtime content was not preserved: %+v", document)
	}
	if len(document.Rules) != 1 || document.Rules[0] != "MATCH,Proxy" {
		t.Fatalf("runtime rules were not preserved: %+v", document.Rules)
	}
}
