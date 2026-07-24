package pihole

import (
	"context"
	"net/http"
	"testing"
)

func TestHistory(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/history" {
			t.Errorf("path = %q", r.URL.Path)
		}
		writeJSON(w, http.StatusOK,
			`{"history":[{"timestamp":1000,"total":10,"cached":3,"blocked":2,"forwarded":5}],"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.History(context.Background())
	if err != nil {
		t.Fatalf("History error: %v", err)
	}
	if len(got.History) != 1 || got.History[0].Total != 10 || got.History[0].Blocked != 2 {
		t.Errorf("history = %+v", got.History)
	}
}

func TestHistoryClientsBuildsNParam(t *testing.T) {
	var gotQuery string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(w, http.StatusOK,
			`{"clients":{"10.0.0.1":{"name":"host","total":9}},"history":[{"timestamp":1000,"data":{"10.0.0.1":9}}],"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.HistoryClients(context.Background(), 3)
	if err != nil {
		t.Fatalf("HistoryClients error: %v", err)
	}
	if gotQuery != "N=3" {
		t.Errorf("query = %q, want N=3", gotQuery)
	}
	if got.History[0].Data["10.0.0.1"] != 9 {
		t.Errorf("history data = %+v", got.History)
	}
}

func TestHistoryClientsOmitsNWhenZero(t *testing.T) {
	var gotQuery string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(w, http.StatusOK, `{"clients":{},"history":[],"took":0.0}`)
	})
	client.setSID("S")

	if _, err := client.HistoryClients(context.Background(), 0); err != nil {
		t.Fatalf("HistoryClients error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty", gotQuery)
	}
}
