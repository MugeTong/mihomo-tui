package runtimeconfig

import (
	"strings"
	"testing"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/subscription"

	"go.yaml.in/yaml/v3"
)

type generatedDocument struct {
	Port       int              `yaml:"port"`
	SOCKSPort  int              `yaml:"socks-port"`
	MixedPort  int              `yaml:"mixed-port"`
	Controller string           `yaml:"external-controller"`
	Proxies    []map[string]any `yaml:"proxies"`
	Groups     []generatedGroup `yaml:"proxy-groups"`
	Rules      []string         `yaml:"rules"`
}

type generatedGroup struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Proxies []string `yaml:"proxies"`
}

func TestGenerateBuildsManagedMihomoConfig(t *testing.T) {
	cfg := config.Default()
	state := subscription.NewState()
	state.Nodes = []subscription.Node{
		{ID: "ss", Name: "Tokyo", Protocol: subscription.ProtocolShadowsocks, Server: "ss.example.test", Port: 443, UDP: true, Options: map[string]any{"cipher": "aes-128-gcm", "password": "secret"}},
		{ID: "vless", Name: "Tokyo", Protocol: subscription.ProtocolVLESS, Server: "reality.example.test", Port: 8443, Options: map[string]any{
			"uuid": "test-uuid", "encryption": "none", "flow": "xtls-rprx-vision", "security": "reality",
			"sni": "www.example.test", "fp": "chrome", "pbk": "public-key", "sid": "short-id", "type": "tcp",
		}},
	}
	state.Selections = []subscription.PolicySelection{{Policy: "Proxy", NodeID: "vless"}}

	data, err := Generate(cfg, state)
	if err != nil {
		t.Fatal(err)
	}
	var document generatedDocument
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document.Port != 7890 || document.SOCKSPort != 7891 || document.MixedPort != 7892 || document.Controller != "127.0.0.1:9090" {
		t.Fatalf("runtime settings = %+v", document)
	}
	if len(document.Proxies) != 2 || document.Proxies[0]["name"] != "Tokyo" || document.Proxies[1]["name"] != "Tokyo (2)" {
		t.Fatalf("proxies = %+v", document.Proxies)
	}
	reality := document.Proxies[1]
	if reality["type"] != "vless" || reality["network"] != "tcp" || reality["tls"] != true || reality["servername"] != "www.example.test" || reality["client-fingerprint"] != "chrome" {
		t.Fatalf("Reality proxy = %+v", reality)
	}
	if _, exists := reality["pbk"]; exists {
		t.Fatalf("share aliases leaked into generated proxy: %+v", reality)
	}
	if len(document.Groups) != 2 || document.Groups[0].Name != "Proxy" || document.Groups[1].Name != "Final" {
		t.Fatalf("groups = %+v", document.Groups)
	}
	if got := document.Groups[0].Proxies; len(got) != 3 || got[0] != "Tokyo (2)" || got[1] != "Tokyo" || got[2] != "DIRECT" {
		t.Fatalf("Proxy members = %v", got)
	}
	if got := document.Groups[1].Proxies; len(got) != 2 || got[0] != "Proxy" || got[1] != "DIRECT" {
		t.Fatalf("Final members = %v", got)
	}
	if len(document.Rules) < 40 || document.Rules[len(document.Rules)-1] != "MATCH,Final" {
		t.Fatalf("rules = %v", document.Rules)
	}
}

func TestGenerateWorksWithoutImportedNodes(t *testing.T) {
	data, err := Generate(config.Default(), subscription.NewState())
	if err != nil {
		t.Fatal(err)
	}
	var document generatedDocument
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Groups) != 2 || len(document.Groups[0].Proxies) != 1 || document.Groups[0].Proxies[0] != "DIRECT" {
		t.Fatalf("groups = %+v", document.Groups)
	}
	for _, forbidden := range []string{"external-ui", "external-ui-name", "external-ui-url"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("generated config exposes web UI field %q", forbidden)
		}
	}
}

func TestGenerateNormalizesHysteria2ShareOptions(t *testing.T) {
	state := subscription.NewState()
	state.Nodes = []subscription.Node{{ID: "hy", Name: "Hysteria", Protocol: subscription.ProtocolHysteria2, Server: "hy.example.test", Port: 443, Options: map[string]any{
		"password": "secret", "sni": "edge.example.test", "insecure": "1", "obfs": "salamander", "obfs-password": "mask",
	}}}
	data, err := Generate(config.Default(), state)
	if err != nil {
		t.Fatal(err)
	}
	var document generatedDocument
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	proxy := document.Proxies[0]
	if proxy["type"] != "hysteria2" || proxy["sni"] != "edge.example.test" || proxy["skip-cert-verify"] != true || proxy["obfs"] != "salamander" {
		t.Fatalf("Hysteria2 proxy = %+v", proxy)
	}
}

func TestGenerateNormalizesTrojanShareAliases(t *testing.T) {
	state := subscription.NewState()
	state.Nodes = []subscription.Node{{ID: "trojan", Name: "Trojan", Protocol: subscription.ProtocolTrojan, Server: "trojan.example.test", Port: 443, Options: map[string]any{
		"password": "secret", "peer": "edge.example.test", "fp": "chrome", "allowInsecure": "1",
	}}}
	data, err := Generate(config.Default(), state)
	if err != nil {
		t.Fatal(err)
	}
	var document generatedDocument
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	proxy := document.Proxies[0]
	if proxy["tls"] != true || proxy["sni"] != "edge.example.test" || proxy["client-fingerprint"] != "chrome" || proxy["skip-cert-verify"] != true {
		t.Fatalf("Trojan proxy = %+v", proxy)
	}
	for _, alias := range []string{"peer", "fp", "allowInsecure"} {
		if _, exists := proxy[alias]; exists {
			t.Fatalf("alias %q leaked into proxy: %+v", alias, proxy)
		}
	}
}
