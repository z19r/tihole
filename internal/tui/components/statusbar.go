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

// Render returns the styled top status bar for the given theme: a filled brand
// badge, the active instance, the current screen, and a right-aligned theme
// segment — all sitting as colored blocks on the panel, lipgloss-style.
func (b StatusBar) Render(th *theme.Theme) string {
	// A filled brand badge anchors the bar the way lipgloss's "STATUS" pill
	// does.
	badge := lipgloss.NewStyle().
		Foreground(th.Surface).
		Background(th.Accent).
		Bold(true).
		Padding(0, 1).
		Render("⬢ TIHOLE")

	// Interior segments carry the panel background explicitly. A fill-only
	// style
	// (fg without bg) resets to the terminal's own background after its run, so
	// the panel band would otherwise bleed between the badge and the segments.
	instance := lipgloss.NewStyle().
		Foreground(th.Accent).
		Background(th.Panel).
		Bold(true).
		Padding(0, 1).
		Render(orDash(b.Instance))

	center := lipgloss.NewStyle().
		Foreground(th.Text).
		Background(th.Panel).
		Bold(true).
		Padding(0, 1).
		Render(b.ScreenName)

	// Advertise the active theme and how to change it as a filled segment, so
	// appearance stops being a palette-only secret.
	right := lipgloss.NewStyle().
		Foreground(th.Text).
		Background(th.Border).
		Padding(0, 1).
		Render("◐ " + th.Name + " ⌃T")

	left := lipgloss.JoinHorizontal(lipgloss.Top, badge, instance, center)

	gap := b.Width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	spacer := lipgloss.NewStyle().Background(th.Panel).Width(gap).Render("")

	row := lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)

	return lipgloss.NewStyle().
		Width(b.Width).
		Background(th.Panel).
		MaxWidth(b.Width).
		Render(row)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
