package localdns

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
)

// specialKey builds a KeyPressMsg for a non-text key (arrows, delete, etc.) so
// its String() matches the shortcut the handlers switch on.
func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

// sizedModel is a focused model at a comfortable 120x30 with rows loaded so the
// render paths have real content to draw.
func sizedModel(t *testing.T) *Model {
	t.Helper()
	m := newTestModel()
	m.SetSize(120, 30)
	m.focused = true
	return m
}

// --- lifecycle -------------------------------------------------------------

func TestInitReturnsNil(t *testing.T) {
	if cmd := newTestModel().Init(); cmd != nil {
		t.Fatalf("Init should return nil until focus")
	}
}

func TestTitleAndHelpBindings(t *testing.T) {
	m := newTestModel()
	if m.Title() != "Local DNS" {
		t.Fatalf("unexpected title: %q", m.Title())
	}
	if got := len(m.Help()); got != 4 {
		t.Fatalf("expected 4 help bindings, got %d", got)
	}
}

func TestBlurCancelsInflightAndClearsState(t *testing.T) {
	// Arrange: Focus arms a cancel func; open a form + confirm too.
	m := newTestModel()
	m.Focus()
	if m.cancel == nil {
		t.Fatalf("precondition: Focus should arm an in-flight cancel")
	}
	m.form = newHostForm()
	m.confirm = m.confirm.Show("t", "msg", true)
	m.pendingHost = &pihole.HostRecord{IP: "10.0.0.1", Domain: "a.lan"}

	// Act
	m.Blur()

	// Assert
	if m.focused {
		t.Fatalf("Blur should clear focus")
	}
	if m.form != nil || m.confirm.Active || m.pendingHost != nil {
		t.Fatalf("Blur should clear form, confirm, and pending target")
	}
	if m.cancel != nil {
		t.Fatalf("Blur should cancel and clear the in-flight request")
	}
}

func TestSetSizeNegativeSizesDoNotPanic(t *testing.T) {
	m := newTestModel()
	m.SetSize(-5, -5)
	m.form = newCNAMEForm()
	m.SetSize(-1, -1) // exercises form.setWidth clamp too
	_ = m.View()
}

// --- kind toggle -----------------------------------------------------------

func TestKindToggleArrowsSwitchVisibleTableAndFormFields(t *testing.T) {
	// Arrange
	m := sizedModel(t)

	// Act: right advances to CNAMEs, left wraps back to Hosts.
	_, _ = m.Update(specialKey(tea.KeyRight))
	if m.kind != kindCNAMEs {
		t.Fatalf("right should switch to cnames, got %v", m.kind)
	}
	_, _ = m.Update(specialKey(tea.KeyLeft))
	if m.kind != kindHosts {
		t.Fatalf("left should switch back to hosts, got %v", m.kind)
	}

	// Assert: add-form fields track the active kind.
	_, _ = m.Update(keyPress("a"))
	if len(m.form.inputs) != 2 || m.form.kind != kindHosts {
		t.Fatalf("hosts add form should have 2 fields")
	}
	_, _ = m.Update(keyPress("esc"))
	_, _ = m.Update(specialKey(tea.KeyRight))
	_, _ = m.Update(keyPress("a"))
	if len(m.form.inputs) != 3 || m.form.kind != kindCNAMEs {
		t.Fatalf("cname add form should have 3 fields")
	}
}

func TestRecordKindPrevAndLabelFallback(t *testing.T) {
	if kindHosts.prev() != kindCNAMEs {
		t.Fatalf("prev of hosts should wrap to cnames")
	}
	if kindCNAMEs.prev() != kindHosts {
		t.Fatalf("prev of cnames should be hosts")
	}
	if kindCount.label() != "Hosts (A/AAAA)" {
		t.Fatalf("unknown kind should fall back to hosts label, got %q", kindCount.label())
	}
}

// --- view states -----------------------------------------------------------

func TestViewHostsTableState(t *testing.T) {
	m := sizedModel(t)
	_, _ = m.Update(hostRecordsMsg{epoch: m.epoch, records: sampleHosts()})

	out := m.View().Content
	for _, want := range []string{"Local DNS", "Hosts (A/AAAA)", "records", "navigate"} {
		if !strings.Contains(out, want) {
			t.Fatalf("hosts view missing %q:\n%s", want, out)
		}
	}
}

func TestViewCNAMETableState(t *testing.T) {
	m := sizedModel(t)
	_, _ = m.Update(cnameRecordsMsg{epoch: m.epoch, records: sampleCNAMEs()})
	_, _ = m.Update(keyPress("f"))

	out := m.View().Content
	if !strings.Contains(out, "CNAMEs") {
		t.Fatalf("cname view should show the CNAMEs chip:\n%s", out)
	}
}

func TestViewLoadingState(t *testing.T) {
	m := sizedModel(t)
	m.loading = true // no records yet
	if !strings.Contains(m.View().Content, "loading records") {
		t.Fatalf("empty+loading should render a loading indicator")
	}
}

func TestViewEmptyState(t *testing.T) {
	m := sizedModel(t)
	m.loading = false
	if !strings.Contains(m.View().Content, "no Hosts (A/AAAA) records") {
		t.Fatalf("empty+idle should render the no-records message")
	}
}

func TestViewErrorBanner(t *testing.T) {
	m := sizedModel(t)
	m.err = errors.New("connection refused")
	if !strings.Contains(m.View().Content, "error:") {
		t.Fatalf("a set error should render the banner")
	}
}

func TestViewErrorBannerAtTinyWidth(t *testing.T) {
	// Exercises the maxInt(b>a) branch in errBanner/footer.
	m := newTestModel()
	m.SetSize(3, 6)
	m.focused = true
	m.err = errors.New("boom boom boom boom")
	_ = m.View() // must not panic
}

func TestViewAddFormHostAndCNAME(t *testing.T) {
	// Host form.
	m := sizedModel(t)
	_, _ = m.Update(keyPress("a"))
	out := m.View().Content
	for _, want := range []string{"Add host record", "IP", "Domain", "save"} {
		if !strings.Contains(out, want) {
			t.Fatalf("host form view missing %q:\n%s", want, out)
		}
	}

	// CNAME form.
	m2 := sizedModel(t)
	_, _ = m2.Update(keyPress("f"))
	_, _ = m2.Update(keyPress("a"))
	out2 := m2.View().Content
	for _, want := range []string{"Add CNAME record", "Target", "TTL"} {
		if !strings.Contains(out2, want) {
			t.Fatalf("cname form view missing %q:\n%s", want, out2)
		}
	}
}

func TestViewConfirmDialog(t *testing.T) {
	m := sizedModel(t)
	_, _ = m.Update(hostRecordsMsg{epoch: m.epoch, records: sampleHosts()})
	_, _ = m.Update(keyPress("x"))
	if !strings.Contains(m.View().Content, "Delete host record?") {
		t.Fatalf("confirm dialog should render its title")
	}
}

// --- form navigation -------------------------------------------------------

func TestFormNavigationAndTyping(t *testing.T) {
	m := sizedModel(t)
	_, _ = m.Update(keyPress("a"))
	if m.form.focus != 0 {
		t.Fatalf("form should start on field 0")
	}

	// tab / down advance focus.
	_, _ = m.Update(keyPress("tab"))
	if m.form.focus != 1 {
		t.Fatalf("tab should advance focus, got %d", m.form.focus)
	}
	_, _ = m.Update(specialKey(tea.KeyDown))
	if m.form.focus != 0 { // wrapped (2 fields)
		t.Fatalf("down should wrap focus, got %d", m.form.focus)
	}

	// up / shift+tab move back with wrap.
	_, _ = m.Update(specialKey(tea.KeyUp))
	if m.form.focus != 1 {
		t.Fatalf("up should wrap focus back, got %d", m.form.focus)
	}

	// typing routes to the focused input.
	_, _ = m.Update(keyPress("z"))
	if !strings.Contains(m.form.inputs[1].Value(), "z") {
		t.Fatalf("typing should edit the focused input, got %q", m.form.inputs[1].Value())
	}

	// esc closes the form.
	_, _ = m.Update(keyPress("esc"))
	if m.form != nil {
		t.Fatalf("esc should close the form")
	}
}

func TestFormValueOutOfRange(t *testing.T) {
	f := newHostForm()
	if f.value(-1) != "" || f.value(99) != "" {
		t.Fatalf("out-of-range value should be blank")
	}
}

// --- table key routing -----------------------------------------------------

func TestRefreshKeyBumpsEpochAndLoads(t *testing.T) {
	m := sizedModel(t)
	before := m.epoch
	_, cmd := m.Update(keyPress("r"))
	if m.epoch != before+1 || !m.loading || cmd == nil {
		t.Fatalf("r should refetch: epoch=%d loading=%v cmd=%v", m.epoch, m.loading, cmd)
	}
}

func TestNavigationKeyRoutesToActiveTable(t *testing.T) {
	m := sizedModel(t)
	_, _ = m.Update(hostRecordsMsg{epoch: m.epoch, records: sampleHosts()})
	_, _ = m.Update(specialKey(tea.KeyDown)) // hosts table
	_, _ = m.Update(keyPress("f"))
	_, _ = m.Update(cnameRecordsMsg{epoch: m.epoch, records: sampleCNAMEs()})
	_, _ = m.Update(specialKey(tea.KeyDown)) // cname table, no panic
}

func TestDeleteKeyViaDeleteKeyCode(t *testing.T) {
	m := sizedModel(t)
	_, _ = m.Update(hostRecordsMsg{epoch: m.epoch, records: sampleHosts()})
	_, _ = m.Update(specialKey(tea.KeyDelete))
	if !m.confirm.Active {
		t.Fatalf("delete key should open the confirm dialog")
	}
}

func TestCNAMESubmitWithTTL(t *testing.T) {
	m := sizedModel(t)
	_, _ = m.Update(keyPress("f"))
	_, _ = m.Update(keyPress("a"))
	m.form.inputs[0].SetValue("alias.lan")
	m.form.inputs[1].SetValue("host.lan")
	m.form.inputs[2].SetValue("600")
	_, cmd := m.Update(keyPress("enter"))
	if m.form != nil || cmd == nil {
		t.Fatalf("valid cname submit with ttl should close form and return cmd")
	}
}

func TestBlankCNAMESubmitKeepsForm(t *testing.T) {
	m := sizedModel(t)
	_, _ = m.Update(keyPress("f"))
	_, _ = m.Update(keyPress("a"))
	_, _ = m.Update(keyPress("enter"))
	if m.form == nil || m.err == nil {
		t.Fatalf("blank cname submit should keep form and set error")
	}
}

// --- Update: spinner + mutation error --------------------------------------

func TestSpinnerTickAdvancesOnlyWhileLoading(t *testing.T) {
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

func TestMutationErrorSetsBanner(t *testing.T) {
	m := newTestModel()
	m.epoch = 7
	m.loading = true
	_, cmd := m.Update(mutationMsg{epoch: 7, err: errors.New("nope")})
	if m.err == nil || m.loading || cmd != nil {
		t.Fatalf("mutation error should set banner, clear loading, no cmd")
	}
}

func TestStaleMutationDropped(t *testing.T) {
	m := newTestModel()
	m.epoch = 5
	_, cmd := m.Update(mutationMsg{epoch: 4, err: nil})
	if cmd != nil || m.epoch != 5 {
		t.Fatalf("stale mutation should be dropped without refetch")
	}
}

func TestStaleCNAMEResultDropped(t *testing.T) {
	m := newTestModel()
	m.epoch = 3
	m.loading = true
	_, _ = m.Update(cnameRecordsMsg{epoch: 2, records: sampleCNAMEs()})
	if len(m.cnames) != 0 || !m.loading {
		t.Fatalf("stale cname result should not populate or clear loading")
	}
}

// --- fetch / mutate closure bodies (cancelled context, no network) ---------

// runBatch executes a (possibly batched) command and all of its child commands
// synchronously, discarding the resulting messages. Used to drive fetch/mutate
// closures whose captured context has already been cancelled — the pihole
// client short-circuits on ctx.Err() so no real network I/O occurs.
func runBatch(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				_ = c()
			}
		}
	}
}

func TestFetchClosuresRunAgainstCancelledContext(t *testing.T) {
	m := newTestModel()
	m.epoch = 1
	cmd := m.fetch()
	// Pre-cancel the context the closures captured so the API calls return
	// immediately without touching the network.
	if m.cancel != nil {
		m.cancel()
	}
	runBatch(cmd) // must not panic
}

func TestMutateClosureRunsAgainstCancelledContext(t *testing.T) {
	m := newTestModel()
	called := false
	cmd := m.mutate(func(ctx context.Context, api *pihole.Client) error {
		called = true
		return ctx.Err()
	})
	if !m.loading {
		t.Fatalf("mutate should set loading")
	}
	if m.cancel != nil {
		m.cancel()
	}
	runBatch(cmd)
	if !called {
		t.Fatalf("mutate closure should have executed")
	}
}

// --- pure helpers ----------------------------------------------------------

func TestTruncateEdgeCases(t *testing.T) {
	cases := []struct {
		s     string
		width int
		want  string
	}{
		{"hello", 0, ""},
		{"hello", 1, "…"},
		{"hello", 5, "hello"},
		{"hello", 3, "he…"},
		{"héllo", 3, "hé…"},
	}
	for _, c := range cases {
		if got := truncate(c.s, c.width); got != c.want {
			t.Fatalf("truncate(%q,%d)=%q want %q", c.s, c.width, got, c.want)
		}
	}
}

func TestDisplayCNAMEIncludesTTLWhenPositive(t *testing.T) {
	with := displayCNAME(pihole.CNAMERecord{Domain: "a.lan", Target: "b.lan", TTL: 300})
	if !strings.Contains(with, "ttl 300") {
		t.Fatalf("positive ttl should appear, got %q", with)
	}
	without := displayCNAME(pihole.CNAMERecord{Domain: "a.lan", Target: "b.lan"})
	if strings.Contains(without, "ttl") {
		t.Fatalf("zero ttl should be omitted, got %q", without)
	}
}

func TestMaxIntBothBranches(t *testing.T) {
	if maxInt(3, 7) != 7 || maxInt(9, 2) != 9 {
		t.Fatalf("maxInt wrong")
	}
}

func TestFormTitleAndSetWidthClamp(t *testing.T) {
	if newHostForm().title() != "Add host record" {
		t.Fatalf("host form title wrong")
	}
	if newCNAMEForm().title() != "Add CNAME record" {
		t.Fatalf("cname form title wrong")
	}
	f := newHostForm()
	f.setWidth(2) // below the 8-cell floor
	f.setWidth(100)
}
