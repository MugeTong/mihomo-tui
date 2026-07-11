package app

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))

	tabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("241"))

	tabActiveStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Background(lipgloss.Color("63")).
			Foreground(lipgloss.Color("230")).
			Bold(true)

	tabSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	tabBarBorderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("63"))

	contentStyle = lipgloss.NewStyle().
			Padding(0, 2)

	frameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true)

	onStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("120")).
		Bold(true)

	offStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	messageBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238"))

	statusPrefixStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			MarginBottom(1)

	sectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	sectionCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	sectionSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	nodeActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("120"))

	nodeInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	nodeDelayGood = lipgloss.NewStyle().
			Foreground(lipgloss.Color("120"))

	nodeDelayMed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	nodeDelayBad = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	nodeDelayNone = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	btnDisabledStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Background(lipgloss.Color("238")).
				Foreground(lipgloss.Color("241"))

	btnStartStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Background(lipgloss.Color("120")).
			Foreground(lipgloss.Color("0")).
			Bold(true)

	btnStopStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Background(lipgloss.Color("196")).
			Foreground(lipgloss.Color("230")).
			Bold(true)
)
