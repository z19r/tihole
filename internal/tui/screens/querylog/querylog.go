// Package querylog implements the PiHole v6 query-log screen: a cursor-paginated
// results table with /-triggered domain search, a row-detail pane, live ~2s
// polling while focused, and an inline error banner. It satisfies core.Screen.
package querylog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
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

	headerHeight = 2 // title/chip line + spacer
	footerHeight = 1 // hint line
)

// tickMsg drives the live poll. It carries the epoch active when scheduled so
// stale ticks (from a prior focus cycle) can be dropped.
type tickMsg struct{ epoch int }

// queriesMsg carries a page fetch result, tagged with its issuing epoch.
type queriesMsg struct {
	epoch int
	page  pihole.QueriesPage
	err   error
}

// classifyMsg reports the result of a quick allow/block. verb is the past-tense
// label for the success toast ("allowed"/"blocked").
type classifyMsg struct {
	domain string
	verb   string
	err    error
}

// noteExpireMsg clears the transient success toast.
type noteExpireMsg struct{}

// noteTTL is how long a quick-classify success toast stays on screen.
const noteTTL = 4 * time.Second

// classifyComment tags entries added from the query log so their origin is
// obvious in the Domains screen.
const classifyComment = "added from query log (tihole)"

// Model is the query-log screen.
type Model struct {
	ctx *core.AppContext

	w, h int

	table   table.Model
	search  textinput.Model
	spinner spinner.Model

	queries []pihole.Query // raw rows, parallel to the table
	detail  *pihole.Query  // non-nil while the detail pane is open

	// Quick-classify: allow/block the selected domain straight from the log.
	confirm       components.ConfirmDialog
	pendingType   pihole.DomainType // allow/deny awaiting confirmation
	pendingDomain string            // the domain awaiting confirmation
	note          string            // transient success toast (e.g. "allowed x")

	filterDomain string
	filtered     int
	total        int

	searching bool
	focused   bool
	loading   bool
	err       error

	epoch  int
	cancel context.CancelFunc
}

// New constructs the query-log screen from the shared app context.
func New(ctx *core.AppContext) *Model {
	widths := computeColumnWidths(80)

	t := table.New(
		table.WithColumns(tableColumns(widths)),
		table.WithFocused(true),
	)

	si := textinput.New()
	si.Prompt = "/"
	si.Placeholder = "domain"
	si.CharLimit = 253

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))

	return &Model{
		ctx:     ctx,
		table:   t,
		search:  si,
		spinner: sp,
	}
}

// Init satisfies tea.Model; work begins on Focus.
func (m *Model) Init() tea.Cmd { return nil }

// Title is shown in the header/status bar.
func (m *Model) Title() string { return "Query Log" }

// CapturesInput reports whether the screen wants raw keys instead of the root's
// global single-key shortcuts (see core.InputCapturer): while the search field
// is focused, or while the quick-classify confirm is up so y/n/esc stay local
// rather than firing d=toggle-block or the esc-to-rail climb.
func (m *Model) CapturesInput() bool { return m.searching || m.confirm.Active }

// Focus activates the screen: start the poller and fetch a fresh page.
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	m.epoch++
	m.loading = true
	m.err = nil
	return tea.Batch(m.fetch(), m.tick(), m.spinner.Tick)
}

// Blur deactivates the screen and cancels any in-flight request.
func (m *Model) Blur() {
	m.focused = false
	m.searching = false
	m.search.Blur()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

// Help returns screen-local key bindings for the help bar.
func (m *Model) Help() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "detail")),
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "allow")),
		key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "block")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
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

	sw := w - 4
	if sw < 4 {
		sw = 4
	}
	m.search.SetWidth(sw)
}

// tick schedules the next poll, stamped with the current epoch.
func (m *Model) tick() tea.Cmd {
	e := m.epoch
	return tea.Tick(pollInterval, func(time.Time) tea.Msg { return tickMsg{e} })
}

// fetch issues a fresh-cursor page request. The context+cancel are stored on
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
	filter := buildFilter(m.filterDomain, defaultPage, nil)

	return func() tea.Msg {
		page, err := api.Queries(ctx, filter)
		return queriesMsg{epoch: e, page: page, err: err}
	}
}

// Update handles messages. It never performs I/O — all network access is
// deferred to tea.Cmd closures, and stale (wrong-epoch) results are dropped.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if !m.focused || m.searching || msg.epoch != m.epoch {
			return m, nil
		}
		return m, tea.Batch(m.fetch(), m.tick())

	case queriesMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.queries = msg.page.Queries
		m.filtered = msg.page.RecordsFiltered
		m.total = msg.page.RecordsTotal
		m.syncRows()
		return m, nil

	case classifyMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.note = msg.verb + " " + msg.domain
		return m, m.expireNote()

	case noteExpireMsg:
		m.note = ""
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

// expireNote schedules the success toast to clear after noteTTL.
func (m *Model) expireNote() tea.Cmd {
	return tea.Tick(noteTTL, func(time.Time) tea.Msg { return noteExpireMsg{} })
}

// handleKey routes keystrokes through the search box, detail pane, and table.
func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		switch msg.String() {
		case "enter":
			m.filterDomain = strings.TrimSpace(m.search.Value())
			m.searching = false
			m.search.Blur()
			m.detail = nil
			m.epoch++
			m.loading = true
			return m, tea.Batch(m.fetch(), m.tick(), m.spinner.Tick)
		case "esc":
			m.searching = false
			m.search.Blur()
			m.search.SetValue(m.filterDomain)
			return m, nil
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	}

	if m.confirm.Active {
		return m.handleConfirmKey(msg)
	}

	// Quick-classify works from both the table and the detail pane.
	switch msg.String() {
	case "a":
		return m.promptClassify(pihole.DomainAllow)
	case "b":
		return m.promptClassify(pihole.DomainDeny)
	}

	if m.detail != nil {
		if msg.String() == "esc" {
			m.detail = nil
		}
		return m, nil
	}

	switch msg.String() {
	case "/":
		m.searching = true
		m.search.SetValue(m.filterDomain)
		m.search.CursorEnd()
		return m, m.search.Focus()
	case "enter":
		idx := m.table.Cursor()
		if idx >= 0 && idx < len(m.queries) {
			q := m.queries[idx]
			m.detail = &q
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// selectedDomain resolves the domain the quick actions target: the detail
// pane's domain when it's open, otherwise the highlighted table row.
func (m *Model) selectedDomain() (string, bool) {
	if m.detail != nil {
		d := strings.TrimSpace(m.detail.Domain)
		return d, d != ""
	}
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.queries) {
		d := strings.TrimSpace(m.queries[idx].Domain)
		return d, d != ""
	}
	return "", false
}

// promptClassify opens the confirm dialog for allowing/blocking the selected
// domain. A blank selection is a no-op.
func (m *Model) promptClassify(dt pihole.DomainType) (tea.Model, tea.Cmd) {
	domain, ok := m.selectedDomain()
	if !ok {
		return m, nil
	}
	m.pendingType = dt
	m.pendingDomain = domain

	title, danger := "Allow domain?", false
	if dt == pihole.DomainDeny {
		title, danger = "Block domain?", true
	}
	m.confirm = m.confirm.Show(title, domain, danger)
	return m, nil
}

// handleConfirmKey interprets y/n on the quick-classify confirmation.
func (m *Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		dt, domain := m.pendingType, m.pendingDomain
		m.confirm = m.confirm.Hide()
		m.pendingDomain = ""
		if domain == "" {
			return m, nil
		}
		return m, m.classify(dt, domain)
	case "n", "esc":
		m.confirm = m.confirm.Hide()
		m.pendingDomain = ""
	}
	return m, nil
}

// classify dispatches the allow/deny add as an exact-match domain. It runs on a
// self-contained context so it never disturbs the live poll's cancel handle.
func (m *Model) classify(dt pihole.DomainType, domain string) tea.Cmd {
	api := m.ctx.API
	verb := "allowed"
	if dt == pihole.DomainDeny {
		verb = "blocked"
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		_, err := api.AddDomain(ctx, dt, pihole.KindExact, domain, classifyComment, nil)
		return classifyMsg{domain: domain, verb: verb, err: err}
	}
}

// syncRows rebuilds the table rows from the current queries and theme,
// preserving the cursor position.
func (m *Model) syncRows() {
	widths := columnWidthsFrom(m.table.Columns())
	idx := m.table.Cursor()
	m.table.SetRows(styledRows(m.ctx.Theme, m.queries, widths))
	m.table.SetCursor(idx)
}

// columnWidthsFrom extracts the content widths from the table's columns.
func columnWidthsFrom(cols []table.Column) []int {
	w := make([]int, len(cols))
	for i, c := range cols {
		w[i] = c.Width
	}
	return w
}

// View renders the screen. The theme is read here so live re-themes and
// per-row status colours take effect immediately.
func (m *Model) View() tea.View {
	th := m.ctx.Theme

	// Re-apply row styling with the current theme every frame.
	if len(m.queries) > 0 {
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

	chipLabel := "all domains"
	if m.filterDomain != "" {
		chipLabel = "domain: " + m.filterDomain
	}
	chip := lipgloss.NewStyle().
		Foreground(th.Surface).
		Background(th.Accent).
		Padding(0, 1).
		Render(chipLabel)

	count := th.SubtleStyle().Render(fmt.Sprintf("%d shown · %d total", m.filtered, m.total))

	left := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", chip)
	gap := m.w - lipgloss.Width(left) - lipgloss.Width(count)
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + count

	searchLine := ""
	switch {
	case m.searching:
		searchLine = m.search.View()
	case m.err != nil:
		searchLine = m.errBanner(th)
	case m.note != "":
		searchLine = th.AllowStyle().Bold(true).Render(truncate("✓ "+m.note, maxInt(m.w-2, 8)))
	}
	return lipgloss.JoinVertical(lipgloss.Left, line, searchLine)
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

	if m.detail != nil {
		return m.renderDetail(th, bodyH)
	}

	if m.loading && len(m.queries) == 0 {
		line := m.spinner.View() + " " + th.SubtleStyle().Render("loading queries…")
		return lipgloss.Place(m.w, bodyH, lipgloss.Center, lipgloss.Center, line)
	}

	if len(m.queries) == 0 {
		empty := th.SubtleStyle().Render("no queries match this filter")
		return lipgloss.Place(m.w, bodyH, lipgloss.Center, lipgloss.Center, empty)
	}

	return m.table.View()
}

func (m *Model) renderDetail(th *theme.Theme, bodyH int) string {
	q := m.detail

	rows := [][2]string{
		{"Domain", q.Domain},
		{"CNAME", orDash(q.CNAME)},
		{"Client", clientDetail(q.Client)},
		{"Type", q.Type},
		{"Status", q.Status},
		{"Reply", fmt.Sprintf("%s (%.1fms)", orDash(q.Reply.Type), q.Reply.Time*1000)},
		{"Upstream", orDash(q.Upstream)},
		{"DNSSEC", orDash(q.DNSSEC)},
		{"Time", formatQueryTime(q.Time)},
	}

	keyStyle := th.SubtleStyle().Width(10)
	valStyle := styleForToken(th, statusToken(q.Status))

	var lines []string
	heading := th.AccentStyle().Bold(true).Render("Query detail")
	lines = append(lines, heading, "")
	for _, r := range rows {
		vs := th.TextStyle()
		if r[0] == "Status" || r[0] == "Domain" {
			vs = valStyle
		}
		lines = append(lines, keyStyle.Render(r[0])+"  "+vs.Render(r[1]))
	}
	lines = append(lines, "", th.SubtleStyle().Render("esc  close"))

	panel := th.PanelStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Border).
		Padding(1, 2).
		Width(maxInt(minInt(m.w-2, 60), 20)).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	return lipgloss.Place(m.w, bodyH, lipgloss.Center, lipgloss.Top, panel)
}

func (m *Model) renderFooter(th *theme.Theme) string {
	hint := "↑↓ navigate · / search · enter detail · a allow · b block · esc close"
	return th.SubtleStyle().Render(truncate(hint, maxInt(m.w, 8)))
}

func clientDetail(c pihole.QueryClient) string {
	if strings.TrimSpace(c.Name) != "" {
		return fmt.Sprintf("%s (%s)", c.Name, c.IP)
	}
	return c.IP
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
