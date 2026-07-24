package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

// captureStub is a minimal core.Screen that also implements core.InputCapturer.
// It records the last key string it received so tests can assert delegation.
type captureStub struct {
	captures bool
	lastKey  string
}

func (s *captureStub) Init() tea.Cmd { return nil }

func (s *captureStub) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		s.lastKey = k.String()
	}
	return s, nil
}

func (s *captureStub) View() tea.View      { return tea.NewView("") }
func (s *captureStub) Title() string       { return "Stub" }
func (s *captureStub) Focus() tea.Cmd      { return nil }
func (s *captureStub) Blur()               {}
func (s *captureStub) Help() []key.Binding { return nil }
func (s *captureStub) SetSize(_, _ int)    {}
func (s *captureStub) CapturesInput() bool { return s.captures }

// installStub swaps the active screen for a capture stub and returns it.
func installStub(m *AppModel, captures bool) *captureStub {
	stub := &captureStub{captures: captures}
	m.screens[m.active] = stub
	return stub
}

func TestCapturingScreenReceivesGlobalShortcutKeys(t *testing.T) {
	// Arrange: the active screen is capturing text input.
	m := sized(t, newTestModel(t), 120, 36)
	stub := installStub(m, true)
	activeBefore := m.active

	// Act: send 's' — normally the switch-instance shortcut.
	updated, _ := m.Update(keyPress("s", 's'))
	m = updated.(*AppModel)

	// Assert: the key was delegated to the screen, not consumed as a shortcut.
	if stub.lastKey != "s" {
		t.Fatalf("capturing screen should receive 's', got %q", stub.lastKey)
	}
	if m.ctx.InstanceName != "home" {
		t.Fatalf("switch-instance must not fire while capturing, instance=%q", m.ctx.InstanceName)
	}
	if m.active != activeBefore {
		t.Fatalf("navigation must not fire while capturing, active=%v", m.active)
	}
}

func TestCapturingScreenDoesNotFireQuitOrNav(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	stub := installStub(m, true)

	// 'q' would normally quit; a digit would normally jump pages.
	for _, tc := range []struct {
		str string
		r   rune
	}{
		{"q", 'q'},
		{"2", '2'},
		{"j", 'j'},
	} {
		stub.lastKey = ""
		updated, cmd := m.Update(keyPress(tc.str, tc.r))
		m = updated.(*AppModel)
		if cmd != nil {
			t.Fatalf("key %q must not produce a global command while capturing", tc.str)
		}
		if stub.lastKey != tc.str {
			t.Fatalf("key %q should reach the screen, got %q", tc.str, stub.lastKey)
		}
	}
	if m.active != core.PageDashboard {
		t.Fatalf("no navigation should occur while capturing, active=%v", m.active)
	}
}

func TestCapturingScreenStillQuitsOnCtrlC(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	installStub(m, true)

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("ctrl+c must still quit even while a screen captures input")
	}
}

func TestNonCapturingScreenFiresGlobalShortcut(t *testing.T) {
	// Control case: a screen that does NOT capture input lets shortcuts fire.
	m := sized(t, newTestModel(t), 120, 36)
	installStub(m, false)

	updated, _ := m.Update(keyPress("2", '2'))
	m = updated.(*AppModel)

	order := core.PageOrder()
	if m.active != order[1].ID {
		t.Fatalf("digit shortcut should navigate when not capturing, active=%v", m.active)
	}
}
