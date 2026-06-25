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

func TestPaletteUpMovesCursorAndClampsAtTop(t *testing.T) {
	// Arrange
	p := NewPalette()
	p, _ = p.Open(sampleCommands())
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	// Act: move up once (to 0), then again (should clamp at 0).
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyUp})

	// Assert
	if p.cursor != 0 {
		t.Fatalf("expected cursor clamped at 0, got %d", p.cursor)
	}
}

func TestPaletteDownClampsAtBottom(t *testing.T) {
	// Arrange: 4 commands, cursor maxes at index 3.
	p := NewPalette()
	p, _ = p.Open(sampleCommands())

	// Act: press down more times than there are items.
	for i := 0; i < 10; i++ {
		p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}

	// Assert
	if p.cursor != 3 {
		t.Fatalf("expected cursor clamped at 3, got %d", p.cursor)
	}
}

func TestPaletteUpdateInactiveIsNoop(t *testing.T) {
	// Arrange
	p := NewPalette()

	// Act
	p2, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	// Assert
	if p2.Active() || cmd != nil {
		t.Fatal("inactive palette should ignore updates")
	}
}

func TestPaletteUpdateNonKeyMsgForwardsToInput(t *testing.T) {
	// Arrange
	p := NewPalette()
	p, _ = p.Open(sampleCommands())

	// Act: a non-key message is forwarded to the text input branch.
	p, _ = p.Update(struct{ tea.Msg }{})

	// Assert: still active, cursor untouched.
	if !p.Active() || p.cursor != 0 {
		t.Fatal("non-key msg should leave palette active with cursor unchanged")
	}
}

func TestPaletteTypingFiltersAndClampsCursor(t *testing.T) {
	// Arrange: move cursor down, then type a query that matches nothing so the
	// filtered list empties and the cursor clamps to 0 via maxInt0.
	p := NewPalette()
	p, _ = p.Open(sampleCommands())
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	// Act: type "zzz" (no command matches).
	for _, r := range "zzz" {
		p, _ = p.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	// Assert
	if len(p.filtered) != 0 {
		t.Fatalf("expected no matches for zzz, got %d", len(p.filtered))
	}
	if p.cursor != 0 {
		t.Fatalf("expected cursor clamped to 0, got %d", p.cursor)
	}
}

func TestPaletteRenderShowsNoMatchingCommands(t *testing.T) {
	// Arrange: an empty filtered list renders the placeholder.
	p := NewPalette()
	p, _ = p.Open(sampleCommands())
	for _, r := range "zzz" {
		p, _ = p.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	// Act
	out := p.Render(theme.DeepNight(), 80, 24)

	// Assert
	if !strings.Contains(out, "no matching commands") {
		t.Fatalf("expected empty-state message, got %q", out)
	}
}

func TestPaletteRenderScrollsWithManyCommands(t *testing.T) {
	// Arrange: more commands than paletteMaxRows forces windowBounds scrolling.
	var cmds []Command
	for i := 0; i < 20; i++ {
		cmds = append(cmds, Command{Title: "Command " + string(rune('A'+i)), Desc: "desc"})
	}
	p := NewPalette()
	p, _ = p.Open(cmds)

	// Act: move the cursor deep into the list so the window slides.
	for i := 0; i < 15; i++ {
		p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	out := p.Render(theme.DeepNight(), 80, 30)

	// Assert: a later command is visible; the first is scrolled out of view.
	if !strings.Contains(out, "Command P") {
		t.Fatalf("expected scrolled window to show a later command, got %q", out)
	}
}

func TestWindowBounds(t *testing.T) {
	// Arrange / Act / Assert
	if s, e := windowBounds(0, 5, 8); s != 0 || e != 5 {
		t.Errorf("small list: got (%d,%d), want (0,5)", s, e)
	}
	if s, e := windowBounds(0, 20, 8); s != 0 || e != 8 {
		t.Errorf("cursor at top: got (%d,%d), want (0,8)", s, e)
	}
	if s, e := windowBounds(10, 20, 8); s != 6 || e != 14 {
		t.Errorf("cursor middle: got (%d,%d), want (6,14)", s, e)
	}
	if s, e := windowBounds(19, 20, 8); s != 12 || e != 20 {
		t.Errorf("cursor at end: got (%d,%d), want (12,20)", s, e)
	}
}

func TestMaxInt0(t *testing.T) {
	// Arrange / Act / Assert
	if got := maxInt0(3, 1); got != 3 {
		t.Errorf("maxInt0(3,1) = %d, want 3", got)
	}
	if got := maxInt0(-1, 0); got != 0 {
		t.Errorf("maxInt0(-1,0) = %d, want 0", got)
	}
}

func TestClampInt(t *testing.T) {
	// Arrange / Act / Assert
	if got := clampInt(5, 10, 70); got != 10 {
		t.Errorf("below lo: got %d, want 10", got)
	}
	if got := clampInt(100, 10, 70); got != 70 {
		t.Errorf("above hi: got %d, want 70", got)
	}
	if got := clampInt(40, 10, 70); got != 40 {
		t.Errorf("in range: got %d, want 40", got)
	}
}
