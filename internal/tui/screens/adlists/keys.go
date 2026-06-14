package adlists

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
)

// handleGravityLine appends a streamed line to the log viewport and pumps the
// next read. A closed channel (ok=false) ends the pump.
func (m *Model) handleGravityLine(msg gravityLineMsg) (tea.Model, tea.Cmd) {
	if msg.epoch != m.epoch {
		return m, nil
	}
	if !msg.ok {
		return m, nil
	}
	m.gravityLines = append(m.gravityLines, msg.line)
	m.viewport.SetContent(strings.Join(m.gravityLines, "\n"))
	m.viewport.GotoBottom()
	return m, readGravityLine(m.gravityChan, m.epoch)
}

// handleKey routes keystrokes through the form, confirm dialog, gravity pane,
// and finally the table.
func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.form.active {
		return m.handleFormKey(msg)
	}
	if m.confirm.Active {
		return m.handleConfirmKey(msg)
	}
	if m.gravityOpen {
		if msg.String() == "esc" && !m.gravityRunning {
			m.gravityOpen = false
			m.gravityDone = false
		}
		return m, nil
	}
	return m.handleTableKey(msg)
}

// handleTableKey handles the primary table-mode bindings.
func (m *Model) handleTableKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "f", "left", "right", "h", "l":
		m.toggleVisible()
		return m, nil
	case "r":
		m.epoch++
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.fetch(), m.spinner.Tick)
	case "a":
		m.form = m.form.openAdd(m.visible)
		m.form.setWidth(m.w)
		return m, nil
	case "e":
		if l, ok := m.selected(); ok {
			m.form = m.form.openEdit(l)
			m.form.setWidth(m.w)
		}
		return m, nil
	case "space", "t":
		return m, m.toggleEnabled()
	case "x", "delete":
		if l, ok := m.selected(); ok {
			m.pendingAddr = l.Address
			m.pendingType = pihole.ListType(l.Type)
			m.confirmKind = confirmDelete
			m.confirm = m.confirm.Show("Delete list?", l.Address, true)
		}
		return m, nil
	case "g":
		m.confirmKind = confirmGravity
		m.confirm = m.confirm.Show("Run gravity update?", "Re-download and rebuild all adlists.", false)
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// handleFormKey drives the inline add/edit form.
func (m *Model) handleFormKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.form = m.form.close()
		return m, nil
	case "enter":
		return m, m.submitForm()
	case "tab", "down":
		m.form = m.form.advance(1)
		return m, nil
	case "shift+tab", "up":
		m.form = m.form.advance(-1)
		return m, nil
	case "left", "right", "space":
		switch m.form.current() {
		case fieldType, fieldEnabled:
			m.form = m.form.toggle()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.form, cmd = m.form.update(msg)
	return m, cmd
}

// handleConfirmKey resolves the y/n on a pending confirm dialog.
func (m *Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		kind := m.confirmKind
		m.confirm = m.confirm.Hide()
		m.confirmKind = confirmNone
		switch kind {
		case confirmDelete:
			return m, m.deleteSelected()
		case confirmGravity:
			return m, m.startGravity()
		}
		return m, nil
	case "n", "esc":
		m.confirm = m.confirm.Hide()
		m.confirmKind = confirmNone
		return m, nil
	}
	return m, nil
}
