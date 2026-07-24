package pihole

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockFTL spins up an httptest server whose handler is supplied by the test,
// and returns a Client pointed at it. The client uses the test server's own
// http.Client so TLS (for TLS servers) is trusted.
func mockFTL(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := New(srv.URL, "secret-pw", WithHTTPClient(srv.Client()))
	return client, srv
}

// authOK writes a valid session envelope, used by handlers to satisfy the
// transparent login step.
func authOK(w http.ResponseWriter, sid string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"session":{"valid":true,"sid":"` + sid + `","csrf":"c","validity":1800,"totp":false},"took":0.1}`))
}

// writeJSON is a small helper for handlers.
func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
