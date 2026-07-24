// Package adlists implements the PiHole v6 adlists screen: block/allow list CRUD
// over a bubbles table plus the streamed gravity update. It satisfies
// core.Screen. All I/O happens inside tea.Cmd closures; Update never blocks.
package adlists

import (
	"context"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/tui/components"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

const (
	fetchTimeout   = 8 * time.Second
	gravityTimeout = 5 * time.Minute

	headerHeight = 2 // title/chip line + spacer/banner line
	footerHeight = 1 // hint line
)

// confirmKind records which action a pending confirm dialog will run on "y".
type confirmKind int

const (
	confirmNone confirmKind = iota
	confirmDelete
	confirmGravity
)

// listsMsg carries a fetch of both list types, tagged with its issuing epoch.
type listsMsg struct {
	epoch int
	block []pihole.List
	allow []pihole.List
	err   error
}

// mutatedMsg reports the result of a create/update/delete, tagged with epoch.
// On success the screen refetches; on error it raises the inline banner.
type mutatedMsg struct {
	epoch int
	err   error
}

// Model is the adlists screen.
type Model struct {
	ctx *core.AppContext

	w, h int

	table    table.Model
	spinner  spinner.Model
	viewport viewport.Model
	form     formModel
	confirm  components.ConfirmDialog

	lists   map[pihole.ListType][]pihole.List
	visible pihole.ListType

	// pending target for a confirmed delete.
	pendingAddr string
	pendingType pihole.ListType
	confirmKind confirmKind

	// gravity streaming state.
	gravityOpen    bool
	gravityRunning bool
	gravityDone    bool
	gravityErr     error
	gravityLines   []string
	gravityChan    chan string
	gravityCancel  context.CancelFunc

	focused bool
	loading bool
	err     error

	epoch  int
	cancel context.CancelFunc
}

// New constructs the adlists screen from the shared app context.
func New(ctx *core.AppContext) *Model {
	t := table.New(
		table.WithColumns(tableColumns(computeColumnWidths(80))),
		table.WithFocused(true),
	)

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	vp := viewport.New()

	return &Model{
		ctx:      ctx,
		table:    t,
		spinner:  sp,
		viewport: vp,
		form:     newForm(),
		lists:    map[pihole.ListType][]pihole.List{},
		visible:  pihole.ListBlock,
	}
}

// Init satisfies tea.Model; work begins on Focus.
func (m *Model) Init() tea.Cmd { return nil }

// Title is shown in the header/status bar.
func (m *Model) Title() string { return "Adlists" }

// Focus activates the screen: bump epoch and fetch both list types.
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	m.epoch++
	m.loading = true
	m.err = nil
	return tea.Batch(m.fetch(), m.spinner.Tick)
}

// Blur deactivates the screen and cancels any in-flight request, including a
// running gravity stream (cancelling its context stops the stream).
func (m *Model) Blur() {
	m.focused = false
	m.form = m.form.close()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.gravityCancel != nil {
		m.gravityCancel()
		m.gravityCancel = nil
	}
}

// Help returns screen-local key bindings for the help bar.
func (m *Model) Help() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
		key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
		key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "gravity")),
		key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "type")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

// SetSize informs the screen of its inner content area and reflows children.
func (m *Model) SetSize(w, h int) {
	m.w, m.h = w, h

	m.table.SetColumns(tableColumns(computeColumnWidths(w)))
	m.table.SetWidth(w)

	bodyH := h - headerHeight - footerHeight
	if bodyH < 1 {
		bodyH = 1
	}
	m.table.SetHeight(bodyH)

	m.viewport.SetWidth(maxInt(w-2, 4))
	m.viewport.SetHeight(maxInt(bodyH-2, 1))

	m.form.setWidth(w)
}

// fetch issues a request for both list types. The context+cancel are stored on
// the model (Update runs on the main loop, so this mutation is safe); the
// closure performs the only I/O, off the Update path.
func (m *Model) fetch() tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	m.cancel = cancel

	e := m.epoch
	api := m.ctx.API

	return func() tea.Msg {
		block, err := api.Lists(ctx, pihole.ListBlock)
		if err != nil {
			return listsMsg{epoch: e, err: err}
		}
		allow, err := api.Lists(ctx, pihole.ListAllow)
		if err != nil {
			return listsMsg{epoch: e, err: err}
		}
		return listsMsg{epoch: e, block: block, allow: allow}
	}
}

// Update handles messages. It never performs I/O — all network access is
// deferred to tea.Cmd closures, and stale (wrong-epoch) results are dropped.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case listsMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.lists[pihole.ListBlock] = msg.block
		m.lists[pihole.ListAllow] = msg.allow
		m.syncRows()
		return m, nil

	case mutatedMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.loading = true
		return m, tea.Batch(m.fetch(), m.spinner.Tick)

	case gravityLineMsg:
		return m.handleGravityLine(msg)

	case gravityDoneMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.gravityRunning = false
		m.gravityDone = true
		m.gravityErr = msg.err
		return m, nil

	case spinner.TickMsg:
		if !m.loading && !m.gravityRunning {
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

// toggleVisible switches the visible list type and re-syncs the table rows.
func (m *Model) toggleVisible() {
	if m.visible == pihole.ListBlock {
		m.visible = pihole.ListAllow
	} else {
		m.visible = pihole.ListBlock
	}
	m.syncRows()
}

// selected returns the list under the table cursor for the visible type.
func (m *Model) selected() (pihole.List, bool) {
	rows := m.lists[m.visible]
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(rows) {
		return pihole.List{}, false
	}
	return rows[idx], true
}

// syncRows rebuilds the table rows from the visible lists and theme, preserving
// the cursor position.
func (m *Model) syncRows() {
	widths := computeColumnWidths(m.w)
	if m.w == 0 {
		widths = computeColumnWidths(80)
	}
	idx := m.table.Cursor()
	m.table.SetRows(styledRows(m.ctx.Theme, m.lists[m.visible], widths))
	m.table.SetCursor(idx)
}
