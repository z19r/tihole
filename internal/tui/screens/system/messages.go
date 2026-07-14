package system

import (
	"context"
	"strconv"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/components"
)

const (
	msgColID   = 5
	msgColType = 16
	msgColTime = 8
	msgMinFlex = 8
	cellPad    = 2
)

var msgColumnTitles = []string{"ID", "Type", "Message", "Time"}

// messagesMsg carries a diagnosis-messages fetch result plus the active count,
// tagged with its issuing epoch.
type messagesMsg struct {
	epoch    int
	messages []pihole.DiagnosisMessage
	count    int
	err      error
}

// computeMsgWidths returns per-column content widths that never overflow total.
func computeMsgWidths(total int) []int {
	avail := total - len(msgColumnTitles)*cellPad
	fixed := msgColID + msgColType + msgColTime
	flex := avail - fixed
	if flex < msgMinFlex {
		flex = msgMinFlex
	}
	return []int{msgColID, msgColType, flex, msgColTime}
}

func msgColumns(widths []int) []table.Column {
	cols := make([]table.Column, len(msgColumnTitles))
	for i, title := range msgColumnTitles {
		w := msgColID
		if i < len(widths) {
			w = widths[i]
		}
		cols[i] = table.Column{Title: title, Width: w}
	}
	return cols
}

// messageToRow maps a diagnosis message to its ordered, truncated table cells.
func messageToRow(msg pihole.DiagnosisMessage, widths []int) []string {
	typeW, plainW := msgColType, msgMinFlex
	if len(widths) == len(msgColumnTitles) {
		typeW, plainW = widths[1], widths[2]
	}
	return []string{
		strconv.Itoa(msg.ID),
		truncate(msg.Type, typeW),
		truncate(msg.Plain, plainW),
		formatEpoch(msg.Timestamp),
	}
}

// formatEpoch renders epoch-seconds as HH:MM:SS.
func formatEpoch(sec float64) string {
	whole := int64(sec)
	nsec := int64((sec - float64(whole)) * 1e9)
	return time.Unix(whole, nsec).Format("15:04:05")
}

// fetchMessages requests the diagnosis messages and their active count.
func (m *Model) fetchMessages() tea.Cmd {
	ctx := m.newCtx()
	e := m.epoch
	api := m.ctx.API
	m.loading = true

	return tea.Batch(func() tea.Msg {
		msgs, err := api.Messages(ctx)
		if err != nil {
			return messagesMsg{epoch: e, err: err}
		}
		count, cErr := api.MessageCount(ctx)
		return messagesMsg{epoch: e, messages: msgs, count: count, err: cErr}
	}, m.spinner.Tick)
}

// selectedMessage returns the diagnosis message under the table cursor, if any.
func (m *Model) selectedMessage() (pihole.DiagnosisMessage, bool) {
	idx := m.msgTable.Cursor()
	if idx >= 0 && idx < len(m.messages) {
		return m.messages[idx], true
	}
	return pihole.DiagnosisMessage{}, false
}

// syncMsgRows rebuilds the messages table rows from the current data.
func (m *Model) syncMsgRows() {
	widths := columnWidthsFrom(m.msgTable.Columns())
	idx := m.msgTable.Cursor()
	rows := make([]table.Row, len(m.messages))
	for i, msg := range m.messages {
		rows[i] = table.Row(messageToRow(msg, widths))
	}
	m.msgTable.SetStyles(components.TableStyles(m.ctx.Theme))
	m.msgTable.SetRows(rows)
	if idx >= len(rows) {
		idx = len(rows) - 1
	}
	if idx < 0 {
		idx = 0
	}
	m.msgTable.SetCursor(idx)
}

// handleMessagesKey routes Messages-tab shortcuts, then defers to the table.
func (m *Model) handleMessagesKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "x", "delete":
		if dm, ok := m.selectedMessage(); ok {
			id := dm.ID
			m.pendingOp = func(ctx context.Context, api *pihole.Client) error {
				return api.DeleteMessages(ctx, []int{id})
			}
			m.pendingReload = true
			m.confirm = m.confirm.Show(
				"Dismiss message?",
				truncate(dm.Plain, 60),
				true,
			)
		}
		return m, nil
	case "X":
		if len(m.messages) > 0 {
			ids := make([]int, len(m.messages))
			for i, dm := range m.messages {
				ids[i] = dm.ID
			}
			m.pendingOp = func(ctx context.Context, api *pihole.Client) error {
				return api.DeleteMessages(ctx, ids)
			}
			m.pendingReload = true
			m.confirm = m.confirm.Show(
				"Dismiss all messages?",
				strconv.Itoa(len(ids))+" messages",
				true,
			)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.msgTable, cmd = m.msgTable.Update(msg)
	return m, cmd
}

// renderMessages draws the diagnosis-messages table.
func (m *Model) renderMessages(th *theme.Theme, bodyH int) string {
	if m.loading && len(m.messages) == 0 {
		return m.spinnerLine(th, "loading messages…", bodyH)
	}
	if len(m.messages) == 0 {
		empty := th.SubtleStyle().Render("no diagnosis messages")
		return lipgloss.Place(
			m.w,
			bodyH,
			lipgloss.Center,
			lipgloss.Center,
			empty,
			th.SurfaceWhitespace(),
		)
	}
	m.syncMsgRows()
	return m.msgTable.View()
}

// columnWidthsFrom extracts content widths from a table's columns.
func columnWidthsFrom(cols []table.Column) []int {
	w := make([]int, len(cols))
	for i, c := range cols {
		w[i] = c.Width
	}
	return w
}
