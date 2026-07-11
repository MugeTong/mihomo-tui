package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/runtimeconfig"
	"mihomo-tui/internal/subscription"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sourcesPage struct {
	store        subscription.Store
	cfg          config.Config
	runtimePath  string
	pathErr      error
	state        subscription.State
	cursor       int
	nameInput    string
	input        string
	inputField   int
	focused      bool
	renaming     bool
	renameBuffer string
	loading      bool
	status       string
	err          string
}

type sourcesLoadedMsg struct {
	state  subscription.State
	report subscription.ReconcileReport
	path   string
	err    error
}

type sourceAddedMsg struct {
	state  subscription.State
	source subscription.Source
	nodes  int
	issues int
	path   string
	err    error
}

type sourceRenamedMsg struct {
	state subscription.State
	err   error
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
	return sourcesPage{store: store, cfg: cfg, runtimePath: cfg.ConfigPath, pathErr: pathErr, state: subscription.NewState(), status: "Press a to add a subscription"}
}

func (p sourcesPage) Init() tea.Cmd {
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
		if msg.err != nil {
			p.err = msg.err.Error()
			return p, nil
		}
		p.state = msg.state
		p.runtimePath = msg.path
		p.clampCursor()
		if len(msg.report.Issues) > 0 {
			p.status = fmt.Sprintf("Loaded with %d repaired state issues", len(msg.report.Issues))
		}
	case sourceAddedMsg:
		p.loading = false
		if msg.err != nil {
			if msg.state.Version != 0 {
				p.state = msg.state
			}
			p.err = msg.err.Error()
			p.status = "Add failed"
			return p, nil
		}
		p.state = msg.state
		p.runtimePath = msg.path
		p.nameInput = ""
		p.input = ""
		p.focused = false
		p.err = ""
		p.cursor = len(p.state.Sources) - 1
		p.status = fmt.Sprintf("Added %s: %d nodes, %d skipped", msg.source.Name, msg.nodes, msg.issues)
	case sourceRenamedMsg:
		p.loading = false
		if msg.err != nil {
			p.err = msg.err.Error()
			return p, nil
		}
		p.state = msg.state
		p.renaming = false
		p.renameBuffer = ""
		p.err = ""
		p.status = "Subscription renamed"
	}
	return p, nil
}

func (p sourcesPage) updateKey(key tea.KeyMsg) (Page, tea.Cmd) {
	if p.loading {
		return p, nil
	}
	if p.renaming {
		return p.updateRenameKey(key)
	}
	if p.focused {
		switch key.String() {
		case "esc":
			p.focused = false
			p.nameInput = ""
			p.input = ""
			p.status = "Input unfocused"
		case "tab", "shift+tab", "left", "right":
			p.inputField = 1 - p.inputField
		case "enter", "ctrl+s":
			if p.inputField == 0 && key.String() == "enter" {
				p.inputField = 1
				return p, nil
			}
			if strings.TrimSpace(p.input) == "" {
				p.err = "paste a subscription URL or share link"
				return p, nil
			}
			p.loading = true
			p.err = ""
			p.status = "Adding subscription"
			return p, p.addSource()
		case "backspace":
			if p.inputField == 0 {
				p.nameInput = trimLastRune(p.nameInput)
			} else {
				p.input = trimLastRune(p.input)
			}
		default:
			if key.Type == tea.KeyRunes {
				if p.inputField == 0 {
					p.nameInput += string(key.Runes)
				} else {
					p.input += string(key.Runes)
				}
			}
		}
		return p, nil
	}

	switch key.String() {
	case "a", "i":
		p.focused = true
		p.inputField = 0
		p.err = ""
		p.status = "Paste a subscription URL or share link"
	case "j", "down":
		if p.cursor < len(p.state.Sources)-1 {
			p.cursor++
		}
	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
		}
	case "e":
		if len(p.state.Sources) > 0 {
			p.renaming = true
			p.renameBuffer = p.state.Sources[p.cursor].Name
			p.status = "Rename subscription"
		}
	case "r":
		p.loading = true
		return p, p.load()
	}
	return p, nil
}

func (p sourcesPage) updateRenameKey(key tea.KeyMsg) (Page, tea.Cmd) {
	switch key.String() {
	case "esc":
		p.renaming = false
		p.renameBuffer = ""
		p.status = "Rename canceled"
	case "enter":
		name := strings.TrimSpace(p.renameBuffer)
		if name == "" {
			p.err = "subscription name cannot be empty"
			return p, nil
		}
		next := cloneSubscriptionState(p.state)
		next.Sources[p.cursor].Name = uniqueSourceName(name, next.Sources, p.cursor)
		p.loading = true
		return p, p.saveRenamed(next)
	case "backspace":
		p.renameBuffer = trimLastRune(p.renameBuffer)
	default:
		if key.Type == tea.KeyRunes {
			p.renameBuffer += string(key.Runes)
		}
	}
	return p, nil
}

func (p sourcesPage) View(width, _ int) string {
	name := p.nameInput
	input := p.input
	if p.focused {
		if name == "" {
			name = "Name"
		}
		if input == "" {
			input = "Subscription URL or share link"
		}
		if p.inputField == 0 {
			name += "_"
		} else {
			input += "_"
		}
	} else {
		name = "Name"
		input = "Press a to add subscription"
	}
	nameWidth := min(48, max(width-lipgloss.Width("  Name: []")-1, 1))
	subWidth := min(48, max(width-lipgloss.Width("  Sub:  []")-1, 1))
	pathWidth := max(width-lipgloss.Width("  Config: ")-1, 1)
	lines := []string{
		titleStyle.Render("Sources"),
		labelStyle.Render("  State: ") + valueStyle.Render(padOrTruncate(p.store.Path, pathWidth+1)),
		labelStyle.Render("  Config: ") + valueStyle.Render(padOrTruncate(p.runtimePath, pathWidth)),
		labelStyle.Render("  Name: ") + valueStyle.Render("["+padOrTruncate(name, nameWidth)+"]"),
		labelStyle.Render("  Sub:  ") + valueStyle.Render("["+padOrTruncate(input, subWidth)+"]"),
		labelStyle.Render("  Press a to add • Enter confirm • Esc cancel"),
		"",
		headerStyle.Render("Subscriptions"),
	}
	if len(p.state.Sources) == 0 {
		lines = append(lines, labelStyle.Render("  No subscriptions added"))
		return strings.Join(lines, "\n")
	}

	listNameWidth := max(width-27, 12)
	lines = append(lines, labelStyle.Render("  "+padOrTruncate("NAME", listNameWidth)+" "+padOrTruncate("TYPE", 8)+" NODES"))
	counts := p.sourceNodeCounts()
	for index, source := range p.state.Sources {
		marker := "  "
		if index == p.cursor && !p.focused {
			marker = "> "
		}
		name := source.Name
		if p.renaming && index == p.cursor {
			name = p.renameBuffer + "_"
		}
		lines = append(lines, sectionStyle.Render(marker)+
			valueStyle.Render(padOrTruncate(name, listNameWidth))+" "+
			labelStyle.Render(padOrTruncate(strings.ToUpper(string(source.Type)), 8))+" "+
			valueStyle.Render(fmt.Sprintf("%d", counts[source.ID])))
	}
	return strings.Join(lines, "\n")
}

func (p sourcesPage) Help() string {
	if p.renaming {
		return "type name • enter save • esc cancel"
	}
	if p.focused {
		return "paste source • enter add • esc list"
	}
	return "up/down source • a add • e rename • r refresh"
}

func (p sourcesPage) Message() string {
	if p.err != "" {
		return p.err
	}
	return p.status
}

func (p sourcesPage) InputActive() bool { return p.focused || p.renaming }

func (p sourcesPage) addSource() tea.Cmd {
	input := strings.TrimSpace(p.input)
	requestedName := strings.TrimSpace(p.nameInput)
	state := cloneSubscriptionState(p.state)
	store := p.store
	cfg := p.cfg
	return func() tea.Msg {
		sourceID, err := randomSourceID()
		if err != nil {
			return sourceAddedMsg{err: err}
		}
		sourceType := subscription.SourceShare
		location := ""
		var result subscription.ImportResult
		if parsed, parseErr := url.Parse(input); parseErr == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
			sourceType = subscription.SourceURL
			location = input
			result, err = subscription.DefaultFetcher().Import(input, sourceID)
		} else {
			result, err = subscription.ImportShareLinks([]byte(input), sourceID)
		}
		if err != nil {
			return sourceAddedMsg{err: err}
		}
		name := requestedName
		if name == "" {
			name = "My Subscription"
		}
		source := subscription.Source{
			ID: sourceID, Name: uniqueSourceName(name, state.Sources, -1), Type: sourceType,
			Location: location, Enabled: true,
		}
		if err := state.AddImport(source, result); err != nil {
			return sourceAddedMsg{err: err}
		}
		generated, err := runtimeconfig.Generate(cfg, state)
		if err != nil {
			return sourceAddedMsg{err: err}
		}
		if err := store.Save(state); err != nil {
			return sourceAddedMsg{err: err}
		}
		path, err := runtimeconfig.Write(cfg.ConfigPath, generated)
		if err != nil {
			return sourceAddedMsg{state: state, err: err}
		}
		return sourceAddedMsg{state: state, source: source, nodes: len(result.Nodes), issues: len(result.Issues), path: path}
	}
}

func (p sourcesPage) saveRenamed(state subscription.State) tea.Cmd {
	store := p.store
	return func() tea.Msg { return sourceRenamedMsg{state: state, err: store.Save(state)} }
}

func (p sourcesPage) load() tea.Cmd {
	store := p.store
	cfg := p.cfg
	return func() tea.Msg {
		state, report, err := store.Load()
		if err != nil {
			return sourcesLoadedMsg{err: err}
		}
		generated, err := runtimeconfig.Generate(cfg, state)
		if err != nil {
			return sourcesLoadedMsg{state: state, report: report, err: err}
		}
		path, err := runtimeconfig.Write(cfg.ConfigPath, generated)
		return sourcesLoadedMsg{state: state, report: report, path: path, err: err}
	}
}

func (p sourcesPage) sourceNodeCounts() map[string]int {
	counts := make(map[string]int, len(p.state.Sources))
	for _, link := range p.state.Links {
		counts[link.SourceID]++
	}
	return counts
}

func (p *sourcesPage) clampCursor() {
	if len(p.state.Sources) == 0 {
		p.cursor = 0
	} else if p.cursor >= len(p.state.Sources) {
		p.cursor = len(p.state.Sources) - 1
	}
}

func cloneSubscriptionState(state subscription.State) subscription.State {
	state.Sources = append([]subscription.Source(nil), state.Sources...)
	state.Nodes = append([]subscription.Node(nil), state.Nodes...)
	state.Links = append([]subscription.SourceNode(nil), state.Links...)
	state.Selections = append([]subscription.PolicySelection(nil), state.Selections...)
	return state
}

func nextSourceName(sources []subscription.Source) string {
	return uniqueSourceName("My Subscription", sources, -1)
}

func uniqueSourceName(base string, sources []subscription.Source, skip int) string {
	used := make(map[string]struct{}, len(sources))
	for index, source := range sources {
		if index != skip {
			used[source.Name] = struct{}{}
		}
	}
	if _, exists := used[base]; !exists {
		return base
	}
	for number := 2; ; number++ {
		candidate := fmt.Sprintf("%s (%d)", base, number)
		if _, exists := used[candidate]; !exists {
			return candidate
		}
	}
}

func randomSourceID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate source ID: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
}
