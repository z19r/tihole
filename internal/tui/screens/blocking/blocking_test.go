package blocking

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

// newModel builds a Blocking screen wired to an httptest FTL server. The handler
// answers the transparent /api/auth login plus GET/POST /api/dns/blocking. The
// returned capture pointer records the last POST body so tests can assert the
// timer/blocking values sent.
func newModel(t *testing.T, get string) (*Model, *string) {
	t.Helper()
	var lastPost string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/auth"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"session":{"valid":true,"sid":"s","csrf":"c","validity":1800,"totp":false},"took":0.1}`))
		case strings.HasSuffix(r.URL.Path, "/dns/blocking"):
			if r.Method == http.MethodPost {
				b, _ := io.ReadAll(r.Body)
				lastPost = string(b)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(get))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	api := pihole.New(srv.URL, "pw", pihole.WithHTTPClient(srv.Client()))
	ctx := &core.AppContext{
		API:          api,
		Theme:        theme.DeepNight(),
		Keys:         core.DefaultKeyMap(),
		InstanceName: "home",
	}
	m := New(ctx)
	m.SetSize(80, 24)
	return m, &lastPost
}

func run(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func press(code rune, text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: text}
}

func TestTitleAndInit(t *testing.T) {
	m, _ := newModel(t, `{"blocking":"enabled","took":0.1}`)
	if m.Title() != "Blocking" {
		t.Fatalf("title = %q, want Blocking", m.Title())
	}
	if m.Init() != nil {
		t.Fatal("Init should be a no-op cmd")
	}
	if len(m.Help()) == 0 {
		t.Fatal("Help should expose bindings")
	}
}

func TestFocusFetchesStatus(t *testing.T) {
	// Arrange
	m, _ := newModel(t, `{"blocking":"enabled","took":0.1}`)

	// Act: Focus returns a fetch cmd; run it and feed the result back.
	msg := run(m.Focus())
	updated, _ := m.Update(msg)
	m = updated.(*Model)

	// Assert
	if !m.known || !m.status.Blocking {
		t.Fatalf("expected known+blocking status, got known=%v status=%+v", m.known, m.status)
	}
}

func TestStatusMsgErrorSetsErr(t *testing.T) {
	m, _ := newModel(t, `{}`)
	updated, _ := m.Update(statusMsg{err: io.EOF})
	m = updated.(*Model)
	if m.err == nil {
		t.Fatal("expected err to be set")
	}
	if strings.Contains(m.View().Content, "blocking is") {
		t.Fatal("error state should not render an ON/OFF banner")
	}
}

func TestCursorMovesAndClamps(t *testing.T) {
	m, _ := newModel(t, `{"blocking":"enabled"}`)

	// Up at the top is a no-op.
	updated, _ := m.Update(press(tea.KeyUp, "up"))
	m = updated.(*Model)
	if m.cursor != 0 {
		t.Fatalf("cursor should stay at 0, got %d", m.cursor)
	}

	// Down walks to the last action and clamps.
	for i := 0; i < len(actions)+3; i++ {
		updated, _ = m.Update(press(tea.KeyDown, "down"))
		m = updated.(*Model)
	}
	if m.cursor != len(actions)-1 {
		t.Fatalf("cursor should clamp to %d, got %d", len(actions)-1, m.cursor)
	}

	// Up steps back.
	updated, _ = m.Update(press(tea.KeyUp, "up"))
	m = updated.(*Model)
	if m.cursor != len(actions)-2 {
		t.Fatalf("cursor should be %d, got %d", len(actions)-2, m.cursor)
	}
}

func TestEnterAppliesTimedDisable(t *testing.T) {
	// Arrange: land on "Disable for 5 minutes" (index 1).
	m, last := newModel(t, `{"blocking":"disabled","timer":300}`)
	updated, _ := m.Update(press(tea.KeyDown, "down"))
	m = updated.(*Model)

	// Act: Enter applies; run the returned cmd against the server.
	_, cmd := m.Update(press(tea.KeyEnter, "enter"))
	msg := run(cmd)
	updated, _ = m.Update(msg)
	m = updated.(*Model)

	// Assert: the POST carried blocking=false with a 300s timer.
	var body map[string]any
	if err := json.Unmarshal([]byte(*last), &body); err != nil {
		t.Fatalf("post body not JSON: %q", *last)
	}
	if body["blocking"] != false {
		t.Fatalf("expected blocking=false, got %v", body["blocking"])
	}
	if body["timer"] != float64(300) {
		t.Fatalf("expected timer=300, got %v", body["timer"])
	}
	if m.status.Blocking {
		t.Fatal("status should reflect disabled after apply")
	}
}

func TestEnterEnableSendsNoTimer(t *testing.T) {
	// Arrange: cursor at index 0 = "Enable blocking".
	m, last := newModel(t, `{"blocking":"enabled"}`)

	// Act
	_, cmd := m.Update(press(tea.KeyEnter, "enter"))
	run(cmd)

	// Assert: enable carries blocking=true and a nil timer.
	var body map[string]any
	if err := json.Unmarshal([]byte(*last), &body); err != nil {
		t.Fatalf("post body not JSON: %q", *last)
	}
	if body["blocking"] != true {
		t.Fatalf("expected blocking=true, got %v", body["blocking"])
	}
	if body["timer"] != nil {
		t.Fatalf("enable should send nil timer, got %v", body["timer"])
	}
}

func TestRefreshRefetches(t *testing.T) {
	m, _ := newModel(t, `{"blocking":"enabled"}`)
	_, cmd := m.Update(press('r', "r"))
	if cmd == nil {
		t.Fatal("refresh should return a fetch cmd")
	}
	updated, _ := m.Update(run(cmd))
	m = updated.(*Model)
	if !m.known {
		t.Fatal("status should be known after refresh")
	}
}

func TestBlurCancelsInFlight(t *testing.T) {
	m, _ := newModel(t, `{"blocking":"enabled"}`)
	m.Focus() // primes loading; cancel is set lazily, Blur must be safe regardless
	m.Blur()  // must not panic when there is no in-flight request
}

func TestViewStates(t *testing.T) {
	cases := []struct {
		name string
		set  func(*Model)
		want string
	}{
		{"loading", func(m *Model) {}, "loading status"},
		{"on", func(m *Model) { m.known = true; m.status = pihole.BlockingStatus{Blocking: true} }, "blocking is ON"},
		{"off", func(m *Model) { m.known = true; m.status = pihole.BlockingStatus{Blocking: false} }, "blocking is OFF"},
		{"error", func(m *Model) { m.err = io.EOF }, "error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := newModel(t, `{}`)
			tc.set(m)
			out := m.View().Content
			if !strings.Contains(out, tc.want) {
				t.Fatalf("view %q missing %q", tc.name, tc.want)
			}
			// The action menu is always visible — the whole point of the pane.
			if !strings.Contains(out, "Disable for 5 minutes") {
				t.Fatal("action menu should always render")
			}
		})
	}
}

func TestOffViewShowsCountdown(t *testing.T) {
	m, _ := newModel(t, `{}`)
	secs := 125.0
	m.known = true
	m.status = pihole.BlockingStatus{Blocking: false, Timer: &secs}
	out := m.View().Content
	if !strings.Contains(out, "re-enables in") || !strings.Contains(out, "2m05s") {
		t.Fatalf("expected countdown in view, got: %s", out)
	}
}

func TestHumanCountdown(t *testing.T) {
	cases := map[int]string{-5: "0s", 0: "0s", 45: "45s", 60: "1m00s", 125: "2m05s"}
	for in, want := range cases {
		if got := humanCountdown(in); got != want {
			t.Fatalf("humanCountdown(%d) = %q, want %q", in, got, want)
		}
	}
}
