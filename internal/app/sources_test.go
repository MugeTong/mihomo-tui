package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mihomo-tui/internal/runtimeconfig"
	"mihomo-tui/internal/subscription"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestSourcesPageShareInputAddsNodesWithoutSourceRecord(t *testing.T) {
	store := subscription.Store{Path: filepath.Join(t.TempDir(), "state.json")}
	p := newSourcesPageWithStore(store, nil).(sourcesPage)
	p.focused = true
	p.input = "trojan://test@jp.example.test:443#Tokyo"
	page, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = page.(sourcesPage)
	if cmd == nil {
		t.Fatal("enter did not import source")
	}
	page, _ = p.Update(cmd())
	p = page.(sourcesPage)
	if len(p.state.Sources) != 1 || p.state.Sources[0].Type != subscription.SourceURI || len(p.state.Nodes) != 1 {
		t.Fatalf("state = %+v", p.state)
	}
	generated, err := os.ReadFile(store.Path + ".yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "jp.example.test") {
		t.Fatalf("generated config = %s", generated)
	}
}

func TestSourcesInputLineFitsTerminalWidth(t *testing.T) {
	p := newSourcesPageWithStore(subscription.Store{}, nil).(sourcesPage)
	for _, width := range []int{48, 80, 120} {
		for index, line := range strings.Split(p.View(width, 20), "\n") {
			if got := lipgloss.Width(line); got >= width && index < 3 {
				t.Fatalf("width %d rendered line %d at %d cells", width, index+1, got)
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
	if view := p.View(80, 20); strings.Contains(view, "Press a to add subscription") {
		t.Fatalf("focused input still contains placeholder: %q", view)
	}
	page, _ = p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if page.(sourcesPage).InputActive() {
		t.Fatal("esc did not return to source list")
	}
}

func TestSourcesPageDoesNotReloadAfterInitialization(t *testing.T) {
	p := newSourcesPageWithStore(subscription.Store{}, nil).(sourcesPage)
	p.initialized = true
	if cmd := p.Init(); cmd != nil {
		t.Fatal("initialized Sources page reloaded on navigation")
	}
}

func TestSourcesInitialLoadUsesOfflineSnapshot(t *testing.T) {
	directory := t.TempDir()
	store := subscription.Store{Path: filepath.Join(directory, "state.json")}
	state := subscription.NewState()
	state.Sources = []subscription.Source{{Type: subscription.SourceURL, Location: "https://offline.example.test/token"}}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	p := newSourcesPageWithStore(store, nil).(sourcesPage)
	configData := []byte("proxies:\n  - {name: Cached, type: trojan, server: example.test, port: 443}\n")
	if _, err := runtimeconfig.Write(p.cfg.ConfigPath, configData); err != nil {
		t.Fatal(err)
	}
	page, _ := p.Update(p.Init()())
	p = page.(sourcesPage)
	if p.err != "" || len(p.state.Sources) != 1 || p.nodeCount != 1 {
		t.Fatalf("sources = %+v", p)
	}
}

func TestSourcesViewRedactsSubscriptionToken(t *testing.T) {
	p := sourcesPage{state: subscription.State{Version: subscription.CurrentStateVersion, Sources: []subscription.Source{{Type: subscription.SourceURL, Location: "https://sub.example.test/private-token"}}}}
	view := p.View(80, 20)
	for _, want := range []string{"Sub:", "Subscriptions", "sub.example.test", "Nodes"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view does not contain %q: %q", want, view)
		}
	}
	if strings.Contains(view, "private-token") {
		t.Fatalf("subscription token leaked in view: %q", view)
	}
}
