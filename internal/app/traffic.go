package app

import (
	"fmt"
	"strings"
	"time"

	"mihomo-tui/internal/core"
	"mihomo-tui/internal/mihomo"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const trafficPollInterval = time.Second

type trafficPage struct {
	client        *mihomo.Client
	coreManager   core.Manager
	uploadSpeed   int64
	downloadSpeed int64
	totalUpload   int64
	totalDownload int64
	connections   int
	tcp           int
	udp           int
	peakSpeed     int64
	lastSample    time.Time
	initialized   bool
	status        string
	err           string
}

type trafficLoadedMsg struct {
	snapshot mihomo.ConnectionsSnapshot
	at       time.Time
	err      error
}

type trafficTickMsg struct{}

func newTrafficPage(client *mihomo.Client, coreManager core.Manager) Page {
	return trafficPage{client: client, coreManager: coreManager, status: "Waiting for traffic data"}
}

func (p trafficPage) Init() tea.Cmd {
	return p.load()
}

func (p trafficPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case trafficTickMsg:
		return p, p.load()
	case trafficLoadedMsg:
		p.applySnapshot(msg)
		return p, trafficTick()
	}
	return p, nil
}

func (p *trafficPage) applySnapshot(msg trafficLoadedMsg) {
	if msg.err != nil {
		p.uploadSpeed = 0
		p.downloadSpeed = 0
		p.err = msg.err.Error()
		p.status = "Traffic unavailable"
		return
	}
	if p.initialized {
		elapsed := msg.at.Sub(p.lastSample).Seconds()
		if elapsed > 0 {
			p.uploadSpeed = bytesPerSecond(msg.snapshot.UploadTotal, p.totalUpload, elapsed)
			p.downloadSpeed = bytesPerSecond(msg.snapshot.DownloadTotal, p.totalDownload, elapsed)
		}
	}
	p.totalUpload = msg.snapshot.UploadTotal
	p.totalDownload = msg.snapshot.DownloadTotal
	p.connections = msg.snapshot.Connections
	p.tcp = msg.snapshot.TCP
	p.udp = msg.snapshot.UDP
	p.lastSample = msg.at
	p.initialized = true
	currentSpeed := p.uploadSpeed + p.downloadSpeed
	if currentSpeed > p.peakSpeed {
		p.peakSpeed = currentSpeed
	}
	p.err = ""
	p.status = "Live traffic"
}

func (p trafficPage) View(width, _ int) string {
	// Leave the final terminal cell unused; filling it can trigger an automatic
	// wrap in terminals even when the measured content width is exact.
	barLabel := "  Bandwidth:    "
	barWidth := max(width-lipgloss.Width(barLabel)-1, 10)
	percentage := 0.0
	if p.peakSpeed > 0 {
		percentage = float64(p.uploadSpeed+p.downloadSpeed) / float64(p.peakSpeed)
	}

	return headerStyle.Render("Traffic") + "\n\n" +
		labelStyle.Render("  Up:           ") + valueStyle.Render(formatBytes(p.uploadSpeed)+"/s") + "\n" +
		labelStyle.Render("  Down:         ") + valueStyle.Render(formatBytes(p.downloadSpeed)+"/s") + "\n" +
		labelStyle.Render("  Total Up:     ") + valueStyle.Render(formatBytes(p.totalUpload)) + "\n" +
		labelStyle.Render("  Total Down:   ") + valueStyle.Render(formatBytes(p.totalDownload)) + "\n\n" +
		labelStyle.Render(barLabel) + renderBar(barWidth, percentage) + "\n\n" +
		headerStyle.Render("Connections") + "\n\n" +
		labelStyle.Render("  Active:       ") + valueStyle.Render(fmt.Sprintf("%d", p.connections)) + "\n" +
		labelStyle.Render("  TCP:          ") + valueStyle.Render(fmt.Sprintf("%d", p.tcp)) + "\n" +
		labelStyle.Render("  UDP:          ") + valueStyle.Render(fmt.Sprintf("%d", p.udp))
}

func (p trafficPage) Help() string { return "live traffic • updates every second" }

func (p trafficPage) Message() string {
	if p.err != "" {
		return p.err
	}
	return p.status
}

func (p trafficPage) load() tea.Cmd {
	client, manager := p.client, p.coreManager
	return func() tea.Msg {
		if client == nil || manager == nil || manager.Status() != core.StatusRunning {
			return trafficLoadedMsg{at: time.Now(), err: fmt.Errorf("start Mihomo to view live traffic")}
		}
		snapshot, err := client.Connections()
		return trafficLoadedMsg{snapshot: snapshot, at: time.Now(), err: err}
	}
}

func trafficTick() tea.Cmd {
	return tea.Tick(trafficPollInterval, func(time.Time) tea.Msg { return trafficTickMsg{} })
}

func bytesPerSecond(current, previous int64, elapsed float64) int64 {
	if current < previous || elapsed <= 0 {
		return 0
	}
	return int64(float64(current-previous) / elapsed)
}

func formatBytes(value int64) string {
	if value < 0 {
		value = 0
	}
	const unit = int64(1024)
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	divisor := unit
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	unitIndex := 0
	for value >= divisor*unit && unitIndex < len(units)-1 {
		divisor *= unit
		unitIndex++
	}
	return fmt.Sprintf("%.1f %s", float64(value)/float64(divisor), units[unitIndex])
}

func renderBar(width int, pct float64) string {
	if width < 4 {
		width = 4
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	filled := int(float64(width) * pct)
	empty := width - filled
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color("120")).Render(strings.Repeat("#", filled))
	bar += lipgloss.NewStyle().Foreground(lipgloss.Color("237")).Render(strings.Repeat(".", empty))
	return bar
}
