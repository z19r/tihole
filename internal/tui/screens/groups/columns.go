package groups

import (
	"strconv"

	"charm.land/bubbles/v2/table"

	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
)

// Column layout constants. The table has four columns in a fixed order:
// Name, Comment, Enabled, ID. Enabled/ID take a fixed content width; Name and
// Comment share the remaining space.
const (
	numCols = 4

	colEnabledWidth = 7 // "Enabled" header / ● or ○ glyph
	colIDWidth      = 6 // small integer id

	cellPad    = 2 // default table cell padding (left+right)
	minFlexCol = 6 // minimum width for a flexible column
	ellipsis   = "…"

	enabledGlyph  = "●"
	disabledGlyph = "○"
)

// columnTitles is the header row, in render order.
var columnTitles = []string{"Name", "Comment", "Enabled", "ID"}

// computeColumnWidths returns the content width of each column for a given
// total
// width. Widths never sum past the available (padding-adjusted) space, and each
// flexible column is clamped to a sane minimum so nothing collapses.
func computeColumnWidths(total int) []int {
	avail := total - numCols*cellPad
	fixed := colEnabledWidth + colIDWidth

	flex := avail - fixed
	minFlex := 2 * minFlexCol
	if flex < minFlex {
		flex = minFlex
	}

	name := flex / 2
	comment := flex - name

	name = clampMin(name, minFlexCol)
	comment = clampMin(comment, minFlexCol)

	return []int{name, comment, colEnabledWidth, colIDWidth}
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
		w := colEnabledWidth
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

// truncate shortens s to at most width display cells, appending an ellipsis
// when
// it must cut. Rune-aware so multibyte names are not split mid-rune.
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

// enabledLabel returns the glyph for a group's enabled state (pure, unstyled).
func enabledLabel(enabled bool) string {
	if enabled {
		return enabledGlyph
	}
	return disabledGlyph
}

// groupToRow maps a single group into the ordered cell strings for the table,
// truncating free-text columns to their computed widths. Pure: no styling.
func groupToRow(g pihole.Group, widths []int) []string {
	nameW, commentW := minFlexCol, minFlexCol
	if len(widths) == numCols {
		nameW, commentW = widths[0], widths[1]
	}

	return []string{
		truncate(g.Name, nameW),
		truncate(g.Comment, commentW),
		enabledLabel(g.Enabled),
		strconv.Itoa(g.ID),
	}
}

// styledRows builds table rows from groups, colouring the Enabled cell green
// (allowed) or subtle (disabled) using the current theme. Read at View() time.
func styledRows(
	th *theme.Theme,
	groups []pihole.Group,
	widths []int,
) []table.Row {
	rows := make([]table.Row, len(groups))
	for i, g := range groups {
		cells := groupToRow(g, widths)
		if g.Enabled {
			cells[2] = th.AllowStyle().Render(enabledGlyph)
		} else {
			cells[2] = th.SubtleStyle().Render(disabledGlyph)
		}
		rows[i] = table.Row(cells)
	}
	return rows
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
