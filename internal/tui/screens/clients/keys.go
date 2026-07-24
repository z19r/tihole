package clients

import (
	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
)

// handleKey routes keystrokes based on the active overlay (confirm > form >
// suggestions), falling back to the list-level bindings and the table.
func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.confirm.Active:
		return m.handleConfirmKey(msg)
	case m.form.active:
		return m.handleFormKey(msg)
	case m.suggest.active:
		return m.handleSuggestKey(msg)
	}

	switch msg.String() {
	case "a":
		return m, m.openAddForm()
	case "e":
		if c, ok := m.selected(); ok {
			return m, m.openEditForm(c)
		}
		return m, nil
	case "x", "delete":
		if c, ok := m.selected(); ok {
			m.pendingDelete = c.Client
			m.confirm = m.confirm.Show("Delete client?", c.Client, true)
		}
		return m, nil
	case "S":
		return m, m.openSuggestions()
	case "r":
		return m, m.reload()
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// handleConfirmKey interprets the delete confirmation dialog.
func (m *Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		client := m.pendingDelete
		m.confirm = m.confirm.Hide()
		m.pendingDelete = ""
		if client == "" {
			return m, nil
		}
		return m, m.deleteCmd(client)
	case "n", "esc":
		m.confirm = m.confirm.Hide()
		m.pendingDelete = ""
	}
	return m, nil
}

// handleFormKey drives the add/edit form: tab cycles fields, enter submits,
// esc cancels, everything else is forwarded to the focused input.
func (m *Model) handleFormKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.form.close()
		return m, nil
	case "enter":
		return m.submitForm()
	case "tab":
		m.form.focusNext()
		return m, m.form.focusCmd()
	case "shift+tab":
		m.form.focusPrev()
		return m, m.form.focusCmd()
	}
	cmd := m.form.updateFocused(msg)
	return m, cmd
}

// handleSuggestKey drives the suggestions pane list.
func (m *Model) handleSuggestKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.suggest.active = false
		return m, nil
	case "up", "k":
		m.suggest.moveUp()
		return m, nil
	case "down", "j":
		m.suggest.moveDown()
		return m, nil
	case "enter":
		if s, ok := m.suggest.selected(); ok {
			m.suggest.active = false
			cmd := m.openAddForm()
			m.form.client.SetValue(firstAddress(s.Addresses))
			m.form.client.CursorEnd()
			return m, cmd
		}
		return m, nil
	}
	return m, nil
}

// openAddForm resets and activates the form in add mode.
func (m *Model) openAddForm() tea.Cmd {
	m.form.mode = formAdd
	m.form.active = true
	m.form.focusIdx = 0
	m.form.err = nil
	m.form.client.SetValue("")
	m.form.comment.SetValue("")
	m.form.groups.SetValue("")
	return m.form.focusCmd()
}

// openEditForm activates the form in edit mode, pre-filled and with the client
// id read-only.
func (m *Model) openEditForm(c pihole.ClientEntry) tea.Cmd {
	m.form.mode = formEdit
	m.form.active = true
	m.form.focusIdx = 0
	m.form.err = nil
	m.form.client.SetValue(c.Client)
	m.form.comment.SetValue(c.Comment)
	m.form.groups.SetValue(groupsToString(c.Groups))
	return m.form.focusCmd()
}

// openSuggestions activates the pane and fetches candidates.
func (m *Model) openSuggestions() tea.Cmd {
	m.suggest.active = true
	m.suggest.loading = true
	m.suggest.err = nil
	m.suggest.cursor = 0
	return tea.Batch(m.fetchSuggestions(), m.spinner.Tick)
}

// submitForm validates the form, closes it, and returns the matching mutation
// command. Validation failures set an inline form error and keep the form open.
func (m *Model) submitForm() (tea.Model, tea.Cmd) {
	client, comment, groups, err := m.form.formValues()
	if err != nil {
		m.form.err = err
		return m, nil
	}
	mode := m.form.mode
	m.form.close()
	if mode == formEdit {
		return m, m.updateCmd(client, comment, groups)
	}
	return m, m.addCmd(client, comment, groups)
}
