package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

func TestSplashRendersWordmarkAndTagline(t *testing.T) {
	// Arrange
	s := Splash{Instance: "home", Status: "starting up…", Width: 80, Height: 24}

	// Act: strip ANSI so the gradient art matches as plain text.
	out := ansi.Strip(s.Render(theme.DeepNight()))

	// Assert
	for _, want := range []string{"PiHole v6", "home", "starting up…"} {
		if !strings.Contains(out, want) {
			t.Errorf("splash missing %q\n%s", want, out)
		}
	}
	// The block-font banner should contribute several full-width rows.
	if strings.Count(out, "█") == 0 {
		t.Errorf("expected block-font wordmark glyphs\n%s", out)
	}
}

func TestSplashOmitsInstanceWhenEmpty(t *testing.T) {
	// Arrange
	s := Splash{Instance: "", Status: "", Width: 60, Height: 20}

	// Act
	out := ansi.Strip(s.Render(theme.DeepNight()))

	// Assert: no stray instance marker when there's nothing to show.
	if strings.Contains(out, "▸") {
		t.Errorf("expected no instance marker for empty instance\n%s", out)
	}
}

func TestVerticalGradientAlignsRaggedRows(t *testing.T) {
	// Arrange: two rows of differing width.
	art := "████\n██"

	// Act
	out := ansi.Strip(verticalGradient(art, theme.DeepNight().Accent, theme.DeepNight().Allow))

	// Assert: the shorter row is right-padded to match, so both are equal width.
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if len([]rune(lines[0])) != len([]rune(lines[1])) {
		t.Errorf("rows not aligned: %d vs %d", len([]rune(lines[0])), len([]rune(lines[1])))
	}
}
