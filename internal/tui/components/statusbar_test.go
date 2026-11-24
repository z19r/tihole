package components

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/z19r/tihole/internal/theme"
)

func TestStatusBarRenderShowsInstanceAndScreen(t *testing.T) {
	// Arrange
	b := StatusBar{Instance: "home-pi", ScreenName: "Dashboard", Width: 80}

	// Act
	out := b.Render(theme.DeepNight())

	// Assert
	for _, want := range []string{"home-pi", "Dashboard"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q", want)
		}
	}
}

func TestStatusBarRenderShowsDashWhenNoInstance(t *testing.T) {
	// Arrange
	b := StatusBar{Instance: "", ScreenName: "Dashboard", Width: 80}

	// Act / Assert
	if out := b.Render(theme.DeepNight()); !strings.Contains(out, "—") {
		t.Fatalf("expected em dash placeholder for empty instance, got %q", out)
	}
}

func TestStatusBarRenderNarrowWidthDoesNotPanic(t *testing.T) {
	// Arrange: width narrower than the content clamps the gap to zero.
	b := StatusBar{Instance: "home-pi", ScreenName: "Dashboard", Width: 5}

	// Act / Assert
	if out := b.Render(theme.DeepNight()); out == "" {
		t.Fatal("expected non-empty render at narrow width")
	}
}

func TestStatusBarShowsBlockingOnWhenEnabled(t *testing.T) {
	// Arrange
	b := StatusBar{
		Instance:   "home-pi",
		ScreenName: "Query Log",
		Width:      120,
		Blocking:   BlockState{Known: true, Enabled: true},
	}

	// Act
	out := b.Render(theme.DeepNight())

	// Assert
	if !strings.Contains(out, "BLOCKING ON") {
		t.Fatalf("expected on-state pill, got %q", out)
	}
}

func TestStatusBarShowsBlockingOffWhenDisabled(t *testing.T) {
	// Arrange
	b := StatusBar{
		Instance:   "home-pi",
		ScreenName: "Query Log",
		Width:      120,
		Blocking:   BlockState{Known: true, Enabled: false},
	}

	// Act
	out := b.Render(theme.DeepNight())

	// Assert
	if !strings.Contains(out, "BLOCKING OFF") {
		t.Fatalf("expected off-state pill, got %q", out)
	}
}

func TestStatusBarShowsCountdownOnTemporaryDisable(t *testing.T) {
	// Arrange: 90s remaining renders as 1:30 alongside the OFF marker.
	b := StatusBar{
		Instance:   "home-pi",
		ScreenName: "Query Log",
		Width:      120,
		Blocking:   BlockState{Known: true, Enabled: false, CountdownSecs: 90},
	}

	// Act
	out := b.Render(theme.DeepNight())

	// Assert
	if !strings.Contains(out, "1:30") {
		t.Fatalf("expected 1:30 countdown, got %q", out)
	}
}

func TestStatusBarShowsPendingBlockingBeforeFirstFetch(t *testing.T) {
	// Arrange: zero BlockState (Known == false) reads as a muted placeholder.
	b := StatusBar{Instance: "home-pi", ScreenName: "Dashboard", Width: 120}

	// Act
	out := b.Render(theme.DeepNight())

	// Assert
	if !strings.Contains(out, "blocking …") {
		t.Fatalf("expected pending placeholder, got %q", out)
	}
}

func TestFmtCountdown(t *testing.T) {
	// Arrange / Act / Assert
	cases := map[int]string{
		0:    "0:00",
		9:    "0:09",
		90:   "1:30",
		600:  "10:00",
		3661: "1:01:01",
		-5:   "0:00",
	}
	for secs, want := range cases {
		if got := fmtCountdown(secs); got != want {
			t.Errorf("fmtCountdown(%d) = %q, want %q", secs, got, want)
		}
	}
}

func TestOrDash(t *testing.T) {
	// Arrange / Act / Assert
	if got := orDash(""); got != "—" {
		t.Errorf("orDash(\"\") = %q, want em dash", got)
	}
	if got := orDash("pi"); got != "pi" {
		t.Errorf("orDash(\"pi\") = %q, want pi", got)
	}
}

func TestStatusBarCombinedPill(t *testing.T) {
	cases := []struct {
		name  string
		state BlockState
		want  string
	}{
		{"all on", BlockState{Known: true, Enabled: true, On: 2, Total: 2},
			"ON 2/2"},
		{"all off", BlockState{Known: true, Total: 2, CountdownSecs: 30},
			"OFF 0/2 0:30"},
		{"partial", BlockState{Known: true, On: 1, Total: 2},
			"PARTIAL 1/2"},
		{"single", BlockState{Known: true, Enabled: true}, "BLOCKING ON"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := StatusBar{Width: 120, Blocking: tc.state}
			out := ansi.Strip(b.blockingPill(theme.DeepNight()))
			if !strings.Contains(out, tc.want) {
				t.Fatalf("pill = %q, want %q", out, tc.want)
			}
		})
	}
}

func TestStatusBarPartialPillUsesWarn(t *testing.T) {
	th := theme.DeepNight()
	b := StatusBar{Blocking: BlockState{Known: true, On: 1, Total: 2}}
	want := lipgloss.NewStyle().Bold(true).Padding(0, 1).
		Foreground(th.Surface).Background(th.Warn).Render("◐ PARTIAL 1/2")
	if got := b.blockingPill(th); got != want {
		t.Fatalf("pill = %q, want %q", got, want)
	}
}
