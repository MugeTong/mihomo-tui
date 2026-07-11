package subscription

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

type clashDocument struct {
	Proxies []map[string]any `yaml:"proxies"`
}

// ImportClashYAML imports only the top-level proxies list. All other Clash or
// Mihomo configuration sections are intentionally ignored.
func ImportClashYAML(data []byte, sourceID string) (ImportResult, error) {
	if strings.TrimSpace(sourceID) == "" {
		return ImportResult{}, fmt.Errorf("source ID is required")
	}
	var document clashDocument
	if err := yaml.Unmarshal(data, &document); err != nil {
		return ImportResult{}, fmt.Errorf("parse subscription YAML: %w", err)
	}

	result := ImportResult{
		Nodes: make([]Node, 0, len(document.Proxies)),
		Links: make([]SourceNode, 0, len(document.Proxies)),
	}
	seenIDs := make(map[string]struct{}, len(document.Proxies))
	usedNames := make(map[string]int, len(document.Proxies))

	for index, raw := range document.Proxies {
		node, err := normalizeClashNode(raw)
		if err != nil {
			result.Issues = append(result.Issues, ImportIssue{
				Index: index,
				Name:  safeString(raw["name"]),
				Err:   err,
			})
			continue
		}
		if _, exists := seenIDs[node.ID]; exists {
			result.Issues = append(result.Issues, ImportIssue{
				Index: index,
				Name:  node.Name,
				Err:   fmt.Errorf("duplicate node ignored"),
			})
			continue
		}

		node.Name = uniqueName(node.Name, usedNames)
		seenIDs[node.ID] = struct{}{}
		result.Nodes = append(result.Nodes, node)
		result.Links = append(result.Links, SourceNode{SourceID: sourceID, NodeID: node.ID, Alias: node.Name})
	}

	return result, nil
}

func normalizeClashNode(raw map[string]any) (Node, error) {
	name := strings.TrimSpace(safeString(raw["name"]))
	if name == "" {
		return Node{}, fmt.Errorf("name is required")
	}

	protocol := Protocol(strings.ToLower(strings.TrimSpace(safeString(raw["type"]))))
	if !supportedProtocol(protocol) {
		return Node{}, fmt.Errorf("unsupported protocol %q", protocol)
	}

	server := strings.TrimSpace(safeString(raw["server"]))
	if server == "" {
		return Node{}, fmt.Errorf("server is required")
	}
	port, err := integer(raw["port"])
	if err != nil || port < 1 || port > 65535 {
		return Node{}, fmt.Errorf("port must be between 1 and 65535")
	}

	options := cloneMap(raw)
	delete(options, "name")
	delete(options, "type")
	delete(options, "server")
	delete(options, "port")
	delete(options, "udp")

	node := Node{
		Name:     name,
		Protocol: protocol,
		Server:   server,
		Port:     port,
		UDP:      boolean(raw["udp"]),
		Options:  options,
	}
	node.ID, err = stableNodeID(node)
	if err != nil {
		return Node{}, fmt.Errorf("identify node: %w", err)
	}
	return node, nil
}

func stableNodeID(node Node) (string, error) {
	identity := struct {
		Protocol Protocol       `json:"protocol"`
		Server   string         `json:"server"`
		Port     int            `json:"port"`
		Options  map[string]any `json:"options"`
	}{node.Protocol, strings.ToLower(node.Server), node.Port, node.Options}
	payload, err := json.Marshal(identity)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func supportedProtocol(protocol Protocol) bool {
	switch protocol {
	case ProtocolShadowsocks, ProtocolTrojan, ProtocolVLESS, ProtocolVMess,
		ProtocolHysteria2, ProtocolTUIC, ProtocolWireGuard:
		return true
	default:
		return false
	}
}

func uniqueName(name string, used map[string]int) string {
	used[name]++
	if used[name] == 1 {
		return name
	}
	return fmt.Sprintf("%s (%d)", name, used[name])
}

func safeString(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return ""
	}
}

func integer(value any) (int, error) {
	switch value := value.(type) {
	case int:
		return value, nil
	case int64:
		return int(value), nil
	case uint64:
		return int(value), nil
	case float64:
		if value != float64(int(value)) {
			return 0, fmt.Errorf("not an integer")
		}
		return int(value), nil
	case string:
		return strconv.Atoi(strings.TrimSpace(value))
	default:
		return 0, fmt.Errorf("not an integer")
	}
}

func boolean(value any) bool {
	result, _ := value.(bool)
	return result
}

func cloneMap(source map[string]any) map[string]any {
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
