package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/pihole"
)

// countingHole is a fake Pi-hole that counts requests per path.
type countingHole struct {
	mu   sync.Mutex
	hits map[string]int
}

func (h *countingHole) count(path string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hits[path]
}

func (h *countingHole) reset() {
	h.mu.Lock()
	clear(h.hits)
	h.mu.Unlock()
}

func newCountingHole(
	t *testing.T,
	gravity string,
) (*pihole.Client, *countingHole) {
	t.Helper()
	h := &countingHole{hits: map[string]int{}}
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			h.mu.Lock()
			h.hits[r.URL.Path]++
			h.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			body := `{}`
			if r.URL.Path == "/api/stats/summary" {
				body = `{"gravity":{"domains_being_blocked":` + gravity + `}}`
			}
			_, _ = w.Write([]byte(body))
		}))
	t.Cleanup(srv.Close)
	return pihole.New(srv.URL, "pw", pihole.WithHTTPClient(srv.Client())), h
}

// runRound executes one fetch round synchronously, feeding every reply back.
func runRound(t *testing.T, m *Model, force bool) {
	t.Helper()
	m.inFlight = 0
	cmd := m.fetchAll(force)
	if cmd == nil {
		t.Fatal("expected a fetch round")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatal("fetchAll should return a batch")
	}
	for _, c := range batch {
		m.Update(c())
	}
}

func TestCombinedRoundFetchesEachMemberOnce(t *testing.T) {
	// Arrange
	a, ha := newCountingHole(t, "100")
	b, hb := newCountingHole(t, "200")
	m := newTestModel()
	m.ctx.API = a
	m.ctx.Fleet = pihole.NewFleet([]pihole.Member{
		{Name: "a", Client: a}, {Name: "b", Client: b},
	}, nil)
	m.ctx.Combined = true
	m.ctx.InstanceName = "a"
	m.epoch = 1
	m.baseCtx = context.Background()

	// Act / Assert: a full round.
	runRound(t, m, true)
	for name, h := range map[string]*countingHole{"a": ha, "b": hb} {
		if n := h.count("/api/stats/summary"); n != 1 {
			t.Errorf("full round: %s summary = %d, want 1", name, n)
		}
		if n := h.count("/api/info/system"); n != 1 {
			t.Errorf("full round: %s system = %d, want 1", name, n)
		}
	}
	if len(m.instances) != 2 || m.instances[0].gravity != 100 ||
		m.instances[1].gravity != 200 {
		t.Fatalf("instance gravity not from the summary: %+v", m.instances)
	}
	if m.instances[0].err != nil {
		t.Fatalf(
			"active row err = %v, want its fetchSystem result",
			m.instances[0].err,
		)
	}

	// Act / Assert: a fast round.
	ha.reset()
	hb.reset()
	m.lastSlow = time.Now()
	runRound(t, m, false)
	if ha.count("/api/stats/summary") != 1 ||
		hb.count("/api/stats/summary") != 1 {
		t.Errorf("fast round summaries = %d, %d; want 1, 1",
			ha.count("/api/stats/summary"), hb.count("/api/stats/summary"))
	}
	if ha.count("/api/info/system") != 1 || hb.count("/api/info/system") != 0 {
		t.Errorf("fast round systems = %d, %d; want 1, 0",
			ha.count("/api/info/system"), hb.count("/api/info/system"))
	}
}
