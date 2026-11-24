package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/z19r/tihole/internal/config"
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

	f := buildFleet(cfg, "home", api)

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
	}
	for _, tc := range cases {
		if got := desiredBlocking(tc.b); got != tc.want {
			t.Errorf("%s: desired = %v, want %v", tc.name, got, tc.want)
		}
	}
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
