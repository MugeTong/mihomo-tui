package app

import (
	"mihomo-tui/internal/config"
	"mihomo-tui/internal/core"
	"mihomo-tui/internal/mihomo"

	tea "github.com/charmbracelet/bubbletea"
)

type pageID int

const (
	pageHome pageID = iota
	pageRules
	pageTraffic
	pageSettings
)

type Page interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Page, tea.Cmd)
	View(width, height int) string
	Help() string
}

type MessageProvider interface {
	Message() string
}

// InputProvider tells the root model that the current page is consuming text.
// While active, printable keys (including q) belong to the page.
type InputProvider interface {
	InputActive() bool
}

type pageEntry struct {
	id    pageID
	label string
	page  Page
}

func newPages(client *mihomo.Client, coreManager core.Manager, cfg config.Config) []pageEntry {
	return []pageEntry{
		{id: pageHome, label: "Home", page: newHomePage(client, coreManager, cfg)},
		{id: pageRules, label: "Rules", page: newRulesPage()},
		{id: pageTraffic, label: "Traffic", page: newTrafficPage()},
		{id: pageSettings, label: "Settings", page: newSettingsPage(cfg)},
	}
}
