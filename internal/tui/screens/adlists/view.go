package adlists

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
	"github.com/z19r/tihole/internal/tui/components"
)

// View renders the screen. The theme is read here so live re-themes and
// per-row status colours take effect immediately.
func (m *Model) View() tea.View {
	th := m.ctx.Theme

	if len(m.lists[m.visible]) > 0 {
		m.syncRows()
	}

	header := m.renderHeader(th)
	body := m.renderBody(th)
	footer := m.renderFooter(th)

	content := strings.Join([]string{header, body, footer}, "\n")
	view := th.SurfaceStyle().Width(m.w).Height(m.h).Render(content)
	return tea.NewView(view)
}

// listTypeOrder is the fixed left-to-right order of the adlist type tabs.
var listTypeOrder = []pihole.ListType{pihole.ListBlock, pihole.ListAllow}

func (m *Model) renderHeader(th *theme.Theme) string {
	labels := make([]string, len(listTypeOrder))
	active := 0
	for i, lt := range listTypeOrder {
		labels[i] = string(lt)
		if lt == m.visible {
			active = i
		}
	}
	count := th.SubtleStyle().Render(fmt.Sprintf("block %d · allow %d",
		len(m.lists[pihole.ListBlock]), len(m.lists[pihole.ListAllow])))

	line := components.SectionTabs{
		Title:  m.Title(),
		Labels: labels,
		Active: active,
		Right:  count,
		Width:  m.w,
	}.Render(th)

	second := ""
	if m.err != nil {
		second = m.errBanner(th)
	}
	return strings.Join([]string{line, second}, "\n")
}

func (m *Model) errBanner(th *theme.Theme) string {
	msg := truncate("error: "+m.err.Error(), maxInt(m.w-2, 8))
	return th.BlockStyle().Bold(true).Render(msg)
}

func (m *Model) renderBody(th *theme.Theme) string {
	bodyH := m.h - headerHeight - footerHeight
	if bodyH < 1 {
		bodyH = 1
	}

	if m.form.active {
		return lipgloss.Place(
			m.w,
			bodyH,
			lipgloss.Center,
			lipgloss.Top,
			m.form.view(th, m.w),
			th.SurfaceWhitespace(),
		)
	}
	if m.confirm.Active {
		return m.confirm.Render(th, m.w, bodyH)
	}
	if m.gravityOpen {
		return m.renderGravity(th, bodyH)
	}

	if m.loading && len(m.lists[m.visible]) == 0 {
		line := m.spinner.View() + " " + th.SubtleStyle().
			Render("loading lists…")
		return lipgloss.Place(
			m.w,
			bodyH,
			lipgloss.Center,
			lipgloss.Center,
			line,
			th.SurfaceWhitespace(),
		)
	}
	if len(m.lists[m.visible]) == 0 {
		empty := th.SubtleStyle().
			Render("no " + string(m.visible) + " lists configured")
		return lipgloss.Place(
			m.w,
			bodyH,
			lipgloss.Center,
			lipgloss.Center,
			empty,
			th.SurfaceWhitespace(),
		)
	}
	return m.table.View()
}

func (m *Model) renderGravity(th *theme.Theme, bodyH int) string {
	var status string
	switch {
	case m.gravityRunning:
		status = m.spinner.View() + " " + th.SubtleStyle().
			Render(gravitySummary(false, nil))
	case m.gravityErr != nil:
		status = th.BlockStyle().
			Bold(true).
			Render(gravitySummary(true, m.gravityErr))
	default:
		status = th.AllowStyle().Render(gravitySummary(true, nil))
	}

	heading := th.AccentStyle().Bold(true).Render("Gravity update")
	closeHint := ""
	if !m.gravityRunning {
		closeHint = th.SubtleStyle().Render("  esc close")
	}
	top := lipgloss.JoinHorizontal(
		lipgloss.Center,
		heading,
		"   ",
		status,
		closeHint,
	)

	body := strings.Join([]string{top, m.viewport.View()}, "\n")
	return lipgloss.Place(
		m.w,
		bodyH,
		lipgloss.Left,
		lipgloss.Top,
		body,
		th.SurfaceWhitespace(),
	)
}

func (m *Model) renderFooter(th *theme.Theme) string {
	hint := "a add · e edit · space toggle · x delete · g gravity · f type · r refresh"
	if m.form.active {
		hint = "tab next · ←/→ toggle · enter save · esc cancel"
	} else if m.confirm.Active {
		hint = "y confirm · n/esc cancel"
	} else if m.gravityOpen {
		hint = "streaming gravity update…"
		if !m.gravityRunning {
			hint = "esc close"
		}
	}
	return th.SubtleStyle().Render(truncate(hint, maxInt(m.w, 8)))
}
