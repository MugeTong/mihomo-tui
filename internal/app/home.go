package app

import (
	"fmt"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/core"
	"mihomo-tui/internal/mihomo"

	tea "github.com/charmbracelet/bubbletea"
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
	connected   bool
	connecting  bool
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
	connecting := coreManager != nil && coreManager.Status() == core.StatusRunning
	return homePage{
		client:      client,
		coreManager: coreManager,
		cfg:         cfg,
		proxyMode:   "Rule",
		connecting:  connecting,
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
		p.connecting = false
		if msg.err != nil {
			p.connected = false
			p.err = msg.err.Error()
			p.status = "Controller unavailable"
			return p, nil
		}
		p.err = ""
		p.status = "Controller connected"
		p.groups = visibleHomeGroups(msg.groups)
		p.snapshot = msg.snapshot
		p.connected = !msg.snapshot
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
			p.connecting = false
			p.err = msg.err.Error()
			p.status = "Core start failed"
			return p, nil
		}
		p.err = ""
		p.status = "Connecting to controller"
		p.connecting = true
		return p, p.loadProxyGroups()
	case coreStoppedMsg:
		p.loading = false
		p.connecting = false
		if msg.err != nil {
			p.err = msg.err.Error()
			p.status = "Core stop failed"
			return p, nil
		}
		p.err = ""
		p.status = "Core stopped"
		return p, p.loadProxyGroups()
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
		case "Final", "Direct", "DIRECT", "GLOBAL":
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
	if p.coreStatus() == core.StatusRunning {
		return p.startLoading("Stopping core"), p.stopCore()
	}
	return p.startLoading("Starting core"), p.startCore()
}

func firstDelayError(errs map[string]error) error {
	for name, err := range errs {
		return fmt.Errorf("%s: %w", name, err)
	}
	return fmt.Errorf("delay test failed")
}
