package app

import (
	"fmt"
	"strconv"
	"strings"

	"mihomo-tui/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

type settingsField int

const (
	fieldHTTPPort settingsField = iota
	fieldSOCKSPort
	fieldMixedPort
	fieldConfigPath
	fieldBinaryPath
)

const settingsFieldCount = int(fieldBinaryPath) + 1

type settingsPage struct {
	cfg     config.Config
	cursor  settingsField
	editing bool
	buffer  string
	status  string
	err     string
}

type settingsSavedMsg struct{ err error }

func newSettingsPage(cfg config.Config) Page {
	return settingsPage{cfg: cfg, status: "Ready"}
}

func (p settingsPage) Init() tea.Cmd { return nil }

func (p settingsPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if p.editing {
			return p.updateEditKey(msg)
		}
		return p.updateNavKey(msg)
	case settingsSavedMsg:
		if msg.err != nil {
			p.err = msg.err.Error()
			p.status = "Save failed"
			return p, nil
		}
		p.err = ""
		p.status = "Settings saved; restart to apply runtime changes"
	}
	return p, nil
}

func (p settingsPage) InputActive() bool { return p.editing }

func (p settingsPage) updateNavKey(key tea.KeyMsg) (Page, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if int(p.cursor)+1 < settingsFieldCount {
			p.cursor++
		}
	case "k", "up":
		if p.cursor > fieldHTTPPort {
			p.cursor--
		}
	case "enter":
		p.editing = true
		p.buffer = p.fieldValue(p.cursor)
		p.status = "Editing"
	case "s":
		return p, p.save()
	}
	return p, nil
}

func (p settingsPage) updateEditKey(key tea.KeyMsg) (Page, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc:
		p.editing = false
		p.buffer = ""
		p.status = "Edit canceled"
	case tea.KeyEnter:
		if err := p.applyBuffer(); err != nil {
			p.err = err.Error()
			p.status = "Invalid value"
			return p, nil
		}
		p.editing = false
		p.buffer = ""
		p.err = ""
		p.status = "Value updated; press s to save"
	case tea.KeyBackspace:
		p.buffer = trimLastRune(p.buffer)
	case tea.KeyRunes:
		p.buffer += string(key.Runes)
	}
	return p, nil
}

func (p settingsPage) View(_, _ int) string {
	lines := []string{
		headerStyle.Render("General"),
		p.renderField(fieldHTTPPort, "HTTP Port", strconv.Itoa(p.cfg.HTTPPort)),
		p.renderField(fieldSOCKSPort, "SOCKS Port", strconv.Itoa(p.cfg.SOCKSPort)),
		p.renderField(fieldMixedPort, "Mixed Port", strconv.Itoa(p.cfg.MixedPort)),
		"",
		headerStyle.Render("Advanced"),
		p.renderField(fieldConfigPath, "Config File", p.cfg.ConfigPath),
		p.renderField(fieldBinaryPath, "Mihomo Bin", p.cfg.BinaryPath),
		"",
		headerStyle.Render("About"),
		p.renderReadOnly("Platform", p.cfg.Platform),
		p.renderReadOnly("Scope", "Proxy only"),
		p.renderReadOnly("App Version", "dev"),
		p.renderReadOnly("License", "GPL-3.0-only"),
		p.renderReadOnly("Homepage", "github.com/MugeTong/mihomo-tui"),
		p.renderReadOnly("Thanks", "Mihomo contributors"),
		p.renderReadOnly("Inspired by", "Shadowrocket"),
	}
	return strings.Join(lines, "\n")
}

func (p settingsPage) Help() string {
	if p.editing {
		return "type value • enter apply • esc cancel"
	}
	return "up/down field • enter edit • s save"
}

func (p settingsPage) Message() string {
	if p.err != "" {
		return p.err
	}
	return p.status
}

func (p settingsPage) renderReadOnly(label, value string) string {
	return "  " + labelStyle.Render(fmt.Sprintf("%-14s", label+":")) + valueStyle.Render(value)
}

func (p settingsPage) renderField(field settingsField, label, value string) string {
	marker := "  "
	if p.cursor == field {
		marker = "> "
	}
	if p.editing && p.cursor == field {
		value = p.buffer + "_"
	}
	return sectionStyle.Render(marker) +
		labelStyle.Render(fmt.Sprintf("%-14s", label+":")) +
		valueStyle.Render(value)
}

func (p settingsPage) fieldValue(field settingsField) string {
	switch field {
	case fieldHTTPPort:
		return strconv.Itoa(p.cfg.HTTPPort)
	case fieldSOCKSPort:
		return strconv.Itoa(p.cfg.SOCKSPort)
	case fieldMixedPort:
		return strconv.Itoa(p.cfg.MixedPort)
	case fieldConfigPath:
		return p.cfg.ConfigPath
	case fieldBinaryPath:
		return p.cfg.BinaryPath
	default:
		return ""
	}
}

func (p *settingsPage) applyBuffer() error {
	value := strings.TrimSpace(p.buffer)
	if value == "" {
		return fmt.Errorf("value cannot be empty")
	}

	switch p.cursor {
	case fieldHTTPPort, fieldSOCKSPort, fieldMixedPort:
		port, err := parsePort(value)
		if err != nil {
			return err
		}
		switch p.cursor {
		case fieldHTTPPort:
			p.cfg.HTTPPort = port
		case fieldSOCKSPort:
			p.cfg.SOCKSPort = port
		case fieldMixedPort:
			p.cfg.MixedPort = port
		}
	case fieldConfigPath:
		p.cfg.ConfigPath = value
	case fieldBinaryPath:
		p.cfg.BinaryPath = value
	}
	return nil
}

func (p settingsPage) save() tea.Cmd {
	cfg := p.cfg
	return func() tea.Msg { return settingsSavedMsg{err: config.Save(cfg)} }
}

func parsePort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535")
	}
	return port, nil
}

func trimLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	return string(runes[:len(runes)-1])
}
