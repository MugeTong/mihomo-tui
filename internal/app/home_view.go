package app

import (
	"fmt"
	"strings"

	"mihomo-tui/internal/core"

	"github.com/charmbracelet/lipgloss"
)

func (p homePage) View(width, height int) string {
	contentWidth := max(width-4, 10)

	statusSection := p.renderStatusSection()
	groupSection := p.renderGroupsSection(contentWidth)
	nodesHeight := p.nodesViewportHeight(height)
	p.ensureNodeVisible(nodesHeight)
	nodesSection := p.renderNodesSection(contentWidth, nodesHeight)

	return statusSection + "\n\n" + groupSection + "\n\n" + nodesSection
}

func (p homePage) Help() string {
	return "space core • d delay • enter select • arrows navigate"
}

func (p homePage) renderStatusSection() string {
	title := titleStyle.Render("Status")
	body := labelStyle.Render("  State:      ") + renderHomeState(p) + "\n" +
		labelStyle.Render("  Mode:       ") + valueStyle.Render(p.proxyMode)

	return title + "\n\n" + body
}

func renderHomeState(p homePage) string {
	status := p.coreStatus()
	switch {
	case status == core.StatusStarting || p.connecting && status == core.StatusRunning:
		return warningStyle.Render("Starting")
	case status == core.StatusStopping:
		return warningStyle.Render("Stopping")
	case status == core.StatusRunning && p.connected:
		return onStyle.Render("Running")
	case status == core.StatusStopped:
		return offStyle.Render("Stopped")
	default:
		return offStyle.Render("Error")
	}
}

func (p homePage) renderGroupsSection(_ int) string {
	title := titleStyle.Render("Proxy Groups")
	if len(p.groups) == 0 {
		return title + "\n\n" + labelStyle.Render("  No proxy groups loaded")
	}

	groupLabels := make([]string, 0, len(p.groups))
	for i, group := range p.groups {
		text := group.Name
		if group.Now != "" {
			text += ":" + group.Now
		}
		if i == p.groupCursor {
			groupLabels = append(groupLabels, tabActiveStyle.Render(text))
		} else {
			groupLabels = append(groupLabels, tabStyle.Render(text))
		}
	}

	line := strings.Join(groupLabels, tabSepStyle.Render(" "))
	return title + "\n\n" + line
}

func (p homePage) renderNodesSection(width, height int) string {
	title := titleStyle.Render("Nodes")
	buttons := btnDisabledStyle.Render("Delay")
	gap := max(width-lipgloss.Width(title)-lipgloss.Width(buttons), 1)
	titleLine := title + strings.Repeat(" ", gap) + buttons

	var nodeList string
	nodes := p.visibleNodes()
	current := p.currentGroup()
	if len(nodes) == 0 {
		nodeList = labelStyle.Render("  No nodes loaded")
	} else {
		bodyHeight := max(height-2, 1)
		windowInfo := nodeWindow(bodyHeight, len(nodes), p.nodeCursor, p.nodeOffset)
		p.nodeOffset = windowInfo.start
		window := nodes[windowInfo.start:windowInfo.end]
		var lines []string
		for i, n := range window {
			absoluteIndex := windowInfo.start + i
			nodeMarker := "  "
			nameStyle := nodeInactiveStyle
			if absoluteIndex == p.nodeCursor {
				nodeMarker = "> "
			}
			if n.Name == current.Now {
				nameStyle = nodeActiveStyle
			}

			prefix := labelStyle.Render(fmt.Sprintf("  %s%d. ", nodeMarker, absoluteIndex+1))
			name := nameStyle.Render(n.Name)
			typ := labelStyle.Render(fmt.Sprintf(" [%s]", n.Type))
			delay := renderDelay(n.Delay)

			lines = append(lines, prefix+name+typ+"  "+delay)
		}
		if windowInfo.hasAbove {
			lines = append([]string{labelStyle.Render("  ...")}, lines...)
		}
		if windowInfo.hasBelow {
			lines = append(lines, labelStyle.Render("  ..."))
		}
		lines = append(lines, labelStyle.Render(fmt.Sprintf("  %d/%d", p.nodeCursor+1, len(nodes))))
		nodeList = strings.Join(lines, "\n")
	}

	return titleLine + "\n\n" + nodeList
}

func (p homePage) nodesViewportHeight(contentHeight int) int {
	// Status is four lines, groups are three, and the two section gaps each
	// contribute one blank line.
	const rowsBeforeNodes = 4 + 3 + 2
	return max(contentHeight-rowsBeforeNodes, 1)
}

func renderDelay(ms int) string {
	if ms < 0 {
		return nodeDelayNone.Render("--")
	}
	text := fmt.Sprintf("%dms", ms)
	switch {
	case ms < 200:
		return nodeDelayGood.Render(text)
	case ms < 500:
		return nodeDelayMed.Render(text)
	default:
		return nodeDelayBad.Render(text)
	}
}
