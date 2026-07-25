// Package settings implements the PiHole v6 Settings screen. It has two modes:
// a config-tree browser/editor over FTL's live configuration (GET/PATCH
// /config), and a Connections sub-screen for managing configured instances and
// the active theme. Config data is loaded on focus and on manual refresh; all
// network I/O is deferred to tea.Cmd closures and guarded by an epoch so stale
// results are dropped. It satisfies core.Screen.
package settings

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/config"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/components"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

// screenMode selects between the config-tree view and the Connections view.
type screenMode int

const (
	modeConfig screenMode = iota
	modeConnections
)

// configMsg carries a config-tree fetch result, tagged with its issuing epoch.
type configMsg struct {
	epoch int
	tree  map[string]any
	err   error
}

// patchMsg carries the result of a config PATCH, tagged with its epoch. On
// success the screen refetches the tree.
type patchMsg struct {
	epoch int
	err   error
}

// Model is the Settings screen.
type Model struct {
	ctx *core.AppContext

	w, h int
	mode screenMode

	spinner spinner.Model

	// Mode A — config tree.
	tree        table.Model
	filterInput textinput.Model
	leaves      []leaf // full flattened set
	visible     []leaf // filtered subset, parallel to the tree rows
	filter      string
	filtering   bool
	edit        *leafEdit

	// Mode B — connections.
	connTable   table.Model
	connForm    *instanceForm
	confirm     components.ConfirmDialog
	pendingDel  int
	themePicker *themePicker

	focused bool
	loading bool
	err     error

	epoch  int
	cancel context.CancelFunc
}

// New constructs the Settings screen from the shared app context.
func New(ctx *core.AppContext) *Model {
	tree := table.New(
		table.WithColumns(treeColumns(computeTreeWidths(80))),
		table.WithFocused(true),
	)
	conn := table.New(
		table.WithColumns(connColumns(computeConnWidths(80))),
		table.WithFocused(true),
	)

	fi := textinput.New()
	fi.Prompt = "/"
	fi.Placeholder = "path"
	fi.CharLimit = 128

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))

	return &Model{
		ctx:         ctx,
		mode:        modeConfig,
		spinner:     sp,
		tree:        tree,
		filterInput: fi,
		connTable:   conn,
		pendingDel:  -1,
	}
}

// Init satisfies tea.Model; work begins on Focus.
func (m *Model) Init() tea.Cmd { return nil }

// Title is shown in the header/status bar.
func (m *Model) Title() string { return "Settings" }

// CapturesInput reports whether a text field or modal is focused, so the root
// delivers raw keys instead of firing global shortcuts (see
// core.InputCapturer).
func (m *Model) CapturesInput() bool {
	return m.filtering || m.edit != nil || m.connForm != nil ||
		m.themePicker != nil
}

// Focus activates the screen and fetches the live config tree.
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	m.epoch++
	m.loading = true
	m.err = nil
	if m.ctx.Config != nil {
		m.rebuildConnRows()
	}
	return tea.Batch(m.fetch(), m.spinner.Tick)
}

// Blur deactivates the screen and cancels any in-flight request.
func (m *Model) Blur() {
	m.focused = false
	m.filtering = false
	m.filterInput.Blur()
	m.edit = nil
	m.connForm = nil
	m.confirm = m.confirm.Hide()
	m.pendingDel = -1
	m.themePicker = nil
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

// Help returns screen-local key bindings for the help bar. The set differs by
// mode: config-tree bindings versus Connections bindings.
func (m *Model) Help() []key.Binding {
	if m.mode == modeConnections {
		return []key.Binding{
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
			key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
			key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "theme")),
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "config")),
		}
	}
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "edit")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "connections")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

// SetSize informs the screen of its inner content area and reflows both tables.
func (m *Model) SetSize(w, h int) {
	m.w, m.h = w, h

	m.tree.SetColumns(treeColumns(computeTreeWidths(w)))
	m.tree.SetWidth(w)
	m.connTable.SetColumns(connColumns(computeConnWidths(w)))
	m.connTable.SetWidth(w)

	bodyH := h - headerHeight - footerHeight
	if bodyH < 1 {
		bodyH = 1
	}
	m.tree.SetHeight(bodyH)
	// Reserve two lines for the theme summary under the instances table.
	connH := bodyH - 2
	if connH < 1 {
		connH = 1
	}
	m.connTable.SetHeight(connH)

	sw := clampMin(w-4, 4)
	m.filterInput.SetWidth(sw)

	if m.connForm != nil {
		m.connForm.setWidth(w)
	}
	if m.edit != nil {
		m.edit.setWidth(w)
	}
	if len(m.visible) > 0 {
		m.syncTreeRows()
	}
	if m.ctx.Config != nil {
		m.rebuildConnRows()
	}
}

// fetch issues a config-tree request off the Update path, guarded by the epoch.
func (m *Model) fetch() tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	m.cancel = cancel

	e := m.epoch
	api := m.ctx.API

	return func() tea.Msg {
		tree, err := api.Config(ctx, true)
		return configMsg{epoch: e, tree: tree, err: err}
	}
}

// patch issues a config PATCH off the Update path, guarded by the epoch.
func (m *Model) patch(p map[string]any) tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	m.cancel = cancel

	e := m.epoch
	api := m.ctx.API
	m.loading = true

	return tea.Batch(func() tea.Msg {
		return patchMsg{epoch: e, err: api.PatchConfig(ctx, p, false)}
	}, m.spinner.Tick)
}

// emitInstancesChanged returns a command that announces a persisted instance
// list change to the root.
func emitInstancesChanged(c *config.Config) tea.Cmd {
	return func() tea.Msg { return core.InstancesChangedMsg{Config: c} }
}

// emitSetTheme returns a command that requests a live re-theme.
func emitSetTheme(t *theme.Theme) tea.Cmd {
	return func() tea.Msg { return core.SetThemeMsg{Theme: t} }
}

// Update handles messages. It never performs I/O — network access is deferred
// to tea.Cmd closures, and stale (wrong-epoch) results are dropped.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case configMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.leaves = flattenConfig(msg.tree)
		m.applyFilter()
		return m, nil

	case patchMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		if msg.err != nil {
			m.loading = false
			m.err = msg.err
			return m, nil
		}
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

// handleKey routes keystrokes through overlays and per-mode handlers, in
// precedence order.
func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.confirm.Active {
		return m.handleConfirmKey(msg)
	}
	if m.connForm != nil {
		return m.handleConnFormKey(msg)
	}
	if m.themePicker != nil {
		return m.handleThemePickerKey(msg)
	}
	if m.edit != nil {
		return m.handleEditKey(msg)
	}
	if m.filtering {
		return m.handleFilterKey(msg)
	}
	switch msg.String() {
	case "tab":
		m.cycleMode(1)
		return m, nil
	case "shift+tab":
		m.cycleMode(-1)
		return m, nil
	}
	if m.mode == modeConnections {
		return m.handleConnKey(msg)
	}
	return m.handleTreeKey(msg)
}

// cycleMode advances the active section tab by delta, wrapping around, and
// rebuilds the connections rows when landing there. The single-key c toggle and
// the tab keys share this so both stay in lockstep.
func (m *Model) cycleMode(delta int) {
	n := len(sectionLabels)
	next := screenMode((int(m.mode) + delta + n) % n)
	if next == m.mode {
		return
	}
	m.mode = next
	m.err = nil
	if next == modeConnections && m.ctx.Config != nil {
		m.rebuildConnRows()
	}
}

// handleTreeKey routes config-tree shortcuts and otherwise defers to the table.
func (m *Model) handleTreeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "c":
		m.mode = modeConnections
		if m.ctx.Config != nil {
			m.rebuildConnRows()
		}
		return m, nil
	case "r":
		m.epoch++
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.fetch(), m.spinner.Tick)
	case "/":
		m.filtering = true
		m.filterInput.SetValue(m.filter)
		m.filterInput.CursorEnd()
		return m, m.filterInput.Focus()
	case "enter":
		if l, ok := m.selectedLeaf(); ok {
			m.edit = newLeafEdit(l)
			m.edit.setWidth(m.w)
			return m, nil
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.tree, cmd = m.tree.Update(msg)
	return m, cmd
}

// handleFilterKey drives the substring filter input.
func (m *Model) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filter = strings.TrimSpace(m.filterInput.Value())
		m.filtering = false
		m.filterInput.Blur()
		m.applyFilter()
		return m, nil
	case "esc":
		m.filtering = false
		m.filterInput.Blur()
		m.filterInput.SetValue(m.filter)
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	return m, cmd
}

// handleEditKey drives the inline leaf editor: bool toggle or text entry, save,
// cancel.
func (m *Model) handleEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.edit = nil
		return m, nil
	case "enter":
		return m.submitEdit()
	}

	if m.edit.isBool {
		switch msg.String() {
		case "left", "right", " ", "space":
			m.edit.boolVal = !m.edit.boolVal
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.edit.input, cmd = m.edit.input.Update(msg)
	return m, cmd
}

// submitEdit parses the edited value, builds a nested patch from the dotted
// path, and dispatches the PATCH. A parse error keeps the editor open with a
// banner.
func (m *Model) submitEdit() (tea.Model, tea.Cmd) {
	val, err := m.edit.value()
	if err != nil {
		m.err = err
		return m, nil
	}
	patch := buildPatch(m.edit.leaf.path, val)
	m.edit = nil
	m.err = nil
	return m, m.patch(patch)
}

// selectedLeaf returns the leaf under the tree cursor, if any.
func (m *Model) selectedLeaf() (leaf, bool) {
	idx := m.tree.Cursor()
	if idx >= 0 && idx < len(m.visible) {
		return m.visible[idx], true
	}
	return leaf{}, false
}

// applyFilter recomputes the visible subset and rebuilds the tree rows.
func (m *Model) applyFilter() {
	m.visible = filterLeaves(m.leaves, m.filter)
	m.syncTreeRows()
}

// syncTreeRows rebuilds the tree table rows from the visible set and theme,
// clamping the cursor to the new bounds.
func (m *Model) syncTreeRows() {
	widths := computeTreeWidths(m.w)
	idx := m.tree.Cursor()
	m.tree.SetStyles(components.TableStyles(m.ctx.Theme))
	m.tree.SetRows(treeRows(m.ctx.Theme, m.visible, widths))
	if idx >= len(m.visible) {
		idx = len(m.visible) - 1
	}
	if idx < 0 {
		idx = 0
	}
	m.tree.SetCursor(idx)
}

// View renders the screen. The theme is read here so live re-themes take effect
// immediately.
func (m *Model) View() tea.View {
	th := m.ctx.Theme

	if m.mode == modeConfig && len(m.visible) > 0 {
		m.syncTreeRows()
	}
	if m.mode == modeConnections && m.ctx.Config != nil {
		m.rebuildConnRows()
	}

	header := m.renderHeader(th)
	body := m.renderBody(th)
	footer := m.renderFooter(th)

	// Join with plain newlines rather than lipgloss.JoinVertical: JoinVertical
	// pads short lines to the block width with *unstyled* spaces, which bleed
	// the terminal background. Newline-joined, the outer SurfaceStyle pads
	// every line
	// itself and re-applies the surface background across the padding.
	content := strings.Join([]string{header, body, footer}, "\n")
	view := th.SurfaceStyle().Width(m.w).Height(m.h).Render(content)
	return tea.NewView(view)
}

// sectionLabels is the ordered tab set, indexed by screenMode.
var sectionLabels = []string{"Configuration", "Connections"}

// sectionBlurb is the one-line explanation shown beneath the tabs for each
// mode,
// so the screen teaches what it does instead of relying on muscle memory.
func sectionBlurb(mode screenMode) string {
	if mode == modeConnections {
		return "Manage Pi-hole instances and the app theme · " +
			"a add · e edit · x delete · t theme"
	}
	return "Live FTL configuration · enter edits a key and writes it " +
		"back instantly · / filter · r refresh"
}

func (m *Model) renderHeader(th *theme.Theme) string {
	var right string
	if m.mode == modeConnections {
		right = fmt.Sprintf("%d instances", m.instanceCount())
	} else {
		right = fmt.Sprintf(
			"%d shown · %d total",
			len(m.visible),
			len(m.leaves),
		)
	}

	tabs := components.SectionTabs{
		Title:  m.Title(),
		Labels: sectionLabels,
		Active: int(m.mode),
		Right:  right,
		Width:  m.w,
	}.Render(th)

	explain := th.SubtleStyle().
		Render(truncate(sectionBlurb(m.mode), maxInt(m.w, 8)))

	second := ""
	if m.filtering {
		second = m.filterInput.View()
	} else if m.err != nil {
		second = m.errBanner(th)
	}
	// Newline-joined, not JoinVertical: the outer SurfaceStyle in View pads
	// each
	// line with the surface background, so short header rows don't bleed.
	return strings.Join([]string{tabs, explain, second}, "\n")
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

	if m.mode == modeConnections {
		return m.renderConnBody(th, bodyH)
	}

	if m.edit != nil {
		return m.edit.render(th, m.w, bodyH)
	}
	if m.loading && len(m.leaves) == 0 {
		line := m.spinner.View() + " " + th.SubtleStyle().
			Render("loading config…")
		return lipgloss.Place(
			m.w,
			bodyH,
			lipgloss.Center,
			lipgloss.Center,
			line,
			surfaceWhitespace(th),
		)
	}
	if len(m.visible) == 0 {
		empty := th.SubtleStyle().Render("no config keys match this filter")
		return lipgloss.Place(
			m.w,
			bodyH,
			lipgloss.Center,
			lipgloss.Center,
			empty,
			surfaceWhitespace(th),
		)
	}
	return m.tree.View()
}

func (m *Model) renderFooter(th *theme.Theme) string {
	hint := "↑↓ navigate · enter edit · / filter · tab section · r refresh"
	if m.mode == modeConnections {
		hint = "↑↓ navigate · a add · e edit · x delete · t theme · tab section"
	}
	return th.SubtleStyle().Render(truncate(hint, maxInt(m.w, 8)))
}
