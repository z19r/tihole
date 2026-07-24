package pihole

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestNewNormalizesBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantAPI string
	}{
		{"plain host", "https://pi.hole", "https://pi.hole/api"},
		{"trailing slash", "https://pi.hole/", "https://pi.hole/api"},
		{"with api suffix", "https://pi.hole/api", "https://pi.hole/api"},
		{"api suffix and slash", "https://pi.hole/api/", "https://pi.hole/api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange & Act
			c := New(tt.baseURL, "pw")
			// Assert
			if c.apiURL != tt.wantAPI {
				t.Errorf("apiURL = %q, want %q", c.apiURL, tt.wantAPI)
			}
		})
	}
}

func TestDoSendsXFTLSIDHeader(t *testing.T) {
	// Arrange
	var gotSID string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotSID = r.Header.Get("X-FTL-SID")
		if r.Header.Get("X-FTL-CSRF") != "" {
			t.Errorf("X-FTL-CSRF should not be sent in header auth mode")
		}
		writeJSON(w, http.StatusOK, `{"took":0.0}`)
	})
	client.setSID("SID-AUTH")

	// Act
	err := client.do(context.Background(), http.MethodGet, "/stats/summary", nil, nil)

	// Assert
	if err != nil {
		t.Fatalf("do error: %v", err)
	}
	if gotSID != "SID-AUTH" {
		t.Errorf("X-FTL-SID = %q, want SID-AUTH", gotSID)
	}
}

func TestDo401TriggersExactlyOneReauthAndRetry(t *testing.T) {
	// Arrange
	var authCalls int32
	var protectedCalls int32
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth":
			atomic.AddInt32(&authCalls, 1)
			authOK(w, "FRESH-SID")
		case "/api/stats/summary":
			n := atomic.AddInt32(&protectedCalls, 1)
			if n == 1 {
				// First hit: pretend the (empty) session is invalid.
				writeJSON(w, http.StatusUnauthorized,
					`{"error":{"key":"unauthorized","message":"no session"},"took":0.0}`)
				return
			}
			// Retry must carry the freshly-minted SID.
			if r.Header.Get("X-FTL-SID") != "FRESH-SID" {
				t.Errorf("retry X-FTL-SID = %q, want FRESH-SID", r.Header.Get("X-FTL-SID"))
			}
			writeJSON(w, http.StatusOK, `{"took":0.0}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})

	// Act
	err := client.do(context.Background(), http.MethodGet, "/stats/summary", nil, nil)

	// Assert
	if err != nil {
		t.Fatalf("do error: %v", err)
	}
	if got := atomic.LoadInt32(&authCalls); got != 1 {
		t.Errorf("auth called %d times, want exactly 1", got)
	}
	if got := atomic.LoadInt32(&protectedCalls); got != 2 {
		t.Errorf("protected endpoint called %d times, want 2 (original + retry)", got)
	}
}

func TestDo401RetryStillFailsReturnsAPIError(t *testing.T) {
	// Arrange: server always 401s the protected endpoint even after re-auth.
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			authOK(w, "SID")
			return
		}
		writeJSON(w, http.StatusUnauthorized,
			`{"error":{"key":"unauthorized","message":"still no"},"took":0.0}`)
	})

	// Act
	err := client.do(context.Background(), http.MethodGet, "/stats/summary", nil, nil)

	// Assert: retry disabled on the second pass, so the 401 surfaces as APIError.
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.Status != http.StatusUnauthorized {
		t.Errorf("Status = %d, want 401", apiErr.Status)
	}
}

func TestDoDecodesErrorEnvelope(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusForbidden,
			`{"error":{"key":"forbidden","message":"Destructive action","hint":"set allow_destructive"},"took":0.0}`)
	})
	client.setSID("S")

	// Act
	err := client.do(context.Background(), http.MethodPost, "/action/restartdns", nil, nil)

	// Assert
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.Status != http.StatusForbidden || apiErr.Key != "forbidden" ||
		apiErr.Message != "Destructive action" || apiErr.Hint != "set allow_destructive" {
		t.Errorf("APIError = %+v, missing decoded fields", apiErr)
	}
}

func TestDoNetworkErrorOnUnreachableHost(t *testing.T) {
	// Arrange: point at a closed port.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // now unreachable
	client := New(url, "pw")

	// Act
	err := client.do(context.Background(), http.MethodGet, "/stats/summary", nil, nil)

	// Assert
	if _, ok := err.(*NetworkError); !ok {
		t.Fatalf("error type = %T, want *NetworkError", err)
	}
}

func TestWithInsecureTLSWiresSkipVerify(t *testing.T) {
	// Arrange: a TLS server with a self-signed cert.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			authOK(w, "SID")
			return
		}
		writeJSON(w, http.StatusOK, `{"took":0.0}`)
	}))
	t.Cleanup(srv.Close)

	// A default client (no insecure opt-in) must fail the TLS handshake.
	strict := New(srv.URL, "pw")
	if err := strict.do(context.Background(), http.MethodGet, "/stats/summary", nil, nil); err == nil {
		t.Fatalf("strict client unexpectedly trusted a self-signed cert")
	}

	// Act: opt into InsecureSkipVerify.
	insecure := New(srv.URL, "pw", WithInsecureTLS(true))

	// Assert: the transport carries InsecureSkipVerify and the call succeeds.
	tr, ok := insecure.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", insecure.httpClient.Transport)
	}
	if tr.TLSClientConfig == nil || !tr.TLSClientConfig.InsecureSkipVerify {
		t.Errorf("InsecureSkipVerify not set on transport")
	}
	if err := insecure.do(context.Background(), http.MethodGet, "/stats/summary", nil, nil); err != nil {
		t.Errorf("insecure client failed against TLS server: %v", err)
	}
}
