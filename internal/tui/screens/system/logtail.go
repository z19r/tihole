package system

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
)

// maxLogLines caps the retained tail so the buffer stays bounded during a long
// live session.
const maxLogLines = 2000

// tickMsg drives the live log poll. It carries the epoch active when scheduled
// so stale ticks (from a prior focus/tab cycle) can be dropped.
type tickMsg struct{ epoch int }

// dnsLogMsg carries a DNS log-tail page, tagged with its issuing epoch.
type dnsLogMsg struct {
	epoch int
	page  pihole.DNSLogPage
	err   error
}

// tick schedules the next poll, stamped with the current epoch.
func (m *Model) tick() tea.Cmd {
	e := m.epoch
	return tea.Tick(pollInterval, func(time.Time) tea.Msg { return tickMsg{e} })
}

// fetchLog requests the next batch of DNS log lines from the carried cursor.
func (m *Model) fetchLog() tea.Cmd {
	ctx := m.newCtx()
	e := m.epoch
	api := m.ctx.API
	nextID := m.nextID
	if len(m.logLines) == 0 {
		m.loading = true
	}

	return func() tea.Msg {
		page, err := api.DNSLog(ctx, nextID)
		return dnsLogMsg{epoch: e, page: page, err: err}
	}
}

// startLogTail begins the live tail: an immediate fetch plus the recurring tick.
func (m *Model) startLogTail() tea.Cmd {
	return tea.Batch(m.fetchLog(), m.tick(), m.spinner.Tick)
}

// appendLog folds a fetched page into the tail: it appends new lines, advances
// the cursor, clamps the buffer, and refreshes/auto-scrolls the viewport.
func (m *Model) appendLog(page pihole.DNSLogPage) {
	for _, line := range page.Log {
		m.logLines = append(m.logLines, formatLogLine(line))
	}
	if page.NextID > 0 {
		m.nextID = page.NextID
	}
	if len(m.logLines) > maxLogLines {
		m.logLines = m.logLines[len(m.logLines)-maxLogLines:]
	}
	m.viewport.SetContent(strings.Join(m.logLines, "\n"))
	m.viewport.GotoBottom()
}

// formatLogLine renders one resolver log line as "HH:MM:SS  message".
func formatLogLine(line pihole.DNSLogLine) string {
	return formatEpoch(line.Timestamp) + "  " + strings.TrimRight(line.Message, "\n")
}

// handleLogKey lets the viewport handle scroll keys on the Log tab.
func (m *Model) handleLogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// renderLog draws the live log viewport.
func (m *Model) renderLog(th *theme.Theme, bodyH int) string {
	if m.loading && len(m.logLines) == 0 {
		return m.spinnerLine(th, "reading log…", bodyH)
	}
	if len(m.logLines) == 0 {
		empty := th.SubtleStyle().Render("waiting for log lines…")
		return lipgloss.Place(m.w, bodyH, lipgloss.Center, lipgloss.Center, empty)
	}
	return m.viewport.View()
}
