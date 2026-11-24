package pihole

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoginSuccess(t *testing.T) {
	// Arrange
	var gotBody loginRequest
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/auth" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		authOK(w, "SID-123")
	})

	// Act
	err := client.login(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("login returned error: %v", err)
	}
	if gotBody.Password != "secret-pw" {
		t.Errorf("password sent = %q, want secret-pw", gotBody.Password)
	}
	if client.sessionID() != "SID-123" {
		t.Errorf("sid = %q, want SID-123", client.sessionID())
	}
}

func TestLoginInvalidCredentialsReturnsAuthError(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(
			w,
			http.StatusUnauthorized,
			`{"error":{"key":"unauthorized","message":"Invalid password"},"took":0.0}`,
		)
	})

	// Act
	err := client.login(context.Background())

	// Assert
	authErr, ok := err.(*AuthError)
	if !ok {
		t.Fatalf("error type = %T, want *AuthError", err)
	}
	if authErr.Status != http.StatusUnauthorized {
		t.Errorf("Status = %d, want 401", authErr.Status)
	}
}

func TestLoginInvalidSessionBody(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(
			w,
			http.StatusOK,
			`{"session":{"valid":false,"message":"nope"},"took":0.0}`,
		)
	})

	// Act
	err := client.login(context.Background())

	// Assert
	if _, ok := err.(*AuthError); !ok {
		t.Fatalf("error type = %T, want *AuthError", err)
	}
}

func TestCloseEndsSession(t *testing.T) {
	// Arrange
	called := false
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/auth" {
			called = true
			if r.Header.Get("X-FTL-SID") != "S" {
				t.Errorf("X-FTL-SID = %q, want S", r.Header.Get("X-FTL-SID"))
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
	})
	client.setSID("S")

	// Act
	err := client.Close(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}
	if !called {
		t.Errorf("DELETE /api/auth was not called")
	}
	if client.sessionID() != "" {
		t.Errorf("sid = %q, want empty after logout", client.sessionID())
	}
}

func TestCloseNoSessionIsNoop(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request expected, got %s %s", r.Method, r.URL.Path)
	})

	// Act & Assert
	if err := client.Close(context.Background()); err != nil {
		t.Fatalf("Close error: %v", err)
	}
}

func TestCloseStopsReauth(t *testing.T) {
	var logins, deletes atomic.Int32
	c, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/auth" && r.Method == http.MethodDelete:
			deletes.Add(1)
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/api/auth":
			logins.Add(1)
			authOK(w, "SID-1")
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	})
	ctx := context.Background()
	if err := c.login(ctx); err != nil {
		t.Fatal(err)
	}

	if err := c.Close(ctx); err != nil {
		t.Fatal(err)
	}
	_, err := c.Summary(ctx)

	if !errors.Is(err, ErrClosed) {
		t.Fatalf("err = %v, want ErrClosed", err)
	}
	if !errors.Is(c.login(ctx), ErrClosed) {
		t.Fatal("login after Close should fail with ErrClosed")
	}
	if logins.Load() != 1 || deletes.Load() != 1 {
		t.Fatalf("logins = %d, deletes = %d; want 1, 1",
			logins.Load(), deletes.Load())
	}
}

func TestCloseDuringLoginDiscardsNewSession(t *testing.T) {
	arrived, release := make(chan struct{}), make(chan struct{})
	deleted := make(chan string, 2)
	c, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleted <- r.Header.Get("X-FTL-SID")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		close(arrived)
		<-release
		authOK(w, "SID-2")
	})
	ctx := context.Background()
	errc := make(chan error, 1)
	go func() { errc <- c.login(ctx) }()
	<-arrived // login is awaiting its response

	if err := c.Close(ctx); err != nil {
		t.Fatal(err)
	}
	close(release)

	if err := <-errc; !errors.Is(err, ErrClosed) {
		t.Fatalf("login err = %v, want ErrClosed", err)
	}
	select {
	case sid := <-deleted:
		if sid != "SID-2" {
			t.Fatalf("DELETE sid = %q, want SID-2", sid)
		}
	default:
		t.Fatal("new session was not deleted")
	}
	if c.sessionID() != "" || !c.isClosed() {
		t.Fatal("client should stay closed with no session")
	}
}

func TestConcurrentCloseDeletesSessionOnce(t *testing.T) {
	// Arrange: the DELETE is slow, so a second Close overlaps the first.
	var deletes atomic.Int32
	c, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		time.Sleep(20 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	})
	c.setSID("S")

	// Act
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() { _ = c.Close(context.Background()) })
	}
	wg.Wait()

	// Assert
	if n := deletes.Load(); n != 1 {
		t.Fatalf("DELETE /api/auth = %d, want 1", n)
	}
}
