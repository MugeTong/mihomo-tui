package app

import (
	"strings"
	"testing"
	"time"

	"mihomo-tui/internal/mihomo"

	"github.com/charmbracelet/lipgloss"
)

func TestTrafficPageCalculatesRates(t *testing.T) {
	start := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	p := trafficPage{}
	p.applySnapshot(trafficLoadedMsg{
		snapshot: mihomo.ConnectionsSnapshot{UploadTotal: 1024, DownloadTotal: 2048, Connections: 2, TCP: 1, UDP: 1},
		at:       start,
	})
	p.applySnapshot(trafficLoadedMsg{
		snapshot: mihomo.ConnectionsSnapshot{UploadTotal: 3072, DownloadTotal: 6144, Connections: 3, TCP: 2, UDP: 1},
		at:       start.Add(2 * time.Second),
	})
	if p.uploadSpeed != 1024 || p.downloadSpeed != 2048 {
		t.Fatalf("speeds = %d/%d, want 1024/2048", p.uploadSpeed, p.downloadSpeed)
	}
	if p.connections != 3 || p.tcp != 2 || p.udp != 1 {
		t.Fatalf("connections = %d tcp=%d udp=%d", p.connections, p.tcp, p.udp)
	}
	if len(p.uploadHistory) != 1 || p.uploadHistory[0] != 1024 || len(p.downloadHistory) != 1 || p.downloadHistory[0] != 2048 {
		t.Fatalf("traffic history = up %v down %v", p.uploadHistory, p.downloadHistory)
	}
	view := p.View(100, 24)
	for _, want := range []string{"1.0 KiB/s", "2.0 KiB/s", "Last 60 Seconds", "1/60 samples", "3", "TCP", "UDP"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q: %s", want, view)
		}
	}
}

func TestTrafficViewLeavesFinalTerminalCellUnused(t *testing.T) {
	const width = 74
	view := trafficPage{
		uploadSpeed:     1024,
		downloadSpeed:   2048,
		uploadHistory:   []int64{512, 1024},
		downloadHistory: []int64{1024, 2048},
	}.View(width, 24)
	for lineNumber, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got >= width {
			t.Fatalf("line %d width = %d, want < %d", lineNumber+1, got, width)
		}
	}
}

func TestTrafficViewFitsCompactContentHeight(t *testing.T) {
	p := trafficPage{
		uploadHistory:   []int64{1024, 2048},
		downloadHistory: []int64{2048, 4096},
	}
	if got := lipgloss.Height(p.View(100, 16)); got > 16 {
		t.Fatalf("traffic view height = %d, want at most 16", got)
	}
}

func TestTrafficInitResetsSamplingHistory(t *testing.T) {
	p := trafficPage{
		initialized:   true,
		uploadSpeed:   1024,
		uploadHistory: []int64{1024},
		totalUpload:   4096,
		connections:   1,
		active:        []mihomo.ConnectionSummary{{Target: "example.com:443"}},
	}
	page, _ := p.Update(p.Init()())
	p = page.(trafficPage)
	if p.initialized || p.uploadSpeed != 0 || len(p.uploadHistory) != 0 || p.totalUpload != 0 || len(p.active) != 0 {
		t.Fatalf("traffic sampling was not reset: %+v", p)
	}
}

func TestTrafficIgnoresMessagesFromPreviousVisit(t *testing.T) {
	p := trafficPage{generation: 2, uploadSpeed: 123}
	page, cmd := p.Update(trafficLoadedMsg{
		generation: 1,
		snapshot:   mihomo.ConnectionsSnapshot{UploadTotal: 9999},
		at:         time.Now(),
	})
	p = page.(trafficPage)
	if cmd != nil || p.uploadSpeed != 123 || p.totalUpload != 0 {
		t.Fatalf("stale traffic message changed page: %+v", p)
	}
}

func TestTrafficShowsCompactActiveConnections(t *testing.T) {
	p := trafficPage{
		connections: 2,
		tcp:         1,
		udp:         1,
		active: []mihomo.ConnectionSummary{
			{Target: "api.openai.com:443", Network: "TCP", Route: "Proxy → Tokyo"},
			{Target: "1.1.1.1:53", Network: "UDP", Rule: "MATCH"},
		},
	}
	view := p.View(100, 20)
	for _, want := range []string{"Active Connections · 2", "api.openai.com:443", "Proxy → Tokyo", "1.1.1.1:53", "MATCH"} {
		if !strings.Contains(view, want) {
			t.Fatalf("traffic view missing %q: %s", want, view)
		}
	}
}

func TestTrafficHistoryUsesSlidingWindowAndDynamicScale(t *testing.T) {
	history := make([]int64, 0, trafficHistorySize+1)
	for sample := int64(1); sample <= trafficHistorySize+1; sample++ {
		history = appendTrafficSample(history, sample)
	}
	if len(history) != trafficHistorySize || history[0] != 2 || history[len(history)-1] != trafficHistorySize+1 {
		t.Fatalf("sliding history = %v", history)
	}
	if got := trafficScale([]int64{1024, 3072}); got != 4096 {
		t.Fatalf("traffic scale = %d, want 4096", got)
	}
	graph := renderTrafficHistory([]int64{0, 1024, 4096}, 3, 4096, valueStyle)
	if got := lipgloss.Width(graph); got != 3 {
		t.Fatalf("graph width = %d, want 3", got)
	}
}

func TestBytesPerSecondHandlesCounterReset(t *testing.T) {
	if got := bytesPerSecond(10, 20, 1); got != 0 {
		t.Fatalf("rate after reset = %d, want 0", got)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := map[int64]string{0: "0 B", 1024: "1.0 KiB", 1024 * 1024: "1.0 MiB"}
	for value, want := range tests {
		if got := formatBytes(value); got != want {
			t.Fatalf("formatBytes(%d) = %q, want %q", value, got, want)
		}
	}
}
