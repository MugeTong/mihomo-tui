package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestRulesViewFitsTerminalWidth(t *testing.T) {
	p := newRulesPage().(rulesPage)
	for _, width := range []int{32, 48, 80} {
		view := p.View(width, 20)
		for lineNumber, line := range strings.Split(view, "\n") {
			if got := lipgloss.Width(line); got > width {
				t.Fatalf("width %d, line %d rendered at %d cells: %q", width, lineNumber+1, got, line)
			}
		}
	}
}

func TestDomainRuleMatchUsesFirstSupportedRule(t *testing.T) {
	rules := []routingRule{
		{Type: "DOMAIN", Value: "api.example.com", Policy: "Direct"},
		{Type: "DOMAIN-SUFFIX", Value: "example.com", Policy: "Proxy"},
		{Type: "DOMAIN-KEYWORD", Value: "example", Policy: "Keyword"},
		{Type: "MATCH", Policy: "Final"},
	}
	tests := []struct {
		domain string
		policy string
	}{
		{"API.EXAMPLE.COM.", "Direct"},
		{"www.example.com", "Proxy"},
		{"notexample.net", "Keyword"},
		{"other.net", "Final"},
	}
	for _, test := range tests {
		matched, ok := matchDomainRule(test.domain, rules)
		if !ok || matched.Policy != test.policy {
			t.Fatalf("matchDomainRule(%q) = %+v, %v; want policy %s", test.domain, matched, ok, test.policy)
		}
	}
}

func TestRulesEnterChecksDomainAndShowsMatchedRule(t *testing.T) {
	p := rulesPage{
		rules: []routingRule{
			{Type: "DOMAIN-SUFFIX", Value: "example.com", Policy: "Proxy"},
			{Type: "MATCH", Policy: "Final"},
		},
		searching: true,
		filter:    "www.example.com",
	}
	page, _ := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = page.(rulesPage)
	if p.match == nil || p.match.Type != "DOMAIN-SUFFIX" {
		t.Fatalf("matched rule = %+v", p.match)
	}
	if got := p.Message(); got != "www.example.com → DOMAIN-SUFFIX example.com → Proxy" {
		t.Fatalf("message = %q", got)
	}
	if visible := p.visibleRules(); len(visible) != 1 || visible[0].Type != "DOMAIN-SUFFIX" {
		t.Fatalf("visible rules = %+v", visible)
	}
}

func TestRulesRejectsNonDomainInput(t *testing.T) {
	p := rulesPage{searching: true, filter: "https://example.com/path"}
	page, _ := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = page.(rulesPage)
	if p.match != nil || !strings.Contains(p.status, "valid domain") {
		t.Fatalf("invalid domain result = match %+v, status %q", p.match, p.status)
	}
}

func TestRulesSearchBarShowsInputState(t *testing.T) {
	p := newRulesPage().(rulesPage)
	p.searching = true
	p.filter = "github.com"
	p.previewDomainMatch()
	view := p.View(60, 20)
	if !strings.Contains(view, "Search:") || !strings.Contains(view, "github.com_") {
		t.Fatalf("search input not rendered: %q", view)
	}
}

func TestRulesDomainMatchUpdatesWhileTyping(t *testing.T) {
	p := rulesPage{
		rules: []routingRule{
			{Type: "DOMAIN-KEYWORD", Value: "google", Policy: "Proxy"},
			{Type: "MATCH", Policy: "Final"},
		},
		searching: true,
	}
	for _, character := range "google.com" {
		page, _ := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{character}})
		p = page.(rulesPage)
	}
	if p.match == nil || p.match.Type != "DOMAIN-KEYWORD" || p.match.Policy != "Proxy" {
		t.Fatalf("live domain match = %+v", p.match)
	}
	visible := p.visibleRules()
	if len(visible) != 1 || visible[0].Value != "google" {
		t.Fatalf("visible rules = %+v", visible)
	}
}

func TestRulesSearchBarDoesNotFillLastTerminalColumn(t *testing.T) {
	p := newRulesPage().(rulesPage)
	firstLine := strings.Split(p.View(120, 20), "\n")[0]
	if got := lipgloss.Width(firstLine); got >= 120 {
		t.Fatalf("search header width = %d, want less than terminal width", got)
	}
}

func TestRulesViewUsesAvailableHeight(t *testing.T) {
	p := newRulesPage().(rulesPage)
	const height = 20
	if got := lipgloss.Height(p.View(80, height)); got != height {
		t.Fatalf("rules view height = %d, want %d", got, height)
	}
}

func TestRuleColumnsUseTypePolicyValueOrder(t *testing.T) {
	p := newRulesPage().(rulesPage)
	view := p.View(80, 20)
	typePos := strings.Index(view, "TYPE")
	policyPos := strings.Index(view, "POLICY")
	valuePos := strings.Index(view, "VALUE")
	if typePos < 0 || policyPos <= typePos || valuePos <= policyPos {
		t.Fatalf("unexpected rule column order: %q", view)
	}
}
