package components

import (
	"strings"
	"testing"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

func sampleSidebarItems() []SidebarItem {
	return []SidebarItem{
		{Title: "Dashboard", Icon: "◆"},
		{Title: "Query Log", Icon: "▤"},
		{Title: "Domains", Icon: "◉"},
	}
}

func TestSidebarRenderShowsAllItems(t *testing.T) {
	// Arrange
	s := Sidebar{Items: sampleSidebarItems(), Selected: 0, Height: 20}

	// Act
	out := s.Render(theme.DeepNight())

	// Assert
	for _, want := range []string{"tihole", "Dashboard", "Query Log", "Domains"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q", want)
		}
	}
}

func TestSidebarRenderMarksSelectedItem(t *testing.T) {
	// Arrange: selecting a middle item should style it differently than the rest.
	selected := Sidebar{Items: sampleSidebarItems(), Selected: 1, Height: 20}
	other := Sidebar{Items: sampleSidebarItems(), Selected: 0, Height: 20}

	// Act
	selOut := selected.Render(theme.DeepNight())
	othOut := other.Render(theme.DeepNight())

	// Assert: the rendered output differs when a different item is selected,
	// proving the selected item receives distinct styling.
	if selOut == othOut {
		t.Fatal("expected different render when a different item is selected")
	}
}

func TestSidebarRenderUsesDefaultWidthWhenZero(t *testing.T) {
	// Arrange: Width 0 should fall back to SidebarWidth.
	zero := Sidebar{Items: sampleSidebarItems(), Height: 10}
	explicit := Sidebar{Items: sampleSidebarItems(), Width: SidebarWidth, Height: 10}

	// Act / Assert
	if zero.Render(theme.DeepNight()) != explicit.Render(theme.DeepNight()) {
		t.Fatal("zero width should render identically to explicit SidebarWidth")
	}
}

func TestSidebarRenderSafeWithTinyHeight(t *testing.T) {
	// Arrange
	s := Sidebar{Items: sampleSidebarItems(), Selected: 2, Height: 1}

	// Act / Assert: must not panic and still produce output.
	if out := s.Render(theme.DeepNight()); out == "" {
		t.Fatal("expected non-empty render at tiny height")
	}
}

func TestSidebarRenderSafeWithNoItems(t *testing.T) {
	// Arrange
	s := Sidebar{Height: 5}

	// Act / Assert: brand still renders even with no nav items.
	if out := s.Render(theme.DeepNight()); !strings.Contains(out, "tihole") {
		t.Fatalf("expected brand in empty sidebar, got %q", out)
	}
}
