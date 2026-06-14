package adlists

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
)

// Column layout constants. The table has six columns in a fixed order:
// Type, Address, Comment, On, Domains, Status. Type/On/Domains/Status take a
// fixed content width; Address and Comment share the remaining space by weight.
const (
	numCols = 6

	colTypeWidth    = 6 // "block" / "allow"
	colEnabledWidth = 3 // ● / ○
	colDomainsWidth = 7 // "Domains" header
	colStatusWidth  = 8 // "unknown" longest label

	cellPad    = 2 // default table cell padding (left+right)
	minFlexCol = 8 // minimum width for a flexible column
	ellipsis   = "…"

	enabledGlyph  = "●"
	disabledGlyph = "○"

	tokenBlock   = "block"
	tokenAllow   = "allow"
	tokenWarn    = "warn"
	tokenNeutral = "neutral"
)

// columnTitles is the header row, in render order.
var columnTitles = []string{"Type", "Address", "Comment", "On", "Domains", "Status"}

// computeColumnWidths returns the content width of each column for a given total
// width. Widths never sum past the available (padding-adjusted) space, and each
// flexible column is clamped to a sane minimum so nothing collapses.
func computeColumnWidths(total int) []int {
	avail := total - numCols*cellPad
	fixed := colTypeWidth + colEnabledWidth + colDomainsWidth + colStatusWidth

	flex := avail - fixed
	minFlex := 2 * minFlexCol
	if flex < minFlex {
		flex = minFlex
	}

	// Address gets two-thirds, Comment the remaining third.
	address := (flex * 2) / 3
	comment := flex - address

	address = clampMin(address, minFlexCol)
	comment = clampMin(comment, minFlexCol)

	return []int{colTypeWidth, address, comment, colEnabledWidth, colDomainsWidth, colStatusWidth}
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

// truncate shortens s to at most width display cells, appending an ellipsis when
// it must cut. Rune-aware so multibyte addresses are not split mid-rune.
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

// enabledCell returns the on/off glyph for a list's enabled state.
func enabledCell(enabled bool) string {
	if enabled {
		return enabledGlyph
	}
	return disabledGlyph
}

// statusLabel maps a gravity list's integer status to a short human label.
// 0 unknown, 1 ok, 2 warn, 3 error; anything else is treated as unknown.
func statusLabel(status int) string {
	switch status {
	case 1:
		return "ok"
	case 2:
		return "warn"
	case 3:
		return "error"
	default:
		return "unknown"
	}
}

// statusToken classifies a list status integer into a semantic style token.
func statusToken(status int) string {
	switch status {
	case 1:
		return tokenAllow
	case 2:
		return tokenWarn
	case 3:
		return tokenBlock
	default:
		return tokenNeutral
	}
}

// typeToken classifies a list type string into a semantic style token.
func typeToken(listType string) string {
	switch strings.ToLower(strings.TrimSpace(listType)) {
	case string(pihole.ListBlock):
		return tokenBlock
	case string(pihole.ListAllow):
		return tokenAllow
	default:
		return tokenNeutral
	}
}

// listToRow maps a single list into the ordered cell strings for the table,
// truncating the free-text columns to their computed widths. Pure: no styling.
func listToRow(l pihole.List, widths []int) []string {
	addressW, commentW := minFlexCol, minFlexCol
	if len(widths) == numCols {
		addressW, commentW = widths[1], widths[2]
	}

	typeLabel := l.Type
	if strings.TrimSpace(typeLabel) == "" {
		typeLabel = "-"
	}

	return []string{
		typeLabel,
		truncate(l.Address, addressW),
		truncate(l.Comment, commentW),
		enabledCell(l.Enabled),
		strconv.Itoa(l.Number),
		statusLabel(l.Status),
	}
}

// styleForToken resolves a style token to a lipgloss style using the live theme.
func styleForToken(th *theme.Theme, token string) lipgloss.Style {
	switch token {
	case tokenBlock:
		return lipgloss.NewStyle().Foreground(th.Block)
	case tokenAllow:
		return lipgloss.NewStyle().Foreground(th.Allow)
	case tokenWarn:
		return lipgloss.NewStyle().Foreground(th.Warn)
	default:
		return lipgloss.NewStyle().Foreground(th.Text)
	}
}

// styledRows builds table rows from lists, colouring the Type, On, and Status
// cells by their tokens using the current theme. Read at View() time.
func styledRows(th *theme.Theme, lists []pihole.List, widths []int) []table.Row {
	rows := make([]table.Row, len(lists))
	for i, l := range lists {
		cells := listToRow(l, widths)
		cells[0] = styleForToken(th, typeToken(l.Type)).Render(cells[0])
		enabledTok := tokenAllow
		if !l.Enabled {
			enabledTok = tokenNeutral
		}
		cells[3] = styleForToken(th, enabledTok).Render(cells[3])
		cells[5] = styleForToken(th, statusToken(l.Status)).Render(cells[5])
		rows[i] = table.Row(cells)
	}
	return rows
}

// parseGroups parses a comma-separated list of integer group IDs, skipping
// blank and non-numeric entries. Pure and order-preserving.
func parseGroups(csv string) []int {
	parts := strings.Split(csv, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}

// groupsOrDefault returns parsed groups, falling back to the default group 0
// when none were provided (PiHole assigns new lists to group 0 by default).
func groupsOrDefault(csv string) []int {
	g := parseGroups(csv)
	if len(g) == 0 {
		return []int{0}
	}
	return g
}

// groupsCSV renders a group-ID slice back to a comma-separated string.
func groupsCSV(groups []int) string {
	parts := make([]string, len(groups))
	for i, g := range groups {
		parts[i] = strconv.Itoa(g)
	}
	return strings.Join(parts, ",")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
