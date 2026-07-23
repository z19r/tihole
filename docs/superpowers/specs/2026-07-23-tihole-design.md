# tihole — a beautiful PiHole v6 TUI

**Status:** Approved design (brainstorming complete) · **Date:** 2026-07-23

## 1. Goals

1. **Feature parity** with the PiHole v6 web UI, as far as the REST API allows.
2. **Absolutely beautiful** — a first-class Charm/Bubble Tea TUI that looks intentional
   and opinionated, not a default template.

Non-goals: PiHole v5 (`admin/api.php`) support; managing the host OS; a web/GUI frontend.

## 2. Decisions (locked during brainstorming)

| Area | Decision |
|------|----------|
| Target API | **PiHole v6 REST API only** (FTL embedded webserver, `/api/...`) |
| Instances | **Multiple, switchable** — configure N, switch the active one; every screen reflects the active instance |
| Config & secrets | **Config file + first-run wizard.** `~/.config/tihole/config.yaml` (`0600`); passwords inline or via `password_env` |
| v1 scope | **Full parity target**, built **foundation-first** and shipped incrementally |
| App layout | **Sidebar + content**, top status bar, bottom help bar; **Ctrl-K command palette** as a secondary accelerator |
| Aesthetic | **2–3 curated built-in themes** + an **Omarchy adapter** that mirrors the user's current desktop theme |

## 3. Stack

Target the **Charm v2 ecosystem** (`charm.land/*/v2`) — current for a new large app in 2026.
Upgrade Bubble Tea, Bubbles, and Lip Gloss in lockstep.

| Library | Module path | Purpose |
|---------|-------------|---------|
| Bubble Tea | `charm.land/bubbletea/v2` | Elm-architecture runtime. Note `View() tea.View`, `tea.KeyPressMsg`, `p.Run()`. |
| Bubbles | `charm.land/bubbles/v2` | list, table, viewport, textinput, help, spinner, paginator, key |
| Lip Gloss | `charm.land/lipgloss/v2` | styling/layout; explicit `LightDark` adaptive color |
| Huh | `charm.land/huh/v2` | first-run wizard + connection forms |
| Glamour | `charm.land/glamour/v2` | markdown rendering for in-app help (optional) |
| Harmonica | `github.com/charmbracelet/harmonica` | spring animation for transitions/sparklines (optional; v1, fine) |
| ntcharts | `github.com/NimbleMarkets/ntcharts` | sparkline/line/bar charts — **verify LG v2 compatibility before adopting**; else hand-roll block-rune bars |

Go: pin a toolchain (mise) — `go@1.25.x` or newer.

## 4. Architecture

Clean layering; **domain packages never import Bubble Tea** so they stay testable and
could drive a CLI later.

```
tihole/
├── cmd/tihole/main.go            # flag parsing, wire deps, tea.NewProgram(...).Run()
├── internal/
│   ├── pihole/                   # v6 API client — pure Go, no TUI imports
│   │   ├── client.go             # Client, transport, X-FTL-SID auth, re-auth on 401
│   │   ├── auth.go stats.go history.go queries.go blocking.go
│   │   ├── domains.go groups.go clients.go lists.go gravity.go
│   │   ├── dns_records.go config.go info.go network.go actions.go
│   │   └── types.go errors.go
│   ├── config/                   # load/save/validate config.yaml (0600), password_env
│   ├── theme/                    # Theme tokens, built-in themes, Omarchy adapter
│   └── tui/
│       ├── app.go                # root AppModel: router, sidebar, status/help chrome, size fan-out
│       ├── nav.go                # PageID enum, NavigateMsg, Screen interface
│       ├── keys.go               # global KeyMap (key.Binding)
│       ├── context.go            # AppContext{API, *Theme, Keys} injected into screens
│       ├── palette.go            # Ctrl-K command palette
│       ├── components/           # statusbar, sidebar, banner/toast, chart wrappers, confirm
│       └── screens/              # one package per screen implementing Screen
│           ├── dashboard/ querylog/ domains/ groups/ clients/
│           ├── adlists/ localdns/ settings/ system/
```

### 4.1 Root model & routing

Root `AppModel` owns the persistent chrome (sidebar, status bar, help bar) and a
`Screen` for the content pane. Navigation via a custom `NavigateMsg`.

```go
type Screen interface {
    tea.Model                 // Init() tea.Cmd; Update(tea.Msg)(tea.Model,tea.Cmd); View() tea.View
    Title() string
    Focus() tea.Cmd           // becomes active: start pollers, fetch
    Blur()                    // leaving: cancel in-flight work
    Help() []key.Binding
}

type AppContext struct {
    API  *pihole.Client       // active instance's client
    Theme *theme.Theme        // pointer → live re-theme visible everywhere
    Keys  KeyMap
}
```

**Rules (from Charm-idiom research):**
- On `tea.WindowSizeMsg`, root subtracts chrome and **broadcasts the inner content size to
  every screen** (not just the active one) so backgrounded screens stay correctly sized.
- Only the active screen receives key input; `Focus()`/`Blur()` start/stop pollers and
  cancel commands. Gate delegation on focus (some Bubbles like `list` react to all msgs).
- Shared context injected at construction — **no package-level globals**. Theme is a pointer.

### 4.2 Instance switching

Config holds N instances. The active instance owns a `*pihole.Client`. A picker
(hotkey `s`, also in the command palette) switches active; on switch, the root swaps
`ctx.API` and re-`Focus()`es the current screen to reload. Each instance authenticates
lazily on first use and keeps one session (see 5.2).

### 4.3 Async data & refresh

All I/O in `tea.Cmd` (never in `Update`). A per-screen poller re-issues a `tea.Tick` from
its own handler (dashboard ~5s, query log ~2s, static screens on-demand / manual `r`).
Polling only re-schedules while the screen is focused. Each fetch is `context.Context`-aware;
`Blur()` calls `cancel()`. Loading → self-driving `spinner`; errors → inline non-fatal
banner, never a crash. Writes are optimistic with rollback + visible error on failure.

## 5. PiHole v6 client (`internal/pihole`)

Pure Go, `context`-aware, one `Client` per instance. Verified against the FTL OpenAPI specs
(`pi-hole/FTL/src/api/docs/content/specs/*.yaml`). The live instance self-documents at
`http(s)://<host>/api/docs`.

### 5.1 Transport & errors

- Base URL `http(s)://<host>/api`. Universal `took` float on every response; **no global
  data wrapper** — each endpoint returns its own top-level keys.
- Error envelope: `{ "error": { "key", "message", "hint" }, "took" }` → typed
  `APIError{Status, Key, Message, Hint}`. Also `AuthError`, `NetworkError`.
- Self-signed TLS is common → `verify_tls: false` per instance opts into
  `InsecureSkipVerify` (default verify on).

### 5.2 Auth (`/api/auth`) — `X-FTL-SID` header flow

- Login: `POST /api/auth` `{ "password": "<app-or-web-password>", "totp": <int?> }` →
  `session{ valid, sid, csrf, validity, message, totp }`.
- Subsequent requests send **`X-FTL-SID: <sid>`** (header auth needs **no** CSRF token;
  `X-FTL-CSRF` is only for cookie mode). SID is **bound to client IP**.
- Session check `GET /api/auth`; logout `DELETE /api/auth` **on exit** (don't leak seats —
  `webserver.api.max_sessions`, ~16, else `429 api_seats_exceeded`).
- On any `401`, transparently re-auth and retry once; renew before `validity` (default
  1800s) elapses.
- **App passwords preferred** for a headless client (bypass TOTP/2FA). Web password also
  supported (with `totp` when 2FA is on). No secrets logged.

### 5.3 Endpoint groups → client methods → screens

| Group | Endpoints (verified) | Screen |
|-------|----------------------|--------|
| Stats | `GET /api/stats/summary`, `/query_types`, `/upstreams`, `/top_domains?blocked&count`, `/top_clients`, `/recent_blocked`; `/stats/database/*` (`from`,`until`) | Dashboard |
| History | `GET /api/history`, `/history/clients?N`, `/history/database` | Dashboard charts |
| Queries | `GET /api/queries` (cursor pagination; filters: `length,start,cursor,from,until,domain,client_ip,client_name,upstream,type,status,reply,dnssec,disk`) | Query Log |
| Blocking | `GET/POST /api/dns/blocking` `{blocking:bool, timer:sec|null}` | global `d`, palette, status bar |
| Domains | `GET/POST/PUT/DELETE /api/domains/{type}/{kind}/{domain}` (`allow\|deny` × `exact\|regex`); `POST /api/domains:batchDelete` | Domains |
| Groups | `GET/POST/PUT/DELETE /api/groups[/{name}]`; `:batchDelete` | Groups |
| Clients | `GET/POST/PUT/DELETE /api/clients[/{client}]`; `/clients/_suggestions`; `:batchDelete` | Clients |
| Lists | `GET/POST/PUT/DELETE /api/lists[/{list}]?type=allow\|block`; `:batchDelete` | Adlists |
| Gravity | `POST /api/action/gravity` — **chunked `text/plain` stream** (not SSE); parse lines for live progress | Adlists (update action) |
| Local DNS | config elements: `GET/PUT/DELETE /api/config/dns/hosts[/{value}]` (A/AAAA), `/api/config/dns/cnameRecords[/{value}]` (CNAME) | Local DNS |
| Config | `GET /api/config?detailed`, `GET /api/config/{element}`, `PATCH /api/config?restart`, `PUT/DELETE /api/config/{element}/{value}` | Settings |
| Info | `GET /api/info/{ftl,host,system,version,metrics,sensors,database,messages}`; `/info/messages/count`, `DELETE /info/messages/{ids}`; `/info/logs/dns?nextID` (poll) | System/Tools |
| Network | `GET /api/network/devices`, `/gateway`, `/interfaces`, `/routes`; `DELETE /network/devices/{id}` | System/Tools |
| Actions | `POST /api/action/{restartdns,flush/logs,flush/network}` (may `403` unless `allow_destructive`) | System/Tools |

Notes: batch delete uses the literal `:batchDelete` verb with a `[{item,...}]` body.
Creates return `201` + `Location`; deletes `204`. Regex domains must be JSON/URI-escaped.

## 6. Screen catalog (full parity)

Each screen is a self-contained package implementing `Screen`. Built foundation-first;
each is independently shippable.

- **Dashboard** — summary tiles (queries/blocked %/clients/gravity), queries-over-time
  sparkline (from `/history`), query-type + upstream breakdowns, top domains / top clients /
  top blocked. Live poll ~5s.
- **Query Log** — cursor-paginated table with `/`-search and filters (domain, client, type,
  status, upstream, time). Row detail pane. Quick allow/deny a domain from a row. Live ~2s.
- **Blocking toggle** — global `d` and palette actions ("disable 30s / 5m / 30m / until
  enabled"); state + countdown shown in the status bar.
- **Domains** — allow/deny × exact/regex management; add/edit/delete, comment, group
  assignment, enable toggle; batch delete.
- **Groups** — CRUD, enable toggle.
- **Clients** — CRUD, group assignment, `_suggestions` for unconfigured clients.
- **Adlists** — block/allow lists CRUD; trigger **gravity update** with a streamed live log
  view.
- **Local DNS** — A/AAAA hosts + CNAME records via config elements.
- **Settings** — browse/edit config tree (DNS, DHCP, privacy, resolver, webserver, misc);
  respect `read_only` keys; **Connections** sub-screen to add/edit/remove instances +
  theme selection.
- **System/Tools** — FTL/host/system info, diagnosis messages (view + dismiss), network
  device table, live DNS log tail (`/info/logs/dns` poll), destructive actions
  (restart DNS, flush logs/network) behind confirmation.

## 7. Config & secrets (`internal/config`)

`~/.config/tihole/config.yaml`, mode `0600`.

```yaml
active: home
theme: omarchy            # deep-night | light-luxury | pihole-classic | omarchy
refresh:                  # optional per-screen overrides (seconds)
  dashboard: 5
  querylog: 2
instances:
  - name: home
    url: https://192.168.1.2
    password: "…"         # inline (0600) …
    verify_tls: false
  - name: office
    url: https://10.0.0.2
    password_env: TIHOLE_OFFICE_PW   # … or read from env
    verify_tls: true
```

- **First-run wizard** (Huh): prompt name + URL + password, **validate by attempting auth**,
  write the file. Add/edit/remove later from Settings → Connections.
- `password_env` reads a secret from the environment; inline passwords never logged.
- Validation at load: required fields, URL well-formed, unique names, exactly one `active`.

## 8. Theming (`internal/theme`)

A `Theme` is a struct of **semantic tokens** (surface, panel, text, subtle, accent,
allow=green, block=red, warn, border) → reused `lipgloss.Style`s. Screens read tokens at
`View()` time so re-theming is instant.

- **Built-in themes:** `deep-night` (signature dark), `light-luxury`, `pihole-classic` (red).
- **Adaptive color:** Lip Gloss v2 explicit `LightDark` — detect background once, pick
  light/dark per token.
- **Omarchy adapter** (`theme: omarchy`): parse
  `~/.config/omarchy/current/theme/alacritty.toml` (`[colors.primary]` bg/fg +
  `[colors.normal]`/`[colors.bright]` 16 ANSI). Detect light vs dark via presence of the
  `light.mode` marker file in the same dir. Map ANSI → semantic tokens (e.g. red→block,
  green→allow, a bright accent→accent). Falls back to `deep-night` if Omarchy isn't present.
- **Live re-theme:** theme held behind a pointer in `AppContext`; a `SetThemeMsg` rebuilds
  and triggers a full re-render. (Nice-to-have: re-read Omarchy on window focus.)

## 9. Error handling

- Typed errors at the client; surfaced in the TUI as inline banners/toasts, never a panic
  that drops the terminal.
- Logging goes to a **file** (e.g. `~/.local/state/tihole/tihole.log`), never stdout (the
  TUI owns the screen). Secrets redacted.
- Auth expiry → transparent re-auth. `403 forbidden` on destructive actions → explain the
  `allow_destructive` requirement. `429` → back off and surface a hint.

## 10. Testing (house target: 80%)

- **`internal/pihole`** — unit-tested against an `httptest` mock FTL per endpoint group,
  including the auth flow, 401 re-auth, error envelope decoding, cursor pagination, and the
  chunked gravity stream parser.
- **`internal/config`** — load/save/validate, `password_env` resolution, `0600` enforcement.
- **`internal/theme`** — ANSI→token mapping, light/dark selection, Omarchy `alacritty.toml`
  parse + `light.mode` detection, fallback behavior.
- **`internal/tui`** — pure `Update` reducer tests with synthetic msgs (bulk of logic
  coverage); `teatest` golden/snapshot tests for key screens rendered with the ASCII color
  profile (`-update` to refresh; `.gitattributes` to protect goldens).

## 11. Build order (foundation-first)

The implementation plan will phase this; sketch:

1. **Foundation** — module + toolchain, `pihole.Client` (transport + auth + errors),
   `config` (+ first-run wizard), `theme` (tokens + one built-in), root `AppModel` shell
   (sidebar/status/help, routing, size fan-out), instance switcher, command palette skeleton.
2. **Monitoring** — Dashboard + Query Log + blocking toggle. First genuinely useful build.
3. **List management** — Domains, Adlists (+ streamed gravity), Groups, Clients.
4. **Records & config** — Local DNS, Settings (config tree + Connections).
5. **System/Tools** — info, messages, network, log tail, destructive actions.
6. **Polish** — remaining themes + Omarchy adapter refinements, animations, help/markdown,
   snapshot-test hardening.

## Appendix A — source references

- FTL OpenAPI specs: `github.com/pi-hole/FTL/tree/master/src/api/docs/content/specs/`
  (`auth, stats, queries, dns, domains, groups, clients, lists, history, config, info,
  network, action, common`).
- `docs.pi-hole.net/api/`; live self-docs at `http(s)://<host>/api/docs`.
- Charm v2: bubbletea/bubbles/lipgloss/huh `UPGRADE_GUIDE_V2.md`; teatest
  (`github.com/charmbracelet/x/exp/teatest`).
