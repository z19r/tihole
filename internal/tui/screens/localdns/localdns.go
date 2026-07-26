// Package localdns implements the PiHole v6 Local DNS screen: local A/AAAA host
// records and CNAME records. It presents two filterable tables (toggled with a
// header chip) with an inline add form and a guarded delete. The config API has
// no single-record update, so records are add/delete only. Data is loaded on
// focus and on manual refresh (no continuous poll). It satisfies core.Screen.
package localdns

import (
	"context"
	"errors"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/tui/components"
	"github.com/z19r/tihole/internal/tui/core"
)

const (
	fetchTimeout = 8 * time.Second

	headerHeight = 2 // title/chip line + spacer
	footerHeight = 1 // hint line
)

// errRecordRequired is surfaced in the banner when a form is submitted with a
// missing required field.
var errRecordRequired = errors.New("all required fields must be filled")

// hostRecordsMsg carries a host-records fetch result, tagged with its epoch.
type hostRecordsMsg struct {
	epoch   int
	records []pihole.HostRecord
	err     error
}

// cnameRecordsMsg carries a CNAME-records fetch result, tagged with its epoch.
type cnameRecordsMsg struct {
	epoch   int
	records []pihole.CNAMERecord
	err     error
}

// mutationMsg carries the result of an add/delete, tagged with its epoch. On
// success the screen refetches both record lists.
type mutationMsg struct {
	epoch int
	err   error
}

// Model is the Local DNS screen.
type Model struct {
	ctx *core.AppContext

	w, h int

	hostTable  table.Model
	cnameTable table.Model
	spinner    spinner.Model

	hosts  []pihole.HostRecord
	cnames []pihole.CNAMERecord

	kind    recordKind
	form    *formState
	confirm components.ConfirmDialog

	pendingHost  *pihole.HostRecord
	pendingCNAME *pihole.CNAMERecord

	focused bool
	loading bool
	err     error

	epoch  int
	cancel context.CancelFunc
}

// New constructs the Local DNS screen from the shared app context.
func New(ctx *core.AppContext) *Model {
	ht := table.New(
		table.WithColumns(
			tableColumns(hostColumnTitles, computeHostColumnWidths(80)),
		),
		table.WithFocused(true),
	)
	ct := table.New(
		table.WithColumns(
			tableColumns(cnameColumnTitles, computeCNAMEColumnWidths(80)),
		),
		table.WithFocused(true),
	)

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))

	return &Model{
		ctx:        ctx,
		hostTable:  ht,
		cnameTable: ct,
		spinner:    sp,
		kind:       kindHosts,
	}
}

// Init satisfies tea.Model; work begins on Focus.
func (m *Model) Init() tea.Cmd { return nil }

// Title is shown in the header/status bar.
func (m *Model) Title() string { return "Local DNS" }

// CapturesInput reports whether the add form is open, so the root delivers raw
// keys instead of firing global shortcuts (see core.InputCapturer).
func (m *Model) CapturesInput() bool { return m.form != nil }

// Focus activates the screen and fetches both record lists.
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
	m.pendingHost = nil
	m.pendingCNAME = nil
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

// Help returns screen-local key bindings for the help bar.
func (m *Model) Help() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		key.NewBinding(
			key.WithKeys("x", "delete"),
			key.WithHelp("x", "delete"),
		),
		key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "switch kind")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

// SetSize informs the screen of its inner content area and reflows both tables.
func (m *Model) SetSize(w, h int) {
	m.w, m.h = w, h

	m.hostTable.SetColumns(
		tableColumns(hostColumnTitles, computeHostColumnWidths(w)),
	)
	m.hostTable.SetWidth(w)
	m.cnameTable.SetColumns(
		tableColumns(cnameColumnTitles, computeCNAMEColumnWidths(w)),
	)
	m.cnameTable.SetWidth(w)

	tableH := h - headerHeight - footerHeight
	if tableH < 1 {
		tableH = 1
	}
	m.hostTable.SetHeight(tableH)
	m.cnameTable.SetHeight(tableH)

	if m.form != nil {
		m.form.setWidth(w)
	}

	m.syncRows()
}

// fetch issues both list requests under a single timeout context. The
// context+cancel are stored on the model (Update runs on the main loop, so this
// mutation is safe); the closures perform the only I/O, off the Update path.
func (m *Model) fetch() tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	m.cancel = cancel

	e := m.epoch
	api := m.ctx.API

	hostsCmd := func() tea.Msg {
		rs, err := api.HostRecords(ctx)
		return hostRecordsMsg{epoch: e, records: rs, err: err}
	}
	cnamesCmd := func() tea.Msg {
		rs, err := api.CNAMERecords(ctx)
		return cnameRecordsMsg{epoch: e, records: rs, err: err}
	}
	return tea.Batch(hostsCmd, cnamesCmd)
}

// mutate wraps a write operation in a timeout context and returns a command
// that reports the outcome as a mutationMsg. Loading is set so the spinner
// shows.
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
	case hostRecordsMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.hosts = msg.records
		m.syncHostRows()
		return m, nil

	case cnameRecordsMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.cnames = msg.records
		m.syncCNAMERows()
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
		// Success: reload both lists to reflect the change.
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

// syncRows rebuilds both tables' rows from the current record sets.
func (m *Model) syncRows() {
	m.syncHostRows()
	m.syncCNAMERows()
}

// syncHostRows rebuilds the host table rows, clamping the cursor to bounds.
func (m *Model) syncHostRows() {
	widths := columnWidthsFrom(m.hostTable.Columns())
	idx := m.hostTable.Cursor()
	m.hostTable.SetStyles(components.TableStyles(m.ctx.Theme))
	m.hostTable.SetRows(styledHostRows(m.ctx.Theme, m.hosts, widths))
	m.hostTable.SetCursor(clampCursor(idx, len(m.hosts)))
}

// syncCNAMERows rebuilds the CNAME table rows, clamping the cursor to bounds.
func (m *Model) syncCNAMERows() {
	widths := columnWidthsFrom(m.cnameTable.Columns())
	idx := m.cnameTable.Cursor()
	m.cnameTable.SetStyles(components.TableStyles(m.ctx.Theme))
	m.cnameTable.SetRows(styledCNAMERows(m.ctx.Theme, m.cnames, widths))
	m.cnameTable.SetCursor(clampCursor(idx, len(m.cnames)))
}

func clampCursor(idx, n int) int {
	if idx >= n {
		idx = n - 1
	}
	if idx < 0 {
		idx = 0
	}
	return idx
}

// activeCount is the number of records shown under the active kind.
func (m *Model) activeCount() int {
	if m.kind == kindCNAMEs {
		return len(m.cnames)
	}
	return len(m.hosts)
}
