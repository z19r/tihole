package settings

import (
	"testing"

	"github.com/z19r/tihole/internal/config"
	"github.com/z19r/tihole/internal/tui/core"
)

// addSecond starts from a one-instance config and submits a second instance.
func addSecond(t *testing.T) *Model {
	t.Helper()
	m := newTestModel(t)
	m.ctx.Config.Instances = m.ctx.Config.Instances[:1]
	_, _ = m.Update(keyPress("c"))
	_, _ = m.Update(keyPress("a"))
	m.connForm.name.SetValue("backup")
	m.connForm.url.SetValue("http://pi2.hole")
	m.connForm.password.SetValue("pw2")
	_, _ = m.Update(keyPress("enter"))
	return m
}

func TestAddingSecondInstanceAsksAboutCombined(t *testing.T) {
	m := addSecond(t)

	if !m.confirm.Active || !m.askCombined {
		t.Fatalf("adding the 2nd instance should ask about combined view")
	}
}

func TestCombinedPromptYesSavesAndEmits(t *testing.T) {
	m := addSecond(t)

	_, cmd := m.Update(keyPress("y"))

	if m.confirm.Active || m.askCombined {
		t.Fatalf("answering should close the prompt")
	}
	if !m.ctx.Config.Combined {
		t.Fatalf("yes should turn combined on")
	}
	saved, err := config.Load(m.ctx.ConfigPath)
	if err != nil || !saved.Combined {
		t.Fatalf("yes should persist combined: %+v %v", saved, err)
	}
	changed, ok := cmd().(core.InstancesChangedMsg)
	if !ok || !changed.Config.Combined {
		t.Fatalf("yes should emit the combined config, got %T", cmd())
	}
}

func TestCombinedPromptNoLeavesItOff(t *testing.T) {
	m := addSecond(t)

	_, cmd := m.Update(keyPress("n"))

	if m.confirm.Active || m.ctx.Config.Combined || cmd != nil {
		t.Fatalf("no should close the prompt and change nothing")
	}
}

func TestAddingThirdInstanceDoesNotAsk(t *testing.T) {
	m := newTestModel(t) // starts with two instances
	_, _ = m.Update(keyPress("c"))
	_, _ = m.Update(keyPress("a"))
	m.connForm.name.SetValue("tertiary")
	m.connForm.url.SetValue("http://pi3.hole")
	m.connForm.password.SetValue("pw3")
	_, _ = m.Update(keyPress("enter"))

	if m.confirm.Active {
		t.Fatalf("only the 2nd instance should trigger the prompt")
	}
}

func TestCombinedPromptCapturesInput(t *testing.T) {
	m := addSecond(t)

	if !m.CapturesInput() {
		t.Fatalf("an open prompt should keep global keys like m away")
	}
}
