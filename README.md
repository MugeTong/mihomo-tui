# Mihomo TUI

[![Go](https://img.shields.io/github/go-mod/go-version/MugeTong/mihomo-tui)](https://go.dev/)
[![License](https://img.shields.io/github/license/MugeTong/mihomo-tui)](LICENSE)
[![Release](https://img.shields.io/github/v/release/MugeTong/mihomo-tui?color=green)](https://github.com/MugeTong/mihomo-tui/releases)
![Downloads](https://img.shields.io/github/downloads/MugeTong/mihomo-tui/total)

A keyboard-first Mihomo manager for the terminal, with Linux amd64 as its
primary supported platform.

Manage subscriptions, nodes, proxy groups, rules, and the Mihomo core without a
Web UI. The project supports HTTP, SOCKS, and mixed proxies only.

## Quick Start

Download the installer for your platform from the
[latest release](https://github.com/MugeTong/mihomo-tui/releases/latest):

```bash
# Linux amd64
curl -LO https://github.com/MugeTong/mihomo-tui/releases/latest/download/mihomo-tui-linux-amd64-installer
chmod +x mihomo-tui-linux-amd64-installer
./mihomo-tui-linux-amd64-installer
```

For Linux arm64 or macOS arm64, replace the asset name with
`mihomo-tui-linux-arm64-installer` or
`mihomo-tui-darwin-arm64-installer`.

The installer does not require root. Open a new terminal after installation,
then launch the TUI:

```bash
mhmt
```

Press `a` on the Sources page to import a subscription URL or node URI. Return
to Home and press `space` to start Mihomo. Use the arrow keys to choose a node
and `enter` to select it.

## Commands

```text
mhmt          Open the TUI
mhmt start    Start the managed Mihomo core
mhmt stop     Stop the managed Mihomo core
mhmt on       Enable proxy variables in the current shell
mhmt off      Disable proxy variables in the current shell
mhmt version  Show the installed version
```

The commands are intentionally independent: `on` does not start Mihomo, and
`off` does not stop it. Both commands update proxy variables in the current
shell through the integration configured by the installer.

## Installed Files

```text
~/.config/mihomo-tui/       Settings, subscription state, and generated config
~/.local/share/mihomo-tui/  Mihomo core, GeoIP, shell integration, and licenses
~/.local/state/mihomo-tui/  PID and log files
~/.local/bin/mhmt            Command-line program
```

Existing settings and subscription state are preserved when the installer is
run again.

## Development

### Project structure

```text
cmd/mhmt/          CLI commands and TUI entrypoint
cmd/installer/     Embedded payload declarations and installer entrypoint
internal/app/      Bubble Tea pages, updates, and views
internal/config/   Application settings model and JSON persistence
internal/core/     Managed Mihomo process lifecycle
internal/install/  Installation workflow and initial config generation
internal/local/    Fixed per-user paths, shell integration, and licenses
internal/mihomo/   Mihomo controller API client
internal/rules/    Embedded application-owned routing rules
internal/runtimeconfig/  Mihomo YAML generation and snapshots
internal/subscription/   Source fetching, parsing, and persisted state
```

The dependency direction is intentional: `internal/local` owns paths but does
not know the JSON or YAML formats. `internal/install` combines local paths with
the config, subscription, and runtime-config packages. The installer command
only supplies its embedded platform payload. This keeps future packaging or
uninstall commands from duplicating path and configuration logic.

Run the local TUI against the last generated config snapshot:

```bash
make run
```

Build platform-specific self-contained installers:

```bash
make build
```

The build downloads the pinned official Mihomo core and GeoIP database, verifies
their SHA-256 hashes, embeds them with `mhmt`, and writes only release installers
to `releases/`. Set `MIHOMO_ASSET_DIR` and `GEOIP_ASSET` to reuse matching local
files.

Outputs:

```text
releases/mihomo-tui-linux-amd64-installer
releases/mihomo-tui-linux-arm64-installer
releases/mihomo-tui-darwin-arm64-installer
```

Run tests:

```bash
make test
```

## License

Copyright (C) 2025-2026 MugeTong.

Mihomo TUI is free software licensed under the GNU General Public License
version 3 only (`GPL-3.0-only`). See [LICENSE](LICENSE).

This project does not provide proxy servers, subscriptions, network access, or
telecommunications services. Users are responsible for complying with
applicable laws and the terms of their network providers.

Mihomo is an independent GPL-3.0 project. Mihomo TUI is inspired in part by
Shadowrocket but is not affiliated with or endorsed by its developer. See
[third-party notices](internal/local/licenses/THIRD_PARTY_NOTICES.md) for details.

Project homepage: https://github.com/MugeTong/mihomo-tui
