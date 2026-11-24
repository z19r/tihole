package dashboard

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/pihole"
)

const (
	// maxInstanceRows caps the per-instance rows; the rest are counted.
	maxInstanceRows = 4
	// minNameW is the name column width below which columns are dropped.
	minNameW = 12
	colGap   = "  "
)

// instanceHealth is one fleet member's row in the combined health block.
type instanceHealth struct {
	name     string
	active   bool
	system   pihole.SystemInfo
	sensors  pihole.Sensors
	messages int // -1 when unknown
	gravity  int // -1 when unknown
	err      error
}

// instancesMsg carries every member's health, active instance first.
type instancesMsg struct {
	epoch int
	rows  []instanceHealth
}

// instanceCol is one metric column: its header, width and cell text.
type instanceCol struct {
	title string
	w     int
	cell  func(instanceHealth) (string, lipgloss.Style)
}

func fetchInstances(
	base context.Context,
	f *pihole.Fleet,
	active string,
	epoch int,
) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(base, fetchTimeout)
		defer cancel()
		res := pihole.EachMember(ctx, f, probeInstance)
		return instancesMsg{
			epoch: epoch,
			rows:  instanceRows(res, f.Names(), active),
		}
	}
}

// probeInstance reads one member's health. Only a System failure fails the
// member; the other calls degrade to unknown cells.
func probeInstance(
	ctx context.Context,
	c *pihole.Client,
) (instanceHealth, error) {
	sys, err := c.System(ctx)
	if err != nil {
		return instanceHealth{}, err
	}
	h := instanceHealth{system: sys, messages: -1, gravity: -1}
	h.sensors, _ = c.SensorsInfo(ctx) // the zero value renders n/a
	if msgs, err := c.Messages(ctx); err == nil {
		h.messages = len(msgs)
	}
	if s, err := c.Summary(ctx); err == nil {
		h.gravity = s.Gravity.DomainsBeingBlocked
	}
	return h, nil
}

// instanceRows orders results by config (names), active first. Instances
// that failed to build aren't in names; they follow, sorted.
func instanceRows(
	res map[string]pihole.MemberResult[instanceHealth],
	names []string,
	active string,
) []instanceHealth {
	for _, n := range slices.Sorted(maps.Keys(res)) {
		if !slices.Contains(names, n) {
			names = append(names, n)
		}
	}
	rows := make([]instanceHealth, 0, len(names))
	for _, n := range names {
		h := res[n].Value
		h.name, h.err, h.active = n, res[n].Err, n == active
		if h.active {
			rows = slices.Insert(rows, 0, h)
		} else {
			rows = append(rows, h)
		}
	}
	return rows
}

// commonGravity returns the gravity most reachable rows report, or -1.
// Ties go to the earliest row, which is the active instance when reachable.
func commonGravity(rows []instanceHealth) int {
	counts := map[int]int{}
	for _, h := range rows {
		if h.err == nil && h.gravity >= 0 {
			counts[h.gravity]++
		}
	}
	if len(counts) == 0 {
		return -1
	}
	best := slices.Max(slices.Collect(maps.Values(counts)))
	for _, h := range rows {
		if h.err == nil && h.gravity >= 0 && counts[h.gravity] == best {
			return h.gravity
		}
	}
	return -1
}

// padCell truncates s to w, styles it and pads it to w columns.
func padCell(st lipgloss.Style, s string, w int) string {
	s = truncate(s, w)
	return st.Render(s) + strings.Repeat(" ", max(w-lipgloss.Width(s), 0))
}

// instanceCols returns the metric columns, dropping Gravity and then
// Load · Up when the name column would fall below minNameW, plus the name
// column's width.
func (m *Model) instanceCols(inner int) ([]instanceCol, int) {
	th := m.ctx.Theme
	plain, subtle := th.TextStyle(), th.SubtleStyle()
	ref := commonGravity(m.instances)
	pct := func(f func(pihole.SystemInfo) float64) func(
		instanceHealth,
	) (string, lipgloss.Style) {
		return func(h instanceHealth) (string, lipgloss.Style) {
			return fmt.Sprintf("%.0f%%", f(h.system)), plain
		}
	}
	cols := []instanceCol{
		{"CPU", 5, pct(func(s pihole.SystemInfo) float64 {
			return s.CPUPercent
		})},
		{"Mem", 5, pct(func(s pihole.SystemInfo) float64 {
			return s.MemUsedPercent
		})},
		{"Temp", 6, func(h instanceHealth) (string, lipgloss.Style) {
			if !h.sensors.HasTemp {
				return "n/a", subtle
			}
			unit := cmp.Or(h.sensors.Unit, "C")
			return fmt.Sprintf("%.0f°%s", h.sensors.CPUTemp, unit), plain
		}},
		{"Load · Up", 14, func(h instanceHealth) (string, lipgloss.Style) {
			return fmt.Sprintf("%.0f%% · %s", h.system.Load1Percent,
				humanUptime(h.system.Uptime)), plain
		}},
		{"Gravity", 11, func(h instanceHealth) (string, lipgloss.Style) {
			switch {
			case h.gravity < 0:
				return "—", subtle
			case h.gravity != ref:
				return formatCount(h.gravity), th.WarnStyle()
			}
			return formatCount(h.gravity), plain
		}},
		{"Health", 11, func(h instanceHealth) (string, lipgloss.Style) {
			switch {
			case h.messages < 0:
				return "—", subtle
			case h.messages == 0:
				return "✓ clear", th.AllowStyle()
			}
			return pluralIssues(h.messages), th.WarnStyle()
		}},
	}
	nameW := func() int {
		w := inner
		for _, c := range cols {
			w -= c.w + len(colGap)
		}
		return w
	}
	for _, drop := range []string{"Gravity", "Load · Up"} {
		if nameW() >= minNameW {
			break
		}
		cols = slices.DeleteFunc(cols, func(c instanceCol) bool {
			return c.title == drop
		})
	}
	return cols, max(nameW(), 1)
}

// renderInstances draws one plain-text row per fleet member, in place of
// the health strip in combined mode: CPU, memory, temperature, load and
// uptime, gravity and diagnostics can't be merged across machines.
func (m *Model) renderInstances() string {
	th := m.ctx.Theme
	subtle := th.SubtleStyle()
	inner := m.w - cardChrome
	cols, nameW := m.instanceCols(inner)

	head := padCell(subtle, "Pi-holes", nameW)
	for _, c := range cols {
		head += colGap + padCell(subtle, c.title, c.w)
	}
	lines := []string{head}
	for _, h := range m.instances[:min(len(m.instances), maxInstanceRows)] {
		lines = append(lines, instanceRow(m, h, cols, nameW))
	}
	if extra := len(m.instances) - maxInstanceRows; extra > 0 {
		lines = append(lines,
			subtle.Render(fmt.Sprintf("  and %d more…", extra)))
	}
	return card(th, inner, strings.Join(lines, "\n"))
}

// instanceRow renders one member. An unreachable member is dimmed, with
// "—" for CPU, memory and temperature, then "unreachable".
func instanceRow(
	m *Model,
	h instanceHealth,
	cols []instanceCol,
	nameW int,
) string {
	th := m.ctx.Theme
	subtle := th.SubtleStyle()
	marker, mst, nst := "●", th.TextStyle(), th.TextStyle()
	if h.active {
		mst, nst = th.AccentStyle(), th.TextStyle().Bold(true)
	}
	if h.err != nil {
		marker, mst, nst = "○", subtle, subtle
	}
	row := mst.Render(marker) + " " + padCell(nst, h.name, nameW-2)
	for i, c := range cols {
		if h.err != nil {
			if i == 3 { // past CPU, Mem, Temp
				return row + colGap + subtle.Render("unreachable")
			}
			row += colGap + padCell(subtle, "—", c.w)
			continue
		}
		text, st := c.cell(h)
		row += colGap + padCell(st, text, c.w)
	}
	return row
}
