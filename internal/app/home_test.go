package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/core"
	"mihomo-tui/internal/mihomo"
	"mihomo-tui/internal/runtimeconfig"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestHomeWaitsForManagedProxyGroup(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		proxies := map[string]any{
			"Final": map[string]any{"name": "Final", "type": "Selector", "now": "DIRECT", "all": []string{"DIRECT"}},
		}
		if requests > 1 {
			proxies["Proxy"] = map[string]any{"name": "Proxy", "type": "Selector", "now": "Tokyo", "all": []string{"Tokyo", "DIRECT"}}
			proxies["Tokyo"] = map[string]any{"name": "Tokyo", "type": "Trojan"}
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"proxies": proxies}); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()
	client, err := mihomo.NewClient(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	p := homePage{client: client}
	groups, err := p.loadReadyProxyGroups()
	if err != nil {
		t.Fatal(err)
	}
	visible := visibleHomeGroups(groups)
	if requests != 2 || len(visible) != 1 || visible[0].Name != "Proxy" {
		t.Fatalf("requests=%d visible=%+v", requests, visible)
	}
}

func TestHomeUsesDirectGroupAndNodeNavigation(t *testing.T) {
	p := homePage{groups: []mihomo.ProxyGroup{
		{Name: "A", Proxies: []mihomo.Proxy{{Name: "A1"}, {Name: "A2"}}},
		{Name: "B", Proxies: []mihomo.Proxy{{Name: "B1"}, {Name: "B2"}}},
	}}
	page, _ := p.Update(tea.KeyMsg{Type: tea.KeyDown})
	p = page.(homePage)
	if p.nodeCursor != 1 {
		t.Fatalf("down node cursor = %d", p.nodeCursor)
	}
	page, _ = p.Update(tea.KeyMsg{Type: tea.KeyRight})
	p = page.(homePage)
	if p.groupCursor != 1 || p.nodeCursor != 0 {
		t.Fatalf("right navigation = group %d node %d", p.groupCursor, p.nodeCursor)
	}
	page, _ = p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if page.(homePage).nodeCursor != 0 {
		t.Fatal("up moved before first node")
	}
}

func TestHomeLoadsNodesAndPoliciesFromConfigSnapshot(t *testing.T) {
	cfg := config.Default()
	cfg.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")
	data := []byte("proxies:\n  - {name: Tokyo, type: trojan, server: example.test, port: 443}\nproxy-groups:\n  - {name: Proxy, type: select, proxies: [Tokyo, DIRECT]}\n  - {name: Final, type: select, proxies: [Proxy, DIRECT]}\n  - {name: GLOBAL, type: select, proxies: [Proxy, DIRECT]}\n")
	if _, err := runtimeconfig.Write(cfg.ConfigPath, data); err != nil {
		t.Fatal(err)
	}
	p := newHomePage(nil, &recordingManager{status: core.StatusStopped}, cfg).(homePage)
	page, _ := p.Update(p.Init()())
	p = page.(homePage)
	if !p.snapshot || len(p.groups) != 1 || p.groups[0].Name != "Proxy" || len(p.groups[0].Proxies) != 1 {
		t.Fatalf("home = %+v", p)
	}
	view := p.View(100, 30)
	for _, want := range []string{"Proxy", "Tokyo", "Stopped"} {
		if !strings.Contains(view, want) {
			t.Fatalf("home view missing %q: %q", want, view)
		}
	}
	for _, hidden := range []string{"DIRECT", "Direct", "Final", "GLOBAL"} {
		if strings.Contains(view, hidden) {
			t.Fatalf("home view unexpectedly contains %q: %q", hidden, view)
		}
	}
}

func TestHomeShowsErrorWhenRunningCoreHasNoController(t *testing.T) {
	p := homePage{
		coreManager: &recordingManager{status: core.StatusRunning},
		groups:      []mihomo.ProxyGroup{{Name: "Proxy"}},
		snapshot:    true,
	}
	page, _ := p.Update(proxyGroupsLoadedMsg{err: fmt.Errorf("connection refused")})
	p = page.(homePage)
	view := p.View(100, 20)
	if !strings.Contains(view, "Error") {
		t.Fatalf("home view does not report controller failure: %q", view)
	}
}

func TestNodeWindowKeepsCursorVisible(t *testing.T) {
	tests := []struct {
		name        string
		bodyHeight  int
		total       int
		cursor      int
		previous    int
		wantVisible bool
	}{
		{name: "top", bodyHeight: 8, total: 12, cursor: 0, previous: 0, wantVisible: true},
		{name: "middle", bodyHeight: 8, total: 12, cursor: 6, previous: 1, wantVisible: true},
		{name: "end", bodyHeight: 8, total: 12, cursor: 11, previous: 6, wantVisible: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			window := nodeWindow(tt.bodyHeight, tt.total, tt.cursor, tt.previous)
			gotVisible := tt.cursor >= window.start && tt.cursor < window.end
			if gotVisible != tt.wantVisible {
				t.Fatalf("cursor visibility = %v, want %v; window = %+v", gotVisible, tt.wantVisible, window)
			}
		})
	}
}

func TestNodeWindowFitsAvailableRows(t *testing.T) {
	window := nodeWindow(8, 12, 6, 1)
	renderedRows := window.end - window.start + 1 // position line
	if window.hasAbove {
		renderedRows++
	}
	if window.hasBelow {
		renderedRows++
	}

	if renderedRows > 8 {
		t.Fatalf("rendered rows = %d, want <= 8; window = %+v", renderedRows, window)
	}
}

func TestHomeViewFitsProvidedHeight(t *testing.T) {
	nodes := make([]mihomo.Proxy, 20)
	for index := range nodes {
		nodes[index] = mihomo.Proxy{Name: fmt.Sprintf("Node %d", index+1), Type: "trojan", Delay: -1}
	}
	p := homePage{cfg: config.Default(), groups: []mihomo.ProxyGroup{{Name: "Proxy", Proxies: nodes}}}
	for _, height := range []int{16, 20, 24} {
		if got := lipgloss.Height(p.View(100, height)); got > height {
			t.Fatalf("height %d rendered %d rows", height, got)
		}
	}
}
