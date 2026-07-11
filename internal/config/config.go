package config

import (
	"path/filepath"

	"mihomo-tui/internal/xdg"
)

const (
	ControllerURL     = "http://127.0.0.1:9090"
	ControllerAddress = "127.0.0.1:9090"
)

type Config struct {
	Secret     string   `json:"secret"`
	ConfigPath string   `json:"config_path"`
	BinaryPath string   `json:"binary_path"`
	HTTPPort   int      `json:"http_port"`
	SOCKSPort  int      `json:"socks_port"`
	MixedPort  int      `json:"mixed_port"`
	Policies   []Policy `json:"policies"`
}

type PolicyKind string

const (
	PolicySelector PolicyKind = "selector"
	PolicyDirect   PolicyKind = "direct"
	PolicyFinal    PolicyKind = "final"
)

type Policy struct {
	Name    string     `json:"name"`
	Kind    PolicyKind `json:"kind"`
	Enabled bool       `json:"enabled"`
}

func DefaultPolicies() []Policy {
	return []Policy{
		{Name: "Proxy", Kind: PolicySelector, Enabled: true},
		{Name: "Direct", Kind: PolicyDirect, Enabled: true},
		{Name: "Final", Kind: PolicyFinal, Enabled: true},
	}
}

func Default() Config {
	configPath := filepath.Join("~", ".config", "mihomo-tui", "config.yaml")
	if directory, err := xdg.AppConfigDir("mihomo-tui"); err == nil {
		configPath = filepath.Join(directory, "config.yaml")
	}
	return Config{
		ConfigPath: configPath,
		BinaryPath: "mihomo",
		HTTPPort:   7890,
		SOCKSPort:  7891,
		MixedPort:  7892,
		Policies:   DefaultPolicies(),
	}
}
