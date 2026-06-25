package adlists

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zackkitzmiller/tihole/internal/pihole"
)

// --- lifecycle --------------------------------------------------------------

func TestInitReturnsNilUntilFocus(t *testing.T) {
	if cmd := newTestModel().Init(); cmd != nil {
		t.Fatalf("Init should return nil until focus")
	}
}

func TestTitleIsAdlists(t *testing.T) {
	if got := newTestModel().Title(); got != "Adlists" {
		t.Fatalf("unexpected title: %q", got)
	}
}

func TestHelpReturnsBindings(t *testing.T) {
	if got := len(newTestModel().Help()); got != 7 {
		t.Fatalf("expected 7 help bindings, got %d", got)
	}
}

func TestBlurClearsFocusAndCancelsInflight(t *testing.T) {
	// Arrange: Focus arms a fetch cancel; a running gravity arms its own cancel.
	m := newTestModel()
	m.Focus()
	_, gravityCancel := context.WithCancel(context.Background())
	m.gravityCancel = gravityCancel
	m.form = m.form.openAdd(pihole.ListBlock)
	if m.cancel == nil {
		t.Fatalf("precondition: Focus should arm an in-flight cancel")
	}

	// Act
	m.Blur()

	// Assert
	if m.focused {
		t.Fatalf("Blur should clear focus")
	}
	if m.form.active {
		t.Fatalf("Blur should close the form")
	}
	if m.cancel != nil {
		t.Fatalf("Blur should cancel and clear the in-flight request")
	}
	if m.gravityCancel != nil {
		t.Fatalf("Blur should cancel and clear a running gravity stream")
	}
}

func TestSelectedOutOfRangeReturnsFalse(t *testing.T) {
	m := newTestModel()
	// no rows for the visible type
	if _, ok := m.selected(); ok {
		t.Fatalf("selected should be false when there are no rows")
	}
}

func TestMinInt(t *testing.T) {
	if minInt(3, 7) != 3 || minInt(9, 2) != 2 {
		t.Fatalf("minInt wrong")
	}
}

// --- view: states -----------------------------------------------------------

func TestViewLoadedRendersHeaderChipAndFooter(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.Focus()
	m.lists[pihole.ListBlock] = []pihole.List{{Address: "https://a", Type: "block", Enabled: true, Status: 1, Number: 5}}
	m.loading = false
	m.syncRows()

	// Act
	out := m.View().Content

	// Assert
	for _, want := range []string{"Adlists", "block", "allow", "a add", "gravity"} {
		if !strings.Contains(out, want) {
			t.Fatalf("loaded view missing %q:\n%s", want, out)
		}
	}
}

func TestViewLoadingStateShowsSpinnerText(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.loading = true

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "loading lists") {
		t.Fatalf("empty+loading should render a loading indicator:\n%s", out)
	}
}

func TestViewEmptyStateShowsNoLists(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.loading = false

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "no block lists configured") {
		t.Fatalf("empty+idle should render the no-lists message:\n%s", out)
	}
}

func TestViewErrorBannerRendered(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.err = errors.New("connection refused")

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "error:") {
		t.Fatalf("a set error should render the banner:\n%s", out)
	}
}

func TestViewAddFormRendered(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.form = m.form.openAdd(pihole.ListBlock)
	m.form.setWidth(m.w)

	// Act
	out := m.View().Content

	// Assert
	for _, want := range []string{"Add list", "Address", "Comment", "Groups", "Type", "tab next"} {
		if !strings.Contains(out, want) {
			t.Fatalf("add form missing %q:\n%s", want, out)
		}
	}
}

func TestViewEditFormRendersEnabledRow(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.form = m.form.openEdit(pihole.List{Address: "https://a", Comment: "c", Type: "block", Enabled: true, Groups: []int{0}})
	m.form.setWidth(m.w)

	// Act
	out := m.View().Content

	// Assert
	for _, want := range []string{"Edit list", "Enabled"} {
		if !strings.Contains(out, want) {
			t.Fatalf("edit form missing %q:\n%s", want, out)
		}
	}
}

func TestViewConfirmDialogRendered(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.lists[pihole.ListBlock] = []pihole.List{{Address: "https://gone", Type: "block"}}
	m.syncRows()
	_, _ = m.Update(keyPress("x"))

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "y confirm") {
		t.Fatalf("confirm footer hint missing:\n%s", out)
	}
}

func TestViewGravityRunningPaneRendered(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.gravityOpen = true
	m.gravityRunning = true
	m.viewport.SetContent("[i] Building tree")

	// Act
	out := m.View().Content

	// Assert
	for _, want := range []string{"Gravity update", "streaming gravity update"} {
		if !strings.Contains(out, want) {
			t.Fatalf("running gravity pane missing %q:\n%s", want, out)
		}
	}
}

func TestViewGravityDonePaneRendered(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.gravityOpen = true
	m.gravityRunning = false
	m.gravityDone = true

	// Act
	out := m.View().Content

	// Assert
	for _, want := range []string{"Gravity update", "esc close"} {
		if !strings.Contains(out, want) {
			t.Fatalf("finished gravity pane missing %q:\n%s", want, out)
		}
	}
}

func TestViewGravityErrorPaneRendered(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.gravityOpen = true
	m.gravityRunning = false
	m.gravityDone = true
	m.gravityErr = errors.New("boom")

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "gravity failed") {
		t.Fatalf("errored gravity pane should show the failure:\n%s", out)
	}
}

// --- form key handling ------------------------------------------------------

func TestFormTabAndShiftTabAdvanceFocus(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.form = m.form.openAdd(pihole.ListBlock)
	first := m.form.focus

	// Act: forward then back returns to start.
	_, _ = m.Update(keyPress("tab"))
	if m.form.focus == first {
		t.Fatalf("tab should advance focus")
	}
	_, _ = m.Update(keyPress("shift+tab"))

	// Assert
	if m.form.focus != first {
		t.Fatalf("shift+tab should return focus to start, got %d", m.form.focus)
	}
}

func TestFormLeftRightTogglesTypeField(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.form = m.form.openAdd(pihole.ListBlock)
	for m.form.current() != fieldType {
		m.form = m.form.advance(1)
	}

	// Act
	_, _ = m.Update(keyPress("right"))

	// Assert
	if m.form.listType != pihole.ListAllow {
		t.Fatalf("left/right on the type field should toggle it, got %q", m.form.listType)
	}
}

func TestFormTypingEditsFocusedInput(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.form = m.form.openAdd(pihole.ListBlock) // focus starts on address
	if m.form.current() != fieldAddress {
		t.Fatalf("precondition: add form should focus address first")
	}

	// Act
	_, _ = m.Update(keyPress("z"))

	// Assert
	if !strings.Contains(m.form.address.Value(), "z") {
		t.Fatalf("typing should edit the focused input, got %q", m.form.address.Value())
	}
}

func TestFormCurrentDefaultsToTypeWhenOutOfRange(t *testing.T) {
	f := newForm()
	f.fields = nil
	if f.current() != fieldType {
		t.Fatalf("current with no fields should default to fieldType")
	}
}

// --- command closure bodies (no successful network) -------------------------

func TestSubmitFormEditClosureReturnsMutatedMsg(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.form = m.form.openEdit(pihole.List{Address: "https://a", Type: "block", Groups: []int{0}, Enabled: true})
	cmd := m.submitForm()
	if cmd == nil {
		t.Fatalf("edit submit should return an UpdateList cmd")
	}

	// Act
	msg := cmd()

	// Assert
	mm, ok := msg.(mutatedMsg)
	if !ok {
		t.Fatalf("edit submit closure should yield a mutatedMsg, got %T", msg)
	}
	if mm.err == nil {
		t.Fatalf("offline UpdateList should surface an error")
	}
}

func TestSubmitFormAddClosureReturnsMutatedMsg(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.form = m.form.openAdd(pihole.ListBlock)
	m.form.address.SetValue("https://example.com/list.txt")
	cmd := m.submitForm()

	// Act
	msg := cmd()

	// Assert
	if _, ok := msg.(mutatedMsg); !ok {
		t.Fatalf("add submit closure should yield a mutatedMsg, got %T", msg)
	}
}

func TestToggleEnabledNoSelectionReturnsNil(t *testing.T) {
	m := newTestModel()
	if cmd := m.toggleEnabled(); cmd != nil {
		t.Fatalf("toggleEnabled with no selection should return nil")
	}
}

func TestToggleEnabledClosureReturnsMutatedMsg(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.lists[pihole.ListBlock] = []pihole.List{{Address: "https://a", Type: "block", Enabled: true}}
	m.syncRows()
	cmd := m.toggleEnabled()

	// Act
	msg := cmd()

	// Assert
	if _, ok := msg.(mutatedMsg); !ok {
		t.Fatalf("toggleEnabled closure should yield a mutatedMsg, got %T", msg)
	}
}

func TestDeleteSelectedClosureReturnsMutatedMsg(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.pendingAddr = "https://gone"
	m.pendingType = pihole.ListBlock
	cmd := m.deleteSelected()

	// Act
	msg := cmd()

	// Assert
	if _, ok := msg.(mutatedMsg); !ok {
		t.Fatalf("deleteSelected closure should yield a mutatedMsg, got %T", msg)
	}
}

func TestFetchClosureBodyRunsAgainstCancelledContext(t *testing.T) {
	// Arrange
	m := newTestModel()
	cmd := m.fetch()
	m.cancel() // cancel the context the closure captured

	// Act
	msg := cmd()

	// Assert
	lm, ok := msg.(listsMsg)
	if !ok {
		t.Fatalf("fetch closure should yield a listsMsg, got %T", msg)
	}
	if lm.err == nil {
		t.Fatalf("cancelled fetch should surface an error")
	}
}

// --- gravity command closures ----------------------------------------------

func TestRunGravityClosureAgainstCancelledContext(t *testing.T) {
	// Arrange
	api := pihole.New("http://x", "pw")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ch := make(chan string, gravityChanBuffer)
	cmd := runGravity(ctx, api, ch, 7)

	// Act
	msg := cmd()

	// Assert
	dm, ok := msg.(gravityDoneMsg)
	if !ok {
		t.Fatalf("runGravity closure should yield a gravityDoneMsg, got %T", msg)
	}
	if dm.epoch != 7 {
		t.Fatalf("runGravity should carry its epoch, got %d", dm.epoch)
	}
	if dm.err == nil {
		t.Fatalf("cancelled gravity should surface an error")
	}
	if _, open := <-ch; open {
		t.Fatalf("runGravity should close the channel when the stream ends")
	}
}

func TestReadGravityLineReceivesLine(t *testing.T) {
	// Arrange
	ch := make(chan string, 1)
	ch <- "[i] progress"
	cmd := readGravityLine(ch, 3)

	// Act
	msg := cmd().(gravityLineMsg)

	// Assert
	if !msg.ok || msg.line != "[i] progress" || msg.epoch != 3 {
		t.Fatalf("readGravityLine should report the received line: %#v", msg)
	}
}

func TestReadGravityLineClosedChannel(t *testing.T) {
	// Arrange
	ch := make(chan string)
	close(ch)
	cmd := readGravityLine(ch, 3)

	// Act
	msg := cmd().(gravityLineMsg)

	// Assert
	if msg.ok {
		t.Fatalf("readGravityLine on a closed channel should report ok=false")
	}
}
