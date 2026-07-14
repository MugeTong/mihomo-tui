package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mihomo-tui/internal/config"

	"go.yaml.in/yaml/v3"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSettingsShowsLocalStatePathAsReadOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	p := newSettingsPage(config.Default(), "test").(settingsPage)
	view := p.View(120, 30)
	if want := filepath.Join(home, ".config", "mihomo-tui", "state.json"); !containsText(view, want) {
		t.Fatalf("settings view does not contain state path %q", want)
	}
}

func TestSettingsOnlyExposesSupportedEditableFields(t *testing.T) {
	p := newSettingsPage(config.Default(), "test").(settingsPage)
	view := p.View(80, 24)

	for _, want := range []string{"HTTP Port", "SOCKS Port", "Mixed Port", "Config File", "State File", "Mihomo Bin"} {
		if !containsText(view, want) {
			t.Fatalf("settings view does not contain %q", want)
		}
	}
	for _, hidden := range []string{"Controller", "Secret"} {
		if containsText(view, hidden) {
			t.Fatalf("settings view unexpectedly exposes %q", hidden)
		}
	}
}

func TestSettingsAboutIncludesCreditsAndHomepage(t *testing.T) {
	p := newSettingsPage(config.Default(), "test-version").(settingsPage)
	view := p.View(100, 30)
	for _, want := range []string{"test-version", "GPL-3.0-only", "Mihomo contributors", "Bubble Tea", "Shadowrocket", "github.com/MugeTong/mihomo-tui"} {
		if !containsText(view, want) {
			t.Fatalf("about view does not contain %q", want)
		}
	}
}

func TestSettingsAcceptsQWhileEditingPath(t *testing.T) {
	p := newSettingsPage(config.Default(), "test").(settingsPage)
	p.cursor = fieldConfigPath
	p.editing = true
	p.buffer = ""

	page, _ := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := page.(settingsPage)
	if updated.buffer != "q" {
		t.Fatalf("buffer = %q, want q", updated.buffer)
	}
}

func TestSettingsSaveUpdatesRuntimeConfigPorts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default()
	if err := os.MkdirAll(filepath.Dir(cfg.ConfigPath), 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte("port: 7890\nsocks-port: 7891\nmixed-port: 7892\nproxies:\n  - {name: Tokyo, type: trojan, server: example.test, port: 443}\n")
	if err := os.WriteFile(cfg.ConfigPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	p := newSettingsPage(cfg, "test").(settingsPage)
	p.cfg.HTTPPort = 8890
	p.cfg.SOCKSPort = 8891
	p.cfg.MixedPort = 8892
	page, _ := p.Update(p.save()())
	p = page.(settingsPage)
	if p.err != "" {
		t.Fatal(p.err)
	}

	data, err := os.ReadFile(cfg.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var runtime struct {
		HTTPPort  int              `yaml:"port"`
		SOCKSPort int              `yaml:"socks-port"`
		MixedPort int              `yaml:"mixed-port"`
		Proxies   []map[string]any `yaml:"proxies"`
	}
	if err := yaml.Unmarshal(data, &runtime); err != nil {
		t.Fatal(err)
	}
	if runtime.HTTPPort != 8890 || runtime.SOCKSPort != 8891 || runtime.MixedPort != 8892 || len(runtime.Proxies) != 1 {
		t.Fatalf("runtime config = %+v", runtime)
	}
	saved, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.HTTPPort != 8890 || saved.SOCKSPort != 8891 || saved.MixedPort != 8892 {
		t.Fatalf("saved settings ports = %d/%d/%d", saved.HTTPPort, saved.SOCKSPort, saved.MixedPort)
	}
}

func TestParsePort(t *testing.T) {
	for _, value := range []string{"0", "65536", "not-a-port"} {
		if _, err := parsePort(value); err == nil {
			t.Fatalf("parsePort(%q) unexpectedly succeeded", value)
		}
	}
	if got, err := parsePort("7890"); err != nil || got != 7890 {
		t.Fatalf("parsePort(7890) = %d, %v", got, err)
	}
}

func containsText(value, fragment string) bool {
	return strings.Contains(value, fragment)
}
