package runtimeconfig

import (
	"fmt"
	"os"

	"mihomo-tui/internal/mihomo"

	"go.yaml.in/yaml/v3"
)

type snapshotDocument struct {
	Proxies []struct {
		Name string `yaml:"name"`
		Type string `yaml:"type"`
		UDP  bool   `yaml:"udp"`
	} `yaml:"proxies"`
	Groups []struct {
		Name    string   `yaml:"name"`
		Type    string   `yaml:"type"`
		Proxies []string `yaml:"proxies"`
	} `yaml:"proxy-groups"`
}

// LoadProxyGroups reads the last successfully generated config without
// contacting the subscription server or Mihomo controller.
func LoadProxyGroups(path string) ([]mihomo.ProxyGroup, error) {
	document, err := loadSnapshot(path)
	if err != nil {
		return nil, err
	}
	return proxyGroupsFromSnapshot(document), nil
}

func SnapshotNodeCount(path string) (int, error) {
	document, err := loadSnapshot(path)
	if err != nil {
		return 0, err
	}
	return len(document.Proxies), nil
}

func loadSnapshot(path string) (snapshotDocument, error) {
	resolved, err := expandPath(path)
	if err != nil {
		return snapshotDocument{}, err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return snapshotDocument{}, fmt.Errorf("read runtime config snapshot: %w", err)
	}
	var document snapshotDocument
	if err := yaml.Unmarshal(data, &document); err != nil {
		return snapshotDocument{}, fmt.Errorf("parse runtime config snapshot: %w", err)
	}
	return document, nil
}

func proxyGroupsFromSnapshot(document snapshotDocument) []mihomo.ProxyGroup {
	proxyByName := make(map[string]mihomo.Proxy, len(document.Proxies)+2)
	for _, proxy := range document.Proxies {
		proxyByName[proxy.Name] = mihomo.Proxy{Name: proxy.Name, Type: proxy.Type, UDP: proxy.UDP, Delay: -1}
	}
	proxyByName["DIRECT"] = mihomo.Proxy{Name: "DIRECT", Type: "Direct", UDP: true, Delay: -1}
	groups := make([]mihomo.ProxyGroup, 0, len(document.Groups))
	for _, group := range document.Groups {
		members := make([]mihomo.Proxy, 0, len(group.Proxies))
		for _, name := range group.Proxies {
			member, exists := proxyByName[name]
			if !exists {
				member = mihomo.Proxy{Name: name, Type: "Selector", Delay: -1}
			}
			members = append(members, member)
		}
		now := ""
		if len(group.Proxies) > 0 {
			now = group.Proxies[0]
		}
		groups = append(groups, mihomo.ProxyGroup{Name: group.Name, Type: group.Type, Now: now, All: append([]string(nil), group.Proxies...), Proxies: members})
	}
	return groups
}
