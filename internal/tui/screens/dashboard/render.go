package dashboard

import (
	"fmt"

	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
)

// truncate shortens s to at most width display cells, appending an ellipsis when
// it has to cut. A width <= 0 yields "".
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return string(r[:width-1]) + "…"
}

// panelBox is the shared bordered surface for every dashboard panel.
func panelBox(th *theme.Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Border).
		Padding(0, 1)
}

// renderTiles draws the four headline stat tiles in a single row.
func (m *Model) renderTiles() string {
	th := m.ctx.Theme

	outer := m.w / 4
	inner := outer - 4 // border (2) + padding (2)
	if inner < 3 {
		inner = 3
	}

	tiles := []string{
		m.tile("Total Queries", formatCount(m.summary.Queries.Total), inner),
		m.blockedTile(inner),
		m.tile("Active Clients", formatCount(m.summary.Clients.Active), inner),
		m.tile("Gravity Domains", formatCount(m.summary.Gravity.DomainsBeingBlocked), inner),
	}

	if m.errSummary != "" {
		return panelBox(th).Width(m.w - 4).Render(
			th.BlockStyle().Render("summary unavailable: ") + th.SubtleStyle().Render(truncate(m.errSummary, m.w-24)))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tiles...)
}

// blockedTile is the Blocked stat tile with an animated gradient gauge beneath
// the figure. The gauge springs toward the current block rate as fresh summary
// data lands, giving the dashboard a bit of motion.
func (m *Model) blockedTile(inner int) string {
	th := m.ctx.Theme
	value := fmt.Sprintf("%s  %s",
		formatCount(m.summary.Queries.Blocked),
		formatPercent(m.summary.Queries.PercentBlocked))

	lbl := th.SubtleStyle().Render(truncate("Blocked", inner))
	fig := th.AccentStyle().Bold(true).Render(truncate(value, inner))

	m.syncBarTheme()
	m.blockBar.SetWidth(inner)
	gauge := m.blockBar.View()

	body := lipgloss.JoinVertical(lipgloss.Left, lbl, fig, gauge)
	return panelBox(th).Width(inner).Render(body)
}

// tile renders one stat tile: a subtle label above a large accent figure.
func (m *Model) tile(label, value string, inner int) string {
	th := m.ctx.Theme
	lbl := th.SubtleStyle().Render(truncate(label, inner))
	fig := th.AccentStyle().Bold(true).Render(truncate(value, inner))
	body := lipgloss.JoinVertical(lipgloss.Left, lbl, fig)
	return panelBox(th).Width(inner).Render(body)
}

// renderSparkline draws the queries-over-time line across the full width.
func (m *Model) renderSparkline() string {
	th := m.ctx.Theme
	inner := m.w - 4
	if inner < 8 {
		inner = 8
	}

	title := th.TextStyle().Bold(true).Render("Queries over time")

	var line string
	switch {
	case m.errHistory != "":
		line = th.BlockStyle().Render(truncate("history unavailable: "+m.errHistory, inner))
	case len(m.history.History) == 0:
		line = th.SubtleStyle().Render("no history yet")
	default:
		totals := make([]int, len(m.history.History))
		blocked := make([]int, len(m.history.History))
		for i, p := range m.history.History {
			totals[i] = p.Total
			blocked[i] = p.Blocked
		}
		totalLine := th.AccentStyle().Render(sparklineInt(totals, inner))
		blockedLine := th.BlockStyle().Render(sparklineInt(blocked, inner))
		legend := th.SubtleStyle().Render(fmt.Sprintf("total (accent) / blocked  —  %d buckets", len(m.history.History)))
		line = lipgloss.JoinVertical(lipgloss.Left, totalLine, blockedLine, legend)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, title, line)
	return panelBox(th).Width(inner).Render(body)
}

// renderLower composes the breakdown and top-list columns.
func (m *Model) renderLower() string {
	outer := m.w / 3
	inner := outer - 4
	if inner < 6 {
		inner = 6
	}

	col1 := lipgloss.JoinVertical(lipgloss.Left,
		m.breakdownPanel("Query Types", typeItems(m.types.Types), m.errTypes, inner),
		m.breakdownPanel("Upstreams", upstreamItems(m.upstreams.Upstreams), m.errUpstreams, inner),
	)
	col2 := lipgloss.JoinVertical(lipgloss.Left,
		m.listPanel("Top Domains", domainItems(m.domains), m.errDomains, inner),
		m.listPanel("Top Blocked", domainItems(m.blocked), m.errBlocked, inner),
	)
	col3 := m.listPanel("Top Clients", clientItems(m.clients), m.errClients, inner)

	return lipgloss.JoinHorizontal(lipgloss.Top, col1, col2, col3)
}

// breakdownPanel renders labeled counts as mini horizontal bars.
func (m *Model) breakdownPanel(title string, items []labeledCount, errStr string, inner int) string {
	th := m.ctx.Theme
	head := th.TextStyle().Bold(true).Render(truncate(title, inner))

	if errStr != "" {
		body := lipgloss.JoinVertical(lipgloss.Left, head, th.BlockStyle().Render(truncate(errStr, inner)))
		return panelBox(th).Width(inner).Render(body)
	}
	if len(items) == 0 {
		body := lipgloss.JoinVertical(lipgloss.Left, head, th.SubtleStyle().Render("no data"))
		return panelBox(th).Width(inner).Render(body)
	}

	max := items[0].count
	for _, it := range items {
		if it.count > max {
			max = it.count
		}
	}

	labelW := 5
	countW := len(formatCount(max))
	barW := inner - labelW - countW - 2
	if barW < 3 {
		barW = 3
	}

	rows := []string{head}
	for _, it := range items {
		frac := 0.0
		if max > 0 {
			frac = float64(it.count) / float64(max)
		}
		label := th.SubtleStyle().Render(padRight(truncate(it.label, labelW), labelW))
		bar := th.AccentStyle().Render(miniBar(frac, barW))
		count := th.TextStyle().Render(padLeft(formatCount(it.count), countW))
		rows = append(rows, fmt.Sprintf("%s %s %s", label, bar, count))
	}
	return panelBox(th).Width(inner).Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// listPanel renders a ranked label/count list.
func (m *Model) listPanel(title string, items []labeledCount, errStr string, inner int) string {
	th := m.ctx.Theme
	head := th.TextStyle().Bold(true).Render(truncate(title, inner))

	if errStr != "" {
		body := lipgloss.JoinVertical(lipgloss.Left, head, th.BlockStyle().Render(truncate(errStr, inner)))
		return panelBox(th).Width(inner).Render(body)
	}
	if len(items) == 0 {
		body := lipgloss.JoinVertical(lipgloss.Left, head, th.SubtleStyle().Render("no data"))
		return panelBox(th).Width(inner).Render(body)
	}

	countW := 0
	for _, it := range items {
		if w := len(formatCount(it.count)); w > countW {
			countW = w
		}
	}
	labelW := inner - countW - 1
	if labelW < 1 {
		labelW = 1
	}

	rows := []string{head}
	for _, it := range items {
		label := th.TextStyle().Render(padRight(truncate(it.label, labelW), labelW))
		count := th.AccentStyle().Render(padLeft(formatCount(it.count), countW))
		rows = append(rows, label+" "+count)
	}
	return panelBox(th).Width(inner).Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func padRight(s string, w int) string {
	r := []rune(s)
	for len(r) < w {
		r = append(r, ' ')
	}
	return string(r)
}

func padLeft(s string, w int) string {
	r := []rune(s)
	for len(r) < w {
		r = append([]rune{' '}, r...)
	}
	return string(r)
}

// --- item adapters (pure, testable) ---

func typeItems(types map[string]int) []labeledCount {
	return sortedTypes(types)
}

func upstreamItems(servers []pihole.UpstreamServer) []labeledCount {
	out := make([]labeledCount, 0, len(servers))
	for _, s := range servers {
		name := s.Name
		if name == "" {
			name = s.IP
		}
		out = append(out, labeledCount{label: name, count: s.Count})
	}
	return out
}

func domainItems(td pihole.TopDomains) []labeledCount {
	out := make([]labeledCount, 0, len(td.Domains))
	for _, d := range td.Domains {
		out = append(out, labeledCount{label: d.Domain, count: d.Count})
	}
	return out
}

func clientItems(tc pihole.TopClients) []labeledCount {
	out := make([]labeledCount, 0, len(tc.Clients))
	for _, c := range tc.Clients {
		name := c.Name
		if name == "" {
			name = c.IP
		}
		out = append(out, labeledCount{label: name, count: c.Count})
	}
	return out
}
