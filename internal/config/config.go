package config

type Config struct {
	ControllerURL string   `json:"controller_url"`
	Secret        string   `json:"secret"`
	ConfigPath    string   `json:"config_path"`
	BinaryPath    string   `json:"binary_path"`
	Platform      string   `json:"platform"`
	RuntimeMode   string   `json:"runtime_mode"`
	SourceMode    string   `json:"source_mode"`
	HTTPPort      int      `json:"http_port"`
	SOCKSPort     int      `json:"socks_port"`
	MixedPort     int      `json:"mixed_port"`
	Policies      []Policy `json:"policies"`
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
	System  bool       `json:"system"`
}

func DefaultPolicies() []Policy {
	return []Policy{
		{Name: "Proxy", Kind: PolicySelector, Enabled: true, System: true},
		{Name: "Direct", Kind: PolicyDirect, Enabled: true, System: true},
		{Name: "Final", Kind: PolicyFinal, Enabled: true, System: true},
	}
}

func Default() Config {
	return Config{
		ControllerURL: "http://127.0.0.1:9090",
		ConfigPath:    "~/.config/mihomo-tui/config.yaml",
		BinaryPath:    "mihomo",
		Platform:      "ubuntu",
		RuntimeMode:   "proxy",
		SourceMode:    "managed",
		HTTPPort:      7890,
		SOCKSPort:     7891,
		MixedPort:     7892,
		Policies:      DefaultPolicies(),
	}
}
