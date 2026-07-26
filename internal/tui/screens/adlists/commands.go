package adlists

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/pihole"
)

// submitForm builds a create or update command from the form's current values.
func (m *Model) submitForm() tea.Cmd {
	address, comment, groupsVal := m.form.values()
	mode := m.form.mode
	listType := m.form.listType
	enabled := m.form.enabled
	m.form = m.form.close()

	if strings.TrimSpace(address) == "" {
		m.err = fmt.Errorf("address is required")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	e := m.epoch
	api := m.ctx.API

	if mode == formAdd {
		groups := groupsOrDefault(groupsVal)
		return func() tea.Msg {
			defer cancel()
			_, err := api.AddList(ctx, address, listType, comment, groups)
			return mutatedMsg{epoch: e, err: err}
		}
	}

	groups := parseGroups(groupsVal)
	return func() tea.Msg {
		defer cancel()
		_, err := api.UpdateList(
			ctx,
			address,
			listType,
			comment,
			groups,
			enabled,
		)
		return mutatedMsg{epoch: e, err: err}
	}
}

// toggleEnabled flips the enabled flag of the selected list via UpdateList.
func (m *Model) toggleEnabled() tea.Cmd {
	l, ok := m.selected()
	if !ok {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	e := m.epoch
	api := m.ctx.API
	address := l.Address
	listType := pihole.ListType(l.Type)
	comment := l.Comment
	groups := l.Groups
	next := !l.Enabled

	return func() tea.Msg {
		defer cancel()
		_, err := api.UpdateList(ctx, address, listType, comment, groups, next)
		return mutatedMsg{epoch: e, err: err}
	}
}

// deleteSelected removes the pending (confirmed) list.
func (m *Model) deleteSelected() tea.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	e := m.epoch
	api := m.ctx.API
	address := m.pendingAddr
	listType := m.pendingType

	return func() tea.Msg {
		defer cancel()
		err := api.DeleteList(ctx, address, listType)
		return mutatedMsg{epoch: e, err: err}
	}
}

// startGravity opens the live-log pane and kicks off the streaming pump: one
// fire-and-forget producer feeding a channel, one reader command pulling a line
// at a time and re-issuing itself, plus the spinner.
func (m *Model) startGravity() tea.Cmd {
	if m.gravityCancel != nil {
		m.gravityCancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), gravityTimeout)
	m.gravityCancel = cancel

	ch := make(chan string, gravityChanBuffer)
	m.gravityChan = ch
	m.gravityOpen = true
	m.gravityRunning = true
	m.gravityDone = false
	m.gravityErr = nil
	m.gravityLines = nil
	m.viewport.SetContent("")

	return tea.Batch(
		runGravity(ctx, m.ctx.API, ch, m.epoch),
		readGravityLine(ch, m.epoch),
		m.spinner.Tick,
	)
}
