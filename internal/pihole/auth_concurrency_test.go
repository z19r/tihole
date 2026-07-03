package pihole

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
)

// TestConcurrentRequestsLoginOnce reproduces the dashboard's startup pattern:
// many widget fetches fire in parallel against a client with no session. They
// must coalesce into a SINGLE login (single-flight), not one per request —
// otherwise a real FTL exhausts its session seats and starts returning 401s.
func TestConcurrentRequestsLoginOnce(t *testing.T) {
	// Arrange: a mock FTL that 401s unauthenticated GETs, mints exactly one SID
	// per login, and 200s any GET carrying that SID.
	var logins int64
	var validSID atomic.Value // string
	validSID.Store("")

	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/auth" {
			n := atomic.AddInt64(&logins, 1)
			sid := "SID-" + string(rune('A'+n-1))
			validSID.Store(sid)
			authOK(w, sid)
			return
		}
		if r.Header.Get("X-FTL-SID") == validSID.Load().(string) && validSID.Load().(string) != "" {
			writeJSON(w, http.StatusOK, `{"blocking":"enabled","timer":null,"took":0.0}`)
			return
		}
		writeJSON(w, http.StatusUnauthorized,
			`{"session":{"valid":false,"sid":null},"took":0.0}`)
	})

	// Act: 24 concurrent requests, all starting with no session.
	const n = 24
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = client.Blocking(context.Background())
		}(i)
	}
	wg.Wait()

	// Assert: every request succeeded and exactly one login occurred.
	for i, err := range errs {
		if err != nil {
			t.Errorf("request %d failed: %v", i, err)
		}
	}
	if got := atomic.LoadInt64(&logins); got != 1 {
		t.Fatalf("logins = %d, want exactly 1 (single-flight re-auth)", got)
	}
}
