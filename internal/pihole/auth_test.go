package pihole

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
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
	err := client.Login(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if gotBody.Password != "secret-pw" {
		t.Errorf("password sent = %q, want secret-pw", gotBody.Password)
	}
	if client.sessionID() != "SID-123" {
		t.Errorf("sid = %q, want SID-123", client.sessionID())
	}
}

func TestLoginWithTOTP(t *testing.T) {
	// Arrange
	var gotBody loginRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		authOK(w, "SID-TOTP")
	}))
	t.Cleanup(srv.Close)
	client := New(srv.URL, "pw", WithHTTPClient(srv.Client()), WithTOTP(123456))

	// Act
	if err := client.Login(context.Background()); err != nil {
		t.Fatalf("Login error: %v", err)
	}

	// Assert
	if gotBody.TOTP == nil || *gotBody.TOTP != 123456 {
		t.Errorf("totp sent = %v, want 123456", gotBody.TOTP)
	}
}

func TestLoginInvalidCredentialsReturnsAuthError(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnauthorized,
			`{"error":{"key":"unauthorized","message":"Invalid password"},"took":0.0}`)
	})

	// Act
	err := client.Login(context.Background())

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
		writeJSON(w, http.StatusOK, `{"session":{"valid":false,"message":"nope"},"took":0.0}`)
	})

	// Act
	err := client.Login(context.Background())

	// Assert
	if _, ok := err.(*AuthError); !ok {
		t.Fatalf("error type = %T, want *AuthError", err)
	}
}

func TestSessionValid(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/auth" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, http.StatusOK,
			`{"session":{"valid":true,"sid":"S","validity":1800},"took":0.0}`)
	})
	client.setSID("S")

	// Act
	sess, err := client.SessionValid(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("SessionValid error: %v", err)
	}
	if !sess.Valid {
		t.Errorf("session not valid, want valid")
	}
}

func TestLogout(t *testing.T) {
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
	err := client.Logout(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("Logout error: %v", err)
	}
	if !called {
		t.Errorf("DELETE /api/auth was not called")
	}
	if client.sessionID() != "" {
		t.Errorf("sid = %q, want empty after logout", client.sessionID())
	}
}

func TestLogoutNoSessionIsNoop(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request expected, got %s %s", r.Method, r.URL.Path)
	})

	// Act & Assert
	if err := client.Logout(context.Background()); err != nil {
		t.Fatalf("Logout error: %v", err)
	}
}
