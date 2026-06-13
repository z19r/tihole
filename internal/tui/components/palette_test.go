package components

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

func sampleCommands() []Command {
	return []Command{
		{Title: "Go to Dashboard", Desc: "summary tiles"},
		{Title: "Go to Query Log", Desc: "recent queries"},
		{Title: "Toggle blocking", Desc: "enable/disable"},
		{Title: "Theme: deep-night", Desc: "dark"},
	}
}

func TestPaletteOpensAndActivates(t *testing.T) {
	// Arrange
	p := NewPalette()

	// Act
	p, _ = p.Open(sampleCommands())

	// Assert
	if !p.Active() {
		t.Fatal("palette should be active after Open")
	}
	if len(p.filtered) != 4 {
		t.Fatalf("expected 4 commands, got %d", len(p.filtered))
	}
}

func TestPaletteFilterNarrowsByQuery(t *testing.T) {
	// Arrange / Act
	out := filterCommands(sampleCommands(), "query")

	// Assert
	if len(out) != 1 || out[0].Title != "Go to Query Log" {
		t.Fatalf("expected only Query Log, got %+v", out)
	}
}

func TestPaletteFilterMultiTerm(t *testing.T) {
	// Arrange / Act: both terms must match somewhere in title+desc.
	out := filterCommands(sampleCommands(), "theme dark")

	// Assert
	if len(out) != 1 || !strings.Contains(out[0].Title, "deep-night") {
		t.Fatalf("expected deep-night theme, got %+v", out)
	}
}

func TestPaletteEnterRunsSelectedCommand(t *testing.T) {
	// Arrange
	ran := false
	cmds := []Command{{Title: "Do it", Run: func() tea.Msg { ran = true; return nil }}}
	p := NewPalette()
	p, _ = p.Open(cmds)

	// Act
	p, runCmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Assert
	if p.Active() {
		t.Fatal("palette should close after enter")
	}
	if runCmd == nil {
		t.Fatal("expected a command from enter")
	}
	runCmd()
	if !ran {
		t.Fatal("selected command was not executed")
	}
}

func TestPaletteEscCloses(t *testing.T) {
	// Arrange
	p := NewPalette()
	p, _ = p.Open(sampleCommands())

	// Act
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	// Assert
	if p.Active() {
		t.Fatal("palette should close after esc")
	}
}

func TestPaletteDownMovesCursor(t *testing.T) {
	// Arrange
	p := NewPalette()
	p, _ = p.Open(sampleCommands())

	// Act
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	// Assert
	if p.cursor != 1 {
		t.Fatalf("expected cursor 1 after down, got %d", p.cursor)
	}
}

func TestPaletteRenderEmptyWhenInactive(t *testing.T) {
	// Arrange
	p := NewPalette()

	// Act / Assert
	if out := p.Render(theme.DeepNight(), 80, 24); out != "" {
		t.Fatalf("expected empty render when inactive, got %q", out)
	}
}

func TestPaletteRenderShowsCommands(t *testing.T) {
	// Arrange
	p := NewPalette()
	p, _ = p.Open(sampleCommands())

	// Act
	out := p.Render(theme.DeepNight(), 80, 24)

	// Assert
	for _, want := range []string{"Command palette", "Dashboard", "run", "close"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q", want)
		}
	}
}
