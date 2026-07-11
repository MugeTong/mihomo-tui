package subscription

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func ImportShareLinks(data []byte, sourceID string) (ImportResult, error) {
	if strings.TrimSpace(sourceID) == "" {
		return ImportResult{}, fmt.Errorf("source ID is required")
	}
	content := strings.TrimSpace(string(data))
	if decoded, ok := decodeSubscriptionBase64(content); ok {
		content = decoded
	}

	result := ImportResult{}
	seen := make(map[string]struct{})
	index := 0
	for _, line := range strings.FieldsFunc(content, func(r rune) bool { return r == '\n' || r == '\r' }) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		node, err := parseShareLink(line)
		if err != nil {
			result.Issues = append(result.Issues, ImportIssue{Index: index, Err: err})
			index++
			continue
		}
		if _, duplicate := seen[node.ID]; duplicate {
			result.Issues = append(result.Issues, ImportIssue{Index: index, Name: node.Name, Err: fmt.Errorf("duplicate node ignored")})
			index++
			continue
		}
		seen[node.ID] = struct{}{}
		result.Nodes = append(result.Nodes, node)
		result.Links = append(result.Links, SourceNode{SourceID: sourceID, NodeID: node.ID, Alias: node.Name})
		index++
	}
	if len(result.Nodes) == 0 {
		return result, fmt.Errorf("no supported share links found")
	}
	return result, nil
}

func ImportContent(data []byte, sourceID string) (ImportResult, error) {
	content := strings.TrimSpace(string(data))
	if decoded, ok := decodeSubscriptionBase64(content); ok {
		content = decoded
	}
	if strings.Contains(content, "proxies:") {
		return ImportClashYAML([]byte(content), sourceID)
	}
	return ImportShareLinks([]byte(content), sourceID)
}

func parseShareLink(value string) (Node, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return Node{}, fmt.Errorf("invalid share link")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "ss":
		return parseSSLink(parsed)
	case "trojan":
		return parseUserLink(parsed, ProtocolTrojan, "password")
	case "vless":
		return parseUserLink(parsed, ProtocolVLESS, "uuid")
	default:
		return Node{}, fmt.Errorf("unsupported share link scheme %q", parsed.Scheme)
	}
}

func parseUserLink(parsed *url.URL, protocol Protocol, credentialKey string) (Node, error) {
	credential := ""
	if parsed.User != nil {
		credential = parsed.User.Username()
	}
	if credential == "" {
		return Node{}, fmt.Errorf("share link credential is required")
	}
	server, port, err := shareEndpoint(parsed)
	if err != nil {
		return Node{}, err
	}
	options := map[string]any{credentialKey: credential}
	copyQueryOptions(options, parsed.Query())
	return normalizedShareNode(parsed, protocol, server, port, options)
}

func parseSSLink(parsed *url.URL) (Node, error) {
	var method, password, server string
	var port int
	var err error
	if parsed.User != nil && parsed.Host != "" {
		userinfo := parsed.User.Username()
		if decoded, ok := decodeBase64(userinfo); ok {
			userinfo = decoded
		}
		method, password, err = splitCredential(userinfo)
		if err != nil {
			return Node{}, err
		}
		server, port, err = shareEndpoint(parsed)
	} else {
		encoded := strings.TrimPrefix(parsed.Opaque, "//")
		if encoded == "" {
			encoded = strings.TrimPrefix(parsed.Host+parsed.Path, "//")
		}
		decoded, ok := decodeBase64(encoded)
		if !ok {
			return Node{}, fmt.Errorf("invalid Shadowsocks share link")
		}
		at := strings.LastIndex(decoded, "@")
		if at < 1 {
			return Node{}, fmt.Errorf("invalid Shadowsocks share link")
		}
		method, password, err = splitCredential(decoded[:at])
		if err == nil {
			server, port, err = splitHostPort(decoded[at+1:])
		}
	}
	if err != nil {
		return Node{}, err
	}
	options := map[string]any{"cipher": method, "password": password}
	copyQueryOptions(options, parsed.Query())
	return normalizedShareNode(parsed, ProtocolShadowsocks, server, port, options)
}

func normalizedShareNode(parsed *url.URL, protocol Protocol, server string, port int, options map[string]any) (Node, error) {
	name, _ := url.QueryUnescape(parsed.Fragment)
	name = strings.TrimSpace(name)
	if name == "" {
		name = server
	}
	node := Node{Name: name, Protocol: protocol, Server: server, Port: port, Options: options}
	id, err := stableNodeID(node)
	if err != nil {
		return Node{}, fmt.Errorf("identify shared node: %w", err)
	}
	node.ID = id
	return node, nil
}

func shareEndpoint(parsed *url.URL) (string, int, error) {
	return splitHostPort(parsed.Host)
}

func splitHostPort(endpoint string) (string, int, error) {
	host, portText, err := net.SplitHostPort(endpoint)
	if err != nil || strings.TrimSpace(host) == "" {
		return "", 0, fmt.Errorf("share link server and port are required")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("share link port must be between 1 and 65535")
	}
	return host, port, nil
}

func splitCredential(value string) (string, string, error) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("share link credential is invalid")
	}
	return parts[0], parts[1], nil
}

func copyQueryOptions(options map[string]any, values url.Values) {
	for key, entries := range values {
		if len(entries) > 0 {
			options[key] = entries[0]
		}
	}
}

func decodeSubscriptionBase64(value string) (string, bool) {
	if strings.Contains(value, "://") || strings.Contains(value, "\n") || strings.Contains(value, "proxies:") {
		return "", false
	}
	return decodeBase64(value)
}

func decodeBase64(value string) (string, bool) {
	value = strings.TrimSpace(value)
	for _, encoding := range []*base64.Encoding{base64.RawStdEncoding, base64.StdEncoding, base64.RawURLEncoding, base64.URLEncoding} {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return string(decoded), true
		}
	}
	return "", false
}
