package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/z19r/tihole/internal/config"
	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
)

// bootingModel builds a root model in its initial booting state (unlike the
// shared newTestModel helper, which dismisses the splash).
func bootingModel(t *testing.T) *AppModel {
	t.Helper()
	verify := true
	cfg := &config.Config{
		Active: "home",
		Theme:  "deep-night",
		Instances: []config.Instance{
			{
				Name:      "home",
				URL:       "http://192.168.1.2",
				Password:  "pw",
				VerifyTLS: &verify,
			},
		},
	}
	api := pihole.New("http://192.168.1.2", "pw")
	return New(cfg, "/tmp/tihole.yaml", api, theme.DeepNight())
}

func TestBootShowsSplashBeforeDashboard(t *testing.T) {
	// Arrange
	m := sized(t, bootingModel(t), 100, 30)

	// Act
	out := ansi.Strip(m.View().Content)

	// Assert: the splash is up, so the dashboard chrome is not yet visible.
	if !strings.Contains(out, "PiHole v6") {
		t.Errorf("expected boot splash, got:\n%s", out)
	}
	if strings.Contains(out, "Dashboard") {
		t.Errorf(
			"dashboard chrome should be hidden behind the splash:\n%s",
			out,
		)
	}
}

func TestSplashDoneRevealsApp(t *testing.T) {
	// Arrange
	m := sized(t, bootingModel(t), 100, 30)

	// Act: the splash timer elapses.
	updated, _ := m.Update(splashDoneMsg{})
	m = updated.(*AppModel)
	out := ansi.Strip(m.View().Content)

	// Assert
	if m.booting {
		t.Fatal("expected booting to clear after splashDoneMsg")
	}
	if !strings.Contains(out, "Dashboard") {
		t.Errorf("expected dashboard chrome after boot:\n%s", out)
	}
}

func TestAnyKeyDismissesSplash(t *testing.T) {
	// Arrange
	m := sized(t, bootingModel(t), 100, 30)

	// Act: press a key while booting.
	updated, _ := m.Update(keyPress("j", 'j'))
	m = updated.(*AppModel)

	// Assert: the splash is skipped and the key was consumed (no navigation).
	if m.booting {
		t.Fatal("expected any key to dismiss the splash")
	}
}
