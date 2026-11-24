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
		writeJSON(
			w,
			http.StatusOK,
			`{"history":[{"timestamp":1000,"total":10,"cached":3,"blocked":2,"forwarded":5}],"took":0.0}`,
		)
	})
	client.setSID("S")

	got, err := client.History(context.Background())
	if err != nil {
		t.Fatalf("History error: %v", err)
	}
	if len(got.History) != 1 || got.History[0].Total != 10 ||
		got.History[0].Blocked != 2 {
		t.Errorf("history = %+v", got.History)
	}
}
