package app

import (
	"path/filepath"
	"strings"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/core"
	"mihomo-tui/internal/mihomo"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	width       int
	height      int
	cursor      int
	pages       []pageEntry
	coreManager core.Manager
	stopping    bool
	quitError   string
}

type coreStoppedAndQuitMsg struct{ err error }

func newModel(client *mihomo.Client, coreManager core.Manager, cfg config.Config) model {
	return model{
		pages:       newPages(client, coreManager, cfg),
		coreManager: coreManager,
	}
}

func (m model) Init() tea.Cmd {
	if len(m.pages) == 0 {
		return nil
	}
	return m.pages[m.cursor].page.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.inputActive() && msg.String() != "ctrl+c" {
			break
		}
		switch msg.String() {
		case "ctrl+c":
			if m.stopping {
				return m, nil
			}
			m.stopping = true
			return m, m.stopCoreAndQuit()
		case "q":
			return m, tea.Quit
		case "tab":
			m.nextPage()
			return m, m.currentPage().Init()
		case "shift+tab":
			m.prevPage()
			return m, m.currentPage().Init()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case coreStoppedAndQuitMsg:
		m.stopping = false
		if msg.err != nil {
			m.quitError = "Could not stop managed core: " + msg.err.Error()
			return m, nil
		}
		return m, tea.Quit
	}

	if len(m.pages) == 0 {
		return m, nil
	}

	page, cmd := m.currentPage().Update(msg)
	m.pages[m.cursor].page = page
	return m, cmd
}

func (m model) inputActive() bool {
	if len(m.pages) == 0 {
		return false
	}
	provider, ok := m.currentPage().(InputProvider)
	return ok && provider.InputActive()
}

func (m model) stopCoreAndQuit() tea.Cmd {
	return func() tea.Msg {
		return coreStoppedAndQuitMsg{err: m.coreManager.Stop()}
	}
}

func (m *model) nextPage() {
	if len(m.pages) == 0 {
		return
	}
	m.cursor = (m.cursor + 1) % len(m.pages)
}

func (m *model) prevPage() {
	if len(m.pages) == 0 {
		return
	}
	m.cursor--
	if m.cursor < 0 {
		m.cursor = len(m.pages) - 1
	}
}

func (m model) currentPage() Page {
	return m.pages[m.cursor].page
}

func (m model) View() string {
	width := m.width
	height := m.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}

	innerWidth := width - 2
	pageWidth := max(innerWidth-4, 1) // contentStyle has two cells of padding on each side
	contentHeight := max(height-8, 5)

	title := titleStyle.Render(" Mihomo TUI")
	helpWidth := max(innerWidth-lipgloss.Width(title)-1, 0)
	helpText := truncateCells("tab switch • q leave • ^C stop", helpWidth)
	help := helpStyle.Render(helpText)
	rightSide := help
	gap := max(innerWidth-lipgloss.Width(title)-lipgloss.Width(rightSide), 0)
	titleBar := title + strings.Repeat(" ", gap) + rightSide

	tabBar := m.renderTabBar()
	separator := tabBarBorderStyle.Render(strings.Repeat("-", innerWidth))
	content := contentStyle.Width(pageWidth).Height(contentHeight).Render(
		m.currentPage().View(pageWidth, contentHeight),
	)
	message := m.renderMessageBar(innerWidth)

	return frameStyle.Width(innerWidth).Render(
		titleBar + "\n" + tabBar + "\n" + separator + "\n" + content + "\n" + message,
	)
}

func (m model) renderTabBar() string {
	tabs := make([]string, 0, len(m.pages))
	for i, page := range m.pages {
		style := tabStyle
		if i == m.cursor {
			style = tabActiveStyle
		}
		tabs = append(tabs, style.Render(page.label))
	}
	return " " + strings.Join(tabs, tabSepStyle.Render(" | "))
}

func (m model) renderMessageBar(width int) string {
	message := "Ready"
	if m.stopping {
		message = "Stopping managed core…"
	} else if m.quitError != "" {
		message = m.quitError
	} else if provider, ok := m.currentPage().(MessageProvider); ok {
		if pageMessage := strings.TrimSpace(provider.Message()); pageMessage != "" {
			message = pageMessage
		}
	}

	prefix := statusPrefixStyle.Render("Message: ")
	available := max(width-lipgloss.Width(prefix), 0)
	if lipgloss.Width(message) > available {
		message = truncateCells(message, available)
	}

	return messageBarStyle.Width(width).Render(prefix + statusMessageStyle.Render(message))
}

func StartTUI() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client, err := mihomo.NewClient(config.ControllerURL, cfg.Secret)
	if err != nil {
		return err
	}

	settingsPath, err := config.Path()
	if err != nil {
		return err
	}
	appDir := filepath.Dir(settingsPath)
	coreManager := core.NewProcessManager(core.ProcessOptions{
		BinaryPath:        cfg.BinaryPath,
		ConfigPath:        cfg.ConfigPath,
		DataDir:           filepath.Join(appDir, "mihomo"),
		PIDPath:           filepath.Join(appDir, "mihomo.pid"),
		LogPath:           filepath.Join(appDir, "mihomo.log"),
		ControllerAddress: config.ControllerAddress,
	})
	p := tea.NewProgram(newModel(client, coreManager, cfg), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
