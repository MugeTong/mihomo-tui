package runtimeconfig

import (
	"path/filepath"
	"testing"
)

func TestLoadProxyGroupsReadsGeneratedSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte("proxies:\n  - {name: Tokyo, type: trojan, server: example.test, port: 443}\nproxy-groups:\n  - name: Proxy\n    type: select\n    proxies: [Tokyo, DIRECT]\n  - name: Final\n    type: select\n    proxies: [Proxy, DIRECT]\n")
	if _, err := Write(path, data); err != nil {
		t.Fatal(err)
	}
	groups, err := LoadProxyGroups(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || groups[0].Name != "Proxy" || len(groups[0].Proxies) != 2 || groups[0].Proxies[0].Name != "Tokyo" || groups[1].Proxies[0].Type != "Selector" {
		t.Fatalf("groups = %+v", groups)
	}
}
