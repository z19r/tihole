# tihole

[![CI](https://github.com/z19r/tihole/actions/workflows/ci.yml/badge.svg)](https://github.com/z19r/tihole/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/z19r/tihole.svg)](https://pkg.go.dev/github.com/z19r/tihole)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
![Go 1.25+](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)

A fast, keyboard-driven terminal UI for [Pi-hole v6](https://pi-hole.net/),
built in Go on the [Charm](https://charm.sh/) v2 stack (Bubble Tea, Bubbles,
Lip Gloss). It aims for feature parity with the Pi-hole web admin — query log,
allow/deny lists, groups, clients, adlists, local DNS, live config, and system
tools — without leaving your terminal.

## Features

- **Dashboard** — blocking status, query/percentage tiles, top clients &
  domains at a glance.
- **Query Log** — cursor-paginated, filterable live log of DNS queries.
- **Domains** — manage allow/deny lists across exact and regex kinds with
  group assignment.
- **Groups & Clients** — full CRUD, with client suggestions pulled from the
  network table.
- **Adlists** — block/allow lists plus a streamed **gravity update** log.
- **Local DNS** — A/AAAA host records and CNAME records.
- **Settings** — browse and edit the entire FTL config tree, manage
  connections (add/edit/remove instances), and switch themes live.
- **System / Tools** — FTL/host/system info, diagnosis messages, the network
  device table, a live DNS-log tail, and guarded destructive actions
  (restart DNS, flush logs/network).
- **Multi-instance** — configure several Pi-holes and switch between them
  instantly (`s` or the command palette).
- **Command palette** (`ctrl+k`) — fuzzy-jump to any screen, toggle blocking,
  change themes, or switch instances.
- **Help overlay** (`?`) — context-aware cheat-sheet of global and per-screen
  keys.
- **Themes** — the signature **Gloss** default (multi-stop gradient treatment),
  plus `deep-night`, `light-luxury`, `pihole-classic`, and automatic adoption of
  your [Omarchy](https://omarchy.org/) theme when present.

## Install

Requires Go 1.25+.

```bash
go install github.com/z19r/tihole/cmd/tihole@latest
```

Or build from a checkout:

```bash
go build -o tihole ./cmd/tihole
```

## Usage

```bash
tihole           # launch the dashboard
tihole config    # jump straight to the config editor to add or fix an instance
tihole help      # usage
```

On first run, tihole walks you through a short setup wizard and writes a config
file. After that it connects to the active instance and opens the dashboard.

A wrong address or password never aborts startup: authentication happens lazily,
so connection failures show up as in-app error banners, and a structurally
broken instance drops you straight into the config editor to fix it. You can
also open the editor any time with `tihole config` or from the command palette
(`ctrl+k` → Settings).

### Keys

| Key | Action |
| --- | --- |
| `1`–`9` | Jump to a screen by number |
| `↑↓` / `j` `k` | Move the selection (screens on the rail, rows in a panel) |
| `enter` / `→` / `tab` | Descend from the sidebar into the active screen |
| `esc` | Climb back to the sidebar |
| `ctrl+k` | Command palette |
| `s` | Switch to the next instance |
| `d` | Toggle blocking |
| `ctrl+t` | Cycle theme |
| `?` | Help overlay |
| `q` / `ctrl+c` | Quit |

Per-screen actions (add `a`, edit `e`, delete `x`, refresh `r`, …) are shown in
the help bar and the `?` overlay. See [`docs/keys.md`](docs/keys.md) for the
complete reference and an explanation of the two-zone focus model.

## Configuration

Config lives at `~/.config/tihole/config.yaml` (mode `0600`). It is written and
edited by the app, but can also be hand-authored:

```yaml
active: home
theme: deep-night
instances:
  - name: home
    url: https://pi.hole
    password_env: TIHOLE_HOME_PASSWORD   # preferred: read the app password from env
    verify_tls: true
  - name: cabin
    url: http://10.0.0.53
    password: plaintext-ok-but-env-is-better
    verify_tls: false                    # self-signed / no TLS
```

Each instance needs a `name`, a `url`, and an app password supplied either
inline (`password`) or, preferably, via `password_env` naming an environment
variable. Set `verify_tls: false` for self-signed certificates.

### Security

- The config file is created `0600` in a `0700` directory.
- Passwords and session IDs are **never** logged.
- Authentication uses Pi-hole v6's `X-FTL-SID` session header, re-authenticating
  transparently on expiry, and logs out on exit so it doesn't leak a session
  seat.
- Some Pi-hole actions require `webserver.api.allow_destructive` to be enabled
  on the server; tihole surfaces a clear hint when the API rejects them.

## Development

```bash
go build ./...     # compile
go vet ./...       # static checks
gofmt -l .         # formatting (should print nothing)
go test ./...      # tests
go test -cover ./internal/...   # with coverage
```

The codebase separates concerns strictly: the domain packages (`internal/pihole`,
`internal/config`, `internal/theme`) never import the TUI, and screens receive
their dependencies through a shared `core.AppContext`.

### Documentation

- [`docs/architecture.md`](docs/architecture.md) — package layout, dependency
  injection, the `Screen` contract, and message flow.
- [`docs/keys.md`](docs/keys.md) — complete key reference and the focus model.
- [`docs/troubleshooting.md`](docs/troubleshooting.md) — config, TLS, auth, and
  destructive-action issues.
- [`docs/CHARM_V2_API.md`](docs/CHARM_V2_API.md) — Charm v2 specifics.
- [`docs/superpowers/specs/2026-07-23-tihole-design.md`](docs/superpowers/specs/2026-07-23-tihole-design.md)
  — the original design spec.

## License

Released under the [MIT License](LICENSE).
