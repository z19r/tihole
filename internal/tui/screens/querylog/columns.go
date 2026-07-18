package querylog

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
)

// Column layout constants. The table has six columns in a fixed order:
// Time, Domain, Client, Type, Status, Upstream. Time/Type/Status take a fixed
// content width; Domain/Client/Upstream share the remaining space by weight.
const (
	numCols = 6

	colTimeWidth   = 8  // HH:MM:SS
	colTypeWidth   = 5  // A, AAAA, PTR, HTTPS...
	colStatusWidth = 12 // EXTERNAL_BLK etc. (short label)

	cellPad     = 2 // default table cell padding (left+right)
	minFlexCol  = 6 // minimum width for a flexible column
	defaultPage = 100
	ellipsis    = "…"

	tokenBlock   = "block"
	tokenAllow   = "allow"
	tokenNeutral = "neutral"
)

// columnTitles is the header row, in render order.
var columnTitles = []string{
	"Time",
	"Domain",
	"Client",
	"Type",
	"Status",
	"Upstream",
}

// computeColumnWidths returns the content width of each column for a given
// total width. Widths never sum past the available (padding-adjusted) space,
// and each flexible column is clamped to a sane minimum so nothing collapses.
func computeColumnWidths(total int) []int {
	avail := total - numCols*cellPad
	fixed := colTimeWidth + colTypeWidth + colStatusWidth

	flex := avail - fixed
	minFlex := 3 * minFlexCol
	if flex < minFlex {
		flex = minFlex
	}

	// Domain gets half, Client and Upstream a quarter each.
	domain := flex / 2
	client := flex / 4
	upstream := flex - domain - client

	domain = clampMin(domain, minFlexCol)
	client = clampMin(client, minFlexCol)
	upstream = clampMin(upstream, minFlexCol)

	return []int{
		colTimeWidth,
		domain,
		client,
		colTypeWidth,
		colStatusWidth,
		upstream,
	}
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
		w := colTimeWidth
		if i < len(widths) {
			w = widths[i]
		}
		cols[i] = table.Column{Title: title, Width: w}
	}
	return cols
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

// formatQueryTime renders a query's epoch-seconds timestamp as HH:MM:SS.
func formatQueryTime(sec float64) string {
	whole := int64(sec)
	nsec := int64((sec - float64(whole)) * 1e9)
	return time.Unix(whole, nsec).Format("15:04:05")
}

// clientLabel prefers a client's resolved name, falling back to its IP.
func clientLabel(c pihole.QueryClient) string {
	if strings.TrimSpace(c.Name) != "" {
		return c.Name
	}
	return c.IP
}

// queryToRow maps a single query into the ordered cell strings for the table,
// truncating the free-text columns to their computed widths. Pure: no styling.
func queryToRow(q pihole.Query, widths []int) []string {
	domainW, clientW, upstreamW := minFlexCol, minFlexCol, minFlexCol
	if len(widths) == numCols {
		domainW, clientW, upstreamW = widths[1], widths[2], widths[5]
	}

	upstream := q.Upstream
	if strings.TrimSpace(upstream) == "" {
		upstream = "-"
	}

	return []string{
		formatQueryTime(q.Time),
		truncate(q.Domain, domainW),
		truncate(clientLabel(q.Client), clientW),
		q.Type,
		q.Status,
		truncate(upstream, upstreamW),
	}
}

// statusToken classifies a PiHole query status into a semantic style token.
// Blocked families (gravity, denylist, regex, external/special blocks) map to
// tokenBlock; forwarded/cache/retried to tokenAllow; anything else neutral.
func statusToken(status string) string {
	s := strings.ToUpper(strings.TrimSpace(status))
	switch {
	case s == "":
		return tokenNeutral
	case strings.Contains(s, "BLOCK"),
		strings.HasPrefix(s, "GRAVITY"),
		strings.HasPrefix(s, "DENYLIST"),
		strings.HasPrefix(s, "REGEX"),
		strings.HasPrefix(s, "SPECIAL_DOMAIN"):
		return tokenBlock
	case strings.HasPrefix(s, "FORWARD"),
		strings.HasPrefix(s, "CACHE"),
		strings.HasPrefix(s, "RETRIED"),
		s == "OK":
		return tokenAllow
	default:
		return tokenNeutral
	}
}

// styleForToken resolves a style token to a lipgloss style using the live
// theme.
func styleForToken(th *theme.Theme, token string) lipgloss.Style {
	switch token {
	case tokenBlock:
		return lipgloss.NewStyle().Foreground(th.Block)
	case tokenAllow:
		return lipgloss.NewStyle().Foreground(th.Allow)
	default:
		return lipgloss.NewStyle().Foreground(th.Text)
	}
}

// styledRows builds table rows from queries, colouring the Domain and Status
// cells by the row's status token using the current theme. Read at View() time.
func styledRows(
	th *theme.Theme,
	queries []pihole.Query,
	widths []int,
) []table.Row {
	rows := make([]table.Row, len(queries))
	for i, q := range queries {
		cells := queryToRow(q, widths)
		st := styleForToken(th, statusToken(q.Status))
		cells[1] = st.Render(cells[1])
		cells[4] = st.Render(cells[4])
		rows[i] = table.Row(cells)
	}
	return rows
}

// buildFilter constructs a QueryFilter for a fresh page: an optional domain
// search plus the page length. Only set (non-nil) fields are sent by the
// client.
func buildFilter(domain string, length int, cursor *int64) pihole.QueryFilter {
	f := pihole.QueryFilter{}
	if length > 0 {
		l := length
		f.Length = &l
	}
	if cursor != nil {
		f.Cursor = cursor
	}
	if d := strings.TrimSpace(domain); d != "" {
		f.Domain = &d
	}
	return f
}
