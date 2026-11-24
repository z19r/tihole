package dashboard

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/pihole"
)

func TestMaybePromptHotLimit_ShowsOnceAtFactoryDefault(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.sensors = pihole.Sensors{
		HasTemp:  true,
		Unit:     "C",
		HotLimit: hotLimitDefault,
	}

	// Act
	m.maybePromptHotLimit()

	// Assert
	if !m.hotTempPrompt.Active {
		t.Fatal(
			"expected the hot-temperature prompt to show at the factory default",
		)
	}
	if !m.hotTempPrompted {
		t.Fatal("expected hotTempPrompted to latch")
	}

	// Act again: a later poll (still at the default) shouldn't re-arm it after
	// the user hides it.
	m.hotTempPrompt = m.hotTempPrompt.Hide()
	m.maybePromptHotLimit()
	if m.hotTempPrompt.Active {
		t.Fatal(
			"expected the prompt to stay hidden once already shown this session",
		)
	}
}

func TestMaybePromptHotLimit_SkipsNonDefaultOrNonCelsius(t *testing.T) {
	cases := []pihole.Sensors{
		{HasTemp: true, Unit: "C", HotLimit: 75},
		{HasTemp: true, Unit: "F", HotLimit: hotLimitDefault},
		{HasTemp: false, Unit: "C", HotLimit: hotLimitDefault},
	}
	for _, sensors := range cases {
		m := newTestModel()
		m.sensors = sensors
		m.maybePromptHotLimit()
		if m.hotTempPrompt.Active {
			t.Errorf("did not expect a prompt for sensors %+v", sensors)
		}
	}
}

func TestCapturesInput_TracksHotTempPrompt(t *testing.T) {
	m := newTestModel()
	if m.CapturesInput() {
		t.Fatal("should not capture input before the prompt is shown")
	}
	m.sensors = pihole.Sensors{
		HasTemp:  true,
		Unit:     "C",
		HotLimit: hotLimitDefault,
	}
	m.maybePromptHotLimit()
	if !m.CapturesInput() {
		t.Fatal("should capture input while the prompt is active")
	}
}

func TestHandleHotTempKey_DeclineHidesWithoutPatching(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.sensors = pihole.Sensors{
		HasTemp:  true,
		Unit:     "C",
		HotLimit: hotLimitDefault,
	}
	m.maybePromptHotLimit()

	// Act
	_, cmd := m.handleHotTempKey(tea.KeyPressMsg{Code: 'n', Text: "n"})

	// Assert
	if m.hotTempPrompt.Active {
		t.Fatal("'n' should hide the prompt")
	}
	if cmd != nil {
		t.Fatal("declining should not issue a patch command")
	}
}

func TestHandleHotTempKey_ConfirmHidesAndReturnsPatchCmd(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.sensors = pihole.Sensors{
		HasTemp:  true,
		Unit:     "C",
		HotLimit: hotLimitDefault,
	}
	m.maybePromptHotLimit()

	// Act
	_, cmd := m.handleHotTempKey(tea.KeyPressMsg{Code: 'y', Text: "y"})

	// Assert
	if m.hotTempPrompt.Active {
		t.Fatal("'y' should hide the prompt")
	}
	if cmd == nil {
		t.Fatal("confirming should return a patch command")
	}
}

func TestUpdate_HotTempKeyTakesPrecedenceOverRefresh(t *testing.T) {
	// Arrange: focused, so 'r' would normally refresh; the active prompt must
	// intercept it instead since 'r' isn't a bound key on the dialog.
	m := newTestModel()
	m.Focus()
	m.sensors = pihole.Sensors{
		HasTemp:  true,
		Unit:     "C",
		HotLimit: hotLimitDefault,
	}
	m.maybePromptHotLimit()

	// Act
	m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	// Assert
	if !m.hotTempPrompt.Active {
		t.Fatal(
			"an unbound key should leave the prompt showing, " +
				"not fall through to refresh",
		)
	}
}

func TestUpdate_HotLimitPatchMsgDropsStaleEpoch(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 5

	// Act
	_, cmd := m.Update(hotLimitPatchMsg{epoch: 4, err: nil})

	// Assert
	if cmd != nil {
		t.Fatal("a reply from a superseded epoch should be dropped")
	}
}

func TestUpdate_HotLimitPatchMsgErrorKeptApartFromSystem(t *testing.T) {
	// Arrange: a combined dashboard whose active row mirrors fetchSystem.
	m := combinedModel()
	m.epoch = 2
	m.instances = []instanceHealth{
		{name: "primary", active: true, mirror: true},
	}

	// Act
	_, cmd := m.Update(hotLimitPatchMsg{epoch: 2, err: errors.New("boom")})

	// Assert
	if cmd != nil {
		t.Fatal("a failed patch should not trigger a refetch")
	}
	if m.errSystem != "" {
		t.Fatalf("patch error leaked into errSystem: %q", m.errSystem)
	}
	m.overlayInstances()
	if m.instances[0].err != nil {
		t.Fatalf("active row marked unreachable: %v", m.instances[0].err)
	}
	if out := m.View().Content; !strings.Contains(out, "boom") {
		t.Fatalf("patch error not shown:\n%s", out)
	}
}

func TestUpdate_HotLimitPatchMsgSuccessRefetches(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.Focus()
	m.epoch = 1
	m.inFlight = 0 // the Focus round has finished

	// Act
	_, cmd := m.Update(hotLimitPatchMsg{epoch: 1, err: nil})

	// Assert
	if cmd == nil {
		t.Fatal("a successful patch should refetch the dashboard data")
	}
}
