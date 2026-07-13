# Mihomo TUI

A keyboard-first terminal UI for managing mihomo proxy nodes on Ubuntu.

The core scope is proxy-only: HTTP, SOCKS, and mixed proxy modes. It does not
manage TUN mode, VPN routes, or desktop proxy settings. The installer adds a
small Bash/Zsh integration so `mhmt on/off` can update the current shell's proxy
environment variables.

## Commands

```text
mhmt          Open the TUI
mhmt start    Start the managed Mihomo core
mhmt stop     Stop the managed Mihomo core
mhmt on       Enable proxy variables in the current shell
mhmt off      Disable proxy variables in the current shell
```

The commands are intentionally independent: `on` does not start Mihomo, and
`off` does not stop it. The installer creates
`~/.local/share/mihomo-tui/env` and adds this idempotent line to an existing
`~/.bashrc` or `~/.zshrc`:

```bash
. "$HOME/.local/share/mihomo-tui/env"
```

Open a new shell session or source its rc file once after installation. Existing
shells cannot be modified by a child process, so the installed `mhmt()` shell
function evaluates only the `on/off` output and forwards every other command to
the real binary.

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

### Local layout

`internal/local.ResolveLayout` is the single source of truth for installed
paths:

```text
~/.config/mihomo-tui/       config.json, state.json, config.yaml
~/.local/share/mihomo-tui/  core, GeoIP, shell integration, licenses
~/.local/state/mihomo-tui/  PID and log files
~/.local/bin/               mhmt
```

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

The installer uses a fixed per-user layout: `mhmt` in `~/.local/bin`, the
versioned Mihomo core and offline GeoIP data under
`~/.local/share/mihomo-tui`, initial settings under
`~/.config/mihomo-tui`, and process PID and log files under
`~/.local/state/mihomo-tui`. Existing settings are not overwritten.

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
