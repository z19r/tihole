package components

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/theme"
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
	// Right is an optional right-aligned summary (counts, totals). Empty hides
	// it.
	Right string
	// Width is the available line width, used to right-align Right.
	Width int
}

// Render returns the single-line tab bar. Callers typically stack an error
// banner or spacer beneath it.
func (s SectionTabs) Render(th *theme.Theme) string {
	// A segmented control: the active chip is a filled brand pill, the rest sit
	// on
	// the raised panel so the whole bar reads as one glossy switch rather than
	// loose words. A hairline between chips keeps the segments legible.
	//
	// Every filler — the inter-chip spaces, the title gutter, the gap before
	// the right summary — is rendered *with* a background. A plain space sits
	// after a styled run's reset and paints in the terminal's own background,
	// so it would
	// bleed through as a gap; carrying a background keeps the band continuous.
	surface := lipgloss.NewStyle().Background(th.Surface)
	panelBG := lipgloss.NewStyle().Background(th.Panel)
	active := lipgloss.NewStyle().
		Foreground(th.Surface).
		Background(th.Accent).
		Bold(true)
	inactive := lipgloss.NewStyle().Foreground(th.Subtle).Background(th.Panel)
	sep := lipgloss.NewStyle().
		Foreground(th.Border).
		Background(th.Panel).
		Render("│")

	chips := make([]string, 0, len(s.Labels)*2)
	for i, l := range s.Labels {
		if i > 0 && i != s.Active && (i-1) != s.Active {
			chips = append(chips, sep)
		} else if i > 0 {
			chips = append(chips, panelBG.Render(" "))
		}
		label := " " + l + " "
		if i == s.Active {
			chips = append(chips, active.Render(label))
		} else {
			chips = append(chips, inactive.Render(label))
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Center, chips...)

	left := bar
	if s.Title != "" {
		title := lipgloss.NewStyle().
			Foreground(th.Accent).
			Background(th.Surface).
			Bold(true).
			Render(s.Title)
		left = lipgloss.JoinHorizontal(
			lipgloss.Center,
			title,
			surface.Render("  "),
			bar,
		)
	}

	line := left
	if s.Right != "" {
		right := lipgloss.NewStyle().
			Foreground(th.Subtle).
			Background(th.Surface).
			Render(s.Right)
		gap := s.Width - lipgloss.Width(left) - lipgloss.Width(s.Right)
		if gap < 1 {
			gap = 1
		}
		line = lipgloss.JoinHorizontal(
			lipgloss.Center,
			left,
			surface.Render(strings.Repeat(" ", gap)),
			right,
		)
	}

	// Clamp to width with MaxWidth only: an oversized bar (many chips on a
	// narrow terminal) is truncated rather than wrapping onto a second line,
	// which would
	// throw off every caller's header-height math. MaxWidth (not Width) avoids
	// soft-wrapping. Every segment already carries its own background, and the
	// caller paints the surrounding surface, so no trailing fill is needed
	// here.
	if s.Width > 0 {
		return lipgloss.NewStyle().MaxWidth(s.Width).Render(line)
	}
	return line
}
