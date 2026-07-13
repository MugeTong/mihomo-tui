package app

import (
	"fmt"
	defaultRules "mihomo-tui/internal/rules"
	"strings"
	"unicode"

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
	match     *routingRule
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
		p.status = "Enter a domain to check routing"
	case "esc":
		p.filter = ""
		p.match = nil
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
		p.match = nil
		p.cursor = 0
		p.offset = 0
		p.status = "Filter cleared"
	case tea.KeyEnter:
		p.searching = false
		p.applyDomainMatch()
	case tea.KeyBackspace:
		if len(p.filter) > 0 {
			p.filter = trimLastRune(p.filter)
			p.previewDomainMatch()
			p.cursor = 0
			p.offset = 0
		}
	case tea.KeyRunes:
		p.filter += key.String()
		p.previewDomainMatch()
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
	searchWidth := min(48, max(width-lipgloss.Width("  ")-1, 1))
	searchBar := p.renderSearchBar(searchWidth)
	typeWidth, valueWidth, policyWidth := ruleColumnWidths(width)
	columns := sectionStyle.Render("  ") +
		labelStyle.Render(padOrTruncate("TYPE", typeWidth)) + " " +
		labelStyle.Render(padOrTruncate("POLICY", policyWidth)) + " " +
		labelStyle.Render(padOrTruncate("VALUE", valueWidth))
	header := title + "\n\n" + sectionStyle.Render("  ") + searchBar + "\n\n" + columns

	rules := p.visibleRules()
	bodyHeight := max(height-5, 1)
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
		return "type domain • enter check • esc clear"
	}
	return "up/down rule • / check domain • esc clear"
}

func (p rulesPage) Message() string {
	if p.match != nil {
		rule := p.match.Type
		if value := displayRuleValue(*p.match); value != "" {
			rule += " " + value
		}
		return fmt.Sprintf("%s → %s → %s", normalizedDomain(p.filter), rule, p.match.Policy)
	}
	if p.filter != "" {
		return p.status
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
	if p.match != nil {
		return []routingRule{*p.match}
	}
	if strings.TrimSpace(p.filter) == "" {
		return p.rules
	}
	return nil
}

func (p *rulesPage) previewDomainMatch() {
	domain := normalizedDomain(p.filter)
	if !validDomain(domain) {
		p.match = nil
		p.status = "Enter a domain to check routing"
		return
	}
	matched, ok := matchDomainRule(domain, p.rules)
	if !ok {
		p.match = nil
		p.status = "No supported domain rule matched"
		return
	}
	p.match = &matched
	p.status = "Domain rule preview"
}

func (p *rulesPage) applyDomainMatch() {
	domain := normalizedDomain(p.filter)
	if !validDomain(domain) {
		p.match = nil
		p.status = "Enter a valid domain, for example www.example.com"
		return
	}
	matched, ok := matchDomainRule(domain, p.rules)
	if !ok {
		p.match = nil
		p.status = "No supported domain rule matched"
		return
	}
	p.match = &matched
	p.cursor = 0
	p.offset = 0
	p.status = "Domain rule matched"
}

func matchDomainRule(domain string, rules []routingRule) (routingRule, bool) {
	domain = normalizedDomain(domain)
	for _, rule := range rules {
		value := normalizedDomain(rule.Value)
		switch strings.ToUpper(rule.Type) {
		case "DOMAIN":
			if domain == value {
				return rule, true
			}
		case "DOMAIN-SUFFIX":
			if domain == value || strings.HasSuffix(domain, "."+value) {
				return rule, true
			}
		case "DOMAIN-KEYWORD":
			if value != "" && strings.Contains(domain, value) {
				return rule, true
			}
		case "MATCH":
			return rule, true
		}
	}
	return routingRule{}, false
}

func normalizedDomain(value string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
}

func validDomain(domain string) bool {
	if domain == "" || len(domain) > 253 {
		return false
	}
	for _, label := range strings.Split(domain, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, character := range label {
			if !unicode.IsLetter(character) && !unicode.IsDigit(character) && character != '-' {
				return false
			}
		}
	}
	return true
}

func displayRuleValue(rule routingRule) string {
	if rule.Type == "MATCH" || rule.Value == "" || rule.Value == "-" {
		return ""
	}
	return rule.Value
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
