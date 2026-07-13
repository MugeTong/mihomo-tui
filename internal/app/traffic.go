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

const trafficHistorySize = 60

type trafficPage struct {
	client          *mihomo.Client
	coreManager     core.Manager
	uploadSpeed     int64
	downloadSpeed   int64
	totalUpload     int64
	totalDownload   int64
	connections     int
	tcp             int
	udp             int
	active          []mihomo.ConnectionSummary
	uploadHistory   []int64
	downloadHistory []int64
	lastSample      time.Time
	initialized     bool
	status          string
	err             string
}

type trafficLoadedMsg struct {
	snapshot mihomo.ConnectionsSnapshot
	at       time.Time
	err      error
}

type trafficTickMsg struct{}
type trafficEnteredMsg struct{}

func newTrafficPage(client *mihomo.Client, coreManager core.Manager) Page {
	return trafficPage{client: client, coreManager: coreManager, status: "Waiting for traffic data"}
}

func (p trafficPage) Init() tea.Cmd {
	return func() tea.Msg { return trafficEnteredMsg{} }
}

func (p trafficPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case trafficEnteredMsg:
		p.resetSampling()
		return p, p.load()
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
		p.uploadHistory = appendTrafficSample(p.uploadHistory, p.uploadSpeed)
		p.downloadHistory = appendTrafficSample(p.downloadHistory, p.downloadSpeed)
	}
	p.totalUpload = msg.snapshot.UploadTotal
	p.totalDownload = msg.snapshot.DownloadTotal
	p.connections = msg.snapshot.Connections
	p.tcp = msg.snapshot.TCP
	p.udp = msg.snapshot.UDP
	p.active = append([]mihomo.ConnectionSummary(nil), msg.snapshot.Active...)
	p.lastSample = msg.at
	p.initialized = true
	p.err = ""
	p.status = "Live traffic"
}

func (p trafficPage) View(width, height int) string {
	historyLabelWidth := lipgloss.Width("  Down:  ")
	historyWidth := min(trafficHistorySize, max(width-historyLabelWidth-1, 10))
	scale := trafficScale(p.uploadHistory, p.downloadHistory)
	downHistory := renderTrafficHistory(p.downloadHistory, historyWidth, scale, nodeDelayGood)
	upHistory := renderTrafficHistory(p.uploadHistory, historyWidth, scale, nodeDelayMed)

	view := titleStyle.Render("Traffic") + "\n\n" +
		labelStyle.Render("  Up:           ") + valueStyle.Render(formatBytes(p.uploadSpeed)+"/s") + "\n" +
		labelStyle.Render("  Down:         ") + valueStyle.Render(formatBytes(p.downloadSpeed)+"/s") + "\n" +
		labelStyle.Render("  Total Up:     ") + valueStyle.Render(formatBytes(p.totalUpload)) + "\n" +
		labelStyle.Render("  Total Down:   ") + valueStyle.Render(formatBytes(p.totalDownload)) + "\n\n" +
		titleStyle.Render("Last 60 Seconds") + "\n\n" +
		labelStyle.Render("  Down:  ") + downHistory + "\n" +
		labelStyle.Render("  Up:    ") + upHistory + "\n" +
		labelStyle.Render("  Scale: ") + valueStyle.Render("0 – "+formatBytes(scale)+"/s") +
		labelStyle.Render(fmt.Sprintf(" · %d/%d samples", max(len(p.uploadHistory), len(p.downloadHistory)), trafficHistorySize)) + "\n\n" +
		titleStyle.Render(fmt.Sprintf("Active Connections · %d (TCP %d / UDP %d)", p.connections, p.tcp, p.udp)) + "\n\n"
	rows := max(height-lipgloss.Height(view), 1)
	return view + p.renderActiveConnections(width, min(rows, 5))
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

func (p *trafficPage) resetSampling() {
	p.uploadSpeed = 0
	p.downloadSpeed = 0
	p.totalUpload = 0
	p.totalDownload = 0
	p.connections = 0
	p.tcp = 0
	p.udp = 0
	p.active = nil
	p.uploadHistory = nil
	p.downloadHistory = nil
	p.lastSample = time.Time{}
	p.initialized = false
	p.err = ""
	p.status = "Waiting for traffic data"
}

func (p trafficPage) renderActiveConnections(width, limit int) string {
	if len(p.active) == 0 {
		return labelStyle.Render("  No active connections")
	}
	limit = min(limit, len(p.active))
	lines := make([]string, 0, limit)
	for _, connection := range p.active[:limit] {
		network := connection.Network
		if network == "" {
			network = "-"
		}
		route := connection.Route
		if route == "" {
			route = connection.Rule
		}
		prefix := "  " + padOrTruncate(connection.Target, max(width/2, 12)) + " "
		remaining := max(width-lipgloss.Width(prefix)-lipgloss.Width(network)-4, 1)
		line := labelStyle.Render(prefix) + sectionStyle.Render("["+network+"] ") + valueStyle.Render(truncateCells(route, remaining))
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
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

func appendTrafficSample(history []int64, sample int64) []int64 {
	history = append(history, max(sample, 0))
	if len(history) > trafficHistorySize {
		history = history[len(history)-trafficHistorySize:]
	}
	return history
}

func trafficScale(histories ...[]int64) int64 {
	var peak int64
	for _, history := range histories {
		for _, sample := range history {
			if sample > peak {
				peak = sample
			}
		}
	}
	const minimum = int64(1024)
	scale := minimum
	for scale < peak {
		scale *= 2
	}
	return scale
}

func renderTrafficHistory(history []int64, width int, scale int64, style lipgloss.Style) string {
	const levels = "▁▂▃▄▅▆▇█"
	if width < 1 {
		return ""
	}
	if len(history) > width {
		history = history[len(history)-width:]
	}
	var graph strings.Builder
	graph.WriteString(strings.Repeat(" ", width-len(history)))
	for _, sample := range history {
		level := 0
		if sample > 0 && scale > 0 {
			level = int(float64(sample) / float64(scale) * float64(len([]rune(levels))-1))
			level = min(max(level, 1), len([]rune(levels))-1)
		}
		graph.WriteRune([]rune(levels)[level])
	}
	return style.Render(graph.String())
}
