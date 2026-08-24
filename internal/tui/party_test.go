package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/theme"
)

// konamiKeys returns the Konami code as the keyPress messages handleKey
// expects, matching the literal arrow-key strings (not vim aliases).
func konamiKeys() []tea.KeyPressMsg {
	return []tea.KeyPressMsg{
		keyPress("up", tea.KeyUp),
		keyPress("up", tea.KeyUp),
		keyPress("down", tea.KeyDown),
		keyPress("down", tea.KeyDown),
		keyPress("left", tea.KeyLeft),
		keyPress("right", tea.KeyRight),
		keyPress("left", tea.KeyLeft),
		keyPress("right", tea.KeyRight),
		keyPress("b", 'b'),
		keyPress("a", 'a'),
		keyPress("enter", tea.KeyEnter),
	}
}

func TestKonamiCodeTogglesPartyMode(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 40)
	original := m.ctx.Theme

	var lastHandled bool
	var lastCmd tea.Cmd
	for _, k := range konamiKeys() {
		lastCmd, lastHandled = m.handleKey(k)
	}

	if !lastHandled {
		t.Fatal("expected the final Konami key to be handled")
	}
	if !m.party {
		t.Fatal("expected party mode to be enabled after the full sequence")
	}
	if m.ctx.Theme.Name != theme.NameParty {
		t.Errorf("expected theme %q, got %q", theme.NameParty, m.ctx.Theme.Name)
	}
	if !m.ctx.Theme.Animated {
		t.Error("expected the party theme to be Animated")
	}
	if m.prevTheme != original {
		t.Error(
			"expected prevTheme to stash the theme active before party mode",
		)
	}
	if lastCmd == nil {
		t.Error(
			"expected toggling party mode on to schedule the animation ticker",
		)
	}
}

func TestKonamiCodeAgainRestoresPreviousTheme(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 40)
	original := m.ctx.Theme

	for _, k := range konamiKeys() {
		m.handleKey(k)
	}
	for _, k := range konamiKeys() {
		m.handleKey(k)
	}

	if m.party {
		t.Fatal(
			"expected party mode to be disabled after a second full sequence",
		)
	}
	if m.ctx.Theme.Name != original.Name {
		t.Errorf(
			"expected theme restored to %q, got %q",
			original.Name,
			m.ctx.Theme.Name,
		)
	}
	if m.prevTheme != nil {
		t.Error("expected prevTheme to be cleared once party mode ends")
	}
}

func TestPartialKonamiSequenceDoesNotTriggerPartyMode(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 40)

	keys := konamiKeys()
	for _, k := range keys[:len(keys)-1] {
		m.handleKey(k)
	}

	if m.party {
		t.Fatal("expected an incomplete sequence to leave party mode off")
	}
}

func TestWrongSequenceDoesNotTriggerPartyMode(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 40)

	for _, k := range []tea.KeyPressMsg{
		keyPress("down", tea.KeyDown),
		keyPress("up", tea.KeyUp),
		keyPress("left", tea.KeyLeft),
		keyPress("right", tea.KeyRight),
	} {
		m.handleKey(k)
	}

	if m.party {
		t.Fatal("expected a mismatched sequence to leave party mode off")
	}
}

func TestPartyTickMsgIsNoopWhenNotPartying(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 40)
	original := m.ctx.Theme

	updated, cmd := m.Update(partyTickMsg{})
	got := updated.(*AppModel)

	if got.ctx.Theme != original {
		t.Error("expected the theme to be untouched when party mode is off")
	}
	if cmd != nil {
		t.Error("expected no rescheduled tick when party mode is off")
	}
}

func TestPartyTickMsgAnimatesAndReschedulesWhilePartying(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 40)
	for _, k := range konamiKeys() {
		m.handleKey(k)
	}

	before := m.ctx.Theme
	updated, cmd := m.Update(partyTickMsg{})
	got := updated.(*AppModel)

	if got.ctx.Theme == before {
		t.Error("expected the tick to regenerate the theme instance")
	}
	if got.ctx.Theme.Name != theme.NameParty {
		t.Errorf(
			"expected theme to stay %q, got %q",
			theme.NameParty,
			got.ctx.Theme.Name,
		)
	}
	if cmd == nil {
		t.Error("expected the tick to reschedule itself while partying")
	}
}
