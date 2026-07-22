// Package groups implements the PiHole v6 groups screen: a CRUD table of FTL
// groups with inline add/edit forms, an enable toggle, and a guarded delete.
// It fetches on focus and on manual refresh only (no continuous poll), runs all
// I/O in tea.Cmd closures with an epoch guard, and satisfies core.Screen.
package groups

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
	"github.com/z19r/tihole/internal/tui/components"
	"github.com/z19r/tihole/internal/tui/core"
)

const (
	fetchTimeout = 8 * time.Second

	headerHeight = 2 // title/count line + spacer
	footerHeight = 1 // hint line
)

// groupsMsg carries a list fetch result, tagged with its issuing epoch.
type groupsMsg struct {
	epoch  int
	groups []pihole.Group
	err    error
}

// mutationMsg carries the result of an add/update/delete, tagged with its
// epoch.
// On success the screen refetches the list.
type mutationMsg struct {
	epoch int
	err   error
}

// Model is the groups screen.
type Model struct {
	ctx *core.AppContext

	w, h int

	table   table.Model
	spinner spinner.Model
	form    groupForm
	confirm components.ConfirmDialog

	groups        []pihole.Group
	pendingDelete string // group name awaiting delete confirmation

	focused bool
	loading bool
	err     error

	epoch  int
	cancel context.CancelFunc
}

// New constructs the groups screen from the shared app context.
func New(ctx *core.AppContext) *Model {
	widths := computeColumnWidths(80)

	t := table.New(
		table.WithColumns(tableColumns(widths)),
		table.WithFocused(true),
	)

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))

	return &Model{
		ctx:     ctx,
		table:   t,
		spinner: sp,
		form:    newGroupForm(),
	}
}

// Init satisfies tea.Model; work begins on Focus.
func (m *Model) Init() tea.Cmd { return nil }

// Title is shown in the header/status bar.
func (m *Model) Title() string { return "Groups" }

// CapturesInput reports whether the add/edit form is focused, so the root
// delivers raw keys instead of firing global shortcuts (see
// core.InputCapturer).
func (m *Model) CapturesInput() bool { return m.form.active }

// Focus activates the screen: fetch a fresh list. No continuous poll.
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	m.epoch++
	m.loading = true
	m.err = nil
	return tea.Batch(m.fetch(), m.spinner.Tick)
}

// Blur deactivates the screen, closes any open form/dialog, and cancels any
// in-flight request.
func (m *Model) Blur() {
	m.focused = false
	m.form.close()
	m.confirm = m.confirm.Hide()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

// Help returns screen-local key bindings for the help bar.
func (m *Model) Help() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
		key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

// SetSize informs the screen of its inner content area and reflows the table.
func (m *Model) SetSize(w, h int) {
	m.w, m.h = w, h

	widths := computeColumnWidths(w)
	m.table.SetColumns(tableColumns(widths))
	m.table.SetWidth(w)

	tableH := h - headerHeight - footerHeight
	if tableH < 1 {
		tableH = 1
	}
	m.table.SetHeight(tableH)

	m.form.setWidth(w)
}

// fetch issues a fresh list request. The context+cancel are stored on the model
// (Update runs on the main loop, so this mutation is safe); the closure
// performs
// the only I/O, off the Update path.
func (m *Model) fetch() tea.Cmd {
	ctx := m.newContext()
	e := m.epoch
	api := m.ctx.API
	return func() tea.Msg {
		gs, err := api.Groups(ctx)
		return groupsMsg{epoch: e, groups: gs, err: err}
	}
}

// newContext cancels any prior request and returns a fresh timeout context
// whose
// cancel is stored on the model.
func (m *Model) newContext() context.Context {
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	m.cancel = cancel
	return ctx
}

// addCmd creates a group off the Update path, emitting a mutationMsg.
func (m *Model) addCmd(name, comment string) tea.Cmd {
	ctx := m.newContext()
	e := m.epoch
	api := m.ctx.API
	return func() tea.Msg {
		_, err := api.AddGroup(ctx, name, comment)
		return mutationMsg{epoch: e, err: err}
	}
}

// updateCmd updates a group off the Update path, emitting a mutationMsg.
func (m *Model) updateCmd(name, comment string, enabled bool) tea.Cmd {
	ctx := m.newContext()
	e := m.epoch
	api := m.ctx.API
	return func() tea.Msg {
		_, err := api.UpdateGroup(ctx, name, comment, enabled)
		return mutationMsg{epoch: e, err: err}
	}
}

// deleteCmd deletes a group off the Update path, emitting a mutationMsg.
func (m *Model) deleteCmd(name string) tea.Cmd {
	ctx := m.newContext()
	e := m.epoch
	api := m.ctx.API
	return func() tea.Msg {
		err := api.DeleteGroup(ctx, name)
		return mutationMsg{epoch: e, err: err}
	}
}

// selected returns the group under the cursor, if any.
func (m *Model) selected() (pihole.Group, bool) {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.groups) {
		return pihole.Group{}, false
	}
	return m.groups[idx], true
}

// Update handles messages. It never performs I/O — all network access is
// deferred to tea.Cmd closures, and stale (wrong-epoch) results are dropped.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case groupsMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.groups = msg.groups
		m.syncRows()
		return m, nil

	case mutationMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		if msg.err != nil {
			m.loading = false
			m.err = msg.err
			return m, nil
		}
		// Success: reload the list to reflect the change.
		m.loading = true
		return m, tea.Batch(m.fetch(), m.spinner.Tick)

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey routes keystrokes through the confirm dialog, the form, and the
// list table in priority order.
func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.confirm.Active {
		return m.handleConfirmKey(msg)
	}
	if m.form.active {
		return m.handleFormKey(msg)
	}
	return m.handleListKey(msg)
}

// handleConfirmKey resolves the delete confirmation dialog.
func (m *Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		name := m.pendingDelete
		m.confirm = m.confirm.Hide()
		m.pendingDelete = ""
		if name == "" {
			return m, nil
		}
		return m, m.deleteCmd(name)
	case "n", "esc":
		m.confirm = m.confirm.Hide()
		m.pendingDelete = ""
		return m, nil
	}
	return m, nil
}

// handleFormKey resolves the inline add/edit form.
func (m *Model) handleFormKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if !m.form.canSubmit() {
			return m, nil
		}
		name, comment, enabled, editing := m.form.values()
		m.form.close()
		if editing {
			return m, m.updateCmd(name, comment, enabled)
		}
		return m, m.addCmd(name, comment)
	case "esc":
		m.form.close()
		return m, nil
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.update(msg)
	return m, cmd
}

// handleListKey handles the list-view bindings and otherwise drives the table.
func (m *Model) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		m.epoch++
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.fetch(), m.spinner.Tick)
	case "a":
		return m, m.form.openAdd(m.w)
	case "e":
		if g, ok := m.selected(); ok {
			return m, m.form.openEdit(g.Name, g.Comment, g.Enabled, m.w)
		}
		return m, nil
	case "space", "t":
		if g, ok := m.selected(); ok {
			return m, m.updateCmd(g.Name, g.Comment, !g.Enabled)
		}
		return m, nil
	case "x", "delete":
		if g, ok := m.selected(); ok {
			m.pendingDelete = g.Name
			m.confirm = m.confirm.Show("Delete group?", g.Name, true)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// syncRows rebuilds the table rows from the current groups and theme,
// preserving
// the cursor position.
func (m *Model) syncRows() {
	widths := columnWidthsFrom(m.table.Columns())
	idx := m.table.Cursor()
	m.table.SetStyles(components.TableStyles(m.ctx.Theme))
	m.table.SetRows(styledRows(m.ctx.Theme, m.groups, widths))
	m.table.SetCursor(idx)
}

// View renders the screen. The theme is read here so live re-themes and enabled
// glyph colours take effect immediately.
func (m *Model) View() tea.View {
	th := m.ctx.Theme

	if len(m.groups) > 0 {
		m.syncRows()
	}

	header := m.renderHeader(th)
	body := m.renderBody(th)
	footer := m.renderFooter(th)

	content := strings.Join([]string{header, body, footer}, "\n")
	view := th.SurfaceStyle().Width(m.w).Height(m.h).Render(content)

	if m.confirm.Active {
		view = m.confirm.Render(th, m.w, m.h)
	}
	return tea.NewView(view)
}

func (m *Model) renderHeader(th *theme.Theme) string {
	title := th.AccentStyle().Bold(true).Render(m.Title())

	enabled := 0
	for _, g := range m.groups {
		if g.Enabled {
			enabled++
		}
	}
	count := th.SubtleStyle().
		Render(fmt.Sprintf("%d groups · %d enabled", len(m.groups), enabled))

	gap := m.w - lipgloss.Width(title) - lipgloss.Width(count)
	if gap < 1 {
		gap = 1
	}
	line := title + strings.Repeat(" ", gap) + count

	sub := ""
	if m.err != nil {
		sub = m.errBanner(th)
	}
	return strings.Join([]string{line, sub}, "\n")
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

	if m.loading && len(m.groups) == 0 {
		line := m.spinner.View() + " " + th.SubtleStyle().
			Render("loading groups…")
		return lipgloss.Place(
			m.w,
			bodyH,
			lipgloss.Center,
			lipgloss.Center,
			line,
			th.SurfaceWhitespace(),
		)
	}

	if len(m.groups) == 0 {
		empty := th.SubtleStyle().
			Render("no groups configured · press a to add")
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

func (m *Model) renderFooter(th *theme.Theme) string {
	hint := "↑↓ navigate · a add · e edit · space toggle · x delete · r refresh"
	return th.SubtleStyle().Render(truncate(hint, maxInt(m.w, 8)))
}
