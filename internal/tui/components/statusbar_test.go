package components

import (
	"strings"
	"testing"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

func TestStatusBarRenderShowsInstanceAndScreen(t *testing.T) {
	// Arrange
	b := StatusBar{Instance: "home-pi", ScreenName: "Dashboard", Width: 80,
		Block: BlockState{Known: true, Enabled: true}}

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
	b := StatusBar{Instance: "", ScreenName: "Dashboard", Width: 80,
		Block: BlockState{Known: true, Enabled: true}}

	// Act / Assert
	if out := b.Render(theme.DeepNight()); !strings.Contains(out, "—") {
		t.Fatalf("expected em dash placeholder for empty instance, got %q", out)
	}
}

func TestStatusBarBlockingBadgeUnknown(t *testing.T) {
	// Arrange
	b := StatusBar{Instance: "x", ScreenName: "s", Width: 80,
		Block: BlockState{Known: false}}

	// Act / Assert
	if out := b.Render(theme.DeepNight()); !strings.Contains(out, "blocking …") {
		t.Fatalf("expected pending badge when unknown, got %q", out)
	}
}

func TestStatusBarBlockingBadgeEnabled(t *testing.T) {
	// Arrange
	b := StatusBar{Instance: "x", ScreenName: "s", Width: 80,
		Block: BlockState{Known: true, Enabled: true}}

	// Act / Assert
	if out := b.Render(theme.DeepNight()); !strings.Contains(out, "blocking on") {
		t.Fatalf("expected on badge, got %q", out)
	}
}

func TestStatusBarBlockingBadgeDisabled(t *testing.T) {
	// Arrange
	b := StatusBar{Instance: "x", ScreenName: "s", Width: 80,
		Block: BlockState{Known: true, Enabled: false}}

	// Act / Assert
	if out := b.Render(theme.DeepNight()); !strings.Contains(out, "blocking off") {
		t.Fatalf("expected off badge, got %q", out)
	}
}

func TestStatusBarBlockingBadgeCountdownSeconds(t *testing.T) {
	// Arrange: a sub-minute countdown formats as (Ns).
	b := StatusBar{Instance: "x", ScreenName: "s", Width: 80,
		Block: BlockState{Known: true, Enabled: false, CountdownSecs: 45}}

	// Act / Assert
	if out := b.Render(theme.DeepNight()); !strings.Contains(out, "(45s)") {
		t.Fatalf("expected (45s) countdown, got %q", out)
	}
}

func TestStatusBarBlockingBadgeCountdownMinutes(t *testing.T) {
	// Arrange: a multi-minute countdown formats as (Nm SSs).
	b := StatusBar{Instance: "x", ScreenName: "s", Width: 80,
		Block: BlockState{Known: true, Enabled: false, CountdownSecs: 125}}

	// Act / Assert
	if out := b.Render(theme.DeepNight()); !strings.Contains(out, "(2m05s)") {
		t.Fatalf("expected (2m05s) countdown, got %q", out)
	}
}

func TestStatusBarRenderNarrowWidthDoesNotPanic(t *testing.T) {
	// Arrange: width narrower than the content clamps the gap to zero.
	b := StatusBar{Instance: "home-pi", ScreenName: "Dashboard", Width: 5,
		Block: BlockState{Known: true, Enabled: true}}

	// Act / Assert
	if out := b.Render(theme.DeepNight()); out == "" {
		t.Fatal("expected non-empty render at narrow width")
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

func TestHumanCountdown(t *testing.T) {
	// Arrange / Act / Assert
	cases := map[int]string{
		-5:   "(0s)", // negative clamps to zero rather than printing "(-5s)"
		0:    "(0s)",
		59:   "(59s)",
		60:   "(1m00s)",
		125:  "(2m05s)",
		3661: "(61m01s)",
	}
	for secs, want := range cases {
		if got := humanCountdown(secs); got != want {
			t.Errorf("humanCountdown(%d) = %q, want %q", secs, got, want)
		}
	}
}
