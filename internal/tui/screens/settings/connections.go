package settings

import (
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/config"
	"github.com/zackkitzmiller/tihole/internal/theme"
)

const (
	glyphActive = "●"
)

// connColumnTitles is the header row for the instances table.
var connColumnTitles = []string{"Name", "URL", "Auth", "TLS", "Active"}

// authLabel describes an instance's credential source without ever revealing the
// secret: "inline" when a password is stored, "env:NAME" when sourced from the
// environment, and "—" when neither is configured.
func authLabel(inst config.Instance) string {
	if inst.Password != "" {
		return "inline"
	}
	if inst.PasswordEnv != "" {
		return "env:" + inst.PasswordEnv
	}
	return "—"
}

// computeConnWidths splits the total width across the five instance columns.
func computeConnWidths(total int) []int {
	const fixedTLS, fixedActive = 4, 6
	avail := total - 5*treeCellPad - fixedTLS - fixedActive
	if avail < 3*minTreeCol {
		avail = 3 * minTreeCol
	}
	name := avail * 2 / 10
	auth := avail * 3 / 10
	url := avail - name - auth
	return []int{
		clampMin(name, minTreeCol),
		clampMin(url, minTreeCol),
		clampMin(auth, minTreeCol),
		fixedTLS,
		fixedActive,
	}
}

// connColumns builds table columns from computed widths.
func connColumns(widths []int) []table.Column {
	cols := make([]table.Column, len(connColumnTitles))
	for i, title := range connColumnTitles {
		w := minTreeCol
		if i < len(widths) {
			w = widths[i]
		}
		cols[i] = table.Column{Title: title, Width: w}
	}
	return cols
}

// connRows builds styled rows for the instances table. The active instance is
// marked with a filled dot; no password value is ever rendered.
func connRows(th *theme.Theme, c *config.Config, widths []int) []table.Row {
	if c == nil {
		return nil
	}
	nameW, urlW, authW := minTreeCol, minTreeCol, minTreeCol
	if len(widths) == 5 {
		nameW, urlW, authW = widths[0], widths[1], widths[2]
	}
	rows := make([]table.Row, len(c.Instances))
	for i, inst := range c.Instances {
		tls := "no"
		if inst.VerifyTLSValue() {
			tls = "yes"
		}
		active := ""
		if inst.Name == c.Active {
			active = lipgloss.NewStyle().Foreground(th.Allow).Render(glyphActive)
		}
		rows[i] = table.Row{
			truncate(inst.Name, nameW),
			truncate(inst.URL, urlW),
			truncate(authLabel(inst), authW),
			tls,
			active,
		}
	}
	return rows
}

// cloneConfig returns a deep-enough copy of c: the instance slice and refresh
// map are freshly allocated so edits never mutate the shared config in place.
func cloneConfig(c *config.Config) *config.Config {
	insts := make([]config.Instance, len(c.Instances))
	copy(insts, c.Instances)

	var refresh map[string]int
	if c.Refresh != nil {
		refresh = make(map[string]int, len(c.Refresh))
		for k, v := range c.Refresh {
			refresh[k] = v
		}
	}

	return &config.Config{
		Active:    c.Active,
		Theme:     c.Theme,
		Refresh:   refresh,
		Instances: insts,
	}
}

// themePicker is the inline theme selector overlay.
type themePicker struct {
	names  []string
	cursor int
}

// newThemePicker lists the built-in theme names with the cursor on current.
func newThemePicker(current string) *themePicker {
	names := theme.Names()
	cursor := 0
	for i, n := range names {
		if n == current {
			cursor = i
			break
		}
	}
	return &themePicker{names: names, cursor: cursor}
}

func (p *themePicker) up() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *themePicker) down() {
	if p.cursor < len(p.names)-1 {
		p.cursor++
	}
}

func (p *themePicker) selected() string {
	if p.cursor >= 0 && p.cursor < len(p.names) {
		return p.names[p.cursor]
	}
	return ""
}

// render draws the theme picker as a centered panel.
func (p *themePicker) render(th *theme.Theme, w, h int, current string) string {
	return p.renderPanel(th, w, h, current)
}

func (p *themePicker) renderPanel(th *theme.Theme, width, height int, current string) string {
	heading := th.AccentStyle().Bold(true).Render("Select theme")
	lines := []string{heading, ""}
	for i, n := range p.names {
		marker := "  "
		style := th.TextStyle()
		if i == p.cursor {
			marker = "▸ "
			style = th.AccentStyle()
		}
		row := marker + style.Render(n)
		if n == current {
			row += " " + th.SubtleStyle().Render("(current)")
		}
		lines = append(lines, row)
	}
	lines = append(lines, "", th.SubtleStyle().Render("↑↓ move · enter select · esc cancel"))

	panelW := clampInt(width-4, 24, 48)
	panel := th.PanelStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Border).
		Padding(1, 2).
		Width(panelW).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Top, panel)
}

// --- Model methods for Connections mode ---

// rebuildConnRows resyncs the instances table from the current config.
func (m *Model) rebuildConnRows() {
	widths := computeConnWidths(m.w)
	m.connTable.SetColumns(connColumns(widths))
	m.connTable.SetWidth(m.w)
	idx := m.connTable.Cursor()
	m.connTable.SetRows(connRows(m.ctx.Theme, m.ctx.Config, widths))
	if n := m.instanceCount(); idx >= n {
		idx = n - 1
	}
	if idx < 0 {
		idx = 0
	}
	m.connTable.SetCursor(idx)
}

func (m *Model) instanceCount() int {
	if m.ctx.Config == nil {
		return 0
	}
	return len(m.ctx.Config.Instances)
}

// selectedInstance returns the highlighted instance and its index, if any.
func (m *Model) selectedInstance() (config.Instance, int, bool) {
	if m.ctx.Config == nil {
		return config.Instance{}, -1, false
	}
	idx := m.connTable.Cursor()
	if idx >= 0 && idx < len(m.ctx.Config.Instances) {
		return m.ctx.Config.Instances[idx], idx, true
	}
	return config.Instance{}, -1, false
}

// handleConnKey routes Connections-mode shortcuts and otherwise defers to the
// instances table.
func (m *Model) handleConnKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "c":
		m.mode = modeConfig
		return m, nil
	case "a":
		m.connForm = newAddInstanceForm()
		m.connForm.setWidth(m.w)
		return m, m.connForm.syncFocus()
	case "e":
		if inst, idx, ok := m.selectedInstance(); ok {
			m.connForm = newEditInstanceForm(inst, idx)
			m.connForm.setWidth(m.w)
			return m, m.connForm.syncFocus()
		}
		return m, nil
	case "x":
		if inst, idx, ok := m.selectedInstance(); ok {
			m.pendingDel = idx
			m.confirm = m.confirm.Show("Delete instance?", inst.Name, true)
		}
		return m, nil
	case "t":
		if m.ctx.Config != nil {
			m.themePicker = newThemePicker(m.ctx.Config.Theme)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.connTable, cmd = m.connTable.Update(msg)
	return m, cmd
}

// handleConnFormKey drives the instance add/edit form.
func (m *Model) handleConnFormKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.connForm = nil
		return m, nil
	case "enter":
		return m.submitConnForm()
	case "tab", "down":
		m.connForm.focusNext()
		return m, m.connForm.syncFocus()
	case "shift+tab", "up":
		m.connForm.focusPrev()
		return m, m.connForm.syncFocus()
	}

	if m.connForm.focus == fieldVerifyTLS {
		switch msg.String() {
		case "left", "right", " ", "space":
			m.connForm.toggleVerify()
		}
		return m, nil
	}

	return m, m.connForm.updateFocused(msg)
}

// submitConnForm validates and persists an add/edit against a config copy. On
// success it swaps the local config, refreshes the table, and emits
// InstancesChangedMsg; on validation/save failure it surfaces the banner and
// keeps the form open without emitting.
func (m *Model) submitConnForm() (tea.Model, tea.Cmd) {
	if m.ctx.Config == nil {
		m.connForm = nil
		return m, nil
	}
	f := m.connForm
	next := cloneConfig(m.ctx.Config)
	inst := f.instance()

	if f.editing && f.index >= 0 && f.index < len(next.Instances) {
		// Preserve active pointer to the (possibly renamed) instance.
		if next.Active == next.Instances[f.index].Name {
			next.Active = inst.Name
		}
		next.Instances[f.index] = inst
	} else {
		next.Instances = append(next.Instances, inst)
		if next.Active == "" {
			next.Active = inst.Name
		}
	}

	if err := config.Validate(next); err != nil {
		m.err = err
		return m, nil
	}
	if err := config.Save(m.ctx.ConfigPath, next); err != nil {
		m.err = err
		return m, nil
	}

	m.connForm = nil
	m.err = nil
	m.ctx.Config = next
	m.rebuildConnRows()
	return m, emitInstancesChanged(next)
}

// handleConfirmKey interprets y/n on the delete-instance confirmation.
func (m *Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		idx := m.pendingDel
		m.confirm = m.confirm.Hide()
		m.pendingDel = -1
		return m.deleteInstance(idx)
	case "n", "esc":
		m.confirm = m.confirm.Hide()
		m.pendingDel = -1
	}
	return m, nil
}

// deleteInstance removes the instance at idx from a config copy, keeps the
// active pointer valid, validates (which enforces ≥1 instance), persists, and
// emits InstancesChangedMsg. Any failure surfaces the banner without emitting.
func (m *Model) deleteInstance(idx int) (tea.Model, tea.Cmd) {
	if m.ctx.Config == nil || idx < 0 || idx >= len(m.ctx.Config.Instances) {
		return m, nil
	}
	next := cloneConfig(m.ctx.Config)
	removed := next.Instances[idx].Name
	next.Instances = append(next.Instances[:idx], next.Instances[idx+1:]...)

	if next.Active == removed && len(next.Instances) > 0 {
		next.Active = next.Instances[0].Name
	}

	if err := config.Validate(next); err != nil {
		m.err = err
		return m, nil
	}
	if err := config.Save(m.ctx.ConfigPath, next); err != nil {
		m.err = err
		return m, nil
	}

	m.err = nil
	m.ctx.Config = next
	m.rebuildConnRows()
	return m, emitInstancesChanged(next)
}

// handleThemePickerKey drives the theme selector overlay.
func (m *Model) handleThemePickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.themePicker = nil
		return m, nil
	case "up", "k":
		m.themePicker.up()
		return m, nil
	case "down", "j":
		m.themePicker.down()
		return m, nil
	case "enter":
		return m.selectTheme()
	}
	return m, nil
}

// selectTheme persists the chosen theme and emits SetThemeMsg for instant
// re-theme. On save failure it surfaces the banner without emitting.
func (m *Model) selectTheme() (tea.Model, tea.Cmd) {
	name := m.themePicker.selected()
	if name == "" || m.ctx.Config == nil {
		m.themePicker = nil
		return m, nil
	}
	next := cloneConfig(m.ctx.Config)
	next.Theme = name

	if err := config.Save(m.ctx.ConfigPath, next); err != nil {
		m.err = err
		return m, nil
	}

	m.themePicker = nil
	m.err = nil
	m.ctx.Config = next
	resolved, _ := theme.Resolve(name)
	return m, emitSetTheme(resolved)
}

// renderConnBody renders the Connections mode body: the form, confirm dialog, or
// theme picker overlay when active, otherwise the instances table plus a theme
// summary line.
func (m *Model) renderConnBody(th *theme.Theme, bodyH int) string {
	if m.confirm.Active {
		return m.confirm.Render(th, m.w, bodyH)
	}
	if m.connForm != nil {
		return m.connForm.render(th, m.w, bodyH)
	}
	if m.themePicker != nil {
		current := ""
		if m.ctx.Config != nil {
			current = m.ctx.Config.Theme
		}
		return m.themePicker.render(th, m.w, bodyH, current)
	}

	if m.instanceCount() == 0 {
		empty := th.SubtleStyle().Render("no instances configured")
		return lipgloss.Place(m.w, bodyH, lipgloss.Center, lipgloss.Center, empty)
	}

	table := m.connTable.View()
	summary := m.renderThemeSummary(th)
	return lipgloss.JoinVertical(lipgloss.Left, table, "", summary)
}

// renderThemeSummary shows the available themes with the active one marked.
func (m *Model) renderThemeSummary(th *theme.Theme) string {
	current := ""
	if m.ctx.Config != nil {
		current = m.ctx.Config.Theme
	}
	label := th.SubtleStyle().Render("Theme:")
	parts := []string{label}
	for _, n := range theme.Names() {
		if n == current {
			parts = append(parts, th.AccentStyle().Render(n+" "+glyphActive))
		} else {
			parts = append(parts, th.SubtleStyle().Render(n))
		}
	}
	line := lipgloss.JoinHorizontal(lipgloss.Left, joinSpaced(parts)...)
	return truncate(line, maxInt(m.w, 8))
}

// joinSpaced interleaves two-space separators between parts for JoinHorizontal.
func joinSpaced(parts []string) []string {
	if len(parts) == 0 {
		return parts
	}
	out := make([]string, 0, len(parts)*2-1)
	for i, p := range parts {
		if i > 0 {
			out = append(out, "  ")
		}
		out = append(out, p)
	}
	return out
}
