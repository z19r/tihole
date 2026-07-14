package pihole

import (
	"context"
	"net/http"
	"testing"
)

func TestSummary(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stats/summary" {
			t.Errorf("path = %q", r.URL.Path)
		}
		writeJSON(
			w,
			http.StatusOK,
			`{"queries":{"total":100,"blocked":25,"percent_blocked":25.0,"unique_domains":40},"clients":{"active":5,"total":8},"gravity":{"domains_being_blocked":1000},"took":0.1}`,
		)
	})
	client.setSID("S")

	// Act
	sum, err := client.Summary(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("Summary error: %v", err)
	}
	if sum.Queries.Total != 100 || sum.Queries.Blocked != 25 {
		t.Errorf("queries = %+v", sum.Queries)
	}
	if sum.Clients.Active != 5 || sum.Gravity.DomainsBeingBlocked != 1000 {
		t.Errorf("clients/gravity mismatch: %+v %+v", sum.Clients, sum.Gravity)
	}
}

func TestUpstreams(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(
			w,
			http.StatusOK,
			`{"upstreams":[{"ip":"1.1.1.1","name":"cloudflare","port":53,"count":42}],"forwarded_queries":42,"total_queries":100,"took":0.0}`,
		)
	})
	client.setSID("S")

	got, err := client.Upstreams(context.Background())
	if err != nil {
		t.Fatalf("Upstreams error: %v", err)
	}
	if len(got.Upstreams) != 1 || got.Upstreams[0].IP != "1.1.1.1" ||
		got.Upstreams[0].Count != 42 {
		t.Errorf("upstreams = %+v", got.Upstreams)
	}
}

func TestQueryTypes(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"types":{"A":80,"AAAA":20},"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.QueryTypes(context.Background())
	if err != nil {
		t.Fatalf("QueryTypes error: %v", err)
	}
	if got.Types["A"] != 80 || got.Types["AAAA"] != 20 {
		t.Errorf("types = %+v", got.Types)
	}
}

func TestTopDomainsBuildsBlockedAndCountParams(t *testing.T) {
	// Arrange
	var gotQuery string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(
			w,
			http.StatusOK,
			`{"domains":[{"domain":"ads.example","count":9}],"total_queries":100,"blocked_queries":9,"took":0.0}`,
		)
	})
	client.setSID("S")

	// Act
	got, err := client.TopDomains(context.Background(), true, 15)

	// Assert
	if err != nil {
		t.Fatalf("TopDomains error: %v", err)
	}
	if gotQuery != "blocked=true&count=15" {
		t.Errorf("query = %q, want blocked=true&count=15", gotQuery)
	}
	if len(got.Domains) != 1 || got.Domains[0].Domain != "ads.example" {
		t.Errorf("domains = %+v", got.Domains)
	}
}

func TestTopClientsOmitsCountWhenZero(t *testing.T) {
	var gotQuery string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(
			w,
			http.StatusOK,
			`{"clients":[{"ip":"10.0.0.5","name":"laptop","count":50}],"total_queries":100,"took":0.0}`,
		)
	})
	client.setSID("S")

	got, err := client.TopClients(context.Background(), false, 0)
	if err != nil {
		t.Fatalf("TopClients error: %v", err)
	}
	if gotQuery != "blocked=false" {
		t.Errorf("query = %q, want blocked=false", gotQuery)
	}
	if got.Clients[0].IP != "10.0.0.5" {
		t.Errorf("clients = %+v", got.Clients)
	}
}

func TestRecentBlocked(t *testing.T) {
	var gotQuery string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(w, http.StatusOK, `{"blocked":["a.ads","b.ads"],"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.RecentBlocked(context.Background(), 2)
	if err != nil {
		t.Fatalf("RecentBlocked error: %v", err)
	}
	if gotQuery != "count=2" {
		t.Errorf("query = %q, want count=2", gotQuery)
	}
	if len(got) != 2 || got[0] != "a.ads" {
		t.Errorf("blocked = %v", got)
	}
}
