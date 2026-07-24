package components

import (
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

// StatusBar renders the top chrome: the active instance and the current screen
// title. Blocking state lives on the sidebar rail now, not here. Pure view
// helper driven by the root model.
type StatusBar struct {
	Instance   string
	ScreenName string
	Width      int
}

// Render returns the styled top status bar for the given theme.
func (b StatusBar) Render(th *theme.Theme) string {
	left := lipgloss.NewStyle().
		Foreground(th.Accent).
		Bold(true).
		Padding(0, 1).
		Render("⬢ " + orDash(b.Instance))

	center := lipgloss.NewStyle().
		Foreground(th.Text).
		Padding(0, 1).
		Render(b.ScreenName)

	row := lipgloss.JoinHorizontal(lipgloss.Top, left, center)

	return lipgloss.NewStyle().
		Width(b.Width).
		Background(th.Panel).
		Render(row)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
