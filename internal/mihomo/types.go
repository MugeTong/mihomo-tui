package mihomo

import "time"

type Version struct {
	Version string `json:"version"`
}

type ConnectionsSnapshot struct {
	DownloadTotal int64
	UploadTotal   int64
	Connections   int
	TCP           int
	UDP           int
}

type connectionsResponse struct {
	DownloadTotal int64        `json:"downloadTotal"`
	UploadTotal   int64        `json:"uploadTotal"`
	Connections   []connection `json:"connections"`
}

type connection struct {
	Metadata connectionMetadata `json:"metadata"`
}

type connectionMetadata struct {
	Network string `json:"network"`
}

type ProxyGroup struct {
	Name    string
	Type    string
	Now     string
	All     []string
	Proxies []Proxy
}

type Proxy struct {
	Name  string
	Type  string
	UDP   bool
	Delay int
}

type proxyResponse struct {
	Proxies map[string]proxyItem `json:"proxies"`
}

type proxyItem struct {
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Now     string         `json:"now"`
	All     []string       `json:"all"`
	UDP     bool           `json:"udp"`
	History []delayHistory `json:"history"`
}

type delayHistory struct {
	Time  time.Time `json:"time"`
	Delay int       `json:"delay"`
}

type delayResponse struct {
	Delay int `json:"delay"`
}
