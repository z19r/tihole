package pihole

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestBlockingGet(t *testing.T) {
	// Arrange
	timer := 42.0
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/dns/blocking" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		// FTL v6 reports the state as a status STRING on GET, not a boolean.
		writeJSON(w, http.StatusOK, `{"blocking":"disabled","timer":42,"took":0.0}`)
	})
	client.setSID("S")

	// Act
	got, err := client.Blocking(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("Blocking error: %v", err)
	}
	if got.Blocking {
		t.Errorf("blocking = true, want false")
	}
	if got.Status != "disabled" {
		t.Errorf("status = %q, want %q", got.Status, "disabled")
	}
	if got.Timer == nil || *got.Timer != timer {
		t.Errorf("timer = %v, want %v", got.Timer, timer)
	}
}

func TestBlockingGetNilTimer(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"blocking":"enabled","timer":null,"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.Blocking(context.Background())
	if err != nil {
		t.Fatalf("Blocking error: %v", err)
	}
	if !got.Blocking || got.Timer != nil {
		t.Errorf("got = %+v, want blocking=true timer=nil", got)
	}
}

func TestSetBlockingPostsBody(t *testing.T) {
	// Arrange
	var gotBody blockingRequest
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/dns/blocking" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		writeJSON(w, http.StatusOK, `{"blocking":false,"timer":300,"took":0.0}`)
	})
	client.setSID("S")

	// Act
	timer := 300.0
	err := client.SetBlocking(context.Background(), false, &timer)

	// Assert
	if err != nil {
		t.Fatalf("SetBlocking error: %v", err)
	}
	if gotBody.Blocking {
		t.Errorf("body.blocking = true, want false")
	}
	if gotBody.Timer == nil || *gotBody.Timer != 300 {
		t.Errorf("body.timer = %v, want 300", gotBody.Timer)
	}
}

func TestSetBlockingNilTimer(t *testing.T) {
	var rawBody string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		rawBody = string(data)
		writeJSON(w, http.StatusOK, `{"blocking":true,"timer":null,"took":0.0}`)
	})
	client.setSID("S")

	if err := client.SetBlocking(context.Background(), true, nil); err != nil {
		t.Fatalf("SetBlocking error: %v", err)
	}
	// timer must be present as null (not omitted) so FTL clears any countdown.
	if rawBody != `{"blocking":true,"timer":null}` {
		t.Errorf("body = %s, want timer:null present", rawBody)
	}
}
