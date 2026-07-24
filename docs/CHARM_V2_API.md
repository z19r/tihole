# Charm v2 API cheat-sheet (verified against installed modules)

**Import paths are `charm.land/*/v2`** — NOT `github.com/charmbracelet/*`.
The modules declare their canonical path as `charm.land`. Mixing `github.com`
paths will fail with "module declares its path as charm.land/...".

Installed versions: bubbletea v2.0.8, bubbles v2.1.1, lipgloss v2.0.5, huh v2.0.3.

## bubbletea/v2 (`import tea "charm.land/bubbletea/v2"`)

```go
type Model interface {
    Init() tea.Cmd
    Update(tea.Msg) (tea.Model, tea.Cmd)
    View() tea.View            // NOTE: returns tea.View, NOT string
}
```

- `View` is a struct: `tea.View{ Content string, ... }`. Build one with
  `tea.NewView("rendered string")`. A screen's `View()` returns `tea.NewView(s)`.
- Key input: `case tea.KeyPressMsg:` — it wraps `tea.Key{ Text string, Mod KeyMod, Code rune }`.
  Use `msg.String()` for a matchable keystroke (e.g. "ctrl+k", "enter", "q").
  Prefer matching via `key.Matches(msg, binding)` from bubbles `key`.
- Resize: `case tea.WindowSizeMsg:` with `.Width`, `.Height`.
- Background color: `case tea.BackgroundColorMsg:` → `msg.IsDark() bool`. Flows through
  Update automatically on start; use it to drive `lipgloss.LightDark`.
- Program: `tea.NewProgram(model, opts...)`; `p.Run() (tea.Model, error)`.
  Common opts: `tea.WithAltScreen()`, `tea.WithContext(ctx)`.
- Quit: return `tea.Quit` command.

## lipgloss/v2 (`import "charm.land/lipgloss/v2"`)

- `lipgloss.Color("#0000ff")` / `lipgloss.Color("1")` → `color.Color`.
- Adaptive: `ld := lipgloss.LightDark(isDark)` then `ld(lightColor, darkColor)` picks one.
  Get `isDark` from `tea.BackgroundColorMsg.IsDark()`.
- `lipgloss.NewStyle().Foreground(c).Background(c).Bold(true).Padding(...).Border(...)`.
- Layout: `lipgloss.JoinHorizontal(pos, ...)`, `lipgloss.JoinVertical(pos, ...)`,
  `lipgloss.Place(w, h, hPos, vPos, str)`. `lipgloss.Width(s)`, `lipgloss.Height(s)`.

## bubbles/v2 (`import "charm.land/bubbles/v2/<component>"`)

- Components: `list`, `table`, `viewport`, `textinput`, `spinner`, `help`, `paginator`,
  `key`, `cursor`.
- `key`: `key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))`;
  `key.Matches(msg tea.KeyPressMsg, b ...key.Binding) bool`.
- Each component's `Update` takes `tea.Msg` and returns `(Model, tea.Cmd)` (v2 signature).
- `spinner.New()`, `s.Tick` is the cmd; `table.New(...)`, `list.New(...)`.

## huh/v2 (`import "charm.land/huh/v2"`)

- `huh.NewForm(huh.NewGroup(huh.NewInput().Title("...").Value(&s), ...))`.
- Run standalone: `form.Run() error`. Or embed as a bubbletea model.
- Fields: `NewInput()`, `NewSelect[T]()`, `NewConfirm()`, `.Validate(func(string) error)`,
  `.EchoMode(huh.EchoModePassword)` for secrets.

## Idioms for this project

- Domain packages (`pihole`, `config`) MUST NOT import bubbletea/lipgloss.
- All I/O happens inside `tea.Cmd` closures, never in `Update`.
- `View()` returns `tea.NewView(renderedString)`.
