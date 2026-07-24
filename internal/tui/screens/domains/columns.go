package domains

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
)

// Column layout constants. The table has six columns in a fixed order:
// Type, Kind, Domain, Comment, Enabled, Groups. Type/Kind/Enabled/Groups take a
// fixed content width; Domain/Comment share the remaining space by weight.
const (
	numCols = 6

	colTypeWidth    = 5 // allow / deny
	colKindWidth    = 5 // exact / regex
	colEnabledWidth = 7 // "Enabled" header
	colGroupsWidth  = 6 // "Groups" header

	cellPad    = 2 // default table cell padding (left+right)
	minFlexCol = 8 // minimum width for a flexible column
	ellipsis   = "…"

	glyphEnabled  = "●"
	glyphDisabled = "○"
)

// columnTitles is the header row, in render order.
var columnTitles = []string{"Type", "Kind", "Domain", "Comment", "Enabled", "Groups"}

// computeColumnWidths returns the content width of each column for a given total
// width. Widths never sum past the available (padding-adjusted) space, and each
// flexible column is clamped to a sane minimum so nothing collapses.
func computeColumnWidths(total int) []int {
	avail := total - numCols*cellPad
	fixed := colTypeWidth + colKindWidth + colEnabledWidth + colGroupsWidth

	flex := avail - fixed
	minFlex := 2 * minFlexCol
	if flex < minFlex {
		flex = minFlex
	}

	// Domain gets the larger share; Comment the remainder.
	domain := flex / 2
	comment := flex - domain

	domain = clampMin(domain, minFlexCol)
	comment = clampMin(comment, minFlexCol)

	return []int{colTypeWidth, colKindWidth, domain, comment, colEnabledWidth, colGroupsWidth}
}

func clampMin(v, min int) int {
	if v < min {
		return min
	}
	return v
}

// tableColumns builds table.Column values from computed widths.
func tableColumns(widths []int) []table.Column {
	cols := make([]table.Column, len(columnTitles))
	for i, title := range columnTitles {
		w := colTypeWidth
		if i < len(widths) {
			w = widths[i]
		}
		cols[i] = table.Column{Title: title, Width: w}
	}
	return cols
}

// columnWidthsFrom extracts the content widths from the table's columns.
func columnWidthsFrom(cols []table.Column) []int {
	w := make([]int, len(cols))
	for i, c := range cols {
		w[i] = c.Width
	}
	return w
}

// truncate shortens s to at most width display cells, appending an ellipsis when
// it must cut. Rune-aware so multibyte domains are not split mid-rune.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width == 1 {
		return ellipsis
	}
	return string(r[:width-1]) + ellipsis
}

// displayDomain prefers a domain's Unicode presentation, falling back to its
// punycode/ASCII form.
func displayDomain(d pihole.Domain) string {
	if strings.TrimSpace(d.Unicode) != "" {
		return d.Unicode
	}
	return d.Domain
}

// enabledGlyph renders a filled/hollow dot for a domain's enabled state.
func enabledGlyph(enabled bool) string {
	if enabled {
		return glyphEnabled
	}
	return glyphDisabled
}

// domainToRow maps a single domain into the ordered cell strings for the table,
// truncating the free-text columns to their computed widths. Pure: no styling.
func domainToRow(d pihole.Domain, widths []int) []string {
	domainW, commentW := minFlexCol, minFlexCol
	if len(widths) == numCols {
		domainW, commentW = widths[2], widths[3]
	}

	comment := d.Comment
	if strings.TrimSpace(comment) == "" {
		comment = "-"
	}

	return []string{
		d.Type,
		d.Kind,
		truncate(displayDomain(d), domainW),
		truncate(comment, commentW),
		enabledGlyph(d.Enabled),
		strconv.Itoa(len(d.Groups)),
	}
}

// typeStyle resolves the allow/deny colour for the Type cell using the live
// theme: allow is rendered in the Allow (green) token, deny in Block (red).
func typeStyle(th *theme.Theme, dtype string) lipgloss.Style {
	if dtype == string(pihole.DomainDeny) {
		return lipgloss.NewStyle().Foreground(th.Block)
	}
	return lipgloss.NewStyle().Foreground(th.Allow)
}

// styledRows builds table rows from domains, colouring the Type cell by
// allow/deny and the Enabled cell by state, using the current theme. Read at
// View() time so live re-themes take effect immediately.
func styledRows(th *theme.Theme, domains []pihole.Domain, widths []int) []table.Row {
	rows := make([]table.Row, len(domains))
	for i, d := range domains {
		cells := domainToRow(d, widths)
		cells[0] = typeStyle(th, d.Type).Render(cells[0])
		if d.Enabled {
			cells[4] = lipgloss.NewStyle().Foreground(th.Allow).Render(cells[4])
		} else {
			cells[4] = lipgloss.NewStyle().Foreground(th.Subtle).Render(cells[4])
		}
		rows[i] = table.Row(cells)
	}
	return rows
}

// parseGroups parses a comma-separated list of group ids, ignoring blanks and
// non-numeric entries. Returns nil for an empty input.
func parseGroups(s string) []int {
	var out []int
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}

// groupsToString renders a group-id slice as a comma-separated string for
// pre-filling the edit form.
func groupsToString(groups []int) string {
	parts := make([]string, len(groups))
	for i, g := range groups {
		parts[i] = strconv.Itoa(g)
	}
	return strings.Join(parts, ", ")
}
