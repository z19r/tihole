package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/zackkitzmiller/tihole/internal/config"
	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

// newTestModel builds a root model without touching the network. The client is
// constructed but never logged in; no test here executes API commands.
func newTestModel(t *testing.T) *AppModel {
	t.Helper()
	verify := true
	cfg := &config.Config{
		Active: "home",
		Theme:  "deep-night",
		Instances: []config.Instance{
			{Name: "home", URL: "http://192.168.1.2", Password: "pw", VerifyTLS: &verify},
			{Name: "office", URL: "http://10.0.0.2", Password: "pw2", VerifyTLS: &verify},
		},
	}
	api := pihole.New("http://192.168.1.2", "pw")
	return New(cfg, "/tmp/tihole.yaml", api, theme.DeepNight())
}

// sized returns the model after an initial window-size message.
func sized(t *testing.T, m *AppModel, w, h int) *AppModel {
	t.Helper()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(*AppModel)
}

func keyPress(s string, code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: s}
}

func TestNewBuildsAllScreensWithDashboardActive(t *testing.T) {
	// Arrange / Act
	m := newTestModel(t)

	// Assert
	if got := len(m.screens); got != len(core.PageOrder()) {
		t.Fatalf("expected %d screens, got %d", len(core.PageOrder()), got)
	}
	if m.active != core.PageDashboard {
		t.Fatalf("expected Dashboard active, got %v", m.active)
	}
}

func TestViewRendersChromeAndActiveTitle(t *testing.T) {
	// Arrange
	m := sized(t, newTestModel(t), 100, 30)

	// Act: strip ANSI so the per-rune gradient wordmark matches as plain text.
	out := ansi.Strip(m.View().Content)

	// Assert
	if out == "" {
		t.Fatal("view is empty")
	}
	for _, want := range []string{"tihole", "Dashboard", "home"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestTabDescendsToPanel(t *testing.T) {
	// Arrange: fresh model starts on the rail (Nav focus).
	m := sized(t, newTestModel(t), 100, 30)

	// Act: tab descends into the active screen rather than cycling pages.
	updated, _ := m.Update(keyPress("tab", tea.KeyTab))
	m = updated.(*AppModel)

	// Assert: focus moved to the panel, the page did not change.
	if m.focus != focusPanel {
		t.Fatalf("expected panel focus after tab, got %v", m.focus)
	}
	if m.active != core.PageDashboard {
		t.Fatalf("tab must not change the page, got %v", m.active)
	}
}

func TestDigitKeyJumpsToPage(t *testing.T) {
	// Arrange
	m := sized(t, newTestModel(t), 100, 30)

	// Act: "3" -> third page (Domains), staying on the rail for fast browsing.
	updated, _ := m.Update(keyPress("3", '3'))
	m = updated.(*AppModel)

	// Assert
	if m.active != core.PageDomains {
		t.Fatalf("expected Domains after '3', got %v", m.active)
	}
	if m.focus != focusNav {
		t.Fatalf("digit jump should stay on the rail, got focus %v", m.focus)
	}
}

func TestEscClimbsFromPanelBackToNav(t *testing.T) {
	// Arrange: descend into the panel first.
	m := sized(t, newTestModel(t), 100, 30)
	updated, _ := m.Update(keyPress("tab", tea.KeyTab))
	m = updated.(*AppModel)
	if m.focus != focusPanel {
		t.Fatalf("precondition: expected panel focus, got %v", m.focus)
	}

	// Act: esc climbs back to the rail.
	updated, _ = m.Update(keyPress("esc", tea.KeyEsc))
	m = updated.(*AppModel)

	// Assert
	if m.focus != focusNav {
		t.Fatalf("expected nav focus after esc, got %v", m.focus)
	}
}

func TestArrowsCycleSidebarInNavFocus(t *testing.T) {
	// Arrange: on the rail.
	m := sized(t, newTestModel(t), 100, 30)

	// Act: down arrow moves the selection to the next page.
	updated, _ := m.Update(keyPress("down", tea.KeyDown))
	m = updated.(*AppModel)

	// Assert
	if m.active != core.PageQueryLog {
		t.Fatalf("expected QueryLog after down in nav, got %v", m.active)
	}
	if m.focus != focusNav {
		t.Fatalf("nav arrows should keep rail focus, got %v", m.focus)
	}
}

func TestBlockingResultUpdatesStatusBar(t *testing.T) {
	// Arrange
	m := sized(t, newTestModel(t), 100, 30)

	// Act
	updated, _ := m.Update(blockingResultMsg{status: pihole.BlockingStatus{Blocking: true}})
	m = updated.(*AppModel)

	// Assert
	if !m.block.Known || !m.block.Enabled {
		t.Fatalf("expected known+enabled blocking state, got %+v", m.block)
	}
	if !strings.Contains(m.View().Content, "blocking on") {
		t.Error("status bar should show 'blocking on'")
	}
}

func TestBlockingErrorSurfacesBanner(t *testing.T) {
	// Arrange
	m := sized(t, newTestModel(t), 100, 30)

	// Act
	updated, _ := m.Update(blockingErrMsg{err: errBoom{}})
	m = updated.(*AppModel)

	// Assert
	if m.err == "" {
		t.Fatal("expected error banner text to be set")
	}
	if !strings.Contains(m.View().Content, "boom") {
		t.Error("help bar should render the error banner")
	}
}

func TestQuitKeyReturnsQuitCommand(t *testing.T) {
	// Arrange
	m := sized(t, newTestModel(t), 100, 30)

	// Act
	_, cmd := m.Update(keyPress("q", 'q'))

	// Assert
	if cmd == nil {
		t.Fatal("expected a command from quit key")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("expected tea.QuitMsg from quit key")
	}
}

func TestContentSizeSubtractsChrome(t *testing.T) {
	// Arrange
	m := sized(t, newTestModel(t), 100, 30)

	// Act
	cw, ch := m.contentSize()

	// Assert
	if ch != 30-chromeTop-chromeBottom {
		t.Errorf("content height = %d, want %d", ch, 30-chromeTop-chromeBottom)
	}
	if cw <= 0 || cw >= 100 {
		t.Errorf("content width %d out of expected range", cw)
	}
}

func TestNextInstanceWraps(t *testing.T) {
	// Arrange
	m := newTestModel(t)

	// Act / Assert
	if got := m.nextInstance(); got != "office" {
		t.Fatalf("nextInstance from home = %q, want office", got)
	}
	m.ctx.InstanceName = "office"
	if got := m.nextInstance(); got != "home" {
		t.Fatalf("nextInstance from office = %q, want home", got)
	}
}

func TestCtrlKOpensPalette(t *testing.T) {
	// Arrange
	m := sized(t, newTestModel(t), 100, 30)

	// Act
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	m = updated.(*AppModel)

	// Assert
	if !m.palette.Active() {
		t.Fatal("expected palette active after ctrl+k")
	}
	if !strings.Contains(m.View().Content, "Command palette") {
		t.Error("view should render the palette overlay")
	}
}

func TestPaletteCapturesKeysWhileOpen(t *testing.T) {
	// Arrange: open the palette, then send a key that would otherwise quit.
	m := sized(t, newTestModel(t), 100, 30)
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	m = updated.(*AppModel)

	// Act: 'q' should be typed into the palette, not quit the app.
	_, cmd := m.Update(keyPress("q", 'q'))

	// Assert: no quit command was produced.
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("q should not quit while the palette is open")
		}
	}
	if !m.palette.Active() {
		t.Fatal("palette should still be open")
	}
}

func TestPaletteCommandsIncludeNavigationAndThemes(t *testing.T) {
	// Arrange
	m := newTestModel(t)

	// Act
	cmds := m.paletteCommands()

	// Assert: at least one nav command, one blocking, one theme, one instance.
	var hasNav, hasBlock, hasTheme, hasInstance bool
	for _, c := range cmds {
		switch {
		case strings.HasPrefix(c.Title, "Go to "):
			hasNav = true
		case strings.Contains(c.Title, "blocking"):
			hasBlock = true
		case strings.HasPrefix(c.Title, "Theme: "):
			hasTheme = true
		case strings.HasPrefix(c.Title, "Switch to "):
			hasInstance = true
		}
	}
	if !hasNav || !hasBlock || !hasTheme || !hasInstance {
		t.Fatalf("missing command kinds: nav=%v block=%v theme=%v instance=%v", hasNav, hasBlock, hasTheme, hasInstance)
	}
}

func TestHelpKeyOpensAndRendersOverlay(t *testing.T) {
	// Arrange
	m := sized(t, newTestModel(t), 100, 30)

	// Act: "?" toggles the help overlay on.
	updated, _ := m.Update(keyPress("?", '?'))
	m = updated.(*AppModel)

	// Assert
	if !m.showHelp {
		t.Fatal("expected showHelp true after '?'")
	}
	out := m.View().Content
	if !strings.Contains(out, "Keyboard shortcuts") || !strings.Contains(out, "Global") {
		t.Error("view should render the help overlay")
	}
}

func TestAnyKeyDismissesHelpOverlay(t *testing.T) {
	// Arrange: open help.
	m := sized(t, newTestModel(t), 100, 30)
	updated, _ := m.Update(keyPress("?", '?'))
	m = updated.(*AppModel)

	// Act: a key that would otherwise navigate just closes the overlay.
	updated, cmd := m.Update(keyPress("3", '3'))
	m = updated.(*AppModel)

	// Assert: help closed, no page change occurred.
	if m.showHelp {
		t.Fatal("expected help dismissed by any key")
	}
	if m.active != core.PageDashboard {
		t.Fatalf("dismiss key should not navigate, got %v", m.active)
	}
	_ = cmd
}

func TestHelpSectionsIncludeGlobalAndActiveScreen(t *testing.T) {
	// Arrange
	m := newTestModel(t)

	// Act
	secs := m.helpSections()

	// Assert: first section is Global; it carries at least one binding.
	if len(secs) == 0 || secs[0].Title != "Global" {
		t.Fatalf("expected Global section first, got %+v", secs)
	}
	if len(secs[0].Keys) == 0 {
		t.Fatal("Global section should have key bindings")
	}
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }
