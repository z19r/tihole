package components

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/z19r/tihole/internal/theme"
)

func TestSectionTabsRendersEveryLabel(t *testing.T) {
	// Arrange
	tabs := SectionTabs{
		Title:  "Domains",
		Labels: []string{"allow · exact", "deny · exact", "all"},
		Active: 1,
		Width:  100,
	}

	// Act
	out := ansi.Strip(tabs.Render(theme.DeepNight()))

	// Assert: the whole point is that inactive sub-sections are visible too.
	for _, want := range []string{"Domains", "allow · exact", "deny · exact", "all"} {
		if !strings.Contains(out, want) {
			t.Errorf("tab bar missing %q\n%s", want, out)
		}
	}
}

func TestSectionTabsRightAligned(t *testing.T) {
	// Arrange
	tabs := SectionTabs{
		Title:  "Adlists",
		Labels: []string{"block", "allow"},
		Active: 0,
		Right:  "block 3 · allow 1",
		Width:  80,
	}

	// Act
	out := ansi.Strip(tabs.Render(theme.DeepNight()))

	// Assert
	if !strings.Contains(out, "block 3 · allow 1") {
		t.Errorf("expected right-aligned summary\n%s", out)
	}
	// The summary sits at the far right, past the chips.
	if idx := strings.Index(out, "block 3"); idx < strings.Index(out, "allow") {
		t.Errorf("summary should follow the chips, got:\n%s", out)
	}
}

func TestSectionTabsClampsToWidth(t *testing.T) {
	// Arrange: many chips plus a summary on a narrow line would overflow.
	tabs := SectionTabs{
		Title:  "Local DNS",
		Labels: []string{"Hosts (A/AAAA)", "CNAMEs"},
		Active: 0,
		Right:  "128 records",
		Width:  30,
	}

	// Act
	out := tabs.Render(theme.DeepNight())

	// Assert: never wider than Width (so it can't wrap and break header math).
	if w := lipgloss.Width(out); w > 30 {
		t.Errorf("rendered width = %d, want <= 30", w)
	}
	if strings.Contains(out, "\n") {
		t.Errorf("tab bar must stay a single line:\n%s", out)
	}
}

func TestSectionTabsWithoutTitle(t *testing.T) {
	// Arrange: a title-less bar (some screens supply their own heading).
	tabs := SectionTabs{
		Labels: []string{"one", "two"},
		Active: 0,
		Width:  40,
	}

	// Act
	out := ansi.Strip(tabs.Render(theme.DeepNight()))

	// Assert
	if !strings.Contains(out, "one") || !strings.Contains(out, "two") {
		t.Errorf("expected both labels without a title\n%s", out)
	}
}
