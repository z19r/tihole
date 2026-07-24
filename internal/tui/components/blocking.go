package components

import (
	"fmt"

	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

// BlockState describes the current global blocking status. It drives the
// sidebar's blocking control (the rail's most prominent live indicator).
type BlockState struct {
	// Known is false until the first fetch resolves.
	Known bool
	// Enabled is true when blocking is active.
	Enabled bool
	// CountdownSecs, when > 0, shows a temporary-disable countdown.
	CountdownSecs int
}

// blockingPill renders the sidebar's blocking control: a colour-coded status
// line plus the toggle-key hint. width is the rail's full content width.
func blockingPill(th *theme.Theme, b BlockState, width int) string {
	inner := width - 2
	if inner < 1 {
		inner = 1
	}
	base := lipgloss.NewStyle().Width(inner).Padding(0, 1)

	var status string
	switch {
	case !b.Known:
		status = lipgloss.NewStyle().Foreground(th.Subtle).Render("● blocking …")
	case b.Enabled:
		status = lipgloss.NewStyle().Foreground(th.Allow).Bold(true).Render("● blocking on")
	default:
		label := "○ blocking off"
		if b.CountdownSecs > 0 {
			label = "○ off " + humanCountdown(b.CountdownSecs)
		}
		status = lipgloss.NewStyle().Foreground(th.Block).Bold(true).Render(label)
	}

	hint := lipgloss.NewStyle().Foreground(th.Subtle).Render("d toggle")
	return base.Render(status + "\n" + hint)
}

// humanCountdown formats seconds as a compact mm:ss / Ns countdown.
func humanCountdown(secs int) string {
	if secs < 0 {
		secs = 0
	}
	if secs < 60 {
		return fmt.Sprintf("(%ds)", secs)
	}
	return fmt.Sprintf("(%dm%02ds)", secs/60, secs%60)
}
