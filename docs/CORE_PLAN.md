# Mihomo TUI Core Plan

## Product Direction

Mihomo TUI is a keyboard-first terminal interface for managing an app-owned mihomo core on Ubuntu.

The product should feel like one complete proxy management tool:

- Users do not need to install or start mihomo themselves.
- `mhmt` owns the core lifecycle in managed mode.
- The app extracts or locates a mihomo core, starts it, waits for the controller, and then manages nodes through the controller API.
- Home is the main workspace for choosing proxy groups, selecting nodes, testing delay, and reloading config.

The core runtime mode is managed and proxy-only. The project focuses on HTTP, SOCKS, and mixed proxy ports. TUN mode, system proxy mutation, VPN routing, user-managed controller setup, and cross-platform service integration are out of scope for the core build.

Mouse support is intentionally out of scope for the initial core. It can return later as a component-level enhancement after layout boundaries are stable.

Build artifacts should be written to `releases/`. The source tree should not depend on generated binaries, installers, archives, or release bundles.

## Design Principles

- Keyboard first: every important action must be available through predictable keys.
- Managed first: the default user path is app-owned core lifecycle, not external controller setup.
- API first: UI pages should call a clear mihomo client, not know HTTP details.
- Home workspace first: everyday node selection lives in Home.
- Ubuntu first: packaging, paths, and runtime assumptions target Ubuntu before other platforms.
- Proxy only: do not change system proxy settings, install VPN routes, or manage TUN mode in the core flow.
- Mock remains available for UI development without a live core.
- Page ownership: each page owns its own cursor, filters, local state, update logic, and rendering.
- Thin root model: the Bubble Tea root model handles global messages, page switching, and window size only.
- Process control is separate from API control: core lifecycle is owned by `internal/core`, node operations by `internal/mihomo`.
- Stop only app-owned processes.

## Target Architecture

```text
cmd/
  main.go              # CLI entrypoint for mhmt
  start.go             # non-interactive start command
  stop.go              # non-interactive stop command
  version.go

internal/
  app/
    app.go              # Bubble Tea root model
    pages.go            # page interface and page registry
    home.go             # main workspace
    settings.go

  core/
    manager.go          # managed core lifecycle interface
    mock.go             # mock manager for UI development
    embedded.go         # future embedded core extraction
    process.go          # future subprocess manager
    paths.go            # future data/log/bin paths

  mihomo/
    client.go           # REST client
    types.go            # API types

  config/
    config.go           # app config model
    store.go            # load/save app config

releases/
  mhmt                  # generated local binary, ignored by git
  linux-amd64/mhmt      # generated Linux amd64 binary
  linux-arm64/mhmt      # generated Linux arm64 binary
```

Future managed data layout:

```text
~/.config/mihomo-tui/
  app.json
  config.yaml
  subscriptions/
  profiles/

~/.local/share/mihomo-tui/
  bin/
    mihomo
  logs/
    mihomo.log
```

## Core Interfaces

The UI should depend on core lifecycle and mihomo controller APIs through small interfaces.

```go
type Manager interface {
    Status() Status
    Start(context.Context) error
    Stop() error
    Restart(context.Context) error
}
```

```go
type Kernel interface {
    Health() error
    Status() (Status, error)
    ProxyGroups() ([]ProxyGroup, error)
    SelectProxy(groupName, proxyName string) error
    TestProxyDelay(proxyName string) (int, error)
    TestProxyGroupDelay(groupName string) error
    Traffic() (Traffic, error)
    ReloadConfig() error
}
```

## Page Model

Each page should follow a simple interface:

```go
type Page interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (Page, tea.Cmd)
    View(width, height int) string
    Help() string
}
```

The root app owns:

- current page
- page list
- terminal size
- global quit
- global page switching
- global message bar

Pages own:

- cursor position
- loading state
- filtering/search state
- page-specific messages
- page-specific key handling

## Keyboard Map

Global:

- `q`, `ctrl+c`: quit TUI only
- `tab`, `shift+tab`: switch page
- `?`: toggle help
- `esc`: cancel local mode or return focus

Home:

- `space`: start or stop managed core
- `x`: stop managed core
- `j/k`, `up/down`: move node selection
- `h/l`, `left/right`: move between proxy groups
- `enter`: select highlighted proxy
- `d`: test selected proxy delay
- `D`: test current group delay
- `/`: filter proxies
- `esc`: clear filter
- `r`: refresh status
- `R`: reload config

Settings:

- `j/k`, `up/down`: move field
- `enter`: edit field or toggle source mode
- `s`: save
- `esc`: cancel edit

## Milestones

### Milestone 0: Repo and Build Hygiene

- Initialize Git.
- Ignore local build output.
- Send local build output to `releases/mhmt`.
- Add Linux amd64 and arm64 build targets.
- Keep `go test ./...` passing.

### Milestone 1: App Shell

- Move the root Bubble Tea model to `internal/app`.
- Introduce a `Page` interface.
- Keep mouse support disabled.
- Add unified bottom message bar.

### Milestone 2: Mihomo REST Client

- Add `internal/mihomo`.
- Implement controller health check.
- Implement proxy groups, proxy selection, delay test, traffic, and config reload endpoints.
- Add unit tests with `httptest`.

### Milestone 3: Home Workspace MVP

- Show runtime status.
- Show proxy groups.
- Show nodes for the selected group.
- Highlight the active node.
- Support node selection with `enter`.
- Support selected-node delay test with `d`.
- Support current-group delay test with `D`.
- Support node search with `/`.
- Support nodes viewport/scrolling.
- Support config reload with `R`.

### Milestone 4: Managed Core Skeleton

- Default `source_mode` to `managed`.
- Keep `source_mode=mock` for local UI development.
- Define `internal/core.Manager`.
- Add core states: unavailable, stopped, starting, running, stopping, failed.
- Add Home start/stop controls.
- Use mock manager before wiring real embedded core.

### Milestone 5: Settings

- Add app config file.
- Support Ubuntu/proxy-only runtime settings.
- Support source mode, controller URL, secret, config path, mihomo binary path, and proxy ports.
- Save settings from the Settings page.

### Milestone 6: Embedded Core

- Embed one mihomo binary per release architecture.
- Extract core to `~/.local/share/mihomo-tui/bin/`.
- Start core as a subprocess.
- Wait for controller readiness.
- Stop only the app-owned process.
- Keep logs under `~/.local/share/mihomo-tui/logs/`.

### Milestone 7: Runtime Polling

- Poll status and traffic on intervals.
- Refresh current proxy group state without blocking input.
- Show non-blocking errors in the root message bar.

### Milestone 8: Traffic and Rules

- Add traffic graph.
- Show connection summary.
- Browse and filter routing rules.
- Keep rule editing out of scope until profile generation is ready.

### Milestone 9: Ubuntu Packaging

- Package an Ubuntu-friendly single binary.
- Add install scripts for common Ubuntu paths.
- Keep install instructions simple.

## Completed

- Disable mouse support.
- Send local build output to `releases/mhmt`.
- Move the root Bubble Tea model to `internal/app`.
- Introduce a `Page` interface.
- Add app config defaults.
- Add `internal/mihomo.Client`.
- Add `source_mode` support.
- Add Home mock proxy groups for local UI development.
- Add Home node filtering with `/`.
- Add Settings editing for source mode, controller URL, secret, paths, and proxy ports.
- Persist Settings changes through `internal/config`.
- Add Home current-group delay testing with `D`.
- Add a unified bottom message bar owned by the root app.
- Add Home nodes viewport/scrolling for larger proxy groups.
- Add Ubuntu/Linux build targets for amd64 and arm64.
- Default source mode to managed.
- Add `internal/core.Manager` and mock managed core lifecycle.
- Add Home core start/stop status controls.
- Replace Rules placeholder with read-only routing rule browser and search.

## Mouse Support Later

Mouse support can return after components are stable. The future design should use hitboxes generated during rendering:

```go
type Hitbox struct {
    ID   string
    Rect Rect
}
```

Mouse handlers should dispatch actions by hitbox ID. They should not recalculate layout positions from scratch.

## Immediate Next Steps

1. Add embedded core extraction skeleton.
2. Add real process manager using extracted mihomo binary.
3. Apply Settings changes without requiring an app restart.
4. Add lightweight tests for Settings validation.
5. Add rule loading from generated profile/config.
