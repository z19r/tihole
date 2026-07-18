package localdns

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/pihole"
)

// handleKey routes keystrokes through the confirm dialog, the form, and the
// table, in that precedence order.
func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.confirm.Active {
		return m.handleConfirmKey(msg)
	}
	if m.form != nil {
		return m.handleFormKey(msg)
	}
	return m.handleTableKey(msg)
}

// handleConfirmKey interprets y/n on the delete confirmation.
func (m *Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		host := m.pendingHost
		cname := m.pendingCNAME
		m.confirm = m.confirm.Hide()
		m.pendingHost = nil
		m.pendingCNAME = nil
		if host != nil {
			h := *host
			return m, m.mutate(
				func(ctx context.Context, api *pihole.Client) error {
					return api.DeleteHostRecord(ctx, h.IP, h.Domain)
				},
			)
		}
		if cname != nil {
			c := *cname
			return m, m.mutate(
				func(ctx context.Context, api *pihole.Client) error {
					return api.DeleteCNAMERecord(ctx, c.Domain, c.Target, c.TTL)
				},
			)
		}
		return m, nil
	case "n", "esc":
		m.confirm = m.confirm.Hide()
		m.pendingHost = nil
		m.pendingCNAME = nil
	}
	return m, nil
}

// handleFormKey drives the add form: navigation, submit, cancel.
func (m *Model) handleFormKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.form = nil
		return m, nil
	case "enter":
		return m.submitForm()
	case "tab", "down":
		m.form.focusNext()
		return m, m.form.syncFocus()
	case "shift+tab", "up":
		m.form.focusPrev()
		return m, m.form.syncFocus()
	}
	return m, m.form.updateFocused(msg)
}

// submitForm validates and dispatches the add mutation, then closes the form.
func (m *Model) submitForm() (tea.Model, tea.Cmd) {
	f := m.form

	if f.kind == kindHosts {
		ip, domain := f.value(0), f.value(1)
		if ip == "" || domain == "" {
			m.err = errRecordRequired
			return m, nil
		}
		m.form = nil
		m.err = nil
		return m, m.mutate(func(ctx context.Context, api *pihole.Client) error {
			return api.AddHostRecord(ctx, ip, domain)
		})
	}

	domain, target := f.value(0), f.value(1)
	ttl := parseTTL(f.value(2))
	if domain == "" || target == "" {
		m.err = errRecordRequired
		return m, nil
	}
	m.form = nil
	m.err = nil
	return m, m.mutate(func(ctx context.Context, api *pihole.Client) error {
		return api.AddCNAMERecord(ctx, domain, target, ttl)
	})
}

// handleTableKey routes list-level shortcuts and otherwise defers to the active
// table.
func (m *Model) handleTableKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		m.epoch++
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.fetch(), m.spinner.Tick)

	case "f", "right":
		m.kind = m.kind.next()
		return m, nil
	case "left":
		m.kind = m.kind.prev()
		return m, nil

	case "a":
		if m.kind == kindCNAMEs {
			m.form = newCNAMEForm()
		} else {
			m.form = newHostForm()
		}
		m.form.setWidth(m.w)
		return m, m.form.syncFocus()

	case "x", "delete":
		return m.requestDelete()
	}

	var cmd tea.Cmd
	if m.kind == kindCNAMEs {
		m.cnameTable, cmd = m.cnameTable.Update(msg)
	} else {
		m.hostTable, cmd = m.hostTable.Update(msg)
	}
	return m, cmd
}

// requestDelete opens the confirm dialog for the highlighted record of the
// active kind.
func (m *Model) requestDelete() (tea.Model, tea.Cmd) {
	if m.kind == kindCNAMEs {
		idx := m.cnameTable.Cursor()
		if idx >= 0 && idx < len(m.cnames) {
			r := m.cnames[idx]
			m.pendingCNAME = &r
			m.confirm = m.confirm.Show(
				"Delete CNAME record?",
				displayCNAME(r),
				true,
			)
		}
		return m, nil
	}
	idx := m.hostTable.Cursor()
	if idx >= 0 && idx < len(m.hosts) {
		r := m.hosts[idx]
		m.pendingHost = &r
		m.confirm = m.confirm.Show("Delete host record?", displayHost(r), true)
	}
	return m, nil
}
