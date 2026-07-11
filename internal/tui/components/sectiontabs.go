package components

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

// SectionTabs renders a screen's sub-section chip bar: a title, the full set of
// labeled tabs with the active one highlighted, and an optional right-aligned
// summary. Rendering *every* tab (not just the active one) is what makes a
// screen's hidden depth — alternate filters, record kinds, list types, tool
// panels — discoverable at a glance instead of buried behind a cycle key.
type SectionTabs struct {
	// Title is the bold screen name shown to the left of the chips.
	Title string
	// Labels is the ordered chip text, one per selectable sub-section.
	Labels []string
	// Active is the index into Labels of the highlighted chip.
	Active int
	// Right is an optional right-aligned summary (counts, totals). Empty hides it.
	Right string
	// Width is the available line width, used to right-align Right.
	Width int
}

// Render returns the single-line tab bar. Callers typically stack an error
// banner or spacer beneath it.
func (s SectionTabs) Render(th *theme.Theme) string {
	active := lipgloss.NewStyle().Foreground(th.Surface).Background(th.Accent).Bold(true)

	chips := make([]string, len(s.Labels))
	for i, l := range s.Labels {
		label := " " + l + " "
		if i == s.Active {
			chips[i] = active.Render(label)
		} else {
			chips[i] = th.SubtleStyle().Render(label)
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Center, chips...)

	left := bar
	if s.Title != "" {
		left = lipgloss.JoinHorizontal(lipgloss.Center, th.AccentStyle().Bold(true).Render(s.Title), "  ", bar)
	}

	line := left
	if s.Right != "" {
		gap := s.Width - lipgloss.Width(left) - lipgloss.Width(s.Right)
		if gap < 1 {
			gap = 1
		}
		line = left + strings.Repeat(" ", gap) + s.Right
	}

	// Clamp to the available width so an oversized bar (many chips on a narrow
	// terminal) is truncated rather than wrapping onto a second line, which would
	// throw off every caller's header-height math.
	if s.Width > 0 {
		line = lipgloss.NewStyle().MaxWidth(s.Width).Render(line)
	}
	return line
}
