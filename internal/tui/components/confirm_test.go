package components

import (
	"strings"
	"testing"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

func TestConfirmShowActivatesWithText(t *testing.T) {
	// Arrange
	var d ConfirmDialog

	// Act
	d = d.Show("Delete domain?", "ads.example.com", true)

	// Assert
	if !d.Active {
		t.Fatal("expected dialog to be active after Show")
	}
	if d.Title != "Delete domain?" || d.Message != "ads.example.com" ||
		!d.Danger {
		t.Fatalf("unexpected dialog state: %+v", d)
	}
}

func TestConfirmHideClearsActive(t *testing.T) {
	// Arrange
	d := ConfirmDialog{Active: true, Title: "x", Message: "y"}

	// Act
	d = d.Hide()

	// Assert
	if d.Active {
		t.Fatal("expected dialog inactive after Hide")
	}
}

func TestConfirmRenderReturnsEmptyWhenInactive(t *testing.T) {
	// Arrange
	var d ConfirmDialog

	// Act
	out := d.Render(theme.DeepNight(), 40, 10)

	// Assert
	if out != "" {
		t.Fatalf("expected empty render when inactive, got %q", out)
	}
}

func TestConfirmRenderShowsTitleMessageAndHint(t *testing.T) {
	// Arrange
	d := ConfirmDialog{}.Show("Delete domain?", "ads.example.com", true)

	// Act
	out := d.Render(theme.DeepNight(), 60, 20)

	// Assert
	for _, want := range []string{"Delete domain?", "ads.example.com", "confirm", "cancel"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q", want)
		}
	}
}
