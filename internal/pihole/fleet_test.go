package pihole

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeMember serves /api/auth itself and hands every other request to h.
// hits counts the non-auth requests.
func fakeMember(
	t *testing.T,
	name string,
	h http.HandlerFunc,
) (Member, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	c, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			authOK(w, "SID-"+name)
			return
		}
		hits.Add(1)
		h(w, r)
	})
	return Member{Name: name, Client: c}, &hits
}

func summaryHandler(total int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(map[string]any{
			"queries": map[string]any{"total": total},
		})
		writeJSON(w, http.StatusOK, string(b))
	}
}

func failHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusInternalServerError,
		`{"error":{"key":"boom","message":"boom"},"took":0}`)
}

func TestFleetSummaryMerges(t *testing.T) {
	a, _ := fakeMember(t, "a", summaryHandler(10))
	b, _ := fakeMember(t, "b", summaryHandler(5))
	g := NewFleet([]Member{a, b}, nil)

	got, err := g.Summary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Queries.Total != 15 {
		t.Fatalf("Total = %d, want 15", got.Queries.Total)
	}
	if n := g.Names(); !slices.Equal(n, []string{"a", "b"}) {
		t.Fatalf("Names = %v", n)
	}
}

func TestFleetPartialFailure(t *testing.T) {
	a, _ := fakeMember(t, "a", summaryHandler(10))
	b, _ := fakeMember(t, "b", failHandler)
	g := NewFleet([]Member{a, b}, nil)

	got, err := g.Summary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Queries.Total != 10 {
		t.Fatalf("Total = %d, want 10", got.Queries.Total)
	}
	if u := g.Unreachable(); !slices.Equal(u, []string{"b"}) {
		t.Fatalf("Unreachable = %v, want [b]", u)
	}
}

func TestFleetAllFailErrors(t *testing.T) {
	a, _ := fakeMember(t, "a", failHandler)
	b, _ := fakeMember(t, "b", failHandler)
	g := NewFleet([]Member{a, b}, nil)

	if _, err := g.Summary(context.Background()); err == nil {
		t.Fatal("want error when every member fails")
	}
}

func TestFleetSlowMemberDeadline(t *testing.T) {
	a, _ := fakeMember(t, "a", summaryHandler(10))
	done := make(chan struct{})
	b, _ := fakeMember(t, "b", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(3 * time.Second):
		case <-done:
		}
		summaryHandler(5)(w, r)
	})
	// Registered after the servers, so it runs before srv.Close waits.
	t.Cleanup(func() { close(done) })
	g := NewFleet([]Member{a, b}, nil)

	start := time.Now()
	got, err := g.Summary(context.Background())
	if d := time.Since(start); d >= 2500*time.Millisecond {
		t.Fatalf("took %v, want < 2.5s", d)
	}
	if err != nil {
		t.Fatal(err)
	}
	if got.Queries.Total != 10 {
		t.Fatalf("Total = %d, want 10", got.Queries.Total)
	}
	if u := g.Unreachable(); !slices.Equal(u, []string{"b"}) {
		t.Fatalf("Unreachable = %v, want [b]", u)
	}
}

func TestFleetCooldownSkipsThenRetries(t *testing.T) {
	a, _ := fakeMember(t, "a", summaryHandler(10))
	var bFail atomic.Bool
	bFail.Store(true)
	b, bHits := fakeMember(t, "b",
		func(w http.ResponseWriter, r *http.Request) {
			if bFail.Load() {
				failHandler(w, r)
				return
			}
			summaryHandler(5)(w, r)
		})
	g := NewFleet([]Member{a, b}, nil)
	now := time.Unix(1000, 0)
	g.now = func() time.Time { return now }
	ctx := context.Background()

	if _, err := g.Summary(ctx); err != nil {
		t.Fatal(err)
	}
	hits := bHits.Load()
	now = now.Add(10 * time.Second)
	if _, err := g.Summary(ctx); err != nil {
		t.Fatal(err)
	}
	if bHits.Load() != hits {
		t.Fatal("b was called during cooldown")
	}
	if u := g.Unreachable(); !slices.Equal(u, []string{"b"}) {
		t.Fatalf("Unreachable = %v, want [b]", u)
	}

	bFail.Store(false)
	now = now.Add(31 * time.Second)
	got, err := g.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if bHits.Load() == hits {
		t.Fatal("b not retried after cooldown")
	}
	if got.Queries.Total != 15 {
		t.Fatalf("Total = %d, want 15", got.Queries.Total)
	}
	if u := g.Unreachable(); len(u) != 0 {
		t.Fatalf("Unreachable = %v, want none", u)
	}
}

func TestFleetTopDomainsOverfetch(t *testing.T) {
	var mu sync.Mutex
	var counts []string
	a, _ := fakeMember(t, "a", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		counts = append(counts, r.URL.Query().Get("count"))
		mu.Unlock()
		writeJSON(w, http.StatusOK,
			`{"domains":[{"domain":"x","count":3},{"domain":"y","count":1}]}`)
	})
	g := NewFleet([]Member{a}, nil)

	got, err := g.TopDomains(context.Background(), false, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(counts, []string{"3"}) {
		t.Fatalf("server saw count=%v, want [3]", counts)
	}
	if len(got.Domains) != 1 || got.Domains[0].Domain != "x" {
		t.Fatalf("Domains = %+v", got.Domains)
	}

	counts = nil
	if _, err := g.TopDomains(context.Background(), true, 10); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(counts, []string{"30"}) {
		t.Fatalf("server saw count=%v, want [30]", counts)
	}
}

func TestFleetQueriesTagsInstance(t *testing.T) {
	page := func(tm string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("length") != "2" {
				t.Errorf("length = %q", r.URL.Query().Get("length"))
			}
			writeJSON(w, http.StatusOK,
				`{"queries":[{"time":`+tm+`,"domain":"d"}]}`)
		}
	}
	a, _ := fakeMember(t, "a", page("1"))
	b, _ := fakeMember(t, "b", page("2"))
	g := NewFleet([]Member{a, b}, nil)

	n := 2
	got, err := g.Queries(context.Background(), QueryFilter{Length: &n})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Queries) != 2 || got.Queries[0].Instance != "b" ||
		got.Queries[1].Instance != "a" {
		t.Fatalf("Queries = %+v", got.Queries)
	}
}

func TestFleetSetBlockingPartialError(t *testing.T) {
	var mu sync.Mutex
	var methods []string
	record := func(fail bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Blocking bool     `json:"blocking"`
				Timer    *float64 `json:"timer"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			mu.Lock()
			methods = append(methods, r.Method)
			if body.Blocking || body.Timer == nil || *body.Timer != 60 {
				t.Errorf("body = %+v", body)
			}
			mu.Unlock()
			if fail {
				failHandler(w, r)
				return
			}
			writeJSON(w, http.StatusOK, `{"blocking":"disabled"}`)
		}
	}
	a, _ := fakeMember(t, "a", record(false))
	b, _ := fakeMember(t, "b", record(true))
	g := NewFleet([]Member{a, b},
		map[string]error{"c": errors.New("no url")})

	timer := 60.0
	err := g.SetBlocking(context.Background(), false, &timer)
	if err == nil ||
		!strings.Contains(err.Error(), "blocking not changed on b") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "blocking not changed on a") {
		t.Fatalf("a should not be named: %v", err)
	}
	if !strings.Contains(err.Error(), "blocking not changed on c: no url") {
		t.Fatalf("failed-at-build c not named: %v", err)
	}
	if !slices.Equal(methods, []string{"POST", "POST"}) {
		t.Fatalf("requests = %v", methods)
	}
}

func TestFleetFailedMemberListed(t *testing.T) {
	a, _ := fakeMember(t, "a", summaryHandler(1))
	g := NewFleet([]Member{a}, map[string]error{"c": errors.New("bad")})

	if u := g.Unreachable(); !slices.Equal(u, []string{"c"}) {
		t.Fatalf("Unreachable = %v, want [c]", u)
	}
	if g.Len() != 2 {
		t.Fatalf("Len = %d, want 2", g.Len())
	}
	if _, err := g.Summary(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestFleetSetBlockingAllFailedAtBuild(t *testing.T) {
	g := NewFleet(nil, map[string]error{"c": errors.New("no url")})
	err := g.SetBlocking(context.Background(), true, nil)
	if err == nil ||
		!strings.Contains(err.Error(), "blocking not changed on c") {
		t.Fatalf("err = %v", err)
	}
}

func TestFleetCallerCancelSkipsCooldown(t *testing.T) {
	done := make(chan struct{})
	slow := func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(3 * time.Second):
		case <-done:
		}
		summaryHandler(1)(w, r)
	}
	a, _ := fakeMember(t, "a", slow)
	b, _ := fakeMember(t, "b", slow)
	t.Cleanup(func() { close(done) })
	g := NewFleet([]Member{a, b}, nil)

	ctx, cancel := context.WithTimeout(
		context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := g.Summary(ctx); err == nil {
		t.Fatal("want error after caller cancel")
	}
	timer := 5.0
	if err := g.SetBlocking(ctx, false, &timer); err == nil {
		t.Fatal("want SetBlocking error after caller cancel")
	}
	if u := g.Unreachable(); len(u) != 0 {
		t.Fatalf("Unreachable = %v, want none", u)
	}
}
