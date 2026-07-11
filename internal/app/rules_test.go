package app

import (
	"strings"
	"testing"

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

func TestRulesSearchBarShowsInputState(t *testing.T) {
	p := newRulesPage().(rulesPage)
	p.searching = true
	p.filter = "github.com"
	view := p.View(60, 20)
	if !strings.Contains(view, "Search:") || !strings.Contains(view, "github.com_") {
		t.Fatalf("search input not rendered: %q", view)
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
