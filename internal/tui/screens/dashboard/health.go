package dashboard

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"
)

// renderHealth draws the host-health strip: CPU, Memory and Temperature gauges,
// a load/uptime cell and a diagnostics badge. It is height-cheap (a single
// bordered row) and degrades gracefully when a metric is unavailable.
func (m *Model) renderHealth() string {
	th := m.ctx.Theme

	// Nothing decoded yet and the fetch failed: a compact inline note rather
	// than an empty strip.
	if m.errSystem != "" && m.system.NProcs == 0 {
		return panelBox(th).Width(m.w - 4).Render(
			th.BlockStyle().Render("system unavailable: ") +
				th.SubtleStyle().Render(truncate(m.errSystem, m.w-24)))
	}

	outer := m.w / 5
	inner := outer - 4 // border (2) + padding (2)
	if inner < 8 {
		inner = 8
	}

	cells := []string{
		m.gaugeCell(
			"CPU",
			fmt.Sprintf("%.1f%%", m.system.CPUPercent),
			&m.cpuBar,
			inner,
		),
		m.gaugeCell("Memory", memLabel(m.system), &m.memBar, inner),
		m.tempCell(inner),
		m.loadCell(inner),
		m.diagnosticsCell(inner),
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

// gaugeCell renders a labeled animated gauge with a value line above it.
func (m *Model) gaugeCell(
	label, value string,
	bar *progress.Model,
	inner int,
) string {
	th := m.ctx.Theme
	lbl := th.SubtleStyle().Render(truncate(label, inner))
	val := th.TextStyle().Bold(true).Render(truncate(value, inner))
	bar.SetWidth(inner)
	body := strings.Join([]string{lbl, val, bar.View()}, "\n")
	return panelBox(th).Width(inner).Render(body)
}

// tempCell renders the temperature gauge, or a muted "n/a" line when the host
// exposes no readable sensor.
func (m *Model) tempCell(inner int) string {
	th := m.ctx.Theme
	if !m.sensors.HasTemp {
		body := strings.Join([]string{
			th.SubtleStyle().Render("Temp"),
			th.SubtleStyle().Render("n/a"),
			th.SubtleStyle().Render("no sensor"),
		}, "\n")
		return panelBox(th).Width(inner).Render(body)
	}
	unit := m.sensors.Unit
	if unit == "" {
		unit = "C"
	}
	value := fmt.Sprintf("%.0f°%s", m.sensors.CPUTemp, unit)
	return m.gaugeCell("Temp", value, &m.tempBar, inner)
}

// loadCell renders load average, process count and uptime as plain text.
func (m *Model) loadCell(inner int) string {
	th := m.ctx.Theme
	lbl := th.SubtleStyle().Render("Load · Up")
	load := th.TextStyle().
		Bold(true).
		Render(truncate(fmt.Sprintf("%.1f%%", m.system.Load1Percent), inner))
	sub := th.SubtleStyle().Render(truncate(
		fmt.Sprintf(
			"%dp · %s",
			m.system.Procs,
			humanUptime(m.system.Uptime),
		),
		inner,
	))
	body := strings.Join([]string{lbl, load, sub}, "\n")
	return panelBox(th).Width(inner).Render(body)
}

// diagnosticsCell renders FTL's active diagnosis state: a green all-clear badge
// when there are none, or a warn count plus the newest message when there are.
func (m *Model) diagnosticsCell(inner int) string {
	th := m.ctx.Theme
	lbl := th.SubtleStyle().Render("Health")

	switch {
	case m.errMessages != "":
		val := th.WarnStyle().Bold(true).Render("? unknown")
		sub := th.SubtleStyle().Render(truncate(m.errMessages, inner))
		body := strings.Join([]string{lbl, val, sub}, "\n")
		return panelBox(th).Width(inner).Render(body)

	case len(m.messages) == 0:
		val := th.AllowStyle().Bold(true).Render("✓ All clear")
		sub := th.SubtleStyle().Render("0 messages")
		body := strings.Join([]string{lbl, val, sub}, "\n")
		return panelBox(th).Width(inner).Render(body)

	default:
		val := th.WarnStyle().
			Bold(true).
			Render(truncate(pluralIssues(len(m.messages)), inner))
		sub := th.BlockStyle().Render(truncate(m.messages[0].Plain, inner))
		body := strings.Join([]string{lbl, val, sub}, "\n")
		return panelBox(th).Width(inner).Render(body)
	}
}
