package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/core"
	"mihomo-tui/internal/mihomo"
	"mihomo-tui/internal/runtimeconfig"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type homePage struct {
	client      *mihomo.Client
	coreManager core.Manager
	cfg         config.Config
	proxyMode   string
	groups      []mihomo.ProxyGroup
	groupCursor int
	nodeCursor  int
	nodeOffset  int
	loading     bool
	status      string
	err         string
	snapshot    bool
}

type proxyGroupsLoadedMsg struct {
	groups   []mihomo.ProxyGroup
	snapshot bool
	err      error
}

type proxySelectedMsg struct {
	groupName string
	proxyName string
	err       error
}

type proxyDelayTestedMsg struct {
	proxyName string
	delay     int
	err       error
}

type proxyGroupDelayTestedMsg struct {
	delays map[string]int
	errs   map[string]error
}

type configReloadedMsg struct {
	err error
}

type coreStartedMsg struct {
	err error
}

type coreStoppedMsg struct {
	err error
}

func newHomePage(client *mihomo.Client, coreManager core.Manager, cfg config.Config) Page {
	return homePage{
		client:      client,
		coreManager: coreManager,
		cfg:         cfg,
		proxyMode:   "Rule",
	}
}

func (p homePage) Init() tea.Cmd {
	return p.loadProxyGroups()
}

func (p homePage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return p.updateKey(msg)
	case proxyGroupsLoadedMsg:
		p.loading = false
		if msg.err != nil {
			p.err = msg.err.Error()
			p.status = "Controller unavailable"
			return p, nil
		}
		p.err = ""
		p.status = "Controller connected"
		p.groups = visibleHomeGroups(msg.groups)
		p.snapshot = msg.snapshot
		if msg.snapshot {
			p.status = "Loaded generated config snapshot"
		}
		p.clampCursors()
	case proxySelectedMsg:
		p.loading = false
		if msg.err != nil {
			p.err = msg.err.Error()
			return p, nil
		}
		p.status = "Proxy selected"
		p.applySelection(msg.groupName, msg.proxyName)
		if p.useMock() {
			return p, nil
		}
		return p, p.loadProxyGroups()
	case proxyDelayTestedMsg:
		p.loading = false
		if msg.err != nil {
			p.err = msg.err.Error()
			return p, nil
		}
		p.status = fmt.Sprintf("%s delay: %dms", msg.proxyName, msg.delay)
		p.applyDelay(msg.proxyName, msg.delay)
	case proxyGroupDelayTestedMsg:
		p.loading = false
		if len(msg.errs) > 0 {
			p.err = firstDelayError(msg.errs).Error()
			p.status = "Group delay partially failed"
		} else {
			p.err = ""
			p.status = fmt.Sprintf("Tested %d nodes", len(msg.delays))
		}
		p.applyDelays(msg.delays)
	case configReloadedMsg:
		p.loading = false
		if msg.err != nil {
			p.err = msg.err.Error()
			return p, nil
		}
		p.status = "Config reloaded"
		return p, p.loadProxyGroups()
	case coreStartedMsg:
		p.loading = false
		if msg.err != nil {
			p.err = msg.err.Error()
			p.status = "Core start failed"
			return p, nil
		}
		p.err = ""
		p.status = "Core started"
		return p, p.loadProxyGroups()
	case coreStoppedMsg:
		p.loading = false
		if msg.err != nil {
			p.err = msg.err.Error()
			p.status = "Core stop failed"
			return p, nil
		}
		p.err = ""
		p.status = "Core stopped"
		if p.useManaged() {
			return p, p.loadProxyGroups()
		}
	}

	return p, nil
}

func (p homePage) updateKey(key tea.KeyMsg) (Page, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if p.nodeCursor < len(p.visibleNodes())-1 {
			p.nodeCursor++
		}
	case "k", "up":
		if p.nodeCursor > 0 {
			p.nodeCursor--
		}
	case "h", "left":
		if p.groupCursor > 0 {
			p.groupCursor--
			p.nodeCursor = 0
			p.nodeOffset = 0
		}
	case "l", "right":
		if p.groupCursor < len(p.groups)-1 {
			p.groupCursor++
			p.nodeCursor = 0
			p.nodeOffset = 0
		}
	case "enter":
		if p.snapshot {
			p.err = "Start Mihomo core before selecting a proxy"
			return p, nil
		}
		return p.startLoading("Selecting proxy"), p.selectCurrentProxy()
	case " ":
		return p.toggleCore()
	case "x":
		return p.startLoading("Stopping core"), p.stopCore()
	case "r":
		return p.startLoading("Refreshing"), p.loadProxyGroups()
	case "R":
		return p.startLoading("Reloading config"), p.reloadConfig()
	case "d":
		if p.snapshot {
			p.err = "Start Mihomo core before testing delay"
			return p, nil
		}
		return p.startLoading("Testing delay"), p.testCurrentProxyDelay()
	case "D":
		if p.snapshot {
			p.err = "Start Mihomo core before testing delay"
			return p, nil
		}
		return p.startLoading("Testing group delay"), p.testCurrentGroupDelay()
	}

	return p, nil
}

func (p homePage) View(width, height int) string {
	contentWidth := max(width-4, 10)

	statusSection := p.renderStatusSection(contentWidth)
	groupSection := p.renderGroupsSection(contentWidth)
	nodesHeight := p.nodesViewportHeight(height)
	p.ensureNodeVisible(nodesHeight)
	nodesSection := p.renderNodesSection(contentWidth, nodesHeight)

	return statusSection + "\n\n" + groupSection + "\n\n" + nodesSection
}

func (p homePage) Help() string {
	return "left/right group • up/down node • enter select • space core"
}

func (p homePage) renderStatusSection(width int) string {
	action := btnDisabledStyle.Render(strings.Title(p.cfg.SourceMode))
	if p.loading {
		action = btnDisabledStyle.Render("Loading")
	}

	title := titleStyle.Render("Status")
	gap := max(width-lipgloss.Width(title)-lipgloss.Width(action), 1)
	titleLine := title + strings.Repeat(" ", gap) + action

	status := offStyle.Render("Disconnected")
	if p.snapshot && len(p.groups) > 0 {
		status = valueStyle.Render("Config Ready")
	} else if p.err == "" && len(p.groups) > 0 {
		status = onStyle.Render("Connected")
	}

	body := labelStyle.Render("  State:     ") + status + "\n" +
		labelStyle.Render("  Core:      ") + valueStyle.Render(string(p.coreStatus())) + "\n" +
		labelStyle.Render("  Mode:      ") + valueStyle.Render(p.proxyMode) + "\n" +
		labelStyle.Render("  Source:    ") + valueStyle.Render(p.cfg.SourceMode)

	return titleLine + "\n\n" + body
}

func (p homePage) renderGroupsSection(width int) string {
	title := titleStyle.Render("Proxy Groups")
	if len(p.groups) == 0 {
		return title + "\n\n" + labelStyle.Render("  No proxy groups loaded")
	}

	groupLabels := make([]string, 0, len(p.groups))
	for i, group := range p.groups {
		text := group.Name
		if group.Now != "" {
			text += ":" + group.Now
		}
		if i == p.groupCursor {
			groupLabels = append(groupLabels, tabActiveStyle.Render(text))
		} else {
			groupLabels = append(groupLabels, tabStyle.Render(text))
		}
	}

	line := strings.Join(groupLabels, tabSepStyle.Render(" "))
	return title + "\n\n" + line
}

func (p homePage) renderNodesSection(width, height int) string {
	title := titleStyle.Render("Nodes")
	buttons := btnDisabledStyle.Render("Delay")
	gap := max(width-lipgloss.Width(title)-lipgloss.Width(buttons), 1)
	titleLine := title + strings.Repeat(" ", gap) + buttons

	var nodeList string
	nodes := p.visibleNodes()
	current := p.currentGroup()
	if len(nodes) == 0 {
		nodeList = labelStyle.Render("  No nodes loaded")
	} else {
		bodyHeight := max(height-2, 1)
		windowInfo := nodeWindow(bodyHeight, len(nodes), p.nodeCursor, p.nodeOffset)
		p.nodeOffset = windowInfo.start
		window := nodes[windowInfo.start:windowInfo.end]
		var lines []string
		for i, n := range window {
			absoluteIndex := windowInfo.start + i
			nodeMarker := "  "
			nameStyle := nodeInactiveStyle
			if absoluteIndex == p.nodeCursor {
				nodeMarker = "> "
			}
			if n.Name == current.Now {
				nameStyle = nodeActiveStyle
			}

			prefix := labelStyle.Render(fmt.Sprintf("  %s%d. ", nodeMarker, absoluteIndex+1))
			name := nameStyle.Render(n.Name)
			typ := labelStyle.Render(fmt.Sprintf(" [%s]", n.Type))
			delay := renderDelay(n.Delay)

			lines = append(lines, prefix+name+typ+"  "+delay)
		}
		if windowInfo.hasAbove {
			lines = append([]string{labelStyle.Render("  ...")}, lines...)
		}
		if windowInfo.hasBelow {
			lines = append(lines, labelStyle.Render("  ..."))
		}
		lines = append(lines, labelStyle.Render(fmt.Sprintf("  %d/%d", p.nodeCursor+1, len(nodes))))
		nodeList = strings.Join(lines, "\n")
	}

	return titleLine + "\n\n" + nodeList
}

func (p homePage) loadProxyGroups() tea.Cmd {
	return func() tea.Msg {
		if p.useMock() {
			return proxyGroupsLoadedMsg{groups: mockProxyGroups(), err: nil}
		}
		if p.useManaged() && p.coreStatus() != core.StatusRunning {
			groups, err := runtimeconfig.LoadProxyGroups(p.cfg.ConfigPath)
			return proxyGroupsLoadedMsg{groups: groups, snapshot: err == nil, err: err}
		}
		groups, err := p.client.ProxyGroups()
		return proxyGroupsLoadedMsg{groups: groups, err: err}
	}
}

func (p homePage) selectCurrentProxy() tea.Cmd {
	group := p.currentGroup()
	proxy := p.currentProxy()
	return func() tea.Msg {
		if group.Name == "" || proxy.Name == "" {
			return proxySelectedMsg{err: fmt.Errorf("no proxy selected")}
		}
		if p.useMock() {
			return proxySelectedMsg{groupName: group.Name, proxyName: proxy.Name, err: nil}
		}
		return proxySelectedMsg{groupName: group.Name, proxyName: proxy.Name, err: p.client.SelectProxy(group.Name, proxy.Name)}
	}
}

func (p homePage) testCurrentProxyDelay() tea.Cmd {
	proxy := p.currentProxy()
	return func() tea.Msg {
		if proxy.Name == "" {
			return proxyDelayTestedMsg{err: fmt.Errorf("no proxy selected")}
		}
		if p.useMock() {
			return proxyDelayTestedMsg{proxyName: proxy.Name, delay: mockDelay(proxy.Name), err: nil}
		}
		delay, err := p.client.TestProxyDelay(proxy.Name, 5*time.Second)
		return proxyDelayTestedMsg{proxyName: proxy.Name, delay: delay, err: err}
	}
}

func (p homePage) testCurrentGroupDelay() tea.Cmd {
	nodes := append([]mihomo.Proxy(nil), p.visibleNodes()...)
	return func() tea.Msg {
		if len(nodes) == 0 {
			return proxyGroupDelayTestedMsg{errs: map[string]error{"group": fmt.Errorf("no nodes selected")}}
		}

		delays := make(map[string]int, len(nodes))
		errs := make(map[string]error)
		if p.useMock() {
			for _, node := range nodes {
				delays[node.Name] = mockDelay(node.Name)
			}
			return proxyGroupDelayTestedMsg{delays: delays}
		}

		for _, node := range nodes {
			delay, err := p.client.TestProxyDelay(node.Name, 5*time.Second)
			if err != nil {
				errs[node.Name] = err
				continue
			}
			delays[node.Name] = delay
		}
		return proxyGroupDelayTestedMsg{delays: delays, errs: errs}
	}
}

func (p homePage) reloadConfig() tea.Cmd {
	return func() tea.Msg {
		if p.useMock() {
			return configReloadedMsg{err: nil}
		}
		if p.useManaged() && p.coreStatus() != core.StatusRunning {
			return configReloadedMsg{err: fmt.Errorf("core is %s", p.coreStatus())}
		}
		return configReloadedMsg{err: p.client.ReloadConfig("", false)}
	}
}

func (p homePage) currentGroup() mihomo.ProxyGroup {
	if len(p.groups) == 0 || p.groupCursor < 0 || p.groupCursor >= len(p.groups) {
		return mihomo.ProxyGroup{}
	}
	return p.groups[p.groupCursor]
}

func (p homePage) currentNodes() []mihomo.Proxy {
	group := p.currentGroup()
	return group.Proxies
}

func (p homePage) currentProxy() mihomo.Proxy {
	nodes := p.visibleNodes()
	if len(nodes) == 0 || p.nodeCursor < 0 || p.nodeCursor >= len(nodes) {
		return mihomo.Proxy{}
	}
	return nodes[p.nodeCursor]
}

func (p homePage) visibleNodes() []mihomo.Proxy {
	return p.currentNodes()
}

func (p homePage) clampCursors() {
	if p.groupCursor >= len(p.groups) {
		p.groupCursor = max(len(p.groups)-1, 0)
	}
	if p.nodeCursor >= len(p.visibleNodes()) {
		p.nodeCursor = max(len(p.visibleNodes())-1, 0)
	}
	if p.nodeCursor < 0 {
		p.nodeCursor = 0
	}
	if p.nodeOffset > p.nodeCursor {
		p.nodeOffset = p.nodeCursor
	}
	if p.nodeOffset < 0 {
		p.nodeOffset = 0
	}
}

func (p *homePage) ensureNodeVisible(height int) {
	nodes := p.visibleNodes()
	if len(nodes) == 0 {
		p.nodeCursor = 0
		p.nodeOffset = 0
		return
	}
	if p.nodeCursor >= len(nodes) {
		p.nodeCursor = len(nodes) - 1
	}
	if p.nodeCursor < 0 {
		p.nodeCursor = 0
	}

	bodyHeight := max(height-1, 1)
	p.nodeOffset = nodeWindow(bodyHeight, len(nodes), p.nodeCursor, p.nodeOffset).start
}

func (p homePage) nodesViewportHeight(contentHeight int) int {
	// Status is six lines, groups are three, and the two section gaps each
	// contribute one blank line.
	const rowsBeforeNodes = 6 + 3 + 2
	return max(contentHeight-rowsBeforeNodes, 1)
}

type nodeWindowInfo struct {
	start    int
	end      int
	hasAbove bool
	hasBelow bool
}

func nodeWindow(bodyHeight, totalNodes, cursor, previousStart int) nodeWindowInfo {
	if totalNodes <= 0 {
		return nodeWindowInfo{}
	}
	if bodyHeight < 2 {
		bodyHeight = 2
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= totalNodes {
		cursor = totalNodes - 1
	}

	best := nodeWindowInfo{start: 0, end: min(totalNodes, bodyHeight-1)}
	bestScore := -1
	for start := 0; start < totalNodes; start++ {
		hasAbove := start > 0
		available := bodyHeight - 1 // position line
		if hasAbove {
			available--
		}
		if available < 1 {
			available = 1
		}

		end := min(totalNodes, start+available)
		hasBelow := end < totalNodes
		if hasBelow {
			end = min(totalNodes, start+max(available-1, 1))
		}
		if cursor < start || cursor >= end {
			continue
		}

		score := end - start
		if hasAbove {
			score++
		}
		if hasBelow {
			score++
		}
		if score > bodyHeight-1 {
			continue
		}

		distance := abs(start - previousStart)
		if score > bestScore || (score == bestScore && distance < abs(best.start-previousStart)) {
			best = nodeWindowInfo{
				start:    start,
				end:      end,
				hasAbove: hasAbove,
				hasBelow: hasBelow,
			}
			bestScore = score
		}
	}

	return best
}

func (p homePage) startLoading(status string) homePage {
	p.loading = true
	p.status = status
	p.err = ""
	return p
}

func (p homePage) Message() string {
	if p.err != "" {
		return p.err
	}
	if p.status != "" {
		return p.status
	}
	return "Ready"
}

func (p *homePage) applyDelay(proxyName string, delay int) {
	for gi := range p.groups {
		for pi := range p.groups[gi].Proxies {
			if p.groups[gi].Proxies[pi].Name == proxyName {
				p.groups[gi].Proxies[pi].Delay = delay
			}
		}
	}
}

func (p *homePage) applyDelays(delays map[string]int) {
	for name, delay := range delays {
		p.applyDelay(name, delay)
	}
}

func (p *homePage) applySelection(groupName, proxyName string) {
	for i := range p.groups {
		if p.groups[i].Name == groupName {
			p.groups[i].Now = proxyName
		}
	}
}

func (p homePage) useMock() bool {
	return p.cfg.SourceMode == "mock"
}

func (p homePage) useManaged() bool {
	return p.cfg.SourceMode == "managed"
}

func (p homePage) coreStatus() core.Status {
	if p.coreManager == nil {
		return core.StatusUnavailable
	}
	return p.coreManager.Status()
}

func visibleHomeGroups(groups []mihomo.ProxyGroup) []mihomo.ProxyGroup {
	result := make([]mihomo.ProxyGroup, 0, len(groups))
	for _, group := range groups {
		switch group.Name {
		case "Final", "Direct", "DIRECT":
			continue
		}
		members := group.Proxies[:0]
		for _, proxy := range group.Proxies {
			if proxy.Name != "DIRECT" {
				members = append(members, proxy)
			}
		}
		group.Proxies = members
		all := group.All[:0]
		for _, name := range group.All {
			if name != "DIRECT" {
				all = append(all, name)
			}
		}
		group.All = all
		if group.Now == "DIRECT" {
			group.Now = ""
		}
		result = append(result, group)
	}
	return result
}

func (p homePage) toggleCore() (Page, tea.Cmd) {
	if !p.useManaged() {
		p.status = "Core controls are only available in managed mode"
		return p, nil
	}
	if p.coreStatus() == core.StatusRunning {
		return p.startLoading("Stopping core"), p.stopCore()
	}
	return p.startLoading("Starting core"), p.startCore()
}

func (p homePage) startCore() tea.Cmd {
	return func() tea.Msg {
		if p.coreManager == nil {
			return coreStartedMsg{err: fmt.Errorf("core manager unavailable")}
		}
		return coreStartedMsg{err: p.coreManager.Start(context.Background())}
	}
}

func (p homePage) stopCore() tea.Cmd {
	return func() tea.Msg {
		if p.coreManager == nil {
			return coreStoppedMsg{err: fmt.Errorf("core manager unavailable")}
		}
		return coreStoppedMsg{err: p.coreManager.Stop()}
	}
}

func firstDelayError(errs map[string]error) error {
	for name, err := range errs {
		return fmt.Errorf("%s: %w", name, err)
	}
	return fmt.Errorf("delay test failed")
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func renderDelay(ms int) string {
	if ms < 0 {
		return nodeDelayNone.Render("--")
	}
	text := fmt.Sprintf("%dms", ms)
	switch {
	case ms < 200:
		return nodeDelayGood.Render(text)
	case ms < 500:
		return nodeDelayMed.Render(text)
	default:
		return nodeDelayBad.Render(text)
	}
}
