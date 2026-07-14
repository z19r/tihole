package domains

import (
	"errors"
	"strings"
	"testing"

	"github.com/zackkitzmiller/tihole/internal/pihole"
)

// --- lifecycle & trivial accessors -----------------------------------------

func TestInitReturnsNil(t *testing.T) {
	if cmd := newTestModel().Init(); cmd != nil {
		t.Fatalf("Init should return nil until focus")
	}
}

func TestTitleIsDomains(t *testing.T) {
	if got := newTestModel().Title(); got != "Domains" {
		t.Fatalf("unexpected title: %q", got)
	}
}

func TestHelpReturnsBindings(t *testing.T) {
	if got := len(newTestModel().Help()); got != 6 {
		t.Fatalf("expected 6 help bindings, got %d", got)
	}
}

func TestBlurClearsStateAndCancelsInflight(t *testing.T) {
	// Arrange: Focus arms an in-flight cancel; open a form + confirm too.
	m := newTestModel()
	m.Focus()
	if m.cancel == nil {
		t.Fatalf("precondition: Focus should arm an in-flight cancel")
	}
	_, _ = m.Update(keyPress("a"))
	m.confirm = m.confirm.Show("t", "d", true)
	dc := sampleDomains()[0]
	m.pending = &dc

	// Act
	m.Blur()

	// Assert
	if m.focused {
		t.Fatalf("Blur should clear focus")
	}
	if m.form != nil {
		t.Fatalf("Blur should drop any open form")
	}
	if m.confirm.Active {
		t.Fatalf("Blur should hide the confirm dialog")
	}
	if m.pending != nil {
		t.Fatalf("Blur should clear the pending target")
	}
	if m.cancel != nil {
		t.Fatalf("Blur should cancel and clear the in-flight request")
	}
}

func TestDoubleFocusIsSafe(t *testing.T) {
	// Arrange: a first Focus arms a cancel; a second must cancel and re-arm.
	m := newTestModel()
	m.Focus()
	first := m.epoch

	// Act
	cmd := m.Focus()

	// Assert
	if m.epoch != first+1 {
		t.Fatalf("second Focus should bump epoch again")
	}
	if m.cancel == nil || cmd == nil {
		t.Fatalf("second Focus should re-arm cancel and return a command")
	}
}

func TestSetSizeNegativeAndReflowWithData(t *testing.T) {
	// Arrange: populate + open a form so SetSize touches the form + rows
	// branches.
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})
	_, _ = m.Update(keyPress("a"))

	// Act / Assert: degenerate sizes must not panic and must reflow.
	m.SetSize(-5, -5)
	m.SetSize(0, 0)
	m.SetSize(120, 30)
	if len(m.visible) == 0 {
		t.Fatalf("data should survive reflow")
	}
	_ = m.View()
}

// --- Update edge cases ------------------------------------------------------

func TestMutationErrorSetsBannerNoRefetch(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 7
	m.loading = true

	// Act
	_, cmd := m.Update(mutationMsg{epoch: 7, err: errors.New("boom")})

	// Assert
	if m.err == nil {
		t.Fatalf("mutation error should set the banner")
	}
	if m.loading {
		t.Fatalf("mutation error should clear loading")
	}
	if m.epoch != 7 {
		t.Fatalf("mutation error should not bump the epoch")
	}
	if cmd != nil {
		t.Fatalf("mutation error should not return a refetch command")
	}
}

func TestUpdateIgnoresStaleMutation(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 9

	// Act
	_, cmd := m.Update(mutationMsg{epoch: 8, err: nil})

	// Assert
	if m.epoch != 9 {
		t.Fatalf("stale mutation should not bump the epoch")
	}
	if cmd != nil {
		t.Fatalf("stale mutation should not trigger a refetch")
	}
}

func TestSpinnerTickIgnoredWhenIdle(t *testing.T) {
	m := newTestModel()
	msg := m.spinner.Tick()
	m.loading = false
	if _, cmd := m.Update(msg); cmd != nil {
		t.Fatalf("spinner tick should be ignored when not loading")
	}
	m.loading = true
	if _, cmd := m.Update(msg); cmd == nil {
		t.Fatalf("spinner tick should reschedule while loading")
	}
}

func TestUnhandledMessageIsNoop(t *testing.T) {
	m := newTestModel()
	if _, cmd := m.Update(struct{}{}); cmd != nil {
		t.Fatalf("unknown message should be a no-op")
	}
}

func TestRefreshKeyBumpsEpochAndLoads(t *testing.T) {
	// Arrange
	m := newTestModel()
	before := m.epoch

	// Act
	_, cmd := m.Update(keyPress("r"))

	// Assert
	if m.epoch != before+1 || !m.loading || cmd == nil {
		t.Fatalf("'r' should bump epoch, set loading, and refetch")
	}
}

// --- filter tab toggling across all combos ----------------------------------

func TestFilterCyclesAllCombosAndRefiltersVisible(t *testing.T) {
	// Arrange: one domain of each concrete type×kind, plus a stray.
	ds := []pihole.Domain{
		{Domain: "a", Type: "allow", Kind: "exact"},
		{Domain: "b", Type: "deny", Kind: "exact"},
		{Domain: "c", Type: "allow", Kind: "regex"},
		{Domain: "d", Type: "deny", Kind: "regex"},
	}
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: ds})

	// starts at filterAll -> 4 visible
	want := []struct {
		tab     filterTab
		visible int
	}{
		{filterAllowExact, 1},
		{filterDenyExact, 1},
		{filterAllowRegex, 1},
		{filterDenyRegex, 1},
		{filterAll, 4},
	}
	for i, w := range want {
		_, _ = m.Update(keyPress("f"))
		if m.filter != w.tab {
			t.Fatalf("step %d: filter=%v want %v", i, m.filter, w.tab)
		}
		if len(m.visible) != w.visible {
			t.Fatalf(
				"step %d (%v): visible=%d want %d",
				i,
				w.tab,
				len(m.visible),
				w.visible,
			)
		}
	}
}

func TestLeftKeyCyclesFilterBackward(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})

	// Act: all -> (prev) deny-regex
	_, _ = m.Update(keyPress("left"))

	// Assert
	if m.filter != filterDenyRegex {
		t.Fatalf("left should move to the previous tab, got %v", m.filter)
	}
}

func TestFilterPrevWraps(t *testing.T) {
	if got := filterAllowExact.prev(); got != filterAll {
		t.Fatalf("prev of first tab should wrap to filterAll, got %v", got)
	}
}

func TestFilterLabelFallback(t *testing.T) {
	if got := filterTab(99).label(); got != "all" {
		t.Fatalf("unknown tab label should fall back to 'all', got %q", got)
	}
}

func TestFilterTypeKindDefault(t *testing.T) {
	dt, dk := filterAll.typeKind()
	if dt != pihole.DomainAllow || dk != pihole.KindExact {
		t.Fatalf("filterAll typeKind default wrong: %v %v", dt, dk)
	}
}

// --- table key routing ------------------------------------------------------

func TestEditKeyOpensPrefilledForm(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})

	// Act
	_, cmd := m.Update(keyPress("e"))

	// Assert
	if m.form == nil || !m.form.editing {
		t.Fatalf("'e' should open the edit form in editing mode")
	}
	if m.form.domain.Value() == "" {
		t.Fatalf("edit form should be pre-filled with the row domain")
	}
	_ = cmd
}

func TestEditKeyNoopWithEmptyTable(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(keyPress("e"))
	if m.form != nil {
		t.Fatalf("'e' with no rows should not open a form")
	}
	if cmd != nil {
		t.Fatalf("'e' with no rows should not return a command")
	}
}

func TestToggleNoopWithEmptyTable(t *testing.T) {
	m := newTestModel()
	if _, cmd := m.Update(keyPress("t")); cmd != nil {
		t.Fatalf("toggle with no selection should be a no-op")
	}
}

func TestNavigationKeyRoutesToTable(t *testing.T) {
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})
	_, _ = m.Update(keyPress("j"))
	_, _ = m.Update(keyPress("k"))
}

func TestConfirmEnterDeletesSelected(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})
	_, _ = m.Update(keyPress("x"))

	// Act
	_, cmd := m.Update(keyPress("enter"))

	// Assert
	if m.confirm.Active || m.pending != nil {
		t.Fatalf("enter should confirm and clear the dialog")
	}
	if cmd == nil {
		t.Fatalf("enter should return a delete command")
	}
}

func TestConfirmEscCancels(t *testing.T) {
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})
	_, _ = m.Update(keyPress("x"))
	if _, cmd := m.Update(keyPress("esc")); cmd != nil {
		t.Fatalf("esc on confirm should not return a command")
	}
	if m.confirm.Active {
		t.Fatalf("esc should hide the confirm dialog")
	}
}

// --- form field cycling & submit --------------------------------------------

func TestFormFieldCyclingForwardAndBack(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(keyPress("a")) // focus starts at fieldDomain
	start := m.form.focus

	// Act: tab forward through all fields, wrapping.
	for i := 0; i < int(fieldCount); i++ {
		_, _ = m.Update(keyPress("tab"))
	}
	if m.form.focus != start {
		t.Fatalf(
			"cycling %d tabs should wrap to start, got %v",
			int(fieldCount),
			m.form.focus,
		)
	}

	// Act: shift+tab back one.
	_, _ = m.Update(keyPress("shift+tab"))
	if m.form.focus != fieldKind {
		t.Fatalf(
			"shift+tab from domain should step back to kind, got %v",
			m.form.focus,
		)
	}
}

func TestFormToggleTypeAndKind(t *testing.T) {
	// Arrange: add form from filterAll defaults to deny/exact.
	m := newTestModel()
	_, _ = m.Update(keyPress("a"))
	m.form.focus = fieldType

	// Act: space flips type.
	_, _ = m.Update(keyPress(" "))
	if m.form.dtype != pihole.DomainAllow {
		t.Fatalf("space on Type should flip deny->allow, got %v", m.form.dtype)
	}

	// Arrange: kind field.
	m.form.focus = fieldKind
	_, _ = m.Update(keyPress("right"))
	if m.form.dkind != pihole.KindRegex {
		t.Fatalf("right on Kind should flip exact->regex, got %v", m.form.dkind)
	}
}

func TestFormTypingEditsFocusedInput(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(keyPress("a")) // focus = domain
	m.form.focus = fieldComment
	_ = m.form.syncFocus()

	// Act
	_, _ = m.Update(keyPress("z"))

	// Assert
	if !strings.Contains(m.form.comment.Value(), "z") {
		t.Fatalf(
			"typing should edit the focused input, got %q",
			m.form.comment.Value(),
		)
	}
}

func TestFormEscCancels(t *testing.T) {
	m := newTestModel()
	_, _ = m.Update(keyPress("a"))
	_, _ = m.Update(keyPress("esc"))
	if m.form != nil {
		t.Fatalf("esc should close the form")
	}
}

func TestSubmitEditFormUpdatesExisting(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})
	_, _ = m.Update(keyPress("e"))
	m.form.comment.SetValue("updated note")

	// Act
	_, cmd := m.Update(keyPress("enter"))

	// Assert
	if m.form != nil {
		t.Fatalf("submitting the edit form should close it")
	}
	if cmd == nil {
		t.Fatalf("submitting an edit should return a mutation command")
	}
}

// --- View render states -----------------------------------------------------

func viewOf(t *testing.T, m *Model) string {
	t.Helper()
	m.focused = true
	m.SetSize(120, 30)
	out := m.View().Content
	if out == "" {
		t.Fatalf("View produced empty output")
	}
	return out
}

func TestViewLoadedShowsHeaderTableFooter(t *testing.T) {
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})
	out := viewOf(t, m)
	for _, want := range []string{"Domains", "shown", "total", "navigate"} {
		if !strings.Contains(out, want) {
			t.Fatalf("loaded view missing %q:\n%s", want, out)
		}
	}
}

func TestViewLoadingShowsSpinnerText(t *testing.T) {
	m := newTestModel()
	m.loading = true
	out := viewOf(t, m)
	if !strings.Contains(out, "loading domains") {
		t.Fatalf("loading view should show a loading indicator:\n%s", out)
	}
}

func TestViewEmptyShowsNoMatch(t *testing.T) {
	m := newTestModel()
	m.loading = false
	out := viewOf(t, m)
	if !strings.Contains(out, "no domains match") {
		t.Fatalf("empty+idle view should show the no-match message:\n%s", out)
	}
}

func TestViewErrorBannerRendered(t *testing.T) {
	m := newTestModel()
	m.err = errors.New("connection refused")
	out := viewOf(t, m)
	if !strings.Contains(out, "error:") {
		t.Fatalf("a set error should render the banner:\n%s", out)
	}
}

func TestViewAddFormRendered(t *testing.T) {
	m := newTestModel()
	_, _ = m.Update(keyPress("a"))
	out := viewOf(t, m)
	for _, want := range []string{"Add domain", "Type", "Kind", "Domain", "Comment", "Groups", "cancel"} {
		if !strings.Contains(out, want) {
			t.Fatalf("add-form view missing %q:\n%s", want, out)
		}
	}
}

func TestViewEditFormRendered(t *testing.T) {
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})
	_, _ = m.Update(keyPress("e"))
	out := viewOf(t, m)
	if !strings.Contains(out, "Edit domain") {
		t.Fatalf("edit-form view should show the edit heading:\n%s", out)
	}
}

func TestViewConfirmDialogRendered(t *testing.T) {
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})
	_, _ = m.Update(keyPress("x"))
	out := viewOf(t, m)
	if !strings.Contains(out, "Delete domain?") {
		t.Fatalf("confirm view should show the delete prompt:\n%s", out)
	}
}

// --- pure helpers -----------------------------------------------------------

func TestGroupsToStringRoundTrips(t *testing.T) {
	if got := groupsToString([]int{0, 2, 5}); got != "0, 2, 5" {
		t.Fatalf("groupsToString wrong, got %q", got)
	}
	if got := groupsToString(nil); got != "" {
		t.Fatalf("empty groups should stringify to empty, got %q", got)
	}
}

func TestDisplayDomainPrefersUnicode(t *testing.T) {
	d := pihole.Domain{Domain: "xn--80ak6aa92e.com", Unicode: "аррӏе.com"}
	if got := displayDomain(d); got != "аррӏе.com" {
		t.Fatalf("displayDomain should prefer Unicode, got %q", got)
	}
	if got := displayDomain(pihole.Domain{Domain: "plain.com"}); got != "plain.com" {
		t.Fatalf("displayDomain should fall back to ASCII, got %q", got)
	}
}

func TestTruncateBehaviour(t *testing.T) {
	if got := truncate("hello", 0); got != "" {
		t.Fatalf("width 0 should truncate to empty, got %q", got)
	}
	if got := truncate("hello", 1); got != ellipsis {
		t.Fatalf("width 1 should be a bare ellipsis, got %q", got)
	}
	if got := truncate("hello", 3); got != "he"+ellipsis {
		t.Fatalf("width 3 should cut+ellipsis, got %q", got)
	}
	if got := truncate("hi", 10); got != "hi" {
		t.Fatalf("short strings should pass through, got %q", got)
	}
}

func TestDomainToRowDefaultsAndDashComment(t *testing.T) {
	// nil widths triggers the minFlex fallback path; blank comment -> "-".
	d := pihole.Domain{
		Domain:  "x.com",
		Type:    "allow",
		Kind:    "regex",
		Comment: "  ",
	}
	row := domainToRow(d, nil)
	if row[3] != "-" {
		t.Fatalf("blank comment should render as dash, got %q", row[3])
	}
	if row[4] != glyphDisabled {
		t.Fatalf("disabled domain should use hollow glyph, got %q", row[4])
	}
}

func TestClampMinBothBranches(t *testing.T) {
	if clampMin(2, 8) != 8 {
		t.Fatalf("clampMin should raise below-min values")
	}
	if clampMin(20, 8) != 20 {
		t.Fatalf("clampMin should leave above-min values")
	}
}

func TestMaxIntBothBranches(t *testing.T) {
	if maxInt(3, 9) != 9 || maxInt(9, 3) != 9 {
		t.Fatalf("maxInt wrong")
	}
}

func TestFormTitleForBothModes(t *testing.T) {
	if newAddForm(filterAll).title() != "Add domain" {
		t.Fatalf("add form title wrong")
	}
	if newEditForm(sampleDomains()[0]).title() != "Edit domain" {
		t.Fatalf("edit form title wrong")
	}
}

func TestNewAddFormSeedsFromConcreteTab(t *testing.T) {
	f := newAddForm(filterAllowRegex)
	if f.dtype != pihole.DomainAllow || f.dkind != pihole.KindRegex {
		t.Fatalf(
			"add form should seed type/kind from the tab, got %v/%v",
			f.dtype,
			f.dkind,
		)
	}
}
