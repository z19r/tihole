package components

import (
	"strings"
	"testing"

	"github.com/z19r/tihole/internal/theme"
)

func sampleSections() []HelpSection {
	return []HelpSection{
		{Title: "Global", Keys: []HelpKey{
			{Key: "tab", Desc: "next screen"},
			{Key: "ctrl+k", Desc: "command palette"},
			{Key: "q", Desc: "quit"},
		}},
		{Title: "Domains", Keys: []HelpKey{
			{Key: "a", Desc: "add"},
			{Key: "x", Desc: "delete"},
		}},
	}
}

func TestHelpSheetRenderEmptyWhenNoSections(t *testing.T) {
	// Arrange
	h := HelpSheet{}

	// Act / Assert
	if out := h.Render(theme.DeepNight(), 80, 24); out != "" {
		t.Fatalf("expected empty render with no sections, got %q", out)
	}
}

func TestHelpSheetRenderShowsSectionsAndKeys(t *testing.T) {
	// Arrange
	h := HelpSheet{Sections: sampleSections()}

	// Act
	out := h.Render(theme.DeepNight(), 80, 30)

	// Assert
	for _, want := range []string{"Keyboard shortcuts", "Global", "Domains", "ctrl+k", "command palette", "delete", "close"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q", want)
		}
	}
}

func TestHelpKeyColumnWidthPicksWidest(t *testing.T) {
	// Arrange
	keys := []HelpKey{{Key: "a"}, {Key: "shift+tab"}, {Key: "x"}}

	// Act
	w := helpKeyColumnWidth(keys)

	// Assert: "shift+tab" is 9 wide, above the 6 minimum.
	if w != 9 {
		t.Fatalf("expected column width 9, got %d", w)
	}
}

func TestHelpKeyColumnWidthHonorsMinimum(t *testing.T) {
	// Arrange
	keys := []HelpKey{{Key: "a"}, {Key: "x"}}

	// Act / Assert: short keys still reserve the 6-wide minimum.
	if w := helpKeyColumnWidth(keys); w != 6 {
		t.Fatalf("expected minimum width 6, got %d", w)
	}
}

func TestHelpSheetRenderSafeAtTinySize(t *testing.T) {
	// Arrange
	h := HelpSheet{Sections: sampleSections()}

	// Act / Assert: must not panic and must still produce output.
	if out := h.Render(theme.DeepNight(), 10, 4); out == "" {
		t.Fatal("expected non-empty render even at tiny size")
	}
}
