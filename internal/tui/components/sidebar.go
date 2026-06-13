// Package components holds reusable presentational pieces of the app chrome:
// the sidebar, status bar, banners/toasts, and confirmation prompts.
package components

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
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
}

// SidebarWidth is the default rail width in columns.
const SidebarWidth = 20

// Render returns the styled sidebar for the given theme.
func (s Sidebar) Render(th *theme.Theme) string {
	width := s.Width
	if width == 0 {
		width = SidebarWidth
	}

	brand := lipgloss.NewStyle().
		Foreground(th.Accent).
		Bold(true).
		Padding(0, 2).
		MarginBottom(1).
		Render("● tihole")

	var rows []string
	rows = append(rows, brand)

	itemBase := lipgloss.NewStyle().Width(width-2).Padding(0, 1)
	for i, item := range s.Items {
		label := item.Icon + "  " + item.Title
		if i == s.Selected {
			rows = append(rows, itemBase.
				Foreground(th.Surface).
				Background(th.Accent).
				Bold(true).
				Render(label))
		} else {
			rows = append(rows, itemBase.
				Foreground(th.Text).
				Render(label))
		}
	}

	body := strings.Join(rows, "\n")

	return lipgloss.NewStyle().
		Width(width).
		Height(s.Height).
		Background(th.Panel).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(th.Border).
		Render(body)
}
