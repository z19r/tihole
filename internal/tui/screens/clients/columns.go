package clients

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/table"

	"github.com/z19r/tihole/internal/pihole"
)

// Column layout constants. The table has four columns in a fixed order:
// Client, Name, Comment, Groups. Groups (a membership count) takes a fixed
// width; Client/Name/Comment share the remaining space by weight.
const (
	numCols = 4

	colGroupsWidth = 7 // "Groups" header / small integer count

	cellPad    = 2 // default table cell padding (left+right)
	minFlexCol = 6 // minimum width for a flexible column
	ellipsis   = "…"
)

// columnTitles is the header row, in render order.
var columnTitles = []string{"Client", "Name", "Comment", "Groups"}

// computeColumnWidths returns the content width of each column for a given
// total width. Widths never sum past the available (padding-adjusted) space,
// and each flexible column is clamped to a sane minimum so nothing collapses.
func computeColumnWidths(total int) []int {
	avail := total - numCols*cellPad
	fixed := colGroupsWidth

	flex := avail - fixed
	minFlex := 3 * minFlexCol
	if flex < minFlex {
		flex = minFlex
	}

	// Client and Name each get a third; Comment takes the remainder.
	client := flex / 3
	name := flex / 3
	comment := flex - client - name

	client = clampMin(client, minFlexCol)
	name = clampMin(name, minFlexCol)
	comment = clampMin(comment, minFlexCol)

	return []int{client, name, comment, colGroupsWidth}
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
		w := minFlexCol
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
// when it must cut. Rune-aware so multibyte hosts are not split mid-rune.
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

// clientToRow maps a client entry into the ordered cell strings for the table,
// truncating the free-text columns to their computed widths. Pure: no styling.
func clientToRow(c pihole.ClientEntry, widths []int) []string {
	clientW, nameW, commentW := minFlexCol, minFlexCol, minFlexCol
	if len(widths) == numCols {
		clientW, nameW, commentW = widths[0], widths[1], widths[2]
	}

	return []string{
		truncate(orDash(c.Client), clientW),
		truncate(orDash(c.Name), nameW),
		truncate(orDash(c.Comment), commentW),
		strconv.Itoa(len(c.Groups)),
	}
}

// clientRows builds table rows from client entries using the current widths.
func clientRows(clients []pihole.ClientEntry, widths []int) []table.Row {
	rows := make([]table.Row, len(clients))
	for i, c := range clients {
		rows[i] = table.Row(clientToRow(c, widths))
	}
	return rows
}

// parseGroups parses a comma-separated list of group ids (e.g. "1, 2,3") into
// a slice of ints. Blank input yields an empty slice; any non-integer token
// returns an error so the form can surface it instead of sending bad data.
func parseGroups(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return []int{}, nil
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, errBadGroups
		}
		out = append(out, n)
	}
	return out, nil
}

// groupsToString renders group ids back into a comma-separated string for the
// edit form (e.g. []int{0,1} -> "0, 1").
func groupsToString(groups []int) string {
	parts := make([]string, len(groups))
	for i, g := range groups {
		parts[i] = strconv.Itoa(g)
	}
	return strings.Join(parts, ", ")
}

// firstAddress returns the first address token from a suggestion's Addresses
// field, which may hold several separated by commas or spaces.
func firstAddress(addresses string) string {
	a := strings.TrimSpace(addresses)
	if a == "" {
		return ""
	}
	for _, sep := range []string{",", " "} {
		if i := strings.Index(a, sep); i >= 0 {
			a = a[:i]
		}
	}
	return strings.TrimSpace(a)
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
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
