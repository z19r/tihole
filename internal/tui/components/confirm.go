package components

import (
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

// ConfirmDialog is a lightweight, centered modal used by CRUD screens to guard
// destructive actions (delete, flush, gravity update). It carries only display
// state; the owning screen interprets the y/n keys and clears Active.
type ConfirmDialog struct {
	// Active is true while the dialog is showing.
	Active bool
	// Title is the bold heading (e.g. "Delete domain?").
	Title string
	// Message is the body text (e.g. the item being acted on).
	Message string
	// Danger renders the confirm affordance in the Block color for
	// irreversible actions; otherwise the Accent color is used.
	Danger bool
}

// Show returns a copy of the dialog activated with the given title/message.
func (d ConfirmDialog) Show(title, message string, danger bool) ConfirmDialog {
	return ConfirmDialog{
		Active:  true,
		Title:   title,
		Message: message,
		Danger:  danger,
	}
}

// Hide returns a copy with Active cleared.
func (d ConfirmDialog) Hide() ConfirmDialog {
	return ConfirmDialog{}
}

// Render draws the modal centered within the given content area. Screens call
// this last so the dialog overlays their body. When inactive it returns "".
func (d ConfirmDialog) Render(th *theme.Theme, width, height int) string {
	if !d.Active {
		return ""
	}

	accent := th.Accent
	if d.Danger {
		accent = th.Block
	}

	title := lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Render(d.Title)

	body := lipgloss.NewStyle().
		Foreground(th.Text).
		Render(d.Message)

	hint := lipgloss.NewStyle().
		Foreground(th.Subtle).
		MarginTop(1).
		Render("y confirm   n / esc cancel")

	inner := lipgloss.JoinVertical(lipgloss.Left, title, "", body, hint)

	box := lipgloss.NewStyle().
		Background(th.Panel).
		Foreground(th.Text).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 3).
		Render(inner)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
