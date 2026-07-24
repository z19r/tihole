// Package domains implements the PiHole v6 Domains screen: allow/deny ×
// exact/regex management. It presents a filterable table of managed domains with
// inline add/edit forms, an enabled toggle, and a guarded delete. Data is loaded
// on focus and on manual refresh (no continuous poll). It satisfies core.Screen.
package domains

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/components"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

const (
	fetchTimeout = 8 * time.Second

	headerHeight = 2 // title/chip line + spacer
	footerHeight = 1 // hint line
)

// errDomainRequired is surfaced in the banner when a form is submitted blank.
var errDomainRequired = errors.New("domain is required")

// domainsMsg carries a list fetch result, tagged with its issuing epoch.
type domainsMsg struct {
	epoch   int
	domains []pihole.Domain
	err     error
}

// mutationMsg carries the result of an add/update/delete, tagged with its epoch.
// On success the screen refetches the full list.
type mutationMsg struct {
	epoch int
	err   error
}

// Model is the Domains screen.
type Model struct {
	ctx *core.AppContext

	w, h int

	table   table.Model
	spinner spinner.Model

	domains []pihole.Domain // full fetched set
	visible []pihole.Domain // filtered subset, parallel to the table rows

	filter  filterTab
	form    *formState
	confirm components.ConfirmDialog
	pending *pihole.Domain // row awaiting delete confirmation

	focused bool
	loading bool
	err     error

	epoch  int
	cancel context.CancelFunc
}

// New constructs the Domains screen from the shared app context.
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
		filter:  filterAll,
	}
}

// Init satisfies tea.Model; work begins on Focus.
func (m *Model) Init() tea.Cmd { return nil }

// Title is shown in the header/status bar.
func (m *Model) Title() string { return "Domains" }

// Focus activates the screen and fetches a fresh list.
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
	m.form = nil
	m.confirm = m.confirm.Hide()
	m.pending = nil
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
		key.NewBinding(key.WithKeys(" ", "t"), key.WithHelp("space", "toggle")),
		key.NewBinding(key.WithKeys("x", "delete"), key.WithHelp("x", "delete")),
		key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter")),
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

	if m.form != nil {
		m.form.setWidth(w)
	}

	// Re-truncate cells against the new widths.
	if len(m.visible) > 0 {
		m.syncRows()
	}
}

// fetch issues a list request. The context+cancel are stored on the model
// (Update runs on the main loop, so this mutation is safe); the closure performs
// the only I/O, off the Update path.
func (m *Model) fetch() tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	m.cancel = cancel

	e := m.epoch
	api := m.ctx.API

	return func() tea.Msg {
		ds, err := api.Domains(ctx)
		return domainsMsg{epoch: e, domains: ds, err: err}
	}
}

// mutate wraps a write operation in a timeout context and returns a command that
// reports the outcome as a mutationMsg. Loading is set so the spinner shows.
func (m *Model) mutate(op func(context.Context, *pihole.Client) error) tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	m.cancel = cancel

	e := m.epoch
	api := m.ctx.API
	m.loading = true

	return tea.Batch(func() tea.Msg {
		return mutationMsg{epoch: e, err: op(ctx, api)}
	}, m.spinner.Tick)
}

// Update handles messages. It never performs I/O — all network access is
// deferred to tea.Cmd closures, and stale (wrong-epoch) results are dropped.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case domainsMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.domains = msg.domains
		m.applyFilter()
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
		m.epoch++
		m.loading = true
		m.err = nil
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
		d := m.pending
		m.confirm = m.confirm.Hide()
		m.pending = nil
		if d == nil {
			return m, nil
		}
		target := *d
		return m, m.mutate(func(ctx context.Context, api *pihole.Client) error {
			return api.DeleteDomain(ctx, pihole.DomainType(target.Type), pihole.DomainKind(target.Kind), target.Domain)
		})
	case "n", "esc":
		m.confirm = m.confirm.Hide()
		m.pending = nil
	}
	return m, nil
}

// handleFormKey drives the add/edit form: navigation, toggles, submit, cancel.
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

	// On the toggle fields, left/right/space flip the value instead of typing.
	if m.form.focus == fieldType || m.form.focus == fieldKind {
		switch msg.String() {
		case "left", "right", " ", "space":
			if m.form.focus == fieldType {
				m.form.toggleType()
			} else {
				m.form.toggleKind()
			}
		}
		return m, nil
	}

	return m, m.form.updateFocused(msg)
}

// submitForm validates and dispatches the add or update mutation, then closes
// the form.
func (m *Model) submitForm() (tea.Model, tea.Cmd) {
	f := m.form
	domain, comment, groups := f.values()
	if domain == "" {
		m.err = errDomainRequired
		return m, nil
	}

	dtype, dkind := f.dtype, f.dkind
	editing := f.editing
	enabled := true
	target := domain
	if editing {
		enabled = f.original.Enabled
		target = f.original.Domain
	}

	m.form = nil
	m.err = nil

	return m, m.mutate(func(ctx context.Context, api *pihole.Client) error {
		if editing {
			_, err := api.UpdateDomain(ctx, dtype, dkind, target, comment, groups, enabled)
			return err
		}
		_, err := api.AddDomain(ctx, dtype, dkind, domain, comment, groups)
		return err
	})
}

// handleTableKey routes list-level shortcuts and otherwise defers to the table.
func (m *Model) handleTableKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		m.epoch++
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.fetch(), m.spinner.Tick)

	case "f", "right":
		m.filter = m.filter.next()
		m.applyFilter()
		return m, nil
	case "left":
		m.filter = m.filter.prev()
		m.applyFilter()
		return m, nil

	case "a":
		m.form = newAddForm(m.filter)
		m.form.setWidth(m.w)
		return m, m.form.syncFocus()

	case "e":
		if d, ok := m.selected(); ok {
			m.form = newEditForm(d)
			m.form.setWidth(m.w)
			return m, m.form.syncFocus()
		}
		return m, nil

	case " ", "space", "t":
		return m.toggleSelected()

	case "x", "delete":
		if d, ok := m.selected(); ok {
			dc := d
			m.pending = &dc
			m.confirm = m.confirm.Show("Delete domain?", displayDomain(d), true)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// toggleSelected flips the enabled flag of the highlighted row.
func (m *Model) toggleSelected() (tea.Model, tea.Cmd) {
	d, ok := m.selected()
	if !ok {
		return m, nil
	}
	return m, m.mutate(func(ctx context.Context, api *pihole.Client) error {
		_, err := api.UpdateDomain(ctx, pihole.DomainType(d.Type), pihole.DomainKind(d.Kind), d.Domain, d.Comment, d.Groups, !d.Enabled)
		return err
	})
}

// selected returns the domain under the table cursor, if any.
func (m *Model) selected() (pihole.Domain, bool) {
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.visible) {
		return m.visible[idx], true
	}
	return pihole.Domain{}, false
}

// applyFilter recomputes the visible subset and rebuilds the table rows.
func (m *Model) applyFilter() {
	m.visible = filterDomains(m.domains, m.filter)
	m.syncRows()
}

// syncRows rebuilds the table rows from the current visible set and theme,
// clamping the cursor to the new bounds.
func (m *Model) syncRows() {
	widths := columnWidthsFrom(m.table.Columns())
	idx := m.table.Cursor()
	m.table.SetRows(styledRows(m.ctx.Theme, m.visible, widths))
	if idx >= len(m.visible) {
		idx = len(m.visible) - 1
	}
	if idx < 0 {
		idx = 0
	}
	m.table.SetCursor(idx)
}

// View renders the screen. The theme is read here so live re-themes and per-row
// colours take effect immediately.
func (m *Model) View() tea.View {
	th := m.ctx.Theme

	if len(m.visible) > 0 {
		m.syncRows()
	}

	header := m.renderHeader(th)
	body := m.renderBody(th)
	footer := m.renderFooter(th)

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
		Render(m.filter.label())

	count := th.SubtleStyle().Render(fmt.Sprintf("%d shown · %d total", len(m.visible), len(m.domains)))

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

func (m *Model) renderBody(th *theme.Theme) string {
	bodyH := m.h - headerHeight - footerHeight
	if bodyH < 1 {
		bodyH = 1
	}

	if m.confirm.Active {
		return m.confirm.Render(th, m.w, bodyH)
	}
	if m.form != nil {
		return m.form.render(th, m.w, bodyH)
	}

	if m.loading && len(m.domains) == 0 {
		line := m.spinner.View() + " " + th.SubtleStyle().Render("loading domains…")
		return lipgloss.Place(m.w, bodyH, lipgloss.Center, lipgloss.Center, line)
	}

	if len(m.visible) == 0 {
		empty := th.SubtleStyle().Render("no domains match this filter")
		return lipgloss.Place(m.w, bodyH, lipgloss.Center, lipgloss.Center, empty)
	}

	return m.table.View()
}

func (m *Model) renderFooter(th *theme.Theme) string {
	hint := "↑↓ navigate · a add · e edit · space toggle · x delete · f filter · r refresh"
	return th.SubtleStyle().Render(truncate(hint, maxInt(m.w, 8)))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
