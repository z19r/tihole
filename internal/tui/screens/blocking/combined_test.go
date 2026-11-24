package blocking

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/z19r/tihole/internal/pihole"
)

const authJSON = `{"session":{"valid":true,"sid":"s","csrf":"c",` +
	`"validity":1800}}`

// hole is a fake Pi-hole recording POST /api/dns/blocking bodies.
type hole struct {
	mu    sync.Mutex
	posts []string
}

func (h *hole) client(t *testing.T) *pihole.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/api/auth" {
				_, _ = w.Write([]byte(authJSON))
				return
			}
			if r.Method == http.MethodPost {
				b, _ := io.ReadAll(r.Body)
				h.mu.Lock()
				h.posts = append(h.posts, string(b))
				h.mu.Unlock()
			}
			_, _ = w.Write([]byte(`{"blocking":"enabled","took":0.1}`))
		}))
	t.Cleanup(srv.Close)
	return pihole.New(srv.URL, "pw", pihole.WithHTTPClient(srv.Client()))
}

func combined(t *testing.T) (*Model, [3]*hole) {
	t.Helper()
	var h [3]*hole
	for i := range h {
		h[i] = &hole{}
	}
	m, _ := newModel(t, `{"blocking":"enabled","took":0.1}`)
	m.ctx.API = h[0].client(t)
	m.ctx.Fleet = pihole.NewFleet([]pihole.Member{
		{Name: "primary", Client: h[1].client(t)},
		{Name: "secondary", Client: h[2].client(t)},
	}, nil)
	m.ctx.Combined = true
	return m, h
}

func TestBlockingCombinedHeading(t *testing.T) {
	m, _ := combined(t)
	if v := m.View().Content; !strings.Contains(v, "all (2)") {
		t.Errorf("view lacks %q:\n%s", "all (2)", v)
	}
}

func TestBlockingCombinedPartialStatus(t *testing.T) {
	m, _ := combined(t)
	_, _ = m.Update(statusMsg{status: pihole.BlockingStatus{On: 1, Total: 2}})
	if v := m.View().Content; !strings.Contains(v, "1/2") {
		t.Errorf("view lacks 1/2:\n%s", v)
	}
}

func TestBlockingApplyUsesSource(t *testing.T) {
	m, h := combined(t)
	_ = run(m.apply(actions[1])) // disable 5 minutes
	_ = run(m.apply(action{label: "30s", secs: 30}))
	for i := 1; i <= 2; i++ {
		if len(h[i].posts) != 2 ||
			!strings.Contains(h[i].posts[1], `"blocking":false`) ||
			!strings.Contains(h[i].posts[1], `"timer":30`) {
			t.Errorf("member %d posts = %v", i, h[i].posts)
		}
	}
	if len(h[0].posts) != 0 {
		t.Errorf("API posts = %v, want none", h[0].posts)
	}
}
