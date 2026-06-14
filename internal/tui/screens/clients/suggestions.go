package clients

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
)

// suggestState holds the suggestions-pane state: candidate clients seen in
// queries but not yet configured (GET /api/clients/_suggestions), plus the
// list cursor and its own loading/error flags.
type suggestState struct {
	active      bool
	loading     bool
	suggestions []pihole.ClientSuggestion
	cursor      int
	err         error
}

// selected returns the highlighted suggestion, if any.
func (s *suggestState) selected() (pihole.ClientSuggestion, bool) {
	if s.cursor >= 0 && s.cursor < len(s.suggestions) {
		return s.suggestions[s.cursor], true
	}
	return pihole.ClientSuggestion{}, false
}

// moveUp / moveDown clamp the cursor within the list bounds.
func (s *suggestState) moveUp() {
	if s.cursor > 0 {
		s.cursor--
	}
}

func (s *suggestState) moveDown() {
	if s.cursor < len(s.suggestions)-1 {
		s.cursor++
	}
}

// render draws the suggestions list as a centered panel. Each row shows the
// candidate address, its resolved name, MAC vendor, and last-seen time.
func (s *suggestState) render(th *theme.Theme, spin string, w, h int) string {
	var lines []string
	lines = append(lines, th.AccentStyle().Bold(true).Render("Client suggestions"), "")

	switch {
	case s.loading:
		lines = append(lines, th.SubtleStyle().Render(spin+" loading suggestions…"))
	case s.err != nil:
		lines = append(lines, th.BlockStyle().Bold(true).Render("error: "+s.err.Error()))
	case len(s.suggestions) == 0:
		lines = append(lines, th.SubtleStyle().Render("no unconfigured clients seen"))
	default:
		for i, sg := range s.suggestions {
			lines = append(lines, s.rowLine(th, i, sg))
		}
	}

	lines = append(lines, "", th.SubtleStyle().Render("↑↓ move · enter add · esc close"))

	panel := th.PanelStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Accent).
		Padding(1, 2).
		Width(maxInt(minInt(w-2, 72), 24)).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Top, panel)
}

// rowLine formats one suggestion, marking the cursor row with the accent color.
func (s *suggestState) rowLine(th *theme.Theme, i int, sg pihole.ClientSuggestion) string {
	addr := orDash(firstAddress(sg.Addresses))
	meta := suggestionMeta(sg)

	label := addr
	if meta != "" {
		label = fmt.Sprintf("%-24s %s", addr, meta)
	}

	if i == s.cursor {
		return th.AccentStyle().Render("› " + label)
	}
	return th.TextStyle().Render("  " + label)
}

// suggestionMeta joins the optional name, vendor, and last-seen fields into a
// compact secondary label.
func suggestionMeta(sg pihole.ClientSuggestion) string {
	var parts []string
	if n := strings.TrimSpace(sg.Name); n != "" {
		parts = append(parts, n)
	}
	if v := strings.TrimSpace(sg.MACVendor); v != "" {
		parts = append(parts, v)
	}
	if sg.LastQuery > 0 {
		parts = append(parts, "last "+formatLastSeen(sg.LastQuery))
	}
	return strings.Join(parts, " · ")
}

// formatLastSeen renders a unix timestamp as a short local date-time.
func formatLastSeen(unix int64) string {
	return time.Unix(unix, 0).Format("Jan 2 15:04")
}
