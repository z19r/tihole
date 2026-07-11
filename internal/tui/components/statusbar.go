package components

import (
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

// StatusBar renders the top chrome: the active instance, the current screen
// title, and the active theme (with its cycle-key hint) on the right. Blocking
// is its own pane now. Pure view helper driven by the root model.
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

	// Advertise the active theme and how to change it, so appearance stops being
	// a palette-only secret.
	right := lipgloss.NewStyle().
		Foreground(th.Subtle).
		Padding(0, 1).
		Render("◐ " + th.Name + " (⌃T)")

	gap := b.Width - lipgloss.Width(left) - lipgloss.Width(center) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	spacer := lipgloss.NewStyle().Width(gap).Render("")

	row := lipgloss.JoinHorizontal(lipgloss.Top, left, center, spacer, right)

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
