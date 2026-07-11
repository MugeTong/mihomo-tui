package app

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/runtimeconfig"
	"mihomo-tui/internal/subscription"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sourcesPage struct {
	store       subscription.Store
	cfg         config.Config
	pathErr     error
	state       subscription.State
	input       string
	focused     bool
	loading     bool
	initialized bool
	status      string
	err         string
}

type sourcesLoadedMsg struct {
	state  subscription.State
	report subscription.ReconcileReport
	err    error
}

type sourceAddedMsg struct {
	state      subscription.State
	nodes      int
	duplicates int
	renamed    int
	issues     int
	err        error
}

func newSourcesPage(cfg config.Config) Page {
	path, err := subscription.DefaultStatePath()
	return newSourcesPageWithConfig(subscription.Store{Path: path}, cfg, err)
}

func newSourcesPageWithStore(store subscription.Store, pathErr error) Page {
	cfg := config.Default()
	cfg.ConfigPath = store.Path + ".yaml"
	return newSourcesPageWithConfig(store, cfg, pathErr)
}

func newSourcesPageWithConfig(store subscription.Store, cfg config.Config, pathErr error) Page {
	return sourcesPage{store: store, cfg: cfg, pathErr: pathErr, state: subscription.NewState(), status: "Press a to add a subscription"}
}

func (p sourcesPage) Init() tea.Cmd {
	if p.initialized {
		return nil
	}
	if p.pathErr != nil {
		err := p.pathErr
		return func() tea.Msg { return sourcesLoadedMsg{err: err} }
	}
	return p.load()
}

func (p sourcesPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return p.updateKey(msg)
	case sourcesLoadedMsg:
		p.loading = false
		p.initialized = true
		if msg.err != nil {
			p.err = msg.err.Error()
			return p, nil
		}
		p.state = msg.state
		if len(msg.report.Issues) > 0 {
			p.status = fmt.Sprintf("Loaded with %d repaired state issues", len(msg.report.Issues))
		}
	case sourceAddedMsg:
		p.loading = false
		p.initialized = true
		if msg.state.Version != 0 {
			p.state = msg.state
		}
		if msg.err != nil {
			p.err = msg.err.Error()
			p.status = "Add failed"
			return p, nil
		}
		p.input = ""
		p.focused = false
		p.err = ""
		p.status = fmt.Sprintf("Added %d nodes, %d duplicate, %d renamed, %d skipped", msg.nodes, msg.duplicates, msg.renamed, msg.issues)
	}
	return p, nil
}

func (p sourcesPage) updateKey(key tea.KeyMsg) (Page, tea.Cmd) {
	if p.loading {
		return p, nil
	}
	if p.focused {
		switch key.String() {
		case "esc":
			p.focused = false
			p.input = ""
			p.status = "Input unfocused"
		case "enter", "ctrl+s":
			if strings.TrimSpace(p.input) == "" {
				p.err = "paste a subscription URL or share link"
				return p, nil
			}
			p.loading = true
			p.err = ""
			p.status = "Importing nodes"
			return p, p.addSource()
		case "backspace":
			p.input = trimLastRune(p.input)
		default:
			if key.Type == tea.KeyRunes {
				p.input += string(key.Runes)
			}
		}
		return p, nil
	}
	switch key.String() {
	case "a", "i":
		p.focused = true
		p.err = ""
		p.status = "Paste a subscription URL or share link"
	case "r":
		p.loading = true
		p.status = "Refreshing subscriptions"
		return p, p.refresh()
	}
	return p, nil
}

func (p sourcesPage) View(width, _ int) string {
	input := "Press a to add subscription"
	if p.focused {
		input = p.input + "_"
	}
	inputWidth := min(64, max(width-lipgloss.Width("  Sub:  []")-1, 1))
	lines := []string{
		titleStyle.Render("Sources"),
		labelStyle.Render("  Sub:  ") + valueStyle.Render("["+padOrTruncate(input, inputWidth)+"]"),
		labelStyle.Render("  Press a to add • Enter confirm • Esc cancel"),
		"",
		headerStyle.Render("Subscriptions"),
	}
	if len(p.state.Sources) == 0 {
		lines = append(lines, labelStyle.Render("  No subscription URLs added"))
	} else {
		for _, source := range p.state.Sources {
			lines = append(lines, valueStyle.Render("  "+padOrTruncate(displaySource(source.Location), max(width-3, 1))))
		}
	}
	lines = append(lines, "", headerStyle.Render("Nodes"), valueStyle.Render(fmt.Sprintf("  %d managed nodes", len(p.state.Nodes))))
	return strings.Join(lines, "\n")
}

func (p sourcesPage) Help() string {
	if p.focused {
		return "paste source • enter import • esc list"
	}
	return "a add • r refresh"
}

func (p sourcesPage) Message() string {
	if p.err != "" {
		return p.err
	}
	return p.status
}
func (p sourcesPage) InputActive() bool { return p.focused }

func (p sourcesPage) addSource() tea.Cmd {
	input := strings.TrimSpace(p.input)
	state := cloneSubscriptionState(p.state)
	store, cfg := p.store, p.cfg
	return func() tea.Msg {
		source := subscription.Source{Type: subscription.SourceURI, Location: input, UpdatedAt: time.Now()}
		if parsed, parseErr := url.Parse(input); parseErr == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
			source.Type = subscription.SourceURL
		}
		state.AddSource(source)
		nodes, report, err := subscription.Rebuild(state.Sources, subscription.DefaultFetcher())
		if err != nil {
			return sourceAddedMsg{err: err}
		}
		state.Nodes = nodes
		if err := saveRuntime(store, cfg, state); err != nil {
			return sourceAddedMsg{state: state, err: err}
		}
		return sourceAddedMsg{state: state, nodes: report.Added, duplicates: report.Duplicates, renamed: report.Renamed, issues: report.Skipped}
	}
}

func (p sourcesPage) refresh() tea.Cmd {
	state := cloneSubscriptionState(p.state)
	store, cfg := p.store, p.cfg
	return func() tea.Msg {
		nodes, report, err := subscription.Rebuild(state.Sources, subscription.DefaultFetcher())
		if err != nil {
			return sourceAddedMsg{state: state, err: err}
		}
		state.Nodes = nodes
		for index := range state.Sources {
			state.Sources[index].UpdatedAt = time.Now()
		}
		if err := saveRuntime(store, cfg, state); err != nil {
			return sourceAddedMsg{state: state, err: err}
		}
		return sourceAddedMsg{state: state, nodes: report.Added, duplicates: report.Duplicates, renamed: report.Renamed, issues: report.Skipped}
	}
}

func (p sourcesPage) load() tea.Cmd {
	store, cfg := p.store, p.cfg
	return func() tea.Msg {
		state, report, err := store.Load()
		if err == nil && len(report.Issues) > 0 {
			err = store.Save(state)
		}
		if err == nil && len(state.Sources) > 0 {
			state.Nodes, _, err = subscription.Rebuild(state.Sources, subscription.DefaultFetcher())
		}
		if err == nil {
			err = writeRuntime(cfg, state)
		}
		return sourcesLoadedMsg{state: state, report: report, err: err}
	}
}

func saveRuntime(store subscription.Store, cfg config.Config, state subscription.State) error {
	generated, err := runtimeconfig.Generate(cfg, state)
	if err != nil {
		return err
	}
	if err := store.Save(state); err != nil {
		return err
	}
	_, err = runtimeconfig.Write(cfg.ConfigPath, generated)
	return err
}

func writeRuntime(cfg config.Config, state subscription.State) error {
	generated, err := runtimeconfig.Generate(cfg, state)
	if err != nil {
		return err
	}
	_, err = runtimeconfig.Write(cfg.ConfigPath, generated)
	return err
}

func cloneSubscriptionState(state subscription.State) subscription.State {
	state.Sources = append([]subscription.Source(nil), state.Sources...)
	state.Nodes = append([]subscription.Node(nil), state.Nodes...)
	state.Selections = append([]subscription.PolicySelection(nil), state.Selections...)
	return state
}

func displaySource(location string) string {
	parsed, err := url.Parse(location)
	if err != nil || parsed.Host == "" {
		return "subscription URL"
	}
	return parsed.Scheme + "://" + parsed.Host + "/…"
}
