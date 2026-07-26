// Package clients implements the PiHole v6 clients screen: a table of
// configured clients with inline add/edit forms, guarded deletes, and a
// suggestions pane sourced from /api/clients/_suggestions. Fetches are
// epoch-guarded and Blur cancels any in-flight request. It satisfies
// core.Screen.
package clients

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

// clientsMsg carries a client-list fetch result, tagged with its issuing epoch.
type clientsMsg struct {
	epoch   int
	clients []pihole.ClientEntry
	err     error
}

// suggestionsMsg carries a suggestions fetch result, tagged with its epoch.
type suggestionsMsg struct {
	epoch       int
	suggestions []pihole.ClientSuggestion
	err         error
}

// mutationMsg carries the result of an add/update/delete, tagged with its
// epoch.
type mutationMsg struct {
	epoch int
	err   error
}

// Model is the clients screen.
type Model struct {
	ctx *core.AppContext

	w, h int

	table   table.Model
	spinner spinner.Model

	clients []pihole.ClientEntry

	form    clientForm
	suggest suggestState
	confirm components.ConfirmDialog

	pendingDelete string // client id awaiting delete confirmation

	focused bool
	loading bool
	err     error

	epoch  int
	cancel context.CancelFunc
}

// New constructs the clients screen from the shared app context.
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
		form:    newForm(),
	}
}

// Init satisfies tea.Model; work begins on Focus.
func (m *Model) Init() tea.Cmd { return nil }

// Title is shown in the header/status bar.
func (m *Model) Title() string { return "Clients" }

// CapturesInput reports whether the add/edit form is focused, so the root
// delivers raw keys instead of firing global shortcuts (see
// core.InputCapturer).
func (m *Model) CapturesInput() bool { return m.form.active }

// Focus activates the screen: bump the epoch and fetch a fresh client list.
// There is no continuous poll — `r` triggers a manual refetch.
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	m.epoch++
	m.loading = true
	m.err = nil
	return tea.Batch(m.fetch(), m.spinner.Tick)
}

// Blur deactivates the screen and cancels any in-flight request.
func (m *Model) Blur() {
	m.focused = false
	m.form.close()
	m.suggest.active = false
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
		key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
		key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "suggestions")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

// SetSize informs the screen of its inner content area and reflows the table
// and form inputs.
func (m *Model) SetSize(w, h int) {
	m.w, m.h = w, h

	widths := computeColumnWidths(w)
	m.table.SetColumns(tableColumns(widths))
	m.table.SetWidth(w)

	m.table.SetHeight(m.bodyHeight())

	m.form.setWidth(minInt(w, 64))
}

// bodyHeight returns the height available for the table/overlay body, i.e. the
// inner area minus the header and footer, clamped to at least one line.
func (m *Model) bodyHeight() int {
	h := m.h - headerHeight - footerHeight
	if h < 1 {
		return 1
	}
	return h
}

// newCtx cancels any prior in-flight request and returns a fresh timeout
// context stored on the model (Update runs on the main loop, so this mutation
// is safe). The closures returned by fetch/mutation cmds perform the only I/O.
func (m *Model) newCtx() context.Context {
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	m.cancel = cancel
	return ctx
}

// fetch issues a fresh client-list request stamped with the current epoch.
func (m *Model) fetch() tea.Cmd {
	ctx := m.newCtx()
	e := m.epoch
	api := m.ctx.API
	return func() tea.Msg {
		clients, err := api.Clients(ctx)
		return clientsMsg{epoch: e, clients: clients, err: err}
	}
}

// fetchSuggestions issues a suggestions request stamped with the current epoch.
func (m *Model) fetchSuggestions() tea.Cmd {
	ctx := m.newCtx()
	e := m.epoch
	api := m.ctx.API
	return func() tea.Msg {
		s, err := api.ClientSuggestions(ctx)
		return suggestionsMsg{epoch: e, suggestions: s, err: err}
	}
}

// addCmd / updateCmd / deleteCmd perform the mutating I/O off the Update path.
func (m *Model) addCmd(client, comment string, groups []int) tea.Cmd {
	ctx := m.newCtx()
	e := m.epoch
	api := m.ctx.API
	return func() tea.Msg {
		_, err := api.AddClient(ctx, client, comment, groups)
		return mutationMsg{epoch: e, err: err}
	}
}

func (m *Model) updateCmd(client, comment string, groups []int) tea.Cmd {
	ctx := m.newCtx()
	e := m.epoch
	api := m.ctx.API
	return func() tea.Msg {
		_, err := api.UpdateClient(ctx, client, comment, groups)
		return mutationMsg{epoch: e, err: err}
	}
}

func (m *Model) deleteCmd(client string) tea.Cmd {
	ctx := m.newCtx()
	e := m.epoch
	api := m.ctx.API
	return func() tea.Msg {
		err := api.DeleteClient(ctx, client)
		return mutationMsg{epoch: e, err: err}
	}
}

// reload bumps the epoch and refetches the client list.
func (m *Model) reload() tea.Cmd {
	m.epoch++
	m.loading = true
	m.err = nil
	return tea.Batch(m.fetch(), m.spinner.Tick)
}

// Update handles messages. It never performs I/O — all network access is
// deferred to tea.Cmd closures, and stale (wrong-epoch) results are dropped.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clientsMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.clients = msg.clients
		m.syncRows()
		return m, nil

	case suggestionsMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.suggest.loading = false
		if msg.err != nil {
			m.suggest.err = msg.err
			return m, nil
		}
		m.suggest.err = nil
		m.suggest.suggestions = msg.suggestions
		if m.suggest.cursor >= len(msg.suggestions) {
			m.suggest.cursor = 0
		}
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
		return m, m.reload()

	case spinner.TickMsg:
		if !m.loading && !m.suggest.loading {
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

// syncRows rebuilds the table rows from the current clients, preserving cursor.
func (m *Model) syncRows() {
	widths := columnWidthsFrom(m.table.Columns())
	idx := m.table.Cursor()
	m.table.SetStyles(components.TableStyles(m.ctx.Theme))
	m.table.SetRows(clientRows(m.clients, widths))
	m.table.SetCursor(idx)
}

// selected returns the client under the table cursor, if any.
func (m *Model) selected() (pihole.ClientEntry, bool) {
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.clients) {
		return m.clients[idx], true
	}
	return pihole.ClientEntry{}, false
}

// View renders the screen. The theme is read here so live re-themes take effect
// immediately.
func (m *Model) View() tea.View {
	th := m.ctx.Theme

	header := m.renderHeader(th)
	body := m.renderBody(th)
	footer := m.renderFooter(th)

	content := strings.Join([]string{header, body, footer}, "\n")
	view := th.SurfaceStyle().Width(m.w).Height(m.h).Render(content)
	return tea.NewView(view)
}

func (m *Model) renderHeader(th *theme.Theme) string {
	title := th.AccentStyle().Bold(true).Render(m.Title())
	count := th.SubtleStyle().Render(fmt.Sprintf("%d clients", len(m.clients)))

	gap := m.w - lipgloss.Width(title) - lipgloss.Width(count)
	if gap < 1 {
		gap = 1
	}
	line := title + strings.Repeat(" ", gap) + count

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
	bodyH := m.bodyHeight()

	if m.confirm.Active {
		return m.confirm.Render(th, m.w, bodyH)
	}
	if m.form.active {
		return m.form.render(th, m.w, bodyH)
	}
	if m.suggest.active {
		return m.suggest.render(th, m.spinner.View(), m.w, bodyH)
	}

	if m.loading && len(m.clients) == 0 {
		line := m.spinner.View() + " " + th.SubtleStyle().
			Render("loading clients…")
		return lipgloss.Place(
			m.w,
			bodyH,
			lipgloss.Center,
			lipgloss.Center,
			line,
			th.SurfaceWhitespace(),
		)
	}

	if len(m.clients) == 0 {
		empty := th.SubtleStyle().
			Render("no clients configured — a to add, S for suggestions")
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
	hint := "↑↓ navigate · a add · e edit · x delete · S suggestions · r refresh"
	return th.SubtleStyle().Render(truncate(hint, maxInt(m.w, 8)))
}
