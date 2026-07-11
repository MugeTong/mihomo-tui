package app

import (
	"path/filepath"
	"strings"
	"testing"

	"mihomo-tui/internal/subscription"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestSourcesPageShareInputAddsImmediately(t *testing.T) {
	store := subscription.Store{Path: filepath.Join(t.TempDir(), "state.json")}
	p := newSourcesPageWithStore(store, nil).(sourcesPage)
	p.focused = true
	p.inputField = 1
	p.nameInput = "Work Subscription"
	p.input = "trojan://test@jp.example.test:443#Tokyo"

	page, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = page.(sourcesPage)
	if cmd == nil {
		t.Fatal("enter did not add source")
	}
	page, _ = p.Update(cmd())
	p = page.(sourcesPage)
	if len(p.state.Sources) != 1 || len(p.state.Nodes) != 1 {
		t.Fatalf("state = %+v", p.state)
	}
	if p.state.Sources[0].Name != "Work Subscription" || p.state.Sources[0].Type != subscription.SourceShare {
		t.Fatalf("source = %+v", p.state.Sources[0])
	}
	if p.focused || p.input != "" {
		t.Fatal("input was not cleared/unfocused after add")
	}
}

func TestSourcesInputLineFitsTerminalWidth(t *testing.T) {
	p := newSourcesPageWithStore(subscription.Store{}, nil).(sourcesPage)
	for _, width := range []int{48, 80, 120} {
		lines := strings.Split(p.View(width, 20), "\n")
		for _, lineNumber := range []int{1, 2} {
			line := lines[lineNumber]
			if got := lipgloss.Width(line); got >= width {
				t.Fatalf("width %d rendered line %d at %d cells", width, lineNumber+1, got)
			}
		}
	}
}

func TestSourcesPageOnlyCapturesInputAfterA(t *testing.T) {
	p := newSourcesPageWithStore(subscription.Store{}, nil).(sourcesPage)
	if p.InputActive() {
		t.Fatal("sources page captured input before add mode")
	}
	page, _ := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	p = page.(sourcesPage)
	if !p.InputActive() {
		t.Fatal("a did not activate source input")
	}
	page, _ = p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	p = page.(sourcesPage)
	if p.InputActive() {
		t.Fatal("esc did not return to source list")
	}
}

func TestSourcesPageGeneratesNonConflictingNames(t *testing.T) {
	sources := []subscription.Source{
		{Name: "My Subscription"},
		{Name: "My Subscription (2)"},
	}
	if got := nextSourceName(sources); got != "My Subscription (3)" {
		t.Fatalf("next name = %q", got)
	}
}

func TestSourcesPageAllowsRename(t *testing.T) {
	store := subscription.Store{Path: filepath.Join(t.TempDir(), "state.json")}
	state := subscription.State{
		Version: subscription.CurrentStateVersion,
		Sources: []subscription.Source{{ID: "a", Name: "My Subscription", Type: subscription.SourceShare, Enabled: true}},
	}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	p := newSourcesPageWithStore(store, nil).(sourcesPage)
	p.state = state
	p.focused = false

	page, _ := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	p = page.(sourcesPage)
	p.renameBuffer = "Work"
	page, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = page.(sourcesPage)
	if cmd == nil {
		t.Fatal("rename did not save")
	}
	page, _ = p.Update(cmd())
	p = page.(sourcesPage)
	if p.state.Sources[0].Name != "Work" {
		t.Fatalf("renamed source = %+v", p.state.Sources[0])
	}
}

func TestSourcesViewContainsInputAndSubscriptionList(t *testing.T) {
	p := sourcesPage{
		state: subscription.State{
			Version: subscription.CurrentStateVersion,
			Sources: []subscription.Source{{ID: "a", Name: "My Subscription", Type: subscription.SourceURL}},
		},
	}
	view := p.View(80, 20)
	for _, want := range []string{"Name:", "Sub:", "Press a to add subscription", "Subscriptions", "My Subscription"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view does not contain %q: %q", want, view)
		}
	}
}
