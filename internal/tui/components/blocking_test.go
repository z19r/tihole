package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

func TestBlockingPillUnknown(t *testing.T) {
	// Arrange
	out := ansi.Strip(blockingPill(theme.DeepNight(), BlockState{Known: false}, 20))

	// Act / Assert
	if !strings.Contains(out, "blocking …") {
		t.Fatalf("expected pending status, got %q", out)
	}
}

func TestBlockingPillEnabledShowsToggleHint(t *testing.T) {
	// Arrange
	out := ansi.Strip(blockingPill(theme.DeepNight(), BlockState{Known: true, Enabled: true}, 20))

	// Act / Assert: state and the affordance both live on the rail now.
	for _, want := range []string{"blocking on", "d toggle"} {
		if !strings.Contains(out, want) {
			t.Errorf("pill missing %q\n%s", want, out)
		}
	}
}

func TestBlockingPillDisabled(t *testing.T) {
	// Arrange / Act / Assert
	out := ansi.Strip(blockingPill(theme.DeepNight(), BlockState{Known: true, Enabled: false}, 20))
	if !strings.Contains(out, "blocking off") {
		t.Fatalf("expected off status, got %q", out)
	}
}

func TestBlockingPillCountdown(t *testing.T) {
	// Arrange / Act / Assert
	out := ansi.Strip(blockingPill(theme.DeepNight(), BlockState{Known: true, Enabled: false, CountdownSecs: 45}, 20))
	if !strings.Contains(out, "(45s)") {
		t.Fatalf("expected (45s) countdown, got %q", out)
	}
}

func TestBlockingPillNarrowWidthDoesNotPanic(t *testing.T) {
	// Arrange / Act / Assert: an absurdly narrow rail must not panic.
	if out := blockingPill(theme.DeepNight(), BlockState{Known: true, Enabled: true}, 1); out == "" {
		t.Fatal("expected non-empty render at narrow width")
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
