package groups

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/pihole"
)

// newViewModel builds a model sized to a realistic terminal and focused, so the
// render helpers and lifecycle hooks exercise their populated paths.
func newViewModel() *Model {
	m := newTestModel()
	m.SetSize(120, 30)
	return m
}

// --- pure helper edge cases ---

func TestTruncateEdgeCases(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{"zero width", "abc", 0, ""},
		{"negative width", "abc", -3, ""},
		{"width one cuts to ellipsis", "abc", 1, ellipsis},
		{"fits exactly", "abcd", 4, "abcd"},
		{"shorter than width", "ab", 5, "ab"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := truncate(tc.in, tc.width); got != tc.want {
				t.Fatalf(
					"truncate(%q,%d)=%q want %q",
					tc.in,
					tc.width,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestMaxIntBothBranches(t *testing.T) {
	if got := maxInt(9, 2); got != 9 {
		t.Fatalf("maxInt(9,2)=%d want 9", got)
	}
	if got := maxInt(2, 9); got != 9 {
		t.Fatalf("maxInt(2,9)=%d want 9", got)
	}
}

func TestMinIntBothBranches(t *testing.T) {
	if got := minInt(3, 8); got != 3 {
		t.Fatalf("minInt(3,8)=%d want 3", got)
	}
	if got := minInt(8, 3); got != 3 {
		t.Fatalf("minInt(8,3)=%d want 3", got)
	}
}

func TestClampMinBelowAndAbove(t *testing.T) {
	if got := clampMin(1, 6); got != 6 {
		t.Fatalf("clampMin(1,6)=%d want 6", got)
	}
	if got := clampMin(10, 6); got != 10 {
		t.Fatalf("clampMin(10,6)=%d want 10", got)
	}
}

// --- form helpers ---

func TestCanSubmitEditingAlwaysTrue(t *testing.T) {
	f := newGroupForm()
	f.openEdit("kids", "phones", true, 80)
	if !f.canSubmit() {
		t.Fatalf("edit form should always be submittable")
	}
}

func TestValuesEditingUsesOriginalName(t *testing.T) {
	f := newGroupForm()
	f.openEdit("kids", "phones", false, 80)
	// Even if the underlying input changed, the original name is authoritative.
	f.name.SetValue("mutated")
	name, comment, enabled, editing := f.values()
	if name != "kids" {
		t.Fatalf("edit values should key on original name, got %q", name)
	}
	if comment != "phones" {
		t.Fatalf("comment wrong: %q", comment)
	}
	if enabled {
		t.Fatalf("enabled should carry through as false")
	}
	if !editing {
		t.Fatalf("editing flag should be true")
	}
}

func TestFocusedOutOfRangeFallsBackToEnabled(t *testing.T) {
	f := newGroupForm()
	f.order = []int{fieldComment, fieldEnabled}
	f.cur = 99
	if f.focused() != fieldEnabled {
		t.Fatalf("out-of-range cursor should fall back to fieldEnabled")
	}
}

// --- form navigation via the Update path ---

func TestFormTabCyclesFocusAndTypes(t *testing.T) {
	m := newViewModel()
	m.form.openAdd(m.w)

	// Tab from name -> comment.
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.form.focused() != fieldComment {
		t.Fatalf("tab should move focus to comment, got %d", m.form.focused())
	}

	// Type a character into the focused comment field.
	_, _ = m.Update(keyPress("z"))
	if m.form.comment.Value() != "z" {
		t.Fatalf(
			"typing should append to focused field, got %q",
			m.form.comment.Value(),
		)
	}

	// Shift+tab wraps back to name.
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if m.form.focused() != fieldName {
		t.Fatalf(
			"shift+tab should move focus back to name, got %d",
			m.form.focused(),
		)
	}
}

func TestFormSpaceTogglesEnabledField(t *testing.T) {
	m := newViewModel()
	m.form.openAdd(m.w)
	// Move focus onto the enabled field (name -> comment -> enabled).
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.form.focused() != fieldEnabled {
		t.Fatalf("expected enabled field focused, got %d", m.form.focused())
	}
	before := m.form.enabled
	_, _ = m.Update(keyPress("space"))
	if m.form.enabled == before {
		t.Fatalf("space on enabled field should toggle it")
	}
}

// --- edit submit exercises the editing branch of values/updateCmd ---

func TestEditFormSubmitReturnsUpdateCmd(t *testing.T) {
	m := newViewModel()
	m.groups = []pihole.Group{
		{Name: "kids", Comment: "phones", Enabled: true, ID: 1},
	}
	m.syncRows()
	_, _ = m.Update(keyPress("e"))
	if !m.form.editing {
		t.Fatalf("edit form should be active")
	}
	_, cmd := m.Update(keyPress("enter"))
	if m.form.active {
		t.Fatalf("edit submit should close the form")
	}
	if cmd == nil {
		t.Fatalf("edit submit should return an update command")
	}
}

func TestEditNoSelectionIsNoop(t *testing.T) {
	m := newViewModel() // empty list
	_, cmd := m.Update(keyPress("e"))
	if m.form.active {
		t.Fatalf("edit with no selection should not open the form")
	}
	if cmd != nil {
		t.Fatalf("edit with no selection should be a no-op")
	}
}

// --- lifecycle ---

func TestInitReturnsNil(t *testing.T) {
	m := newViewModel()
	if m.Init() != nil {
		t.Fatalf("Init should return nil (work begins on Focus)")
	}
}

func TestTitle(t *testing.T) {
	if newViewModel().Title() != "Groups" {
		t.Fatalf("unexpected title")
	}
}

func TestHelpBindings(t *testing.T) {
	if got := len(newViewModel().Help()); got != 5 {
		t.Fatalf("expected 5 help bindings, got %d", got)
	}
}

func TestBlurCancelsAndClosesForm(t *testing.T) {
	m := newViewModel()
	// Focus stores a cancel func; opening the form and confirm gives Blur work.
	_ = m.Focus()
	m.form.openAdd(m.w)
	m.confirm = m.confirm.Show("t", "m", true)

	m.Blur()

	if m.focused {
		t.Fatalf("Blur should clear focused")
	}
	if m.form.active {
		t.Fatalf("Blur should close the form")
	}
	if m.confirm.Active {
		t.Fatalf("Blur should hide the confirm dialog")
	}
	if m.cancel != nil {
		t.Fatalf("Blur should clear the stored cancel")
	}
	// A second Blur with no cancel must not panic.
	m.Blur()
}

func TestDoubleFocusBumpsEpochEachTime(t *testing.T) {
	m := newViewModel()
	_ = m.Focus()
	first := m.epoch
	_ = m.Focus()
	if m.epoch != first+1 {
		t.Fatalf(
			"second Focus should bump epoch again: %d -> %d",
			first,
			m.epoch,
		)
	}
}

func TestSetSizeNegativeIsSafe(t *testing.T) {
	m := newViewModel()
	m.SetSize(-5, -5)
	_ = m.View()
}

// --- spinner tick ---

func TestSpinnerTickAdvancesWhileLoading(t *testing.T) {
	m := newViewModel()
	m.loading = true
	_, cmd := m.Update(spinner.TickMsg{})
	if cmd == nil {
		t.Fatalf("tick while loading should return the spinner's next tick")
	}
}

func TestSpinnerTickIgnoredWhenNotLoading(t *testing.T) {
	m := newViewModel()
	m.loading = false
	_, cmd := m.Update(spinner.TickMsg{})
	if cmd != nil {
		t.Fatalf("tick while idle should be a no-op")
	}
}

// --- confirm dialog empty-name guard ---

func TestConfirmEnterWithNoPendingIsNoop(t *testing.T) {
	m := newViewModel()
	m.confirm = m.confirm.Show("Delete?", "", true)
	m.pendingDelete = ""
	_, cmd := m.Update(keyPress("enter"))
	if m.confirm.Active {
		t.Fatalf("enter should hide the dialog")
	}
	if cmd != nil {
		t.Fatalf("empty pending delete should not issue a delete command")
	}
}

// --- View across states ---

func TestViewLoadedShowsHeaderAndFooter(t *testing.T) {
	m := newViewModel()
	m.groups = []pihole.Group{
		{Name: "default", Comment: "all", Enabled: true, ID: 0},
		{Name: "kids", Comment: "phones", Enabled: false, ID: 1},
	}
	m.syncRows()

	out := m.View().Content

	for _, want := range []string{"Groups", "2 groups", "1 enabled", "refresh"} {
		if !strings.Contains(out, want) {
			t.Fatalf("loaded view missing %q\n%s", want, out)
		}
	}
}

func TestViewLoadingState(t *testing.T) {
	m := newViewModel()
	m.loading = true
	out := m.View().Content
	if !strings.Contains(out, "loading groups") {
		t.Fatalf("loading view should show a loading label\n%s", out)
	}
}

func TestViewEmptyState(t *testing.T) {
	m := newViewModel()
	m.loading = false
	out := m.View().Content
	if !strings.Contains(out, "no groups configured") {
		t.Fatalf("empty view should invite adding a group\n%s", out)
	}
}

func TestViewErrorBanner(t *testing.T) {
	m := newViewModel()
	m.err = errBoom
	out := m.View().Content
	if !strings.Contains(out, "error:") || !strings.Contains(out, "boom") {
		t.Fatalf("error view should render a banner\n%s", out)
	}
}

func TestViewAddFormState(t *testing.T) {
	m := newViewModel()
	m.form.openAdd(m.w)
	out := m.View().Content
	for _, want := range []string{"Add group", "Name", "Comment", "Enabled"} {
		if !strings.Contains(out, want) {
			t.Fatalf("add-form view missing %q\n%s", want, out)
		}
	}
}

func TestViewEditFormShowsReadOnlyName(t *testing.T) {
	m := newViewModel()
	m.form.openEdit("kids", "phones", true, m.w)
	out := m.View().Content
	if !strings.Contains(out, "Edit group") {
		t.Fatalf("edit-form view should show the edit heading\n%s", out)
	}
	if !strings.Contains(out, "read-only") {
		t.Fatalf("edit-form name should render read-only\n%s", out)
	}
}

func TestViewConfirmDialogState(t *testing.T) {
	m := newViewModel()
	m.groups = []pihole.Group{{Name: "kids", ID: 1}}
	m.syncRows()
	_, _ = m.Update(keyPress("x"))
	out := m.View().Content
	if !strings.Contains(out, "Delete group") {
		t.Fatalf("confirm view should render the dialog\n%s", out)
	}
}

// --- command closures execute against a cancelled context (no network) ---

func TestCommandClosuresReturnTaggedMessages(t *testing.T) {
	m := newViewModel()
	m.epoch = 7

	t.Run("fetch", func(t *testing.T) {
		cmd := m.fetch()
		m.cancel() // cancel before running so no real request is made
		msg, ok := cmd().(groupsMsg)
		if !ok {
			t.Fatalf("fetch closure should emit a groupsMsg")
		}
		if msg.epoch != 7 || msg.err == nil {
			t.Fatalf(
				"cancelled fetch should carry epoch 7 and an error, got %+v",
				msg,
			)
		}
	})

	t.Run("add", func(t *testing.T) {
		cmd := m.addCmd("n", "c")
		m.cancel()
		msg, ok := cmd().(mutationMsg)
		if !ok || msg.epoch != 7 || msg.err == nil {
			t.Fatalf(
				"cancelled add should emit an errored mutationMsg, got %+v",
				msg,
			)
		}
	})

	t.Run("update", func(t *testing.T) {
		cmd := m.updateCmd("n", "c", true)
		m.cancel()
		msg, ok := cmd().(mutationMsg)
		if !ok || msg.epoch != 7 || msg.err == nil {
			t.Fatalf(
				"cancelled update should emit an errored mutationMsg, got %+v",
				msg,
			)
		}
	})

	t.Run("delete", func(t *testing.T) {
		cmd := m.deleteCmd("n")
		m.cancel()
		msg, ok := cmd().(mutationMsg)
		if !ok || msg.epoch != 7 || msg.err == nil {
			t.Fatalf(
				"cancelled delete should emit an errored mutationMsg, got %+v",
				msg,
			)
		}
	})
}
