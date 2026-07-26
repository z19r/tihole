// Package components holds reusable presentational pieces of the app chrome:
// the sidebar, status bar, banners/toasts, and confirmation prompts.
package components

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/theme"
)

// SidebarItem is one navigable entry rendered in the sidebar.
type SidebarItem struct {
	Title string
	Icon  string
}

// Sidebar renders the vertical navigation rail. It is a pure view helper: the
// root model owns selection state and passes it in.
type Sidebar struct {
	Items    []SidebarItem
	Selected int
	Width    int
	Height   int
	// Focused is true when the rail owns keyboard input (Nav focus). It drives
	// the visual: a live accent border and a bright selection when focused, a
	// muted treatment when the active screen holds focus instead.
	Focused bool
}

// SidebarWidth is the default rail width in columns.
const SidebarWidth = 20

// Render returns the styled sidebar for the given theme.
func (s Sidebar) Render(th *theme.Theme) string {
	width := s.Width
	if width == 0 {
		width = SidebarWidth
	}

	brand := gradientWordmark(th, s.Focused)

	var rows []string
	rows = append(rows, brand)

	// The selection glows on the accent when the rail is focused; when the
	// panel
	// owns input the rail steps back to a muted, bordered pill so it's clear
	// where the keyboard is going.
	selFg, selBg := th.Surface, th.Accent
	if !s.Focused {
		selFg, selBg = th.Text, th.Border
	}

	// itemBase carries the Panel background so the Width padding after each
	// (fg-only) label paints the rail surface instead of leaking the terminal
	// background through the trailing cells.
	itemBase := lipgloss.NewStyle().
		Width(width-2).
		Padding(0, 1).
		Background(th.Panel)
	for i, item := range s.Items {
		marker := "  "
		if i == s.Selected {
			marker = "▎ "
		}
		label := marker + item.Icon + " " + item.Title
		if i == s.Selected {
			rows = append(rows, itemBase.
				Foreground(selFg).
				Background(selBg).
				Bold(true).
				Render(label))
		} else {
			rows = append(rows, itemBase.
				Foreground(th.Text).
				Render(label))
		}
	}

	body := strings.Join(rows, "\n")

	// A live accent border signals the rail has focus; otherwise the quiet
	// panel border.
	border := th.Border
	if s.Focused {
		border = th.Accent
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(s.Height).
		Background(th.Panel).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(border).
		Render(body)
}
