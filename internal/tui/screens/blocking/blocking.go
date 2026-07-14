// Package blocking is the Blocking screen: a first-class pane for reading the
// global DNS-blocking status and changing it, including timed disables. It puts
// what used to be palette-only actions (disable for N minutes) in plain sight.
package blocking

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

const fetchTimeout = 8 * time.Second

// action is one selectable row. secs semantics: enable=true re-enables
// blocking;
// otherwise blocking is disabled for secs seconds (0 = indefinitely).
type action struct {
	label  string
	enable bool
	secs   float64
}

// actions is the fixed, always-visible menu — the whole point of the pane.
var actions = []action{
	{label: "Enable blocking", enable: true},
	{label: "Disable for 5 minutes", secs: 300},
	{label: "Disable for 10 minutes", secs: 600},
	{label: "Disable for 15 minutes", secs: 900},
	{label: "Disable indefinitely", secs: 0},
}

// statusMsg carries a fetched/refreshed blocking status (or an error).
type statusMsg struct {
	status pihole.BlockingStatus
	err    error
}

// Model is the Blocking screen.
type Model struct {
	ctx    *core.AppContext
	w, h   int
	cursor int

	known   bool
	status  pihole.BlockingStatus
	loading bool
	err     error

	cancel context.CancelFunc
}

// New builds the Blocking screen.
func New(ctx *core.AppContext) *Model { return &Model{ctx: ctx} }

func (m *Model) Init() tea.Cmd    { return nil }
func (m *Model) Title() string    { return "Blocking" }
func (m *Model) SetSize(w, h int) { m.w, m.h = w, h }

// Help returns the screen-specific bindings for the help bar.
func (m *Model) Help() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

// Focus (re)loads the current blocking status each time the pane is shown.
func (m *Model) Focus() tea.Cmd {
	m.loading = true
	m.err = nil
	return m.fetch()
}

// Blur cancels any in-flight request.
func (m *Model) Blur() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

// fetch loads the current status.
func (m *Model) fetch() tea.Cmd {
	api := m.ctx.API
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		st, err := api.Blocking(ctx)
		return statusMsg{status: st, err: err}
	}
}

// apply performs the selected action, then refreshes the status.
func (m *Model) apply(a action) tea.Cmd {
	api := m.ctx.API
	var timer *float64
	if !a.enable && a.secs > 0 {
		s := a.secs
		timer = &s
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		if err := api.SetBlocking(ctx, a.enable, timer); err != nil {
			return statusMsg{err: err}
		}
		st, err := api.Blocking(ctx)
		return statusMsg{status: st, err: err}
	}
}

// Update handles navigation, applying an action, and status results.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.known = true
		m.status = msg.status
		return m, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.ctx.Keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.ctx.Keys.Down):
			if m.cursor < len(actions)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.ctx.Keys.Enter):
			m.loading = true
			return m, m.apply(actions[m.cursor])
		case key.Matches(msg, m.ctx.Keys.Refresh):
			m.loading = true
			return m, m.fetch()
		}
	}
	return m, nil
}

// View renders the status banner and the action menu.
func (m *Model) View() tea.View {
	th := m.ctx.Theme

	header := th.AccentStyle().Bold(true).Render("Blocking")
	status := m.statusLine(th)
	menu := m.renderMenu(th)

	body := strings.Join([]string{header, "", status, "", menu}, "\n")
	content := lipgloss.Place(m.w, m.h, lipgloss.Left, lipgloss.Top,
		lipgloss.NewStyle().Background(th.Surface).Padding(1, 2).Render(body),
		th.SurfaceWhitespace())
	return tea.NewView(th.SurfaceStyle().Width(m.w).Height(m.h).Render(content))
}

// statusLine renders the current blocking state prominently.
func (m *Model) statusLine(th *theme.Theme) string {
	if m.err != nil {
		return th.BlockStyle().Bold(true).Render("error: " + m.err.Error())
	}
	if !m.known {
		return th.SubtleStyle().Render("loading status…")
	}
	if m.status.Blocking {
		return th.AllowStyle().Bold(true).Render("● blocking is ON")
	}
	label := "○ blocking is OFF"
	if m.status.Timer != nil && *m.status.Timer > 0 {
		label += "  — re-enables in " + humanCountdown(int(*m.status.Timer))
	}
	return th.BlockStyle().Bold(true).Render(label)
}

// renderMenu renders the action list with the cursor row highlighted.
func (m *Model) renderMenu(th *theme.Theme) string {
	sel := lipgloss.NewStyle().
		Foreground(th.Surface).
		Background(th.Accent).
		Bold(true).
		Padding(0, 1)
	row := lipgloss.NewStyle().
		Foreground(th.Text).
		Background(th.Surface).
		Padding(0, 1)

	lines := make([]string, len(actions))
	for i, a := range actions {
		marker := "  "
		if i == m.cursor {
			marker = "▸ "
		}
		if i == m.cursor {
			lines[i] = sel.Render(marker + a.label)
		} else {
			lines[i] = row.Render(marker + a.label)
		}
	}
	return strings.Join(lines, "\n")
}

// humanCountdown formats seconds as a compact mm:ss / Ns countdown.
func humanCountdown(secs int) string {
	if secs < 0 {
		secs = 0
	}
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	return fmt.Sprintf("%dm%02ds", secs/60, secs%60)
}
