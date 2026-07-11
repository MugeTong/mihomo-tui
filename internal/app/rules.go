package app

import (
	"fmt"
	defaultRules "mihomo-tui/internal/rules"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type routingRule struct {
	Type   string
	Value  string
	Policy string
}

type rulesPage struct {
	rules     []routingRule
	cursor    int
	offset    int
	searching bool
	filter    string
	status    string
}

func newRulesPage() Page {
	rules, err := loadDefaultRules()
	status := "Read-only routing rules"
	if err != nil {
		status = "Default rules unavailable: " + err.Error()
	}
	return rulesPage{
		rules:  rules,
		status: status,
	}
}

func (p rulesPage) Init() tea.Cmd {
	return nil
}

func (p rulesPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	if p.searching {
		return p.updateSearchKey(key), nil
	}

	switch key.String() {
	case "j", "down":
		if p.cursor < len(p.visibleRules())-1 {
			p.cursor++
		}
	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
		}
	case "/":
		p.searching = true
		p.status = "Filtering rules"
	case "esc":
		p.filter = ""
		p.cursor = 0
		p.offset = 0
		p.status = "Filter cleared"
	}

	return p, nil
}

func (p rulesPage) updateSearchKey(key tea.KeyMsg) rulesPage {
	switch key.Type {
	case tea.KeyEsc:
		p.searching = false
		p.filter = ""
		p.cursor = 0
		p.offset = 0
		p.status = "Filter cleared"
	case tea.KeyEnter:
		p.searching = false
		p.status = "Filter applied"
	case tea.KeyBackspace:
		if len(p.filter) > 0 {
			p.filter = p.filter[:len(p.filter)-1]
			p.cursor = 0
			p.offset = 0
		}
	case tea.KeyRunes:
		p.filter += key.String()
		p.cursor = 0
		p.offset = 0
	}
	p.clamp()
	return p
}

func (p rulesPage) View(width, height int) string {
	title := titleStyle.Render("Routing Rules")
	// Keep the input compact and leave a spare cell on the right. Filling the
	// terminal's final column can trigger an automatic wrap in some terminals.
	searchWidth := min(48, max(width-lipgloss.Width(title)-3, 1))
	searchBar := p.renderSearchBar(searchWidth)
	typeWidth, valueWidth, policyWidth := ruleColumnWidths(width)
	columns := sectionStyle.Render("  ") +
		labelStyle.Render(padOrTruncate("TYPE", typeWidth)) + " " +
		labelStyle.Render(padOrTruncate("POLICY", policyWidth)) + " " +
		labelStyle.Render(padOrTruncate("VALUE", valueWidth))
	header := title + "  " + searchBar + "\n\n" + columns

	rules := p.visibleRules()
	bodyHeight := max(height-3, 1)
	if len(rules) == 0 {
		return header + "\n" + labelStyle.Render("  No rules matched")
	}

	p.ensureVisible(bodyHeight, len(rules))
	window := nodeWindow(bodyHeight, len(rules), p.cursor, p.offset)
	p.offset = window.start

	var lines []string
	if window.hasAbove {
		lines = append(lines, labelStyle.Render("  ..."))
	}
	for i, rule := range rules[window.start:window.end] {
		absoluteIndex := window.start + i
		marker := "  "
		if absoluteIndex == p.cursor {
			marker = "> "
		}
		lines = append(lines, p.renderRuleLine(width, marker, rule))
	}
	if window.hasBelow {
		lines = append(lines, labelStyle.Render("  ..."))
	}
	lines = append(lines, labelStyle.Render(fmt.Sprintf("  %d/%d", p.cursor+1, len(rules))))

	return header + "\n" + strings.Join(lines, "\n")
}

func (p rulesPage) Help() string {
	if p.searching {
		return "type filter • enter apply • esc clear"
	}
	return "up/down rule • / filter • esc clear"
}

func (p rulesPage) Message() string {
	if p.filter != "" {
		return fmt.Sprintf("%s: %d matched", p.status, len(p.visibleRules()))
	}
	return p.status
}

func (p rulesPage) InputActive() bool {
	return p.searching
}

func (p rulesPage) renderSearchBar(width int) string {
	label := labelStyle.Render("Search: ")
	available := max(width-lipgloss.Width(label)-2, 1)
	value := p.filter
	if p.searching {
		value += "_"
	} else if value == "" {
		value = "Press / to search"
	}
	value = padOrTruncate(value, available)
	return label + valueStyle.Render("["+value+"]")
}

func (p rulesPage) renderRuleLine(width int, marker string, rule routingRule) string {
	value := rule.Value
	if value == "" {
		value = "-"
	}
	typeWidth, valueWidth, policyWidth := ruleColumnWidths(width)
	return sectionStyle.Render(marker) +
		valueStyle.Render(padOrTruncate(rule.Type, typeWidth)) + " " +
		policyStyle(rule.Policy).Render(padOrTruncate(rule.Policy, policyWidth)) + " " +
		labelStyle.Render(padOrTruncate(value, valueWidth))
}

func ruleColumnWidths(width int) (typeWidth, valueWidth, policyWidth int) {
	usable := max(width-4, 12) // marker and two column gaps
	policyWidth = min(12, max(6, usable/5))
	typeWidth = min(16, max(6, usable/4))
	valueWidth = max(1, usable-typeWidth-policyWidth)
	return
}

func padOrTruncate(value string, width int) string {
	value = truncateCells(value, width)
	return value + strings.Repeat(" ", max(width-lipgloss.Width(value), 0))
}

func (p rulesPage) visibleRules() []routingRule {
	filter := strings.TrimSpace(strings.ToLower(p.filter))
	if filter == "" {
		return p.rules
	}

	filtered := make([]routingRule, 0, len(p.rules))
	for _, rule := range p.rules {
		haystack := strings.ToLower(rule.Type + " " + rule.Value + " " + rule.Policy)
		if strings.Contains(haystack, filter) {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

func (p *rulesPage) ensureVisible(height, total int) {
	if total == 0 {
		p.cursor = 0
		p.offset = 0
		return
	}
	if p.cursor >= total {
		p.cursor = total - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	p.offset = nodeWindow(height, total, p.cursor, p.offset).start
}

func (p *rulesPage) clamp() {
	total := len(p.visibleRules())
	if total == 0 {
		p.cursor = 0
		p.offset = 0
		return
	}
	if p.cursor >= total {
		p.cursor = total - 1
	}
	if p.offset > p.cursor {
		p.offset = p.cursor
	}
}

func policyStyle(policy string) lipgloss.Style {
	switch strings.ToLower(policy) {
	case "direct":
		return nodeDelayGood
	case "reject":
		return nodeDelayBad
	default:
		return nodeDelayMed
	}
}

func loadDefaultRules() ([]routingRule, error) {
	embedded, err := defaultRules.Default()
	if err != nil {
		return nil, err
	}
	rules := make([]routingRule, 0, len(embedded))
	for _, rule := range embedded {
		policy := rule.Policy
		if policy == "DIRECT" {
			policy = "Direct"
		}
		rules = append(rules, routingRule{Type: rule.Type, Value: rule.Value, Policy: policy})
	}
	return rules, nil
}
