package tui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/z19r/tihole/internal/config"
	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
	"github.com/z19r/tihole/internal/tui/components"
	"github.com/z19r/tihole/internal/tui/core"
)

// withConfigPath points the model at a temp config file so saves stay local.
func withConfigPath(t *testing.T, m *AppModel) *AppModel {
	t.Helper()
	m.cfgPath = filepath.Join(t.TempDir(), "tihole.yaml")
	m.ctx.ConfigPath = m.cfgPath
	return m
}

func TestBuildFleetSkipsUnbuildableMember(t *testing.T) {
	cfg := testConfig()
	api, err := clientFor(cfg, "home")
	if err != nil {
		t.Fatal(err)
	}

	f, _ := buildFleet(cfg, "home", api, nil, nil)

	if f.Len() != 2 {
		t.Fatalf("Len = %d, want 2", f.Len())
	}
	if u := f.Unreachable(); len(u) != 1 || u[0] != "envonly" {
		t.Fatalf("Unreachable = %v, want [envonly]", u)
	}
}

func TestToggleCombinedPersists(t *testing.T) {
	m := withConfigPath(t, sized(t, newTestModel(t), 120, 36))

	updated, cmd := m.Update(keyPress("m", 'm'))
	m = updated.(*AppModel)

	if !m.ctx.Combined || !m.cfg.Combined {
		t.Fatal("combined should be on")
	}
	if cmd == nil {
		t.Fatal("expected refocus + blocking fetch cmd")
	}
	raw, err := os.ReadFile(m.cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "combined: true") {
		t.Fatalf("config not persisted:\n%s", raw)
	}
}

func TestToggleCombinedNeedsTwo(t *testing.T) {
	m := withConfigPath(t, sized(t, newTestModel(t), 120, 36))
	m.cfg.Instances = m.cfg.Instances[:1]

	updated, _ := m.Update(keyPress("m", 'm'))
	m = updated.(*AppModel)

	if m.ctx.Combined {
		t.Fatal("combined should stay off")
	}
	if m.err != "combined view needs two or more instances" {
		t.Fatalf("err = %q", m.err)
	}
}

func TestInstancesChangeRebuildsFleet(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	cfg := &config.Config{
		Active:   "home",
		Combined: true,
		Instances: append(
			append([]config.Instance(nil), m.cfg.Instances...),
			config.Instance{
				Name: "cabin", URL: "http://10.0.0.3", Password: "pw3",
			},
		),
	}

	updated, _ := m.Update(core.InstancesChangedMsg{Config: cfg})
	m = updated.(*AppModel)

	if got := m.ctx.Fleet.Len(); got != 3 {
		t.Fatalf("Fleet.Len = %d, want 3", got)
	}
	if !m.ctx.Combined {
		t.Fatal("Combined should sync from the incoming config")
	}
}

func TestStatusLabelPerPage(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	m.ctx.Combined = true

	if out := ansi.Strip(m.View().Content); !strings.Contains(out, "all (2)") {
		t.Fatalf("dashboard should show all (2):\n%s", out)
	}

	m.switchPage(core.PageAdlists)
	out := ansi.Strip(m.View().Content)
	if strings.Contains(out, "all (2)") || !strings.Contains(out, "home") {
		t.Fatalf("adlists should show the active name:\n%s", out)
	}
}

func TestToggleBlockingPartialEnablesAll(t *testing.T) {
	cases := []struct {
		name string
		b    components.BlockState
		want bool
	}{
		{"unknown", components.BlockState{}, true},
		{"single on", components.BlockState{Known: true, Enabled: true}, false},
		{"single off", components.BlockState{Known: true}, true},
		{"partial", components.BlockState{Known: true, On: 1, Total: 2}, true},
		{"all on", components.BlockState{
			Known: true, Enabled: true, On: 2, Total: 2,
		}, false},
		{"all off", components.BlockState{Known: true, Total: 2}, true},
		{"on, one down", components.BlockState{
			Known: true, Enabled: true, On: 1, Total: 2,
		}, false},
	}
	for _, tc := range cases {
		if got := desiredBlocking(tc.b); got != tc.want {
			t.Errorf("%s: desired = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// hole is a fake Pi-hole reporting blocking on. failPost makes POST
// /api/dns/blocking fail; logouts counts DELETE /api/auth.
type hole struct{ logouts atomic.Int32 }

func newHole(t *testing.T, failPost bool) (*pihole.Client, *hole) {
	t.Helper()
	h := &hole{}
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.URL.Path == "/api/auth" && r.Method == http.MethodDelete:
				h.logouts.Add(1)
				w.WriteHeader(http.StatusNoContent)
			case r.URL.Path == "/api/auth":
				_, _ = w.Write([]byte(`{"session":{"valid":true,` +
					`"sid":"s","csrf":"c","validity":1800}}`))
			case r.Method == http.MethodPost && failPost:
				w.WriteHeader(http.StatusInternalServerError)
			default:
				_, _ = w.Write([]byte(`{"blocking":"enabled","took":0.1}`))
			}
		}))
	t.Cleanup(srv.Close)
	return pihole.New(srv.URL, "pw", pihole.WithHTTPClient(srv.Client())), h
}

func TestToggleCombinedSavesLiveConfig(t *testing.T) {
	// Arrange: Settings' theme picker swaps in a fresh config copy.
	m := withConfigPath(t, sized(t, newTestModel(t), 120, 36))
	next := *m.ctx.Config
	next.Theme = theme.NameGloss
	m.ctx.Config = &next

	// Act
	updated, _ := m.Update(keyPress("m", 'm'))
	m = updated.(*AppModel)

	// Assert: the new theme survives and the live copy holds the toggle.
	raw, err := os.ReadFile(m.cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "theme: gloss") {
		t.Fatalf("theme reverted on save:\n%s", raw)
	}
	if !m.ctx.Config.Combined {
		t.Fatal("ctx.Config.Combined should be true")
	}
}

func TestBuildFleetReusesUnchangedClients(t *testing.T) {
	cfg := testConfig()
	cfg.Instances[1].Password = "pw2"
	api, _ := clientFor(cfg, "home")
	prev, _ := buildFleet(cfg, "home", api, nil, nil)

	next, dropped := buildFleet(cfg, "home", api, prev, cfg)

	if next.Client("envonly") != prev.Client("envonly") {
		t.Fatal("unchanged instance should keep its client")
	}
	if len(dropped) != 0 {
		t.Fatalf("dropped = %v, want none", dropped)
	}
}

func TestBuildFleetReplacesChangedClients(t *testing.T) {
	cfg := testConfig()
	cfg.Instances[1].Password = "pw2"
	api, _ := clientFor(cfg, "home")
	prev, _ := buildFleet(cfg, "home", api, nil, nil)
	old := prev.Client("envonly")
	edited := testConfig()
	edited.Instances[1].Password = "changed"

	next, dropped := buildFleet(edited, "home", api, prev, cfg)

	if c := next.Client("envonly"); c == nil || c == old {
		t.Fatal("changed instance should get a new client")
	}
	if len(dropped) != 1 || dropped[0] != old {
		t.Fatalf("dropped = %v, want the old client", dropped)
	}
}

func TestSwitchInstanceReusesFleetClient(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	want := m.ctx.Fleet.Client("office")

	m.switchInstance("office")

	if m.ctx.API != want {
		t.Fatal("switch should reuse the fleet's client")
	}
	if m.ctx.Fleet.Client("home") == nil {
		t.Fatal("previous active client should stay a member")
	}
}

func TestLogoutAllEndsEachSessionOnce(t *testing.T) {
	a, ha := newHole(t, false)
	b, hb := newHole(t, false)
	for _, c := range []*pihole.Client{a, b} {
		if err := c.Login(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	logoutAll(a, b, a)

	if ha.logouts.Load() != 1 || hb.logouts.Load() != 1 {
		t.Fatalf("logouts = %d, %d; want 1, 1",
			ha.logouts.Load(), hb.logouts.Load())
	}
}

// partialFleet is a combined model whose second member fails writes.
func partialFleet(t *testing.T) *AppModel {
	t.Helper()
	m := sized(t, newTestModel(t), 120, 36)
	ok, _ := newHole(t, false)
	bad, _ := newHole(t, true)
	m.ctx.Fleet = pihole.NewFleet([]pihole.Member{
		{Name: "home", Client: ok},
		{Name: "office", Client: bad},
	}, nil)
	m.ctx.Combined = true
	return m
}

func TestToggleBlockingPartialFailureRefreshes(t *testing.T) {
	for name, cmd := range map[string]func(*AppModel) tea.Cmd{
		"toggle":  (*AppModel).toggleBlocking,
		"disable": func(m *AppModel) tea.Cmd { return m.setBlockingFor(30) },
	} {
		m := partialFleet(t)

		updated, _ := m.Update(cmd(m)())
		m = updated.(*AppModel)

		if m.block.Total != 2 || !m.block.Known {
			t.Errorf("%s: pill not refreshed: %+v", name, m.block)
		}
		if !strings.Contains(m.err, "office") {
			t.Errorf("%s: err = %q, want office failure", name, m.err)
		}
	}
}

func TestInstancesChangeRefetchesPill(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	m.block = components.BlockState{Known: true, Enabled: true}
	cfg := *m.cfg

	_, cmd := m.Update(core.InstancesChangedMsg{Config: &cfg})

	if m.block.Known {
		t.Fatal("pill should reset on an instance edit")
	}
	if cmd == nil {
		t.Fatal("expected a blocking refetch cmd")
	}
}
