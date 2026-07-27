# Architecture

tihole is a Bubble Tea (Charm v2) terminal UI over the Pi-hole v6 REST API. The
guiding rule is a strict one-way dependency: **the domain talks to Pi-hole and
knows nothing about the UI; the UI depends on the domain and never the reverse.**

## Package layout

```
cmd/tihole/            entrypoint, CLI subcommands (config, changelog-sync, help)
internal/
├── pihole/            Pi-hole v6 API client — the only package that speaks HTTP
├── config/            config.yaml load/save/validate (~/.config/tihole)
├── theme/             semantic color themes + Omarchy desktop-theme adapter
└── tui/               everything Bubble Tea
    ├── core/          shared contracts: AppContext, Screen, KeyMap, nav, messages
    ├── components/    reusable widgets: sidebar, statusbar, palette, help,
    │                  section tabs, splash, gradient, confirm, table
    └── screens/       one package per screen (dashboard, querylog, domains, …)
```

The three domain packages — `pihole`, `config`, `theme` — **never import
`internal/tui`.** That boundary is what keeps the client and config testable in
isolation and the TUI swappable.

## Dependency injection: `AppContext`

There are no package-level globals. Every screen is constructed with a shared
[`core.AppContext`](../internal/tui/core/context.go), the single injected
dependency set:

```go
type AppContext struct {
    API          *pihole.Client // active instance's client; swapped on instance switch
    Theme        *theme.Theme   // pointer, so a live re-theme is instant everywhere
    Keys         KeyMap         // global key map
    InstanceName string         // currently active instance
    Config       *config.Config // pointer, shared with the Settings screen
    ConfigPath   string         // where Config persists (0600)
}
```

`Theme` and `Config` are held behind pointers deliberately: a theme change or an
edit to the instance list is visible to every screen immediately, without
re-plumbing anything.

## The `Screen` contract

Every content screen implements [`core.Screen`](../internal/tui/core/nav.go),
which extends `tea.Model` with lifecycle hooks the root uses to manage focus and
pollers:

```go
type Screen interface {
    tea.Model
    Title() string          // shown in the header / status bar
    Focus() tea.Cmd         // becoming active: start pollers, fetch
    Blur()                  // leaving: cancel in-flight work
    Help() []key.Binding    // screen-local bindings for the help bar
    SetSize(w, h int)       // inner content area (chrome already subtracted)
}
```

Two optional capabilities refine behavior:

- **`InputCapturer`** — a screen returns `true` while a text field is focused, so
  the root delivers raw keys to it instead of firing single-key globals.
- **`PanelInteractor`** — a screen returns `false` when it has no actionable
  content (e.g. the read-only Dashboard), so the rail refuses to descend into it.

## The root model and message flow

[`internal/tui/app.go`](../internal/tui/app.go) holds the root `AppModel`: it
owns the sidebar, status bar, palette, help overlay, the boot splash, and the
map of constructed screens. Its `Update` implements the two-zone focus model
(see [keys.md](keys.md)) and routes cross-cutting messages defined in `core`:

- `NavigateMsg` — switch to another page (blurs the old screen, focuses the new).
- `SetThemeMsg` — live re-theme across all screens.
- `SwitchInstanceMsg` — activate a different configured instance.
- `InstancesChangedMsg` — Settings edited the instance list; swap in new config.
- `ErrorMsg` — surface a non-fatal error as an inline banner instead of crashing.

Screens communicate outward by returning these messages as `tea.Cmd`s; they
never reach into the root or each other directly.

## The Pi-hole client

[`internal/pihole`](../internal/pihole) is the sole HTTP boundary. It manages a
Pi-hole v6 session via the `X-FTL-SID` header, re-authenticating transparently
when a session expires and logging out on exit so it doesn't leak a session seat.
Passwords and session IDs are never logged. Destructive endpoints (restart DNS,
flush logs/network) return a `403 APIError` unless the server has
`webserver.api.allow_destructive` enabled; the UI surfaces that as a clear hint
rather than a stack trace.

## Themes

[`internal/theme`](../internal/theme) defines themes as semantic tokens rather
than raw colors, so screens ask for `Surface`/`Accent`/`Muted` and the active
theme resolves them. Built-ins: the signature **Gloss** default (multi-stop
gradient treatment), `deep-night`, `light-luxury`, and `pihole-classic`. When
Omarchy is present, `omarchy` adapts your desktop theme by mapping its Alacritty
ANSI palette onto tihole's tokens; if that load fails it falls back to
`deep-night` rather than erroring.

## Further reading

- [`docs/CHARM_V2_API.md`](CHARM_V2_API.md) — Charm v2 specifics this codebase relies on.
- [`docs/keys.md`](keys.md) — the full key reference and focus model.
- [`docs/troubleshooting.md`](troubleshooting.md) — config, TLS, auth, and destructive-action issues.
