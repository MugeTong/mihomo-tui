package app

import "mihomo-tui/internal/mihomo"

func mockProxyGroups() []mihomo.ProxyGroup {
	nodes := []mihomo.Proxy{
		{Name: "Tokyo 01", Type: "Shadowsocks", UDP: true, Delay: 86},
		{Name: "Hong Kong 02", Type: "Trojan", UDP: true, Delay: 112},
		{Name: "Singapore 03", Type: "VLESS", UDP: true, Delay: 95},
		{Name: "Los Angeles 04", Type: "Hysteria2", UDP: true, Delay: 178},
		{Name: "Seoul 05", Type: "Shadowsocks", UDP: true, Delay: 78},
		{Name: "Taipei 06", Type: "Trojan", UDP: true, Delay: 88},
	}

	return []mihomo.ProxyGroup{
		{
			Name:    "Proxy",
			Type:    "Selector",
			Now:     "Tokyo 01",
			All:     proxyNames(nodes),
			Proxies: append([]mihomo.Proxy(nil), nodes...),
		},
		{
			Name: "Final",
			Type: "Selector",
			Now:  "Proxy",
			All:  []string{"Proxy", "DIRECT"},
			Proxies: []mihomo.Proxy{
				{Name: "Proxy", Type: "Selector", UDP: true, Delay: -1},
				{Name: "DIRECT", Type: "Direct", UDP: true, Delay: -1},
			},
		},
	}
}

func proxyNames(nodes []mihomo.Proxy) []string {
	names := make([]string, 0, len(nodes))
	for _, node := range nodes {
		names = append(names, node.Name)
	}
	return names
}

func mockDelay(name string) int {
	total := 0
	for _, char := range name {
		total += int(char)
	}
	return 60 + total%180
}
