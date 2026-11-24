package dashboard

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/z19r/tihole/internal/config"
	"github.com/z19r/tihole/internal/pihole"
)

// healthClient is a real client against a fake Pi-hole that answers the
// per-instance health calls, reporting gravity blocked domains.
func healthClient(t *testing.T, gravity int) *pihole.Client {
	t.Helper()
	return slowHealthClient(t, gravity, 0)
}

// slowHealthClient is healthClient whose sensors, messages and summary
// answers each take delay.
func slowHealthClient(
	t *testing.T,
	gravity int,
	delay time.Duration,
) *pihole.Client {
	t.Helper()
	bodies := map[string]string{
		"/api/auth": `{"session":{"valid":true,"sid":"s","csrf":"c",` +
			`"validity":1800}}`,
		"/api/info/system": `{"system":{"uptime":356400,"procs":90,` +
			`"memory":{"ram":{"total":100,"used":41,"%used":41}},` +
			`"cpu":{"nprocs":4,"%cpu":12,"load":{"percent":[3]}}}}`,
		"/api/info/sensors": `{"sensors":{"cpu_temp":48,` +
			`"hot_limit":85,"unit":"C"}}`,
		"/api/info/messages": `{"messages":[]}`,
		"/api/stats/summary": fmt.Sprintf(
			`{"gravity":{"domains_being_blocked":%d}}`, gravity),
	}
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path != "/api/auth" && r.URL.Path != "/api/info/system" {
				time.Sleep(delay)
			}
			body, ok := bodies[r.URL.Path]
			if !ok {
				body = `{}`
			}
			_, _ = w.Write([]byte(body))
		}))
	t.Cleanup(srv.Close)
	return pihole.New(srv.URL, "pw", pihole.WithHTTPClient(srv.Client()))
}

// instancesModel builds a combined dashboard over the config instances in
// order; those with a gravity are served over httptest, the rest failed to
// build. It runs one full fetch and returns the model.
func instancesModel(
	t *testing.T,
	active string,
	order []string,
	gravity map[string]int,
) *Model {
	t.Helper()
	m := newTestModel()
	m.ctx.API = healthClient(t, 0)
	m.ctx.Config = &config.Config{}
	var members []pihole.Member
	failed := map[string]error{}
	for _, n := range order {
		m.ctx.Config.Instances = append(m.ctx.Config.Instances,
			config.Instance{Name: n})
		g, ok := gravity[n]
		if !ok {
			failed[n] = errors.New("no url")
			continue
		}
		members = append(members,
			pihole.Member{Name: n, Client: healthClient(t, g)})
		if n == active {
			m.ctx.API = members[len(members)-1].Client
		}
	}
	m.ctx.Fleet = pihole.NewFleet(members, failed)
	m.ctx.Combined = true
	m.ctx.InstanceName = active
	m.epoch = 1
	m.baseCtx = context.Background()

	batch, ok := m.fetchAll(true)().(tea.BatchMsg)
	if !ok {
		t.Fatal("fetchAll should return a batch")
	}
	for _, cmd := range batch {
		_, _ = m.Update(cmd())
	}
	return m
}

// lineWith returns the first line of out containing s, or "".
func lineWith(out, s string) (int, string) {
	for i, l := range strings.Split(out, "\n") {
		if strings.Contains(l, s) {
			return i, l
		}
	}
	return -1, ""
}

func TestInstancesRowsLabelled(t *testing.T) {
	m := instancesModel(t, "primary",
		[]string{"secondary", "primary"},
		map[string]int{"secondary": 100, "primary": 100})

	out := ansi.Strip(m.View().Content)

	for _, want := range []string{"Pi-holes", "CPU", "primary", "secondary"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q\n%s", want, out)
		}
	}
	pi, _ := lineWith(out, "▸ primary")
	si, _ := lineWith(out, "● secondary")
	if pi < 0 || si < 0 || pi > si {
		t.Errorf("active not first and marked (%d, %d)\n%s", pi, si, out)
	}
	accent := m.ctx.Theme.AccentStyle().Render("▸")
	if !strings.Contains(m.renderInstances(), accent) {
		t.Error("active marker not in accent style")
	}
}

func TestInstancesUnreachableRow(t *testing.T) {
	m := instancesModel(t, "primary",
		[]string{"primary", "basement"}, map[string]int{"primary": 100})

	raw := m.renderInstances()
	_, row := lineWith(ansi.Strip(raw), "basement")

	if !strings.Contains(row, "○ basement") ||
		!strings.Contains(row, "unreachable") || !strings.Contains(row, "—") {
		t.Errorf("unreachable row = %q", row)
	}
	subtle := m.ctx.Theme.SubtleStyle()
	for _, want := range []string{"○", "basement", "unreachable"} {
		if !strings.Contains(raw, subtle.Render(want)) {
			t.Errorf("%q not dimmed", want)
		}
	}
}

func TestInstancesUnreachableActiveKeepsMark(t *testing.T) {
	m := instancesModel(t, "primary",
		[]string{"primary", "other"}, map[string]int{"other": 100})
	raw := m.renderInstances()

	i, row := lineWith(ansi.Strip(raw), "▸ primary")

	if i != 2 || !strings.Contains(row, "unreachable") {
		t.Errorf("active row = %d %q\n%s", i, row, ansi.Strip(raw))
	}
	if !strings.Contains(raw, m.ctx.Theme.SubtleStyle().Render("▸")) {
		t.Error("unreachable active marker not dimmed")
	}
}

func TestInstancesConfigOrder(t *testing.T) {
	m := instancesModel(t, "c",
		[]string{"a", "b", "c", "d"}, map[string]int{"a": 1, "c": 1, "d": 1})
	out := ansi.Strip(m.renderInstances())

	var got []int
	for _, n := range []string{"c", "a", "b", "d"} {
		i, _ := lineWith(out, " "+n+" ")
		got = append(got, i)
	}
	if !slices.Equal(got, []int{2, 3, 4, 5}) {
		t.Errorf(
			"row lines for c,a,b,d = %v, want 2..5 (after border, header)\n%s",
			got,
			out,
		)
	}
}

func TestInstancesNarrowKeepsNameInColumn(t *testing.T) {
	m := instancesModel(t, "primary",
		[]string{"primary", "other"},
		map[string]int{"primary": 1, "other": 1})
	m.SetSize(30, 30)

	out := ansi.Strip(m.renderInstances())

	_, row := lineWith(out, "▸ ")
	if !strings.Contains(row, "▸ p") && !strings.Contains(row, "▸ …") {
		t.Errorf("active name dropped at width 30: %q", row)
	}
	// border, header, two rows, border: a wrapped row adds lines.
	if n := strings.Count(out, "\n") + 1; n != 5 {
		t.Errorf("%d lines, want 5 (rows wrapped?)\n%s", n, out)
	}
	for _, l := range strings.Split(out, "\n") {
		if w := lipgloss.Width(l); w > 30 {
			t.Errorf("line width %d > 30: %q", w, l)
		}
	}
}

func TestInstancesOverflow(t *testing.T) {
	names := []string{"p1", "p2", "p3", "p4", "p5", "p6"}
	g := map[string]int{}
	for _, n := range names {
		g[n] = 1
	}
	m := instancesModel(t, "p1", names, g)

	out := ansi.Strip(m.renderInstances())

	rows := 0
	for _, n := range names {
		if i, _ := lineWith(out, " "+n+" "); i >= 0 {
			rows++
		}
	}
	if rows != 4 || !strings.Contains(out, "and 2 more…") {
		t.Errorf("rows = %d, want 4 plus overflow\n%s", rows, out)
	}
}

func TestInstancesGravityOddOneOut(t *testing.T) {
	m := instancesModel(t, "a",
		[]string{"a", "b", "c"}, map[string]int{"a": 100, "b": 100, "c": 90})
	warn := m.ctx.Theme.WarnStyle()

	out := m.renderInstances()

	if !strings.Contains(out, warn.Render("90")) {
		t.Errorf("odd gravity not in warn style\n%q", out)
	}
	if strings.Contains(out, warn.Render("100")) {
		t.Errorf("common gravity in warn style\n%q", out)
	}
}

func TestSingleModeKeepsHealthStrip(t *testing.T) {
	m := combinedModel()
	m.ctx.Combined = false
	m.system = sampleSystem()

	out := m.View().Content

	if !strings.Contains(out, "Memory") || strings.Contains(out, "Pi-holes") {
		t.Errorf("single mode lost the health strip\n%s", out)
	}
}

func TestProbeInstanceRunsDetailsConcurrently(t *testing.T) {
	// Two 800ms calls only fit the 2s member deadline side by side (after
	// System); gravity comes from the round's summary, not the probe.
	c := slowHealthClient(t, 7, 800*time.Millisecond)
	g := pihole.NewFleet([]pihole.Member{{Name: "a", Client: c}}, nil)

	r := pihole.EachMember(context.Background(), g, probeInstance)["a"]

	if r.Err != nil || r.Value.gravity != -1 || r.Value.messages != 0 ||
		!r.Value.sensors.HasTemp {
		t.Errorf("probe = %+v", r)
	}
}

func TestInstancesNarrowKeepsHealth(t *testing.T) {
	for _, w := range []int{40, 24} {
		m := instancesModel(t, "primary",
			[]string{"primary", "other", "down"},
			map[string]int{"primary": 1, "other": 1})
		m.SetSize(w, 30)

		out := ansi.Strip(m.renderInstances())

		for _, want := range []string{"Health", "✓ clear", "unreachable"} {
			if !strings.Contains(out, want) {
				t.Errorf("width %d missing %q\n%s", w, want, out)
			}
		}
		for _, l := range strings.Split(out, "\n") {
			if lw := lipgloss.Width(l); lw > w {
				t.Errorf("width %d: line %d wide: %q", w, lw, l)
			}
		}
	}
}
