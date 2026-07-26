# Key Bindings

tihole is fully keyboard-driven. This is the complete reference. The compact
help bar at the bottom of the screen always shows the most relevant keys, and
`?` opens a context-aware overlay with the global keys plus whatever the active
screen adds.

## The focus model

tihole borrows an [Omarchy](https://omarchy.org/)-style two-zone focus model.
At any moment input belongs to one of two zones:

- **Rail** — the sidebar of screens down the left. This is where focus starts.
- **Panel** — the content of the active screen (a table, a form, a log tail).

You move the selection on the rail, then *descend* into the panel to act on its
contents, and press `esc` to *climb* back to the rail. A handful of keys are
**global** and work in both zones; everything else is scoped to the zone that
owns input, so a screen's keys can never be ambushed by a global shortcut and
vice-versa.

## Global keys (both zones)

These are alive everywhere — on the rail, inside a panel, on any screen.

| Key | Action |
| --- | --- |
| `ctrl+k` | Open the command palette (fuzzy-jump to any screen, toggle blocking, change theme, switch instance) |
| `s` | Switch to the next configured instance |
| `d` | Toggle Pi-hole blocking on the active instance |
| `ctrl+t` | Cycle to the next theme |
| `i` | Replay the ASCII boot splash |
| `?` | Toggle the help overlay |
| `q` / `ctrl+c` | Quit |

> **Note:** while a text field is focused (adding or editing a record), single-key
> globals are suppressed and the keystroke goes to the field instead — so typing
> `s` or `d` into a form does what you'd expect. `ctrl+c` still quits.

## On the rail

The rail owns input on startup and whenever you `esc` out of a panel. It never
forwards keys to the screen.

| Key | Action |
| --- | --- |
| `↑` / `k` | Move selection to the previous screen |
| `↓` / `j` | Move selection to the next screen |
| `1`–`9` | Jump straight to a screen by its sidebar position |
| `enter` / `→` / `l` / `tab` | Descend into the panel (interactive screens only) |
| `r` | Refresh / refetch the active screen |

Read-only screens (like the Dashboard) opt out of panel focus, so `enter`/`→`/
`tab` stay on the rail instead of trapping you in a panel where nothing responds.

## In a panel

Once you've descended, the screen owns input. Only one key is reserved:

| Key | Action |
| --- | --- |
| `esc` | Climb back to the rail |

Everything else falls through to the active screen. The common list/table keys:

| Key | Action |
| --- | --- |
| `↑↓` / `j` `k` | Move within the list |
| `enter` | Select / open the highlighted row |

### Per-screen actions

Management screens (Domains, Groups, Clients, Adlists, Local DNS, …) add their
own bindings on top. The conventional set:

| Key | Action |
| --- | --- |
| `a` | Add a new record |
| `e` | Edit the highlighted record |
| `x` | Delete the highlighted record |
| `r` | Refresh the list |

The Query Log adds one-key allow/block on the highlighted query. The exact set
for the screen you're on is always shown in the help bar and the `?` overlay,
which is the source of truth — this table documents the convention, not an
exhaustive per-screen list.

## Command palette (`ctrl+k`)

The palette is the fastest way around. Open it from anywhere and fuzzy-type:

- a screen name to jump there,
- `blocking` to toggle blocking,
- a theme name to switch themes,
- an instance name to switch instances.
