package app

import (
	"context"
	"fmt"
	"time"

	"mihomo-tui/internal/core"
	"mihomo-tui/internal/mihomo"
	"mihomo-tui/internal/runtimeconfig"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	proxyGroupsReadyAttempts = 50
	proxyGroupsReadyDelay    = 100 * time.Millisecond
)

func (p homePage) loadProxyGroups() tea.Cmd {
	return func() tea.Msg {
		if p.coreStatus() != core.StatusRunning {
			groups, err := runtimeconfig.LoadProxyGroups(p.cfg.ConfigPath)
			return proxyGroupsLoadedMsg{groups: groups, snapshot: err == nil, err: err}
		}
		groups, err := p.loadReadyProxyGroups()
		return proxyGroupsLoadedMsg{groups: groups, err: err}
	}
}

func (p homePage) loadReadyProxyGroups() ([]mihomo.ProxyGroup, error) {
	if p.client == nil {
		return nil, fmt.Errorf("mihomo controller unavailable")
	}
	var lastErr error
	for attempt := 0; attempt < proxyGroupsReadyAttempts; attempt++ {
		groups, err := p.client.ProxyGroups()
		if err == nil && len(visibleHomeGroups(groups)) > 0 {
			return groups, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("proxy groups are not ready")
		}
		if attempt+1 < proxyGroupsReadyAttempts {
			time.Sleep(proxyGroupsReadyDelay)
		}
	}
	return nil, lastErr
}

func (p homePage) selectCurrentProxy() tea.Cmd {
	group := p.currentGroup()
	proxy := p.currentProxy()
	return func() tea.Msg {
		if group.Name == "" || proxy.Name == "" {
			return proxySelectedMsg{err: fmt.Errorf("no proxy selected")}
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
		if p.coreStatus() != core.StatusRunning {
			return configReloadedMsg{err: fmt.Errorf("core is %s", p.coreStatus())}
		}
		return configReloadedMsg{err: p.client.ReloadConfig("", false)}
	}
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
