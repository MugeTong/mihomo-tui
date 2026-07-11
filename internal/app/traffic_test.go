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
	view := p.View(100, 24)
	for _, want := range []string{"1.0 KiB/s", "2.0 KiB/s", "3", "TCP", "UDP"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q: %s", want, view)
		}
	}
}

func TestTrafficViewLeavesFinalTerminalCellUnused(t *testing.T) {
	const width = 74
	view := trafficPage{uploadSpeed: 1024, downloadSpeed: 2048, peakSpeed: 3072}.View(width, 24)
	for lineNumber, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got >= width {
			t.Fatalf("line %d width = %d, want < %d", lineNumber+1, got, width)
		}
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
