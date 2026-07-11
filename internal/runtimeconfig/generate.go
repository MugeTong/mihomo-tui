package runtimeconfig

import (
	"fmt"
	"strings"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/rules"
	"mihomo-tui/internal/subscription"

	"go.yaml.in/yaml/v3"
)

// Generate builds the complete app-owned Mihomo runtime configuration.
func Generate(cfg config.Config, state subscription.State) ([]byte, error) {
	document := map[string]any{
		"mode":                "rule",
		"allow-lan":           false,
		"external-controller": controllerAddress(cfg.ControllerURL),
		"secret":              cfg.Secret,
		"proxies":             []map[string]any{},
	}
	if cfg.HTTPPort > 0 {
		document["port"] = cfg.HTTPPort
	}
	if cfg.SOCKSPort > 0 {
		document["socks-port"] = cfg.SOCKSPort
	}
	if cfg.MixedPort > 0 {
		document["mixed-port"] = cfg.MixedPort
	}

	proxies, nodeNames, names, err := buildProxies(state.Nodes)
	if err != nil {
		return nil, err
	}
	document["proxies"] = proxies
	document["proxy-groups"] = buildGroups(cfg.Policies, state.Selections, nodeNames, names)

	defaultRules, err := rules.Default()
	if err != nil {
		return nil, fmt.Errorf("load default rules: %w", err)
	}
	document["rules"] = renderRules(defaultRules)

	data, err := yaml.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode Mihomo config: %w", err)
	}
	return data, nil
}

func buildProxies(nodes []subscription.Node) ([]map[string]any, []string, map[string]string, error) {
	result := make([]map[string]any, 0, len(nodes))
	orderedNames := make([]string, 0, len(nodes))
	names := make(map[string]string, len(nodes))
	used := make(map[string]int, len(nodes))
	for _, node := range nodes {
		name := uniqueName(node.Name, used)
		proxy, err := buildProxy(node, name)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("generate node %q: %w", node.Name, err)
		}
		result = append(result, proxy)
		orderedNames = append(orderedNames, name)
		names[node.ID] = name
	}
	return result, orderedNames, names, nil
}

func buildProxy(node subscription.Node, name string) (map[string]any, error) {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(node.Server) == "" || node.Port < 1 || node.Port > 65535 {
		return nil, fmt.Errorf("invalid name, server, or port")
	}
	proxy := cloneMap(node.Options)
	proxy["name"] = name
	proxy["server"] = node.Server
	proxy["port"] = node.Port
	if node.UDP {
		proxy["udp"] = true
	}
	normalizeShareOptions(proxy, node.Protocol)
	return proxy, nil
}

func normalizeShareOptions(proxy map[string]any, protocol subscription.Protocol) {
	if value, ok := proxy["sni"]; ok && protocol == subscription.ProtocolVLESS {
		if _, exists := proxy["servername"]; !exists {
			proxy["servername"] = value
		}
		delete(proxy, "sni")
	}
	if value, ok := proxy["type"]; ok && protocol == subscription.ProtocolVLESS {
		// At this point type may be the URI transport parameter. The protocol
		// itself is restored after aliases have been normalized below.
		if text, valid := value.(string); valid && text != string(protocol) {
			proxy["network"] = text
		}
	}
	if protocol == subscription.ProtocolVLESS {
		if value, ok := proxy["fp"]; ok {
			proxy["client-fingerprint"] = value
			delete(proxy, "fp")
		}
		publicKey, hasPublicKey := proxy["pbk"]
		shortID, hasShortID := proxy["sid"]
		if hasPublicKey || hasShortID {
			reality := map[string]any{}
			if hasPublicKey {
				reality["public-key"] = publicKey
			}
			if hasShortID {
				reality["short-id"] = shortID
			}
			proxy["reality-opts"] = reality
			delete(proxy, "pbk")
			delete(proxy, "sid")
		}
		if security, ok := proxy["security"].(string); ok {
			if security == "tls" || security == "reality" {
				proxy["tls"] = true
			}
			delete(proxy, "security")
		}
		delete(proxy, "encryption")
	}
	if protocol == subscription.ProtocolHysteria2 {
		if insecure, ok := proxy["insecure"].(string); ok {
			proxy["skip-cert-verify"] = insecure == "1" || strings.EqualFold(insecure, "true")
			delete(proxy, "insecure")
		}
	}
	if protocol == subscription.ProtocolAnyTLS {
		if fingerprint, ok := proxy["fp"]; ok {
			proxy["client-fingerprint"] = fingerprint
			delete(proxy, "fp")
		}
		if insecure, ok := proxy["insecure"].(string); ok {
			proxy["skip-cert-verify"] = insecure == "1" || strings.EqualFold(insecure, "true")
			delete(proxy, "insecure")
		}
	}
	if protocol == subscription.ProtocolTrojan {
		proxy["tls"] = true
		if fingerprint, ok := proxy["fp"]; ok {
			proxy["client-fingerprint"] = fingerprint
			delete(proxy, "fp")
		}
		if _, exists := proxy["sni"]; !exists {
			if peer, ok := proxy["peer"]; ok {
				proxy["sni"] = peer
			}
		}
		delete(proxy, "peer")
		if insecure, ok := proxy["allowInsecure"].(string); ok {
			proxy["skip-cert-verify"] = insecure == "1" || strings.EqualFold(insecure, "true")
			delete(proxy, "allowInsecure")
		}
	}
	proxy["type"] = string(protocol)
}

func buildGroups(policies []config.Policy, selections []subscription.PolicySelection, nodeNames []string, names map[string]string) []map[string]any {
	allNodes := append([]string(nil), nodeNames...)
	allNodes = append(allNodes, "DIRECT")
	selected := make(map[string]string, len(selections))
	for _, selection := range selections {
		selected[selection.Policy] = names[selection.NodeID]
	}
	groups := make([]map[string]any, 0, len(policies))
	for _, policy := range policies {
		if !policy.Enabled || policy.Kind == config.PolicyDirect {
			continue
		}
		members := append([]string(nil), allNodes...)
		if policy.Kind == config.PolicyFinal {
			members = []string{"Proxy", "DIRECT"}
		}
		if preferred := selected[policy.Name]; preferred != "" {
			members = moveFirst(members, preferred)
		}
		groups = append(groups, map[string]any{"name": policy.Name, "type": "select", "proxies": members})
	}
	return groups
}

func renderRules(source []rules.Rule) []string {
	result := make([]string, 0, len(source))
	for _, rule := range source {
		parts := []string{rule.Type}
		if rule.Type != "MATCH" {
			parts = append(parts, rule.Value)
		}
		parts = append(parts, rule.Policy)
		parts = append(parts, rule.Options...)
		result = append(result, strings.Join(parts, ","))
	}
	return result
}

func controllerAddress(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimPrefix(value, "https://")
	return strings.TrimSuffix(value, "/")
}

func cloneMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source)+5)
	for key, value := range source {
		result[key] = value
	}
	return result
}

func uniqueName(name string, used map[string]int) string {
	used[name]++
	if used[name] == 1 {
		return name
	}
	return fmt.Sprintf("%s (%d)", name, used[name])
}

func moveFirst(values []string, target string) []string {
	result := []string{target}
	for _, value := range values {
		if value != target {
			result = append(result, value)
		}
	}
	return result
}
