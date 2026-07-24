package components

import (
	"fmt"

	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

// BlockState describes the current global blocking status for the status bar.
type BlockState struct {
	// Known is false until the first fetch resolves.
	Known bool
	// Enabled is true when blocking is active.
	Enabled bool
	// CountdownSecs, when > 0, shows a temporary-disable countdown.
	CountdownSecs int
}

// StatusBar renders the top chrome: active instance, current screen title, and
// blocking state. Pure view helper driven by the root model.
type StatusBar struct {
	Instance   string
	ScreenName string
	Block      BlockState
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

	right := lipgloss.NewStyle().
		Padding(0, 1).
		Render(b.blockingBadge(th))

	// Distribute: left | center (flex) | right.
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

func (b StatusBar) blockingBadge(th *theme.Theme) string {
	if !b.Block.Known {
		return lipgloss.NewStyle().Foreground(th.Subtle).Render("blocking …")
	}
	if b.Block.Enabled {
		return lipgloss.NewStyle().Foreground(th.Allow).Bold(true).Render("● blocking on")
	}
	label := "○ blocking off"
	if b.Block.CountdownSecs > 0 {
		label = "○ off " + humanCountdown(b.Block.CountdownSecs)
	}
	return lipgloss.NewStyle().Foreground(th.Block).Bold(true).Render(label)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// humanCountdown formats seconds as a compact mm:ss / Ns countdown.
func humanCountdown(secs int) string {
	if secs < 60 {
		return fmt.Sprintf("(%ds)", secs)
	}
	return fmt.Sprintf("(%dm%02ds)", secs/60, secs%60)
}
