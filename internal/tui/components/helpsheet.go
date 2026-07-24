package components

import (
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

// helpSheetWidth caps the overlay width; sections wrap within it.
const helpSheetWidth = 64

// HelpKey is one key → description row in a help section.
type HelpKey struct {
	Key  string
	Desc string
}

// HelpSection is a titled group of key bindings (e.g. "Global", "Domains").
type HelpSection struct {
	Title string
	Keys  []HelpKey
}

// HelpSheet is a centered, full cheat-sheet overlay toggled with "?". It is a
// pure display component: the root owns the visible flag and passes in the
// sections to render, so it stays trivially testable.
type HelpSheet struct {
	Sections []HelpSection
}

// Render draws the help overlay centered within the given area. Returns "" when
// there are no sections to show.
func (h HelpSheet) Render(th *theme.Theme, width, height int) string {
	if len(h.Sections) == 0 {
		return ""
	}

	boxWidth := clampInt(width-8, 34, helpSheetWidth)
	inner := boxWidth - 6 // account for border + padding

	title := lipgloss.NewStyle().
		Foreground(th.Accent).
		Bold(true).
		Render("Keyboard shortcuts")

	rows := []string{title, ""}
	for _, sec := range h.Sections {
		rows = append(rows, h.renderSection(th, sec, inner))
	}
	rows = append(rows, th.SubtleStyle().Render("? or esc close"))

	box := lipgloss.NewStyle().
		Background(th.Panel).
		Foreground(th.Text).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Accent).
		Padding(1, 2).
		Width(boxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// renderSection renders a section heading followed by aligned key/desc rows.
func (h HelpSheet) renderSection(th *theme.Theme, sec HelpSection, width int) string {
	heading := lipgloss.NewStyle().
		Foreground(th.Text).
		Bold(true).
		Render(sec.Title)

	keyWidth := helpKeyColumnWidth(sec.Keys)
	keyStyle := lipgloss.NewStyle().Foreground(th.Accent).Width(keyWidth)
	descStyle := lipgloss.NewStyle().Foreground(th.Subtle)

	lines := []string{heading}
	for _, k := range sec.Keys {
		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			keyStyle.Render(k.Key),
			"  ",
			descStyle.Render(k.Desc),
		)
		lines = append(lines, row)
	}
	lines = append(lines, "")
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// helpKeyColumnWidth returns the widest key string in a section (min 6) so the
// description column aligns.
func helpKeyColumnWidth(keys []HelpKey) int {
	w := 6
	for _, k := range keys {
		if l := lipgloss.Width(k.Key); l > w {
			w = l
		}
	}
	return w
}
