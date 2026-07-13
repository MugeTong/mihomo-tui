package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/core"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type recordingManager struct {
	stopCalls int
	stopErr   error
	status    core.Status
}

func TestRulesPageDoesNotOverflowRootFrame(t *testing.T) {
	manager := &recordingManager{}
	m := newModel(nil, manager, config.Default())
	m.cursor = int(pageRules)
	m.width = 120
	m.height = 30

	for lineNumber, line := range strings.Split(m.View(), "\n") {
		if got := lipgloss.Width(line); got > m.width {
			t.Fatalf("line %d width = %d, terminal width = %d", lineNumber+1, got, m.width)
		}
	}
}

func TestHeaderShowsCurrentPageHelp(t *testing.T) {
	m := newModel(nil, &recordingManager{}, config.Default())
	m.width = 160
	m.height = 30
	view := m.View()
	for _, want := range []string{"space core", "d delay", "tab switch"} {
		if !strings.Contains(view, want) {
			t.Fatalf("header does not contain %q: %q", want, view)
		}
	}
}

func TestTabBelongsToActivePageInput(t *testing.T) {
	manager := &recordingManager{}
	m := newModel(nil, manager, config.Default())
	m.cursor = int(pageSources)
	sources := m.currentPage().(sourcesPage)
	sources.focused = true
	m.pages[m.cursor].page = sources

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	got := updated.(model)
	if got.cursor != int(pageSources) {
		t.Fatal("tab switched page while source input was active")
	}
}

func (m *recordingManager) Status() core.Status {
	if m.status == "" {
		return core.StatusRunning
	}
	return m.status
}
func (m *recordingManager) Start(context.Context) error { return nil }
func (m *recordingManager) Stop() error {
	m.stopCalls++
	return m.stopErr
}

func TestQLeavesTUIWithoutStoppingCore(t *testing.T) {
	manager := &recordingManager{}
	m := newModel(nil, manager, config.Default())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q did not request TUI quit")
	}
	if manager.stopCalls != 0 {
		t.Fatalf("Stop calls = %d, want 0", manager.stopCalls)
	}
}

func TestQIsTextWhileRulesSearchIsActive(t *testing.T) {
	manager := &recordingManager{}
	m := newModel(nil, manager, config.Default())
	m.cursor = int(pageRules)
	rules := m.currentPage().(rulesPage)
	rules.searching = true
	m.pages[m.cursor].page = rules

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatal("q unexpectedly returned a command while searching")
	}
	got := updated.(model).currentPage().(rulesPage).filter
	if got != "q" {
		t.Fatalf("filter = %q, want q", got)
	}
}

func TestCtrlCStopsCoreBeforeQuitting(t *testing.T) {
	manager := &recordingManager{}
	m := newModel(nil, manager, config.Default())
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	stopResult := cmd()
	if manager.stopCalls != 1 {
		t.Fatalf("Stop calls = %d, want 1", manager.stopCalls)
	}
	_, quitCmd := updated.(model).Update(stopResult)
	if _, ok := quitCmd().(tea.QuitMsg); !ok {
		t.Fatal("successful stop did not request TUI quit")
	}
}

func TestCtrlCStaysOpenWhenCoreStopFails(t *testing.T) {
	manager := &recordingManager{stopErr: errors.New("busy")}
	m := newModel(nil, manager, config.Default())
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result, quitCmd := updated.(model).Update(cmd())
	if quitCmd != nil {
		t.Fatal("failed stop unexpectedly requested TUI quit")
	}
	if result.(model).quitError == "" {
		t.Fatal("failed stop was not exposed to the user")
	}
}
