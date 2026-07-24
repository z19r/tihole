package components

import (
	"strings"
	"testing"

	"github.com/zackkitzmiller/tihole/internal/theme"
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

func TestOrDash(t *testing.T) {
	// Arrange / Act / Assert
	if got := orDash(""); got != "—" {
		t.Errorf("orDash(\"\") = %q, want em dash", got)
	}
	if got := orDash("pi"); got != "pi" {
		t.Errorf("orDash(\"pi\") = %q, want pi", got)
	}
}
