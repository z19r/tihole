// Package system implements the PiHole v6 System/Tools screen: a tabbed surface
// over FTL diagnostics and maintenance. Tabs cover read-only Info panels, the
// diagnosis Messages table, the Network device table, a live DNS Log tail, and a
// menu of destructive Actions. Data loads lazily per tab on focus/switch; only
// the Log tab polls continuously. All I/O is deferred to tea.Cmd closures with a
// timeout context, stale (wrong-epoch) results are dropped, and errors surface in
// an inline banner. It satisfies core.Screen.
package system

import (
	"context"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/components"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

const (
	pollInterval = 2 * time.Second
	fetchTimeout = 8 * time.Second

	headerHeight = 2 // tab bar line + banner/spacer
	footerHeight = 1 // hint line
)

// mutationMsg carries the result of a delete/action write, tagged with its
// epoch. When reload is set the active tab is refetched on success.
type mutationMsg struct {
	epoch  int
	err    error
	reload bool
}

// Model is the System/Tools screen.
type Model struct {
	ctx *core.AppContext

	w, h int

	activeTab tab

	spinner  spinner.Model
	confirm  components.ConfirmDialog
	msgTable table.Model
	netTable table.Model
	viewport viewport.Model

	// Per-tab data.
	info     []infoSection
	messages []pihole.DiagnosisMessage
	msgCount int
	devices  []pihole.NetworkDevice
	logLines []string
	nextID   int

	actionCursor int

	// pendingOp is the write armed behind the confirm dialog; pendingReload
	// records whether success should refetch the active tab.
	pendingOp     func(context.Context, *pihole.Client) error
	pendingReload bool

	focused bool
	loading bool
	err     error

	epoch  int
	cancel context.CancelFunc
}

// New constructs the System screen from the shared app context.
func New(ctx *core.AppContext) *Model {
	mt := table.New(
		table.WithColumns(msgColumns(computeMsgWidths(80))),
		table.WithFocused(true),
	)
	nt := table.New(
		table.WithColumns(netColumns(computeNetWidths(80))),
		table.WithFocused(true),
	)
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	vp := viewport.New()

	return &Model{
		ctx:       ctx,
		activeTab: tabInfo,
		spinner:   sp,
		msgTable:  mt,
		netTable:  nt,
		viewport:  vp,
	}
}

// Init satisfies tea.Model; work begins on Focus.
func (m *Model) Init() tea.Cmd { return nil }

// Title is shown in the header/status bar.
func (m *Model) Title() string { return "System" }

// Focus activates the screen and loads the active tab's data.
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	m.epoch++
	m.err = nil
	if m.activeTab == tabLog {
		m.logLines = nil
		m.nextID = 0
	}
	return m.loadActive()
}

// Blur deactivates the screen, cancels any in-flight request, and stops polling.
func (m *Model) Blur() {
	m.focused = false
	m.confirm = m.confirm.Hide()
	m.pendingOp = nil
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

// Help returns screen-local key bindings for the help bar.
func (m *Model) Help() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab", "f"), key.WithHelp("tab", "next tab")),
		key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←→", "switch tab")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		key.NewBinding(key.WithKeys("x", "delete"), key.WithHelp("x", "delete")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run action")),
	}
}

// SetSize informs the screen of its inner content area and reflows both tables
// and the log viewport. Safe at tiny sizes: widths and heights clamp.
func (m *Model) SetSize(w, h int) {
	m.w, m.h = w, h

	bodyH := h - headerHeight - footerHeight
	if bodyH < 1 {
		bodyH = 1
	}

	mw := computeMsgWidths(w)
	m.msgTable.SetColumns(msgColumns(mw))
	m.msgTable.SetWidth(maxInt(w, 1))
	m.msgTable.SetHeight(bodyH)

	nw := computeNetWidths(w)
	m.netTable.SetColumns(netColumns(nw))
	m.netTable.SetWidth(maxInt(w, 1))
	m.netTable.SetHeight(bodyH)

	m.viewport.SetWidth(maxInt(w, 4))
	m.viewport.SetHeight(maxInt(bodyH, 1))
}

// loadActive returns the fetch command for the active tab (nil for Actions).
func (m *Model) loadActive() tea.Cmd {
	switch m.activeTab {
	case tabInfo:
		return m.fetchInfo()
	case tabMessages:
		return m.fetchMessages()
	case tabNetwork:
		return m.fetchNetwork()
	case tabLog:
		return m.startLogTail()
	default:
		return nil
	}
}

// switchTab activates a different tab and loads its data (epoch-guarded).
func (m *Model) switchTab(t tab) (tea.Model, tea.Cmd) {
	if t == m.activeTab {
		return m, nil
	}
	m.activeTab = t
	m.err = nil
	m.epoch++
	if t == tabLog {
		m.logLines = nil
		m.nextID = 0
		m.viewport.SetContent("")
	}
	return m, m.loadActive()
}

// mutate wraps a write in a timeout context and reports the outcome as a
// mutationMsg; loading is set so the spinner shows.
func (m *Model) mutate(op func(context.Context, *pihole.Client) error, reload bool) tea.Cmd {
	ctx := m.newCtx()
	e := m.epoch
	api := m.ctx.API
	m.loading = true

	return tea.Batch(func() tea.Msg {
		return mutationMsg{epoch: e, err: op(ctx, api), reload: reload}
	}, m.spinner.Tick)
}

// Update handles messages. It never performs I/O — all network access is
// deferred to tea.Cmd closures, and stale (wrong-epoch) results are dropped.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if !m.focused || m.activeTab != tabLog || msg.epoch != m.epoch {
			return m, nil
		}
		return m, tea.Batch(m.fetchLog(), m.tick())

	case infoMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		m.info = msg.sections
		return m, nil

	case messagesMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.messages = msg.messages
		m.msgCount = msg.count
		m.syncMsgRows()
		return m, nil

	case networkMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.devices = msg.devices
		m.syncNetRows()
		return m, nil

	case dnsLogMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.appendLog(msg.page)
		return m, nil

	case mutationMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		if msg.err != nil {
			m.loading = false
			m.err = destructiveHint(msg.err)
			return m, nil
		}
		m.err = nil
		if msg.reload {
			m.epoch++
			return m, m.loadActive()
		}
		m.loading = false
		return m, nil

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

// handleKey routes keystrokes through the confirm dialog, the global tab nav,
// and finally the active tab's handler.
func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.confirm.Active {
		return m.handleConfirmKey(msg)
	}

	switch msg.String() {
	case "tab", "f", "right":
		return m.switchTab(m.activeTab.next())
	case "left":
		return m.switchTab(m.activeTab.prev())
	case "r":
		m.epoch++
		return m, m.loadActive()
	}

	switch m.activeTab {
	case tabMessages:
		return m.handleMessagesKey(msg)
	case tabNetwork:
		return m.handleNetworkKey(msg)
	case tabLog:
		return m.handleLogKey(msg)
	case tabActions:
		return m.handleActionsKey(msg)
	default:
		return m, nil // Info is read-only.
	}
}

// handleConfirmKey interprets y/n on the armed destructive action.
func (m *Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		op := m.pendingOp
		reload := m.pendingReload
		m.confirm = m.confirm.Hide()
		m.pendingOp = nil
		if op == nil {
			return m, nil
		}
		return m, m.mutate(op, reload)
	case "n", "esc":
		m.confirm = m.confirm.Hide()
		m.pendingOp = nil
	}
	return m, nil
}

// View renders the screen. The theme is read here so live re-themes take effect
// immediately.
func (m *Model) View() tea.View {
	th := m.ctx.Theme

	header := m.renderHeader(th)
	body := m.renderBody(th)
	footer := m.renderFooter(th)

	content := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	view := th.SurfaceStyle().Width(m.w).Height(m.h).Render(content)
	return tea.NewView(view)
}

// renderHeader draws the title, the tab bar, and any error banner.
func (m *Model) renderHeader(th *theme.Theme) string {
	title := th.AccentStyle().Bold(true).Render(m.Title())

	var chips []string
	for t := tab(0); t < tabCount; t++ {
		label := " " + t.label() + " "
		style := th.SubtleStyle()
		if t == m.activeTab {
			style = lipgloss.NewStyle().Foreground(th.Surface).Background(th.Accent).Bold(true)
		}
		chips = append(chips, style.Render(label))
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Center, chips...)
	line := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", bar)

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

// renderBody dispatches to the active tab's renderer, with the confirm dialog
// overlaying everything when active.
func (m *Model) renderBody(th *theme.Theme) string {
	bodyH := m.h - headerHeight - footerHeight
	if bodyH < 1 {
		bodyH = 1
	}

	if m.confirm.Active {
		return m.confirm.Render(th, m.w, bodyH)
	}

	switch m.activeTab {
	case tabInfo:
		return m.renderInfo(th, bodyH)
	case tabMessages:
		return m.renderMessages(th, bodyH)
	case tabNetwork:
		return m.renderNetwork(th, bodyH)
	case tabLog:
		return m.renderLog(th, bodyH)
	case tabActions:
		return m.renderActions(th, bodyH)
	default:
		return ""
	}
}

// spinnerLine centers the loading spinner with a label in the body area.
func (m *Model) spinnerLine(th *theme.Theme, label string, bodyH int) string {
	line := m.spinner.View() + " " + th.SubtleStyle().Render(label)
	return lipgloss.Place(m.w, bodyH, lipgloss.Center, lipgloss.Center, line)
}

func (m *Model) renderFooter(th *theme.Theme) string {
	hint := "tab switch · ←→ tabs · r refresh · x delete · enter action"
	return th.SubtleStyle().Render(truncate(hint, maxInt(m.w, 8)))
}

var _ core.Screen = (*Model)(nil)
var _ tea.Model = (*Model)(nil)
