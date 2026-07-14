package pihole

import (
	"context"
	"net/http"
	"testing"
)

func TestUpdateGravityStreamsLines(t *testing.T) {
	// Arrange
	lines := []string{
		"  [i] Building tree",
		"  [i] Updating gravity",
		"  [✓] Done",
	}
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/action/gravity" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter is not a Flusher")
		}
		for _, l := range lines {
			_, _ = w.Write([]byte(l + "\n"))
			flusher.Flush()
		}
	})
	client.setSID("S")

	// Act
	var got []string
	err := client.UpdateGravity(context.Background(), func(line string) {
		got = append(got, line)
	})

	// Assert
	if err != nil {
		t.Fatalf("UpdateGravity error: %v", err)
	}
	if len(got) != len(lines) {
		t.Fatalf("received %d lines, want %d: %v", len(got), len(lines), got)
	}
	for i := range lines {
		if got[i] != lines[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], lines[i])
		}
	}
}

func TestUpdateGravityPropagatesAPIError(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(
			w,
			http.StatusForbidden,
			`{"error":{"key":"forbidden","message":"nope"},"took":0.0}`,
		)
	})
	client.setSID("S")

	err := client.UpdateGravity(context.Background(), func(string) {})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Status != http.StatusForbidden {
		t.Errorf("err = %v, want *APIError status 403", err)
	}
}

func TestUpdateGravityHonorsContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte("first\n"))
		flusher.Flush()
		// Block so the client sees only the first line before cancel.
		<-r.Context().Done()
	})
	client.setSID("S")

	err := client.UpdateGravity(ctx, func(line string) {
		if line == "first" {
			cancel()
		}
	})
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}
