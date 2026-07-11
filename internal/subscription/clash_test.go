package subscription

import (
	"os"
	"strings"
	"testing"
)

func TestImportClashYAMLImportsOnlyProxies(t *testing.T) {
	data, err := os.ReadFile("testdata/full-config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	result, err := ImportClashYAML(data, "provider-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("issues = %+v", result.Issues)
	}
	if len(result.Nodes) != 2 {
		t.Fatalf("node count = %d, want 2", len(result.Nodes))
	}

	first := result.Nodes[0]
	if first.Name != "Tokyo" || first.Protocol != ProtocolShadowsocks || first.Server != "jp.example.test" || first.Port != 443 || !first.UDP {
		t.Fatalf("first node = %+v", first)
	}
	if first.ID == "" {
		t.Fatalf("node identity missing: %+v", first)
	}
	if first.Options["password"] != "test-password" || first.Options["cipher"] != "aes-128-gcm" {
		t.Fatalf("protocol options were not retained: %+v", first.Options)
	}
	if result.Nodes[1].Name != "Tokyo" {
		t.Fatalf("duplicate display name = %q, want Tokyo", result.Nodes[1].Name)
	}
	for _, forbidden := range []string{"rules", "proxy-groups", "dns", "secret", "mixed-port"} {
		if _, exists := first.Options[forbidden]; exists {
			t.Fatalf("global field %q leaked into node options", forbidden)
		}
	}
}

func TestImportClashYAMLReportsBadNodesWithoutSecrets(t *testing.T) {
	data := []byte(`
proxies:
  - name: Missing Server
    type: trojan
    port: 443
    password: do-not-log-this
  - name: Unknown
    type: mystery
    server: example.test
    port: 443
  - name: Bad Port
    type: ss
    server: example.test
    port: 70000
`)
	result, err := ImportClashYAML(data, "bad-provider")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Nodes) != 0 || len(result.Issues) != 3 {
		t.Fatalf("result = %+v", result)
	}
	for _, issue := range result.Issues {
		if strings.Contains(issue.Error(), "do-not-log-this") {
			t.Fatalf("issue leaked a credential: %s", issue.Error())
		}
	}
}

func TestImportClashYAMLDeduplicatesStableIdentity(t *testing.T) {
	data := []byte(`
proxies:
  - {name: First Name, type: vless, server: edge.example.test, port: 443, uuid: 00000000-0000-0000-0000-000000000001}
  - {name: Renamed Node, type: vless, server: EDGE.EXAMPLE.TEST, port: 443, uuid: 00000000-0000-0000-0000-000000000001}
`)
	result, err := ImportClashYAML(data, "provider")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Nodes) != 1 || result.Duplicates != 1 || len(result.Issues) != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestImportClashYAMLRejectsMalformedDocument(t *testing.T) {
	_, err := ImportClashYAML([]byte("proxies: ["), "provider")
	if err == nil || !strings.Contains(err.Error(), "parse subscription YAML") {
		t.Fatalf("error = %v", err)
	}
}
