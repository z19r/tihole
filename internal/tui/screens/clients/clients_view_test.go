package clients

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/pihole"
)

func up() tea.KeyPressMsg   { return tea.KeyPressMsg{Code: tea.KeyUp} }
func down() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyDown} }

// sized returns a model laid out at 120x30, the standard render harness size.
func sized() *Model {
	m := newTestModel()
	m.SetSize(120, 30)
	return m
}

func TestLifecyclePrimitives(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act / Assert
	if cmd := m.Init(); cmd != nil {
		t.Fatalf("Init should return nil cmd")
	}
	if m.Title() != "Clients" {
		t.Fatalf("unexpected title: %q", m.Title())
	}
	if len(m.Help()) != 5 {
		t.Fatalf("expected 5 help bindings, got %d", len(m.Help()))
	}
}

func TestBlurCancelsAndClosesOverlays(t *testing.T) {
	// Arrange
	m := sized()
	m.Focus() // sets cancel + loading
	m.form.active = true
	m.suggest.active = true
	m.confirm = m.confirm.Show("Delete?", "x", true)

	// Act
	m.Blur()

	// Assert
	if m.focused {
		t.Fatalf("Blur should clear focused")
	}
	if m.form.active || m.suggest.active || m.confirm.Active {
		t.Fatalf("Blur should close all overlays")
	}
	if m.cancel != nil {
		t.Fatalf("Blur should clear the cancel func")
	}
}

func TestViewLoadedRendersTitleAndCount(t *testing.T) {
	// Arrange
	m := sized()
	m.clients = []pihole.ClientEntry{{Client: "10.0.0.1", Name: "one"}}
	m.syncRows()

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "Clients") {
		t.Fatalf("view should contain the title")
	}
	if !strings.Contains(out, "1 clients") {
		t.Fatalf("view should contain the count, got:\n%s", out)
	}
}

func TestViewLoadingState(t *testing.T) {
	// Arrange
	m := sized()
	m.loading = true

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "loading clients") {
		t.Fatalf("loading view should show the loading label")
	}
}

func TestViewEmptyState(t *testing.T) {
	// Arrange
	m := sized()

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "no clients configured") {
		t.Fatalf("empty view should show the empty hint")
	}
}

func TestViewErrorBanner(t *testing.T) {
	// Arrange
	m := sized()
	m.err = errors.New("boom")

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "error: boom") {
		t.Fatalf("error banner should surface the message, got:\n%s", out)
	}
}

func TestViewFormOpen(t *testing.T) {
	// Arrange
	m := sized()
	m.openAddForm()

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "Add client") {
		t.Fatalf("form view should render the add heading")
	}
	if !strings.Contains(out, "enter save") {
		t.Fatalf("form view should render the hint line")
	}
}

func TestViewEditFormHeadingAndError(t *testing.T) {
	// Arrange
	m := sized()
	m.openEditForm(pihole.ClientEntry{Client: "10.0.0.8"})
	m.form.err = errors.New("nope")

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "Edit client") {
		t.Fatalf("edit form should render the edit heading")
	}
	if !strings.Contains(out, "error: nope") {
		t.Fatalf("form error should render, got:\n%s", out)
	}
}

func TestViewSuggestionsStates(t *testing.T) {
	// Arrange
	m := sized()
	m.suggest.active = true

	// loading branch
	m.suggest.loading = true
	if out := m.View().Content; !strings.Contains(out, "loading suggestions") {
		t.Fatalf("suggest loading should render label")
	}

	// error branch
	m.suggest.loading = false
	m.suggest.err = errors.New("bad")
	if out := m.View().Content; !strings.Contains(out, "error: bad") {
		t.Fatalf("suggest error should render")
	}

	// empty branch
	m.suggest.err = nil
	if out := m.View().Content; !strings.Contains(
		out,
		"no unconfigured clients",
	) {
		t.Fatalf("suggest empty should render label")
	}

	// populated branch (exercises rowLine + suggestionMeta + formatLastSeen)
	m.suggest.suggestions = []pihole.ClientSuggestion{
		{
			Addresses: "10.0.0.5",
			Name:      "phone",
			MACVendor: "Apple",
			LastQuery: 1_700_000_000,
		},
		{Addresses: "10.0.0.6"},
	}
	m.suggest.cursor = 0
	out := m.View().Content
	if !strings.Contains(out, "10.0.0.5") {
		t.Fatalf("suggest list should render the address, got:\n%s", out)
	}
	if !strings.Contains(out, "phone") || !strings.Contains(out, "Apple") {
		t.Fatalf("suggest row meta should include name and vendor")
	}
}

func TestViewConfirmDialog(t *testing.T) {
	// Arrange
	m := sized()
	m.confirm = m.confirm.Show("Delete client?", "10.0.0.7", true)

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "Delete client?") {
		t.Fatalf("confirm dialog should render its prompt")
	}
}

func TestViewTinySizeDoesNotPanic(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.SetSize(-4, -4)

	// Act / Assert — must not panic
	_ = m.View().Content
}

func TestFormTabCyclesFields(t *testing.T) {
	// Arrange
	m := sized()
	m.openAddForm() // focusIdx 0 across 3 fields

	// Act: tab forward
	_, cmd := m.Update(press("tab"))

	// Assert
	if m.form.focusIdx != 1 {
		t.Fatalf("tab should advance focus, got %d", m.form.focusIdx)
	}
	if cmd == nil {
		t.Fatalf("tab should return a focus command")
	}

	// Act: focusPrev wraps back to 0
	m.form.focusPrev()
	if m.form.focusIdx != 0 {
		t.Fatalf("focusPrev should move focus back, got %d", m.form.focusIdx)
	}
}

func TestFormForwardsTypingToFocusedInput(t *testing.T) {
	// Arrange
	m := sized()
	m.openAddForm()

	// Act: type a character into the focused (client) field
	_, _ = m.Update(press("9"))

	// Assert
	if got := m.form.client.Value(); !strings.Contains(got, "9") {
		t.Fatalf("typed char should reach the focused input, got %q", got)
	}
}

func TestFormEscCancels(t *testing.T) {
	// Arrange
	m := sized()
	m.openAddForm()

	// Act
	_, _ = m.Update(press("esc"))

	// Assert
	if m.form.active {
		t.Fatalf("esc should close the form")
	}
}

func TestEditFormSubmitReturnsUpdateCmd(t *testing.T) {
	// Arrange
	m := sized()
	m.openEditForm(pihole.ClientEntry{Client: "10.0.0.8", Comment: "nas"})

	// Act
	_, cmd := m.Update(press("enter"))

	// Assert
	if m.form.active {
		t.Fatalf("edit submit should close the form")
	}
	if cmd == nil {
		t.Fatalf("edit submit should return an update command")
	}
}

func TestBackspaceDoesNotTriggerDelete(t *testing.T) {
	// Arrange: backspace is not a delete shortcut (only `x`/`delete` are), so
	// it must fall through to the table instead of opening the confirm dialog.
	m := sized()
	m.clients = []pihole.ClientEntry{{Client: "10.0.0.7"}}
	m.syncRows()

	// Act
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})

	// Assert
	if m.confirm.Active {
		t.Fatalf("backspace should not open the delete confirm dialog")
	}
	if m.pendingDelete != "" {
		t.Fatalf("backspace should not stage a pending delete")
	}
}

func TestDeleteKeyOpensConfirm(t *testing.T) {
	// Arrange
	m := sized()
	m.clients = []pihole.ClientEntry{{Client: "10.0.0.7"}}
	m.syncRows()

	// Act
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDelete})

	// Assert
	if !m.confirm.Active {
		t.Fatalf("the delete key should open the confirm dialog")
	}
}

func TestSuggestArrowKeysMoveCursor(t *testing.T) {
	// Arrange
	m := sized()
	m.suggest.active = true
	m.suggest.suggestions = []pihole.ClientSuggestion{
		{Addresses: "10.0.0.1"}, {Addresses: "10.0.0.2"},
	}
	m.suggest.cursor = 0

	// Act: down then up (also covers j/k aliases indirectly via named keys)
	_, _ = m.Update(down())
	if m.suggest.cursor != 1 {
		t.Fatalf("down should advance cursor, got %d", m.suggest.cursor)
	}
	_, _ = m.Update(up())
	if m.suggest.cursor != 0 {
		t.Fatalf("up should retract cursor, got %d", m.suggest.cursor)
	}
}

func TestSuggestEscCloses(t *testing.T) {
	// Arrange
	m := sized()
	m.suggest.active = true

	// Act
	_, _ = m.Update(press("esc"))

	// Assert
	if m.suggest.active {
		t.Fatalf("esc should close the suggestions pane")
	}
}

func TestMutationSuccessTriggersReload(t *testing.T) {
	// Arrange
	m := sized()
	m.epoch = 5
	before := m.epoch

	// Act
	_, cmd := m.Update(mutationMsg{epoch: 5})

	// Assert
	if cmd == nil {
		t.Fatalf("successful mutation should return a reload command")
	}
	if m.epoch != before+1 {
		t.Fatalf("successful mutation should bump epoch via reload")
	}
}

func TestMutationErrorSetsBanner(t *testing.T) {
	// Arrange
	m := sized()
	m.epoch = 5
	m.loading = true

	// Act
	_, _ = m.Update(mutationMsg{epoch: 5, err: errors.New("denied")})

	// Assert
	if m.err == nil {
		t.Fatalf("mutation error should set the banner")
	}
	if m.loading {
		t.Fatalf("mutation error should clear loading")
	}
}

func TestFetchClosuresRunWithoutNetwork(t *testing.T) {
	// Arrange: a pre-cancelled context makes each closure return immediately
	// with an error rather than performing real I/O.
	m := sized()
	m.epoch = 7
	_, cancel := context.WithCancel(context.Background())
	cancel()
	m.cancel = cancel

	// Execute each fetch/mutation command; they must not panic and must yield
	// a message of the expected type.
	if msg := m.fetch()(); msg == nil {
		t.Fatalf("fetch closure should return a message")
	}
	if msg := m.fetchSuggestions()(); msg == nil {
		t.Fatalf("fetchSuggestions closure should return a message")
	}
	if msg := m.addCmd("10.0.0.1", "c", []int{0})(); msg == nil {
		t.Fatalf("addCmd closure should return a message")
	}
	if msg := m.updateCmd("10.0.0.1", "c", nil)(); msg == nil {
		t.Fatalf("updateCmd closure should return a message")
	}
	if msg := m.deleteCmd("10.0.0.1")(); msg == nil {
		t.Fatalf("deleteCmd closure should return a message")
	}
}

func TestTruncateWidthOne(t *testing.T) {
	if got := truncate("abcdef", 1); got != ellipsis {
		t.Fatalf("width 1 should collapse to ellipsis, got %q", got)
	}
	if got := truncate("x", 0); got != "" {
		t.Fatalf("width 0 should yield empty string, got %q", got)
	}
}

func TestMaxIntPicksLarger(t *testing.T) {
	if maxInt(3, 9) != 9 || maxInt(9, 3) != 9 {
		t.Fatalf("maxInt should return the larger operand")
	}
}

func TestSuggestionMetaEmptyWhenNoFields(t *testing.T) {
	if got := suggestionMeta(pihole.ClientSuggestion{}); got != "" {
		t.Fatalf("empty suggestion should yield empty meta, got %q", got)
	}
}
