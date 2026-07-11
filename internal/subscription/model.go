package subscription

import "time"

type Protocol string

const (
	ProtocolShadowsocks Protocol = "ss"
	ProtocolTrojan      Protocol = "trojan"
	ProtocolVLESS       Protocol = "vless"
	ProtocolVMess       Protocol = "vmess"
	ProtocolHysteria2   Protocol = "hysteria2"
	ProtocolTUIC        Protocol = "tuic"
	ProtocolWireGuard   Protocol = "wireguard"
)

// Node is the application-owned representation of an imported proxy node.
// Options retains protocol-specific Mihomo fields for later config generation.
type Node struct {
	ID       string
	Name     string
	Protocol Protocol
	Server   string
	Port     int
	UDP      bool
	Options  map[string]any
}

type SourceType string

const (
	SourceURL   SourceType = "url"
	SourceShare SourceType = "share"
)

type Source struct {
	ID        string
	Name      string
	Type      SourceType
	Location  string
	Enabled   bool
	UpdatedAt time.Time
}

type SourceNode struct {
	SourceID string
	NodeID   string
	Alias    string
}

type PolicySelection struct {
	Policy string
	NodeID string
}

type ImportResult struct {
	Nodes  []Node
	Links  []SourceNode
	Issues []ImportIssue
}

type ImportIssue struct {
	Index int
	Name  string
	Err   error
}

func (i ImportIssue) Error() string {
	if i.Name == "" {
		return i.Err.Error()
	}
	return i.Name + ": " + i.Err.Error()
}
