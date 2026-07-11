package subscription

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestImportShareLinksSupportsPrimarySchemes(t *testing.T) {
	ssUser := base64.RawURLEncoding.EncodeToString([]byte("aes-128-gcm:test-password"))
	content := strings.Join([]string{
		"ss://" + ssUser + "@jp.example.test:443#Tokyo",
		"trojan://test-password@hk.example.test:443?sni=edge.example.test#Hong%20Kong",
		"vless://00000000-0000-0000-0000-000000000001@us.example.test:8443?security=tls&type=ws#US",
	}, "\n")
	result, err := ImportShareLinks([]byte(content), "shared")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Nodes) != 3 || len(result.Links) != 3 || len(result.Issues) != 0 {
		t.Fatalf("result = %+v", result)
	}
	if result.Nodes[0].Protocol != ProtocolShadowsocks || result.Nodes[0].Options["cipher"] != "aes-128-gcm" {
		t.Fatalf("Shadowsocks node = %+v", result.Nodes[0])
	}
	if result.Nodes[1].Name != "Hong Kong" || result.Nodes[1].Options["sni"] != "edge.example.test" {
		t.Fatalf("Trojan node = %+v", result.Nodes[1])
	}
	if result.Nodes[2].Protocol != ProtocolVLESS || result.Nodes[2].Options["uuid"] == "" {
		t.Fatalf("VLESS node = %+v", result.Nodes[2])
	}
}

func TestImportShareLinksSupportsBase64Subscription(t *testing.T) {
	plain := "trojan://password@example.test:443#Node"
	encoded := base64.StdEncoding.EncodeToString([]byte(plain))
	result, err := ImportShareLinks([]byte(encoded), "encoded")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Nodes) != 1 || result.Nodes[0].Name != "Node" {
		t.Fatalf("result = %+v", result)
	}
}

func TestImportShareLinksDoesNotLeakCredentialInIssue(t *testing.T) {
	_, err := ImportShareLinks([]byte("trojan://do-not-log-this@missing-port.example.test"), "bad")
	if err == nil {
		t.Fatal("invalid link unexpectedly imported")
	}
	if strings.Contains(err.Error(), "do-not-log-this") {
		t.Fatalf("credential leaked in error: %v", err)
	}
}
