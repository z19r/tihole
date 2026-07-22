package components

import (
	"fmt"

	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/theme"
)

// StatusBar renders the top chrome: the active instance, a persistent
// blocking-status pill, the current screen title, and the active theme (with
// its cycle-key hint) on the right. The blocking pill rides the top bar on
// EVERY screen so "protection is off" can never hide on another tab. Pure view
// helper driven by the root model.
type StatusBar struct {
	Instance   string
	ScreenName string
	Width      int
	// Blocking is the current global blocking status, surfaced as an always-on
	// pill. Its zero value (Known == false) renders a muted "blocking …" while
	// the first fetch is in flight.
	Blocking BlockState
}

// blockingPill renders the always-on blocking indicator: a calm green badge
// when protection is on, and an alarming filled-red badge (with the
// temporary-disable countdown, when running) when it is off — so a disabled
// state stands out from anywhere in the app.
func (b StatusBar) blockingPill(th *theme.Theme) string {
	base := lipgloss.NewStyle().Bold(true).Padding(0, 1)

	if !b.Blocking.Known {
		return base.
			Foreground(th.Subtle).
			Background(th.Panel).
			Render("◌ blocking …")
	}
	if b.Blocking.Enabled {
		return base.
			Foreground(th.Surface).
			Background(th.Allow).
			Render("● BLOCKING ON")
	}

	label := "⚠ BLOCKING OFF"
	if b.Blocking.CountdownSecs > 0 {
		label = fmt.Sprintf("⚠ OFF %s", fmtCountdown(b.Blocking.CountdownSecs))
	}
	return base.
		Foreground(th.Surface).
		Background(th.Block).
		Render(label)
}

// fmtCountdown renders remaining seconds as m:ss (or h:mm:ss past an hour).
func fmtCountdown(secs int) string {
	if secs < 0 {
		secs = 0
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
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

	pill := b.blockingPill(th)

	left := lipgloss.JoinHorizontal(
		lipgloss.Top,
		badge,
		pill,
		instance,
		center,
	)

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
