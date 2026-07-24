// Package tui contains the root Bubble Tea application: the router, persistent
// chrome (sidebar, status bar, help bar), and cross-screen message handling.
package tui

import (
	"context"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/config"
	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/components"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
	"github.com/zackkitzmiller/tihole/internal/tui/screens/adlists"
	"github.com/zackkitzmiller/tihole/internal/tui/screens/clients"
	"github.com/zackkitzmiller/tihole/internal/tui/screens/dashboard"
	"github.com/zackkitzmiller/tihole/internal/tui/screens/domains"
	"github.com/zackkitzmiller/tihole/internal/tui/screens/groups"
	"github.com/zackkitzmiller/tihole/internal/tui/screens/localdns"
	"github.com/zackkitzmiller/tihole/internal/tui/screens/placeholder"
	"github.com/zackkitzmiller/tihole/internal/tui/screens/querylog"
	"github.com/zackkitzmiller/tihole/internal/tui/screens/settings"
	"github.com/zackkitzmiller/tihole/internal/tui/screens/system"
)

const (
	chromeTop    = 1 // status bar
	chromeBottom = 1 // help bar
	fetchTimeout = 8 * time.Second
)

// blockingResultMsg carries a fresh blocking status.
type blockingResultMsg struct{ status pihole.BlockingStatus }

// blockingErrMsg reports a failed blocking fetch/toggle.
type blockingErrMsg struct{ err error }

// AppModel is the root Bubble Tea model.
type AppModel struct {
	ctx     *core.AppContext
	cfg     *config.Config
	cfgPath string

	screens map[core.PageID]core.Screen
	active  core.PageID

	help     help.Model
	showHelp bool
	palette  components.Palette

	width, height int
	isDark        bool

	block components.BlockState
	err   string
}

// New builds the root model from a validated config and an authenticated
// client for the active instance.
func New(cfg *config.Config, cfgPath string, api *pihole.Client, th *theme.Theme) *AppModel {
	ctx := &core.AppContext{
		API:          api,
		Theme:        th,
		Keys:         core.DefaultKeyMap(),
		InstanceName: cfg.Active,
		Config:       cfg,
		ConfigPath:   cfgPath,
	}
	m := &AppModel{
		ctx:     ctx,
		cfg:     cfg,
		cfgPath: cfgPath,
		active:  core.PageDashboard,
		help:    help.New(),
		palette: components.NewPalette(),
	}
	m.screens = buildScreens(ctx)
	return m
}

// buildScreens constructs one Screen per page. Pages not yet implemented get a
// placeholder so the whole app is navigable.
func buildScreens(ctx *core.AppContext) map[core.PageID]core.Screen {
	screens := make(map[core.PageID]core.Screen)
	for _, p := range core.PageOrder() {
		screens[p.ID] = placeholder.New(ctx, p.Title)
	}
	// Replace placeholders with real screens as they're implemented.
	screens[core.PageDashboard] = dashboard.New(ctx)
	screens[core.PageQueryLog] = querylog.New(ctx)
	screens[core.PageDomains] = domains.New(ctx)
	screens[core.PageGroups] = groups.New(ctx)
	screens[core.PageClients] = clients.New(ctx)
	screens[core.PageAdlists] = adlists.New(ctx)
	screens[core.PageLocalDNS] = localdns.New(ctx)
	screens[core.PageSettings] = settings.New(ctx)
	screens[core.PageSystem] = system.New(ctx)
	return screens
}

// Init focuses the initial screen and kicks off the first blocking fetch.
func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.screens[m.active].Focus(),
		fetchBlocking(m.ctx.API),
	)
}

// Update handles global chrome, routing, and delegation to the active screen.
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.SetWidth(msg.Width)
		cw, ch := m.contentSize()
		for _, s := range m.screens {
			s.SetSize(cw, ch)
		}
		return m, nil

	case tea.BackgroundColorMsg:
		m.isDark = msg.IsDark()
		m.ctx.Theme = m.ctx.Theme.Adapt(m.isDark)
		return m, nil

	case tea.KeyPressMsg:
		// The palette, while open, captures all key input.
		if m.palette.Active() {
			var cmd tea.Cmd
			m.palette, cmd = m.palette.Update(msg)
			return m, cmd
		}
		if cmd, handled := m.handleGlobalKey(msg); handled {
			return m, cmd
		}

	case core.NavigateMsg:
		return m, m.switchPage(msg.To)

	case core.SetThemeMsg:
		m.ctx.Theme = msg.Theme.Adapt(m.isDark)
		return m, nil

	case core.SwitchInstanceMsg:
		return m, m.switchInstance(msg.Name)

	case core.InstancesChangedMsg:
		return m, m.applyInstancesChange(msg.Config)

	case blockingResultMsg:
		st := msg.status
		m.block = components.BlockState{Known: true, Enabled: st.Blocking}
		if st.Timer != nil && *st.Timer > 0 {
			m.block.CountdownSecs = int(*st.Timer)
		}
		return m, nil

	case blockingErrMsg:
		m.err = msg.err.Error()
		return m, nil

	case core.ErrorMsg:
		if msg.Err != nil {
			m.err = msg.Err.Error()
		}
		return m, nil
	}

	// Delegate everything else to the active screen.
	updated, cmd := m.screens[m.active].Update(msg)
	if s, ok := updated.(core.Screen); ok {
		m.screens[m.active] = s
	}
	return m, cmd
}

// handleGlobalKey processes app-level bindings. It returns handled=false when
// the key should fall through to the active screen.
func (m *AppModel) handleGlobalKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	k := m.ctx.Keys
	switch {
	case key.Matches(msg, k.Quit):
		return tea.Quit, true
	case key.Matches(msg, k.Palette):
		var cmd tea.Cmd
		m.palette, cmd = m.palette.Open(m.paletteCommands())
		return cmd, true
	case key.Matches(msg, k.Help):
		m.showHelp = !m.showHelp
		return nil, true
	case key.Matches(msg, k.ToggleBlock):
		return m.toggleBlocking(), true
	case key.Matches(msg, k.SwitchInst):
		return m.switchInstance(m.nextInstance()), true
	case msg.String() == "tab":
		return m.switchPage(m.relPage(1)), true
	case msg.String() == "shift+tab":
		return m.switchPage(m.relPage(-1)), true
	}
	// Digit keys 1..9 jump directly to a page.
	if d := msg.String(); len(d) == 1 && d[0] >= '1' && d[0] <= '9' {
		idx := int(d[0] - '1')
		order := core.PageOrder()
		if idx < len(order) {
			return m.switchPage(order[idx].ID), true
		}
	}
	return nil, false
}

// contentSize returns the inner content area after subtracting chrome.
func (m *AppModel) contentSize() (int, int) {
	cw := m.width - components.SidebarWidth - 1 // -1 for the sidebar's right border
	ch := m.height - chromeTop - chromeBottom
	if cw < 0 {
		cw = 0
	}
	if ch < 0 {
		ch = 0
	}
	return cw, ch
}

// switchPage blurs the current screen and focuses the target.
func (m *AppModel) switchPage(to core.PageID) tea.Cmd {
	if to == m.active {
		return nil
	}
	m.screens[m.active].Blur()
	m.active = to
	m.err = ""
	return m.screens[m.active].Focus()
}

// relPage returns the PageID delta positions from the active page (wrapping).
func (m *AppModel) relPage(delta int) core.PageID {
	order := core.PageOrder()
	cur := 0
	for i, p := range order {
		if p.ID == m.active {
			cur = i
			break
		}
	}
	next := (cur + delta + len(order)) % len(order)
	return order[next].ID
}

// toggleBlocking flips global blocking and refreshes status.
func (m *AppModel) toggleBlocking() tea.Cmd {
	desired := !(m.block.Known && m.block.Enabled)
	api := m.ctx.API
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		if err := api.SetBlocking(ctx, desired, nil); err != nil {
			return blockingErrMsg{err}
		}
		st, err := api.Blocking(ctx)
		if err != nil {
			return blockingErrMsg{err}
		}
		return blockingResultMsg{st}
	}
}

// setBlockingFor disables blocking for the given number of seconds, then
// refreshes status. A zero duration disables until manually re-enabled.
func (m *AppModel) setBlockingFor(secs float64) tea.Cmd {
	api := m.ctx.API
	var timer *float64
	if secs > 0 {
		timer = &secs
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		if err := api.SetBlocking(ctx, false, timer); err != nil {
			return blockingErrMsg{err}
		}
		st, err := api.Blocking(ctx)
		if err != nil {
			return blockingErrMsg{err}
		}
		return blockingResultMsg{st}
	}
}

// paletteCommands builds the Ctrl-K command set from the current app state:
// page navigation, blocking actions, instance switches, and theme changes.
func (m *AppModel) paletteCommands() []components.Command {
	var cmds []components.Command

	for _, p := range core.PageOrder() {
		to := p.ID
		cmds = append(cmds, components.Command{
			Title: "Go to " + p.Title,
			Desc:  "navigate",
			Run:   core.Navigate(to),
		})
	}

	cmds = append(cmds,
		components.Command{Title: "Toggle blocking", Desc: "enable / disable", Run: m.toggleBlocking()},
		components.Command{Title: "Disable blocking 30s", Desc: "temporary", Run: m.setBlockingFor(30)},
		components.Command{Title: "Disable blocking 5m", Desc: "temporary", Run: m.setBlockingFor(300)},
		components.Command{Title: "Disable blocking 30m", Desc: "temporary", Run: m.setBlockingFor(1800)},
		components.Command{Title: "Disable blocking (until enabled)", Desc: "indefinite", Run: m.setBlockingFor(0)},
	)

	for _, inst := range m.cfg.Instances {
		if inst.Name == m.ctx.InstanceName {
			continue
		}
		name := inst.Name
		cmds = append(cmds, components.Command{
			Title: "Switch to " + name,
			Desc:  "instance",
			Run:   func() tea.Msg { return core.SwitchInstanceMsg{Name: name} },
		})
	}

	for _, name := range theme.Names() {
		tn := name
		cmds = append(cmds, components.Command{
			Title: "Theme: " + tn,
			Desc:  "appearance",
			Run:   themeCmd(tn),
		})
	}

	return cmds
}

// themeCmd resolves a theme by name and emits a SetThemeMsg (or an ErrorMsg).
func themeCmd(name string) tea.Cmd {
	return func() tea.Msg {
		th, err := theme.Resolve(name)
		if err != nil {
			return core.ErrorMsg{Screen: "theme", Err: err}
		}
		return core.SetThemeMsg{Theme: th}
	}
}

// nextInstance returns the name of the instance after the active one (wrapping).
func (m *AppModel) nextInstance() string {
	insts := m.cfg.Instances
	if len(insts) < 2 {
		return m.ctx.InstanceName
	}
	cur := 0
	for i, inst := range insts {
		if inst.Name == m.ctx.InstanceName {
			cur = i
			break
		}
	}
	return insts[(cur+1)%len(insts)].Name
}

// switchInstance rebuilds the client for the named instance and re-focuses.
func (m *AppModel) switchInstance(name string) tea.Cmd {
	if name == m.ctx.InstanceName {
		return nil
	}
	client, err := clientFor(m.cfg, name)
	if err != nil {
		m.err = err.Error()
		return nil
	}
	m.ctx.API = client
	m.ctx.InstanceName = name
	m.block = components.BlockState{}
	return tea.Batch(m.screens[m.active].Focus(), fetchBlocking(client))
}

// applyInstancesChange swaps in an edited config after Settings → Connections
// persists it. If the active instance still exists nothing else changes; if it
// was removed, the first remaining instance is activated.
func (m *AppModel) applyInstancesChange(cfg *config.Config) tea.Cmd {
	if cfg == nil {
		return nil
	}
	m.cfg = cfg
	m.ctx.Config = cfg

	for _, inst := range cfg.Instances {
		if inst.Name == m.ctx.InstanceName {
			return nil // active instance survived
		}
	}
	if len(cfg.Instances) == 0 {
		return nil
	}
	return m.switchInstance(cfg.Instances[0].Name)
}

// View composes the full screen: status bar, sidebar+content, help bar.
func (m *AppModel) View() tea.View {
	if m.width == 0 || m.height == 0 {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	th := m.ctx.Theme
	cw, ch := m.contentSize()

	status := components.StatusBar{
		Instance:   m.ctx.InstanceName,
		ScreenName: m.screens[m.active].Title(),
		Block:      m.block,
		Width:      m.width,
	}.Render(th)

	sidebar := components.Sidebar{
		Items:    sidebarItems(),
		Selected: activeIndex(m.active),
		Height:   ch,
	}.Render(th)

	var middle string
	if m.palette.Active() {
		// The palette takes over the body area as a centered modal.
		middle = m.palette.Render(th, m.width, ch)
	} else {
		content := m.screens[m.active].View().Content
		middle = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	}

	bottom := m.helpBar(cw + components.SidebarWidth + 1)

	full := lipgloss.JoinVertical(lipgloss.Left, status, middle, bottom)
	v := tea.NewView(full)
	v.AltScreen = true
	return v
}

// helpBar renders the bottom key hints (or an error banner when present).
func (m *AppModel) helpBar(width int) string {
	th := m.ctx.Theme
	if m.err != "" {
		return th.BlockStyle().Width(width).Render("⚠ " + m.err + "  (esc/r to retry)")
	}
	bindings := m.ctx.Keys.ShortHelp()
	bindings = append(bindings, m.screens[m.active].Help()...)
	view := m.help.ShortHelpView(bindings)
	return lipgloss.NewStyle().Width(width).Background(th.Panel).Render(view)
}

// fetchBlocking loads the current blocking status.
func fetchBlocking(api *pihole.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		st, err := api.Blocking(ctx)
		if err != nil {
			return blockingErrMsg{err}
		}
		return blockingResultMsg{st}
	}
}

func sidebarItems() []components.SidebarItem {
	order := core.PageOrder()
	items := make([]components.SidebarItem, len(order))
	for i, p := range order {
		items[i] = components.SidebarItem{Title: p.Title, Icon: p.Icon}
	}
	return items
}

func activeIndex(id core.PageID) int {
	for i, p := range core.PageOrder() {
		if p.ID == id {
			return i
		}
	}
	return 0
}
