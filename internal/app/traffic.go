package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type trafficPage struct {
	uploadSpeed   string
	downloadSpeed string
	totalUpload   string
	totalDownload string
	connections   int
}

func newTrafficPage() Page {
	return trafficPage{
		uploadSpeed:   "0 B/s",
		downloadSpeed: "0 B/s",
		totalUpload:   "0 B",
		totalDownload: "0 B",
	}
}

func (p trafficPage) Init() tea.Cmd {
	return nil
}

func (p trafficPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	return p, nil
}

func (p trafficPage) View(width, _ int) string {
	barWidth := max(width-16, 10)

	return headerStyle.Render("Traffic / Connections") + "\n" +
		labelStyle.Render("  Up:           ") + valueStyle.Render(p.uploadSpeed) + "\n" +
		labelStyle.Render("  Down:         ") + valueStyle.Render(p.downloadSpeed) + "\n" +
		labelStyle.Render("  Total Up:     ") + valueStyle.Render(p.totalUpload) + "\n" +
		labelStyle.Render("  Total Down:   ") + valueStyle.Render(p.totalDownload) + "\n\n" +
		labelStyle.Render("  Bandwidth:    ") + renderBar(barWidth, 0.0) + "\n\n" +
		labelStyle.Render("  Active Conn:  ") + valueStyle.Render(fmt.Sprintf("%d", p.connections)) + "\n" +
		labelStyle.Render("  TCP:          ") + valueStyle.Render("0") + "\n" +
		labelStyle.Render("  UDP:          ") + valueStyle.Render("0") + "\n"
}

func (p trafficPage) Help() string {
	return "traffic preview"
}

func renderBar(width int, pct float64) string {
	if width < 4 {
		width = 4
	}
	filled := int(float64(width) * pct)
	empty := width - filled
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color("120")).Render(strings.Repeat("#", filled))
	bar += lipgloss.NewStyle().Foreground(lipgloss.Color("237")).Render(strings.Repeat(".", empty))
	return bar
}
