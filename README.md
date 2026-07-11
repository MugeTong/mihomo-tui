# Mihomo TUI

A keyboard-first terminal UI for managing mihomo proxy nodes on Ubuntu.

The core scope is proxy-only: HTTP, SOCKS, and mixed proxy modes. It does not manage TUN mode, VPN routes, or system proxy settings.

## Development

Run with mock data:

```bash
make run
```

Build all supported Linux release binaries:

```bash
make build
```

Outputs:

```text
releases/linux-amd64/mhmt
releases/linux-arm64/mhmt
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
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for details.

Project homepage: https://github.com/MugeTong/mihomo-tui
