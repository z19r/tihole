package localdns

import (
	"strconv"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
)

// Column layout constants. Host records have two columns (IP, Domain); CNAME
// records have three (Domain, Target, TTL). Free-text columns share the
// available width; TTL takes a small fixed width.
const (
	hostNumCols  = 2
	cnameNumCols = 3

	colTTLWidth = 5 // "TTL" header + a few digits

	cellPad    = 2 // default table cell padding (left+right)
	minFlexCol = 8 // minimum width for a flexible column
	ellipsis   = "…"
)

// hostColumnTitles / cnameColumnTitles are the header rows, in render order.
var (
	hostColumnTitles  = []string{"IP", "Domain"}
	cnameColumnTitles = []string{"Domain", "Target", "TTL"}
)

// computeHostColumnWidths returns the content width of each host column for a
// given total width. The IP column takes the smaller share; Domain the rest.
func computeHostColumnWidths(total int) []int {
	avail := total - hostNumCols*cellPad
	flex := avail
	minFlex := 2 * minFlexCol
	if flex < minFlex {
		flex = minFlex
	}

	ip := flex * 2 / 5
	domain := flex - ip

	ip = clampMin(ip, minFlexCol)
	domain = clampMin(domain, minFlexCol)

	return []int{ip, domain}
}

// computeCNAMEColumnWidths returns the content width of each CNAME column. TTL
// is fixed; Domain and Target share the remaining space evenly.
func computeCNAMEColumnWidths(total int) []int {
	avail := total - cnameNumCols*cellPad
	flex := avail - colTTLWidth
	minFlex := 2 * minFlexCol
	if flex < minFlex {
		flex = minFlex
	}

	domain := flex / 2
	target := flex - domain

	domain = clampMin(domain, minFlexCol)
	target = clampMin(target, minFlexCol)

	return []int{domain, target, colTTLWidth}
}

func clampMin(v, min int) int {
	if v < min {
		return min
	}
	return v
}

// tableColumns builds table.Column values from titles and computed widths.
func tableColumns(titles []string, widths []int) []table.Column {
	cols := make([]table.Column, len(titles))
	for i, title := range titles {
		w := minFlexCol
		if i < len(widths) {
			w = widths[i]
		}
		cols[i] = table.Column{Title: title, Width: w}
	}
	return cols
}

// columnWidthsFrom extracts the content widths from a table's columns.
func columnWidthsFrom(cols []table.Column) []int {
	w := make([]int, len(cols))
	for i, c := range cols {
		w[i] = c.Width
	}
	return w
}

// truncate shortens s to at most width display cells, appending an ellipsis
// when it must cut. Rune-aware so multibyte domains are not split mid-rune.
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

// ttlString renders a CNAME TTL, blanking a non-positive value.
func ttlString(ttl int) string {
	if ttl <= 0 {
		return ""
	}
	return strconv.Itoa(ttl)
}

// hostToRow maps a host record into the ordered cell strings for the table,
// truncating to the computed widths. Pure: no styling.
func hostToRow(r pihole.HostRecord, widths []int) []string {
	ipW, domainW := minFlexCol, minFlexCol
	if len(widths) == hostNumCols {
		ipW, domainW = widths[0], widths[1]
	}
	return []string{
		truncate(r.IP, ipW),
		truncate(r.Domain, domainW),
	}
}

// cnameToRow maps a CNAME record into the ordered cell strings for the table.
func cnameToRow(r pihole.CNAMERecord, widths []int) []string {
	domainW, targetW := minFlexCol, minFlexCol
	if len(widths) == cnameNumCols {
		domainW, targetW = widths[0], widths[1]
	}
	return []string{
		truncate(r.Domain, domainW),
		truncate(r.Target, targetW),
		ttlString(r.TTL),
	}
}

// styledHostRows builds table rows from host records, colouring the IP cell in
// the Accent token using the live theme.
func styledHostRows(
	th *theme.Theme,
	records []pihole.HostRecord,
	widths []int,
) []table.Row {
	rows := make([]table.Row, len(records))
	for i, r := range records {
		cells := hostToRow(r, widths)
		cells[0] = lipgloss.NewStyle().Foreground(th.Accent).Render(cells[0])
		rows[i] = table.Row(cells)
	}
	return rows
}

// styledCNAMERows builds table rows from CNAME records, colouring the Target
// cell in the Accent token using the live theme.
func styledCNAMERows(
	th *theme.Theme,
	records []pihole.CNAMERecord,
	widths []int,
) []table.Row {
	rows := make([]table.Row, len(records))
	for i, r := range records {
		cells := cnameToRow(r, widths)
		cells[1] = lipgloss.NewStyle().Foreground(th.Accent).Render(cells[1])
		rows[i] = table.Row(cells)
	}
	return rows
}

// displayHost renders a host record for a confirm-dialog message.
func displayHost(r pihole.HostRecord) string {
	return r.IP + " → " + r.Domain
}

// displayCNAME renders a CNAME record for a confirm-dialog message.
func displayCNAME(r pihole.CNAMERecord) string {
	s := r.Domain + " → " + r.Target
	if r.TTL > 0 {
		s += " (ttl " + strconv.Itoa(r.TTL) + ")"
	}
	return s
}
