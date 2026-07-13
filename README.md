# Mihomo TUI

A keyboard-first terminal UI for managing mihomo proxy nodes on Ubuntu.

The core scope is proxy-only: HTTP, SOCKS, and mixed proxy modes. It does not
manage TUN mode, VPN routes, or desktop proxy settings. The installer adds a
small Bash integration so `mhmt on/off` can update the current shell's proxy
environment variables.

## Commands

```text
mhmt          Open the TUI
mhmt start    Start the managed Mihomo core
mhmt stop     Stop the managed Mihomo core
mhmt on       Enable proxy variables in the current Bash
mhmt off      Disable proxy variables in the current Bash
```

The commands are intentionally independent: `on` does not start Mihomo, and
`off` does not stop it. The installer creates
`~/.local/share/mihomo-tui/env` and adds this idempotent line to `~/.bashrc`:

```bash
. "$HOME/.local/share/mihomo-tui/env"
```

Open a new Bash session or run `. ~/.bashrc` once after installation. Existing
shells cannot be modified by a child process, so the installed `mhmt()` shell
function evaluates only the `on/off` output and forwards every other command to
the real binary.

## Development

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

The installer places `mhmt` in `~/.local/bin`, the versioned Mihomo core and
offline GeoIP data under `$XDG_DATA_HOME/mihomo-tui` (default
`~/.local/share/mihomo-tui`), and initial settings in
`$XDG_CONFIG_HOME/mihomo-tui` without overwriting existing settings.

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
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for details.

Project homepage: https://github.com/MugeTong/mihomo-tui
