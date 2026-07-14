package pihole

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestQueryFilterValuesOnlySetFields(t *testing.T) {
	tests := []struct {
		name   string
		filter QueryFilter
		want   url.Values
	}{
		{
			name:   "empty filter yields no params",
			filter: QueryFilter{},
			want:   url.Values{},
		},
		{
			name: "cursor pagination params",
			filter: QueryFilter{
				Length: ptr(50),
				Start:  ptr(100),
				Cursor: ptr(int64(987654321)),
			},
			want: url.Values{
				"length": {"50"},
				"start":  {"100"},
				"cursor": {"987654321"},
			},
		},
		{
			name: "filters and time window",
			filter: QueryFilter{
				From:       ptr(int64(1000)),
				Until:      ptr(int64(2000)),
				Domain:     ptr("example.com"),
				ClientIP:   ptr("10.0.0.5"),
				ClientName: ptr("laptop"),
				Upstream:   ptr("1.1.1.1#53"),
				Type:       ptr("A"),
				Status:     ptr("GRAVITY"),
				Reply:      ptr("IP"),
				DNSSEC:     ptr("SECURE"),
				Disk:       ptr(true),
			},
			want: url.Values{
				"from": {"1000"}, "until": {"2000"}, "domain": {"example.com"},
				"client_ip": {"10.0.0.5"}, "client_name": {"laptop"},
				"upstream": {
					"1.1.1.1#53",
				}, "type": {"A"}, "status": {"GRAVITY"},
				"reply": {"IP"}, "dnssec": {"SECURE"}, "disk": {"true"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := tt.filter.values()
			// Assert
			if got.Encode() != tt.want.Encode() {
				t.Errorf("values = %q, want %q", got.Encode(), tt.want.Encode())
			}
		})
	}
}

func TestQueriesRoundTrip(t *testing.T) {
	// Arrange
	var gotQuery string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/queries" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		writeJSON(
			w,
			http.StatusOK,
			`{"queries":[{"id":1,"time":123.4,"type":"A","domain":"example.com","status":"FORWARDED","client":{"ip":"10.0.0.5","name":"laptop"},"reply":{"type":"IP","time":1.2},"upstream":"1.1.1.1#53"}],"cursor":42,"recordsFiltered":1,"recordsTotal":500,"took":0.0}`,
		)
	})
	client.setSID("S")

	// Act
	page, err := client.Queries(
		context.Background(),
		QueryFilter{Length: ptr(1), Domain: ptr("example.com")},
	)

	// Assert
	if err != nil {
		t.Fatalf("Queries error: %v", err)
	}
	if gotQuery != "domain=example.com&length=1" {
		t.Errorf("query = %q", gotQuery)
	}
	if len(page.Queries) != 1 || page.Queries[0].Domain != "example.com" {
		t.Errorf("queries = %+v", page.Queries)
	}
	if page.Cursor == nil || *page.Cursor != 42 {
		t.Errorf("cursor = %v, want 42", page.Cursor)
	}
	if page.RecordsTotal != 500 {
		t.Errorf("recordsTotal = %d, want 500", page.RecordsTotal)
	}
}
