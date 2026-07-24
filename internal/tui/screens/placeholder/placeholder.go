// Package placeholder provides a stand-in Screen for pages not yet implemented,
// so the whole app is navigable from the first build.
package placeholder

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

// Model is a minimal Screen that renders a "coming soon" panel.
type Model struct {
	ctx   *core.AppContext
	title string
	w, h  int
}

// New returns a placeholder screen with the given title.
func New(ctx *core.AppContext, title string) *Model {
	return &Model{ctx: ctx, title: title}
}

func (m *Model) Init() tea.Cmd                       { return nil }
func (m *Model) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *Model) Title() string                       { return m.title }
func (m *Model) Focus() tea.Cmd                      { return nil }
func (m *Model) Blur()                               {}
func (m *Model) Help() []key.Binding                 { return nil }
func (m *Model) SetSize(w, h int)                    { m.w, m.h = w, h }

func (m *Model) View() tea.View {
	th := m.ctx.Theme
	heading := th.AccentStyle().Bold(true).Render(m.title)
	sub := th.SubtleStyle().Render("This screen isn't built yet — coming soon.")
	body := lipgloss.JoinVertical(lipgloss.Center, heading, "", sub)

	content := lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, body)
	return tea.NewView(th.SurfaceStyle().Width(m.w).Height(m.h).Render(content))
}
