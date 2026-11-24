package querylog

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/z19r/tihole/internal/pihole"
)

const authBody = `{"session":{"valid":true,"sid":"s","csrf":"c",` +
	`"validity":1800}}`

// fakeHole serves auth plus the given queries body and counts POSTs.
func fakeHole(
	t *testing.T,
	queries string,
	posts *atomic.Int32,
) *pihole.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.URL.Path == "/api/auth":
				_, _ = w.Write([]byte(authBody))
			case r.Method == http.MethodPost:
				posts.Add(1)
				_, _ = w.Write([]byte(`{"domains":[]}`))
			default:
				_, _ = w.Write([]byte(queries))
			}
		}))
	t.Cleanup(srv.Close)
	return pihole.New(srv.URL, "pw", pihole.WithHTTPClient(srv.Client()))
}

func combinedModel(t *testing.T, posts [3]*atomic.Int32) *Model {
	t.Helper()
	row := func(d string) string {
		return `{"queries":[{"time":1700000000,"type":"A","domain":"` + d +
			`","status":"GRAVITY","client":{"ip":"1.1.1.1"},` +
			`"upstream":"up"}]}`
	}
	m := newModel()
	m.ctx.API = fakeHole(t, row("a.example"), posts[0])
	m.ctx.Fleet = pihole.NewFleet([]pihole.Member{
		{Name: "primary", Client: fakeHole(t, row("a.example"), posts[1])},
		{Name: "secondary", Client: fakeHole(t, row("b.example"), posts[2])},
	}, nil)
	m.ctx.Combined = true
	m.ctx.InstanceName = "primary"
	return m
}

func loadCombined(t *testing.T, m *Model) {
	t.Helper()
	m.focused = true
	msg := m.fetch()()
	_, _ = m.Update(msg)
	_ = m.View()
}

func TestCombinedColumnShowsInstance(t *testing.T) {
	var p [3]*atomic.Int32
	for i := range p {
		p[i] = new(atomic.Int32)
	}
	m := combinedModel(t, p)
	_ = m.Focus()
	loadCombined(t, m)

	cols := m.table.Columns()
	if got := cols[numCols-1].Title; got != "Pi-hole" {
		t.Errorf("header = %q, want Pi-hole", got)
	}
	var seen []string
	for _, r := range m.table.Rows() {
		seen = append(seen, r[numCols-1])
	}
	if !strings.Contains(strings.Join(seen, ","), "secondary") {
		t.Errorf("no secondary cell in %v", seen)
	}
}

func TestSingleColumnShowsUpstream(t *testing.T) {
	q := pihole.Query{Upstream: "1.1.1.1", Instance: "secondary"}
	row := queryToRow(q, computeColumnWidths(120), false)
	if row[numCols-1] != "1.1.1.1" {
		t.Errorf("cell = %q", row[numCols-1])
	}
	if tableColumns(nil, false)[numCols-1].Title != "Upstream" {
		t.Error("single header should be Upstream")
	}
}

func TestFooterShowsWriteTarget(t *testing.T) {
	var p [3]*atomic.Int32
	for i := range p {
		p[i] = new(atomic.Int32)
	}
	m := combinedModel(t, p)
	if out := m.View().Content; !strings.Contains(out, "→ primary") {
		t.Errorf("footer missing write target\n%s", out)
	}
	m.ctx.Combined = false
	if out := m.View().Content; strings.Contains(out, "→") {
		t.Error("single mode footer should be unchanged")
	}
}

func TestDetailShowsInstance(t *testing.T) {
	var p [3]*atomic.Int32
	for i := range p {
		p[i] = new(atomic.Int32)
	}
	m := combinedModel(t, p)
	q := pihole.Query{Domain: "x.example", Instance: "secondary"}
	m.detail = &q
	out := m.View().Content
	for _, want := range []string{"Pi-hole", "Upstream", "secondary"} {
		if !strings.Contains(out, want) {
			t.Errorf("detail missing %q\n%s", want, out)
		}
	}
}

func TestBlockActionTargetsActive(t *testing.T) {
	var p [3]*atomic.Int32
	for i := range p {
		p[i] = new(atomic.Int32)
	}
	m := combinedModel(t, p)
	_ = m.Focus()
	loadCombined(t, m)
	m.table.SetCursor(len(m.queries) - 1) // a secondary row

	_, _ = m.Update(keyPress("b"))
	_, cmd := m.Update(keyPress("y"))
	if cmd == nil {
		t.Fatal("expected classify cmd")
	}
	_ = cmd()

	if p[0].Load() != 1 || p[1].Load() != 0 || p[2].Load() != 0 {
		t.Errorf("POSTs API/primary/secondary = %d/%d/%d, want 1/0/0",
			p[0].Load(), p[1].Load(), p[2].Load())
	}
}
