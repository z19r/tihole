package localdns

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

// View renders the screen. The theme is read here so live re-themes and per-row
// colours take effect immediately.
func (m *Model) View() tea.View {
	th := m.ctx.Theme

	m.syncRows()

	header := m.renderHeader(th)
	footer := m.renderFooter(th)
	// Derive the body height from the header/footer we actually rendered so a
	// multi-line error banner can never overflow the surface.
	bodyH := m.h - lipgloss.Height(header) - lipgloss.Height(footer)
	if bodyH < 1 {
		bodyH = 1
	}
	body := m.renderBody(th, bodyH)

	content := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	view := th.SurfaceStyle().Width(m.w).Height(m.h).Render(content)
	return tea.NewView(view)
}

func (m *Model) renderHeader(th *theme.Theme) string {
	title := th.AccentStyle().Bold(true).Render(m.Title())

	chip := lipgloss.NewStyle().
		Foreground(th.Surface).
		Background(th.Accent).
		Padding(0, 1).
		Render(m.kind.label())

	count := th.SubtleStyle().Render(fmt.Sprintf("%d records", m.activeCount()))

	left := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", chip)
	gap := m.w - lipgloss.Width(left) - lipgloss.Width(count)
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + count

	second := ""
	if m.err != nil {
		second = m.errBanner(th)
	}
	return lipgloss.JoinVertical(lipgloss.Left, line, second)
}

func (m *Model) errBanner(th *theme.Theme) string {
	msg := truncate("error: "+m.err.Error(), maxInt(m.w-2, 8))
	return th.BlockStyle().Bold(true).Render(msg)
}

func (m *Model) renderBody(th *theme.Theme, bodyH int) string {
	if m.confirm.Active {
		return m.confirm.Render(th, m.w, bodyH)
	}
	if m.form != nil {
		return m.form.render(th, m.w, bodyH)
	}

	if m.loading && m.activeCount() == 0 {
		line := m.spinner.View() + " " + th.SubtleStyle().Render("loading records…")
		return lipgloss.Place(m.w, bodyH, lipgloss.Center, lipgloss.Center, line)
	}

	if m.activeCount() == 0 {
		empty := th.SubtleStyle().Render("no " + m.kind.label() + " records")
		return lipgloss.Place(m.w, bodyH, lipgloss.Center, lipgloss.Center, empty)
	}

	if m.kind == kindCNAMEs {
		return m.cnameTable.View()
	}
	return m.hostTable.View()
}

func (m *Model) renderFooter(th *theme.Theme) string {
	hint := "↑↓ navigate · a add · x delete · f switch kind · r refresh"
	return th.SubtleStyle().Render(truncate(hint, maxInt(m.w, 8)))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
