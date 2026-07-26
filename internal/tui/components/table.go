package components

import (
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/theme"
)

// TableStyles returns theme-aware styles for a bubbles table so every cell —
// including the width padding the table adds to each column — carries an
// explicit background. Without this the table uses library defaults that set no
// background, and each padded cell leaks whatever is behind the terminal (a
// solid colour, or a background image) straight through the list.
func TableStyles(th *theme.Theme) table.Styles {
	return table.Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(th.Subtle).
			Background(th.Panel).
			Padding(0, 1),
		Cell: lipgloss.NewStyle().
			Foreground(th.Text).
			Background(th.Surface).
			Padding(0, 1),
		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(th.Surface).
			Background(th.Accent).
			Padding(0, 1),
	}
}
