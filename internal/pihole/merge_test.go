package pihole

import (
	"encoding/json"
	"testing"
)

func mustJSON[T any](t *testing.T, s string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestMergeSummarySumsAndRecomputesPercent(t *testing.T) {
	a := mustJSON[Summary](t,
		`{"queries":{"total":100,"blocked":10,"frequency":1.5}}`)
	b := mustJSON[Summary](t,
		`{"queries":{"total":300,"blocked":90,"frequency":2}}`)
	got := mergeSummary([]Summary{a, b})
	if got.Queries.Total != 400 || got.Queries.Blocked != 100 ||
		got.Queries.PercentBlocked != 25.0 ||
		got.Queries.Frequency != 3.5 {
		t.Fatalf("got %+v", got.Queries)
	}
}

func TestMergeSummaryZeroTotal(t *testing.T) {
	got := mergeSummary([]Summary{{}, {}})
	if got.Queries.PercentBlocked != 0 {
		t.Fatalf("got %v", got.Queries.PercentBlocked)
	}
}

func TestMergeSummaryMaxFields(t *testing.T) {
	a := mustJSON[Summary](t, `{"queries":{"unique_domains":3},
		"clients":{"active":5,"total":8},
		"gravity":{"domains_being_blocked":1000,"last_update":50}}`)
	b := mustJSON[Summary](t, `{"queries":{"unique_domains":9},
		"clients":{"active":7,"total":6},
		"gravity":{"domains_being_blocked":1200,"last_update":20}}`)
	got := mergeSummary([]Summary{a, b})
	if got.Clients.Active != 7 || got.Clients.Total != 8 ||
		got.Queries.UniqueDomains != 9 ||
		got.Gravity.DomainsBeingBlocked != 1200 ||
		got.Gravity.LastUpdate != 20 || !got.GravityMismatch {
		t.Fatalf("got %+v", got)
	}
	if mergeSummary([]Summary{a, a}).GravityMismatch {
		t.Fatal("equal gravity flagged")
	}
}

func TestMergeHistorySumsByTimestamp(t *testing.T) {
	mk := func(n int) History {
		var h History
		for _, ts := range []int{0, 600, 1200} {
			h.History = append(h.History, HistoryPoint{ts, n, n, n, n})
		}
		return h
	}
	got := mergeHistory([]History{mk(1), mk(2)})
	if len(got.History) != 3 {
		t.Fatalf("got %+v", got.History)
	}
	for i, p := range got.History {
		if p.Timestamp != i*600 || p.Total != 3 || p.Cached != 3 ||
			p.Blocked != 3 || p.Forwarded != 3 {
			t.Fatalf("point %d: %+v", i, p)
		}
	}
}

func TestMergeHistoryKeepsUnmatchedBuckets(t *testing.T) {
	a := History{
		History: []HistoryPoint{
			{Timestamp: 600, Total: 1},
			{Timestamp: 0, Total: 1},
		},
	}
	b := History{
		History: []HistoryPoint{
			{Timestamp: 600, Total: 2},
			{Timestamp: 1200, Total: 2},
		},
	}
	got := mergeHistory([]History{a, b}).History
	if len(got) != 3 || got[0].Timestamp != 0 || got[1].Total != 3 ||
		got[2].Timestamp != 1200 {
		t.Fatalf("got %+v", got)
	}
}

func TestMergeQueryTypesSumsKeys(t *testing.T) {
	a := QueryTypes{Types: map[string]int{"A": 1, "AAAA": 2}}
	b := QueryTypes{Types: map[string]int{"A": 4, "TXT": 1}}
	got := mergeQueryTypes([]QueryTypes{a, b}).Types
	if got["A"] != 5 || got["AAAA"] != 2 || got["TXT"] != 1 {
		t.Fatalf("got %v", got)
	}
}

func TestMergeUpstreamsWeighted(t *testing.T) {
	mk := func(ip string, count int, resp float64) Upstreams {
		u := UpstreamServer{IP: ip, Port: 53, Count: count}
		u.Statistics.Response = resp
		u.Statistics.Variance = resp
		return Upstreams{Upstreams: []UpstreamServer{u},
			ForwardedQueries: count, TotalQueries: count * 2}
	}
	a, b := mk("1.1.1.1", 1, 4), mk("1.1.1.1", 3, 8)
	c := mk("9.9.9.9", 2, 1)
	got := mergeUpstreams([]Upstreams{a, b, c})
	if len(got.Upstreams) != 2 || got.ForwardedQueries != 6 ||
		got.TotalQueries != 12 {
		t.Fatalf("got %+v", got)
	}
	u := got.Upstreams[0]
	if u.IP != "1.1.1.1" || u.Count != 4 ||
		u.Statistics.Response != 7.0 || u.Statistics.Variance != 7.0 {
		t.Fatalf("got %+v", u)
	}
}

func TestMergeTopDomainsSumsSortsTrims(t *testing.T) {
	a := mustJSON[TopDomains](t, `{"domains":[{"domain":"b.com","count":5},
		{"domain":"a.com","count":2}],"total_queries":10,"blocked_queries":1}`)
	b := mustJSON[TopDomains](t, `{"domains":[{"domain":"c.com","count":5},
		{"domain":"a.com","count":3},{"domain":"z.com","count":1}],
		"total_queries":20,"blocked_queries":2}`)
	got := mergeTopDomains([]TopDomains{a, b}, 3)
	if len(got.Domains) != 3 || got.TotalQueries != 30 ||
		got.BlockedQueries != 3 {
		t.Fatalf("got %+v", got)
	}
	want := []string{"a.com", "b.com", "c.com"} // 5,5,5: ties by domain
	for i, d := range got.Domains {
		if d.Domain != want[i] || d.Count != 5 {
			t.Fatalf("row %d: %+v", i, d)
		}
	}
}

func TestMergeTopClientsKeepsName(t *testing.T) {
	a := mustJSON[TopClients](t,
		`{"clients":[{"ip":"10.0.0.5","name":"","count":1}]}`)
	b := mustJSON[TopClients](t,
		`{"clients":[{"ip":"10.0.0.5","name":"tv","count":2}]}`)
	got := mergeTopClients([]TopClients{a, b}, 5)
	if len(got.Clients) != 1 || got.Clients[0].Name != "tv" ||
		got.Clients[0].Count != 3 {
		t.Fatalf("got %+v", got)
	}
}

func TestMergeQueriesOrderTagTrim(t *testing.T) {
	cur := int64(1)
	a := QueriesPage{Queries: []Query{{Time: 9}, {Time: 5}, {Time: 1}},
		RecordsFiltered: 3, RecordsTotal: 10, Cursor: &cur, DraftCursor: &cur}
	b := QueriesPage{Queries: []Query{{Time: 8}, {Time: 7}, {Time: 2}},
		RecordsFiltered: 4, RecordsTotal: 20}
	got := mergeQueries([]QueriesPage{a, b}, []string{"pi1", "pi2"}, 3)
	times := []float64{9, 8, 7}
	inst := []string{"pi1", "pi2", "pi2"}
	if len(got.Queries) != 3 || got.RecordsFiltered != 7 ||
		got.RecordsTotal != 30 || got.Cursor != nil ||
		got.DraftCursor != nil {
		t.Fatalf("got %+v", got)
	}
	for i, q := range got.Queries {
		if q.Time != times[i] || q.Instance != inst[i] {
			t.Fatalf("row %d: %+v", i, q)
		}
	}
}

func TestMergeBlocking(t *testing.T) {
	t30, t10 := 30.0, 10.0
	got := mergeBlocking([]BlockingStatus{
		{Blocking: true},
		{Timer: &t30},
		{Timer: &t10},
	})
	if got.On != 1 || got.Total != 3 || got.Blocking ||
		got.Timer == nil || *got.Timer != 10 {
		t.Fatalf("got %+v", got)
	}
	got = mergeBlocking([]BlockingStatus{{Blocking: true}, {Blocking: true}})
	if !got.Blocking || got.Timer != nil || got.On != 2 {
		t.Fatalf("got %+v", got)
	}
	// A blocking member's stale timer must not win the minimum.
	t5 := 5.0
	got = mergeBlocking([]BlockingStatus{
		{Blocking: true, Timer: &t5},
		{Timer: &t30},
	})
	if got.Timer == nil || *got.Timer != 30 {
		t.Fatalf("got %+v", got)
	}
}
