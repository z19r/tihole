package dashboard

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/z19r/tihole/internal/pihole"
)

// healthClient is a real client against a fake Pi-hole that answers the
// per-instance health calls, reporting gravity blocked domains.
func healthClient(t *testing.T, gravity int) *pihole.Client {
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
			body, ok := bodies[r.URL.Path]
			if !ok {
				body = `{}`
			}
			_, _ = w.Write([]byte(body))
		}))
	t.Cleanup(srv.Close)
	return pihole.New(srv.URL, "pw", pihole.WithHTTPClient(srv.Client()))
}

// instancesModel builds a combined dashboard over members (named
// gravities, in config order) plus failed-at-build instances, runs one
// full fetch and returns the model.
func instancesModel(
	t *testing.T,
	active string,
	names []string,
	gravity []int,
	failed ...string,
) *Model {
	t.Helper()
	m := newTestModel()
	members := make([]pihole.Member, len(names))
	for i, n := range names {
		members[i] = pihole.Member{Name: n, Client: healthClient(t, gravity[i])}
		if n == active {
			m.ctx.API = members[i].Client
		}
	}
	f := map[string]error{}
	for _, n := range failed {
		f[n] = errors.New("no url")
	}
	m.ctx.Fleet = pihole.NewFleet(members, f)
	m.ctx.Combined = true
	m.ctx.InstanceName = active
	m.epoch = 1
	m.baseCtx = context.Background()

	batch, ok := m.fetchAll()().(tea.BatchMsg)
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
		[]string{"secondary", "primary"}, []int{100, 100})

	out := ansi.Strip(m.View().Content)

	for _, want := range []string{"Pi-holes", "CPU", "primary", "secondary"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q\n%s", want, out)
		}
	}
	pi, _ := lineWith(out, "● primary")
	si, _ := lineWith(out, "secondary")
	if pi < 0 || si < 0 || pi > si {
		t.Errorf("active not first and marked (%d, %d)\n%s", pi, si, out)
	}
}

func TestInstancesUnreachableRow(t *testing.T) {
	m := instancesModel(t, "primary",
		[]string{"primary"}, []int{100}, "basement")

	_, row := lineWith(ansi.Strip(m.renderInstances()), "basement")

	if !strings.Contains(row, "unreachable") || !strings.Contains(row, "—") {
		t.Errorf("unreachable row = %q", row)
	}
}

func TestInstancesOverflow(t *testing.T) {
	names := []string{"p1", "p2", "p3", "p4", "p5", "p6"}
	m := instancesModel(t, "p1", names, []int{1, 1, 1, 1, 1, 1})

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
		[]string{"a", "b", "c"}, []int{100, 100, 90})
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
