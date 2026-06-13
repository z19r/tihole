// Package dashboard implements the PiHole v6 overview Screen: summary stat
// tiles, a queries-over-time sparkline, query-type/upstream breakdowns and top
// domain/client/blocked lists. It polls the FTL API roughly every five seconds
// while focused and renders every panel defensively so a single failing request
// never blanks the screen.
package dashboard

import (
	"context"
	"sort"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

const (
	pollInterval = 5 * time.Second
	fetchTimeout = 8 * time.Second
	topCount     = 8
	sparkHeight  = 1
)

// tickMsg drives the poll loop. epoch is captured when the tick is scheduled so
// stale ticks left over from a previous Focus are ignored.
type tickMsg struct{ epoch int }

// Result messages. Each carries the epoch of the fetch that produced it so the
// Update loop can discard results that belong to a superseded focus session.
type summaryMsg struct {
	epoch int
	data  pihole.Summary
	err   error
}

type historyMsg struct {
	epoch int
	data  pihole.History
	err   error
}

type breakdownMsg struct {
	epoch     int
	types     pihole.QueryTypes
	upstreams pihole.Upstreams
	typesErr  error
	upErr     error
}

type topMsg struct {
	epoch      int
	domains    pihole.TopDomains
	blocked    pihole.TopDomains
	clients    pihole.TopClients
	domainsErr error
	blockedErr error
	clientsErr error
}

// Model is the dashboard Screen.
type Model struct {
	ctx *core.AppContext

	w, h    int
	focused bool
	epoch   int
	loaded  bool

	spinner spinner.Model
	cancel  context.CancelFunc
	baseCtx context.Context

	summary   pihole.Summary
	history   pihole.History
	types     pihole.QueryTypes
	upstreams pihole.Upstreams
	domains   pihole.TopDomains
	blocked   pihole.TopDomains
	clients   pihole.TopClients

	errSummary   string
	errHistory   string
	errTypes     string
	errUpstreams string
	errDomains   string
	errBlocked   string
	errClients   string
}

// Compile-time assertion that the dashboard satisfies the Screen contract.
var _ core.Screen = (*Model)(nil)

// New constructs a dashboard Screen bound to the shared app context.
func New(ctx *core.AppContext) *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return &Model{ctx: ctx, spinner: sp}
}

// Init is a no-op: fetching starts on Focus.
func (m *Model) Init() tea.Cmd { return nil }

// Title implements core.Screen.
func (m *Model) Title() string { return "Dashboard" }

// SetSize records the inner content area.
func (m *Model) SetSize(w, h int) { m.w, m.h = w, h }

// Help returns the screen-specific key bindings.
func (m *Model) Help() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

// Focus becomes active: bump the epoch, start a fresh cancelable context, kick
// an immediate fetch, the poll tick and the spinner.
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	m.epoch++
	if m.cancel != nil {
		m.cancel()
	}
	m.baseCtx, m.cancel = context.WithCancel(context.Background())
	return tea.Batch(m.fetchAll(), m.tick(), m.spinner.Tick)
}

// Blur leaves the screen: stop scheduling and cancel any in-flight requests.
func (m *Model) Blur() {
	m.focused = false
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

// tick schedules the next poll, tagging it with the current epoch.
func (m *Model) tick() tea.Cmd {
	e := m.epoch
	return tea.Tick(pollInterval, func(time.Time) tea.Msg { return tickMsg{e} })
}

// shouldHandleTick reports whether a tick should be acted upon: only while
// focused and only when its epoch matches the current one. Extracted as a pure
// function so the epoch-guard logic is unit-testable without a live model.
func shouldHandleTick(focused bool, tickEpoch, current int) bool {
	return focused && tickEpoch == current
}

// Update handles ticks and fetch results. All I/O happens inside the commands
// returned here; Update itself never touches the network.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if !shouldHandleTick(m.focused, msg.epoch, m.epoch) {
			return m, nil
		}
		return m, tea.Batch(m.fetchAll(), m.tick())

	case tea.KeyPressMsg:
		if m.focused && msg.String() == "r" {
			return m, m.fetchAll()
		}

	case summaryMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loaded = true
		if msg.err != nil {
			m.errSummary = msg.err.Error()
		} else {
			m.errSummary, m.summary = "", msg.data
		}
		return m, nil

	case historyMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loaded = true
		if msg.err != nil {
			m.errHistory = msg.err.Error()
		} else {
			m.errHistory, m.history = "", msg.data
		}
		return m, nil

	case breakdownMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loaded = true
		if msg.typesErr != nil {
			m.errTypes = msg.typesErr.Error()
		} else {
			m.errTypes, m.types = "", msg.types
		}
		if msg.upErr != nil {
			m.errUpstreams = msg.upErr.Error()
		} else {
			m.errUpstreams, m.upstreams = "", msg.upstreams
		}
		return m, nil

	case topMsg:
		if msg.epoch != m.epoch {
			return m, nil
		}
		m.loaded = true
		if msg.domainsErr != nil {
			m.errDomains = msg.domainsErr.Error()
		} else {
			m.errDomains, m.domains = "", msg.domains
		}
		if msg.blockedErr != nil {
			m.errBlocked = msg.blockedErr.Error()
		} else {
			m.errBlocked, m.blocked = "", msg.blocked
		}
		if msg.clientsErr != nil {
			m.errClients = msg.clientsErr.Error()
		} else {
			m.errClients, m.clients = "", msg.clients
		}
		return m, nil

	case spinner.TickMsg:
		if m.loaded || !m.focused {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// fetchAll returns a batch of independent fetch commands, each capturing the
// current epoch and cancelable context.
func (m *Model) fetchAll() tea.Cmd {
	if m.ctx == nil || m.ctx.API == nil || m.baseCtx == nil {
		return nil
	}
	api := m.ctx.API
	epoch := m.epoch
	base := m.baseCtx

	return tea.Batch(
		fetchSummary(base, api, epoch),
		fetchHistory(base, api, epoch),
		fetchBreakdown(base, api, epoch),
		fetchTop(base, api, epoch),
	)
}

func fetchSummary(base context.Context, api *pihole.Client, epoch int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(base, fetchTimeout)
		defer cancel()
		data, err := api.Summary(ctx)
		return summaryMsg{epoch: epoch, data: data, err: err}
	}
}

func fetchHistory(base context.Context, api *pihole.Client, epoch int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(base, fetchTimeout)
		defer cancel()
		data, err := api.History(ctx)
		return historyMsg{epoch: epoch, data: data, err: err}
	}
}

func fetchBreakdown(base context.Context, api *pihole.Client, epoch int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(base, fetchTimeout)
		defer cancel()
		types, tErr := api.QueryTypes(ctx)
		up, uErr := api.Upstreams(ctx)
		return breakdownMsg{epoch: epoch, types: types, upstreams: up, typesErr: tErr, upErr: uErr}
	}
}

func fetchTop(base context.Context, api *pihole.Client, epoch int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(base, fetchTimeout)
		defer cancel()
		domains, dErr := api.TopDomains(ctx, false, topCount)
		blocked, bErr := api.TopDomains(ctx, true, topCount)
		clients, cErr := api.TopClients(ctx, false, topCount)
		return topMsg{
			epoch:      epoch,
			domains:    domains,
			blocked:    blocked,
			clients:    clients,
			domainsErr: dErr,
			blockedErr: bErr,
			clientsErr: cErr,
		}
	}
}

// sortedTypes returns query-type entries sorted by descending count for a stable
// breakdown render (Go map order is otherwise random). Extracted for testing.
func sortedTypes(types map[string]int) []labeledCount {
	out := make([]labeledCount, 0, len(types))
	for k, v := range types {
		out = append(out, labeledCount{label: k, count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].count != out[j].count {
			return out[i].count > out[j].count
		}
		return out[i].label < out[j].label
	})
	return out
}

type labeledCount struct {
	label string
	count int
}

// View renders the full dashboard, defensively clamping to the content area.
func (m *Model) View() tea.View {
	th := m.ctx.Theme
	surface := th.SurfaceStyle().Width(m.w).Height(m.h)

	if m.w < 24 || m.h < 8 {
		return tea.NewView(surface.Render(""))
	}

	if !m.loaded {
		msg := m.spinner.View() + " " + th.SubtleStyle().Render("Loading dashboard…")
		body := lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, msg)
		return tea.NewView(surface.Render(body))
	}

	sections := []string{
		m.renderTiles(),
		m.renderSparkline(),
		m.renderLower(),
	}
	body := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return tea.NewView(surface.Render(body))
}
