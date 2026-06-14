package domains

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

// newTestModel builds a Domains model with a non-nil client. No login is
// performed and no network command is executed by these tests — assertions are
// on state transitions and whether a command is returned.
func newTestModel() *Model {
	th := theme.DeepNight()
	api := pihole.New("http://x", "pw")
	m := New(&core.AppContext{API: api, Theme: th, Keys: core.DefaultKeyMap()})
	m.SetSize(100, 24)
	return m
}

// keyPress builds a KeyPressMsg whose String() matches the given keystroke.
func keyPress(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEsc}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	default:
		r := []rune(s)[0]
		return tea.KeyPressMsg{Code: r, Text: s}
	}
}

func sampleDomains() []pihole.Domain {
	return []pihole.Domain{
		{Domain: "ads.example.com", Type: "deny", Kind: "exact", Enabled: true, ID: 1},
		{Domain: "good.example.com", Type: "allow", Kind: "exact", Enabled: true, ID: 2},
		{Domain: "(^|\\.)tracker\\.com$", Type: "deny", Kind: "regex", Enabled: false, ID: 3},
	}
}

func TestComputeColumnWidthsFitsWithinTotal(t *testing.T) {
	// Arrange
	total := 120

	// Act
	widths := computeColumnWidths(total)

	// Assert
	if len(widths) != numCols {
		t.Fatalf("expected %d columns, got %d", numCols, len(widths))
	}
	sum := numCols * cellPad
	for _, w := range widths {
		sum += w
	}
	if sum > total {
		t.Fatalf("columns overflow total width: sum=%d total=%d", sum, total)
	}
}

func TestFilterDomainsSelectsMatchingTypeAndKind(t *testing.T) {
	// Arrange
	ds := sampleDomains()

	// Act
	denyRegex := filterDomains(ds, filterDenyRegex)

	// Assert
	if len(denyRegex) != 1 {
		t.Fatalf("expected 1 deny-regex domain, got %d", len(denyRegex))
	}
	if denyRegex[0].ID != 3 {
		t.Fatalf("wrong domain selected: %+v", denyRegex[0])
	}
}

func TestDomainToRowMapsFieldsInOrder(t *testing.T) {
	// Arrange
	widths := computeColumnWidths(120)
	d := pihole.Domain{Domain: "ads.example.com", Type: "deny", Kind: "exact", Enabled: true, Groups: []int{0, 2}}

	// Act
	row := domainToRow(d, widths)

	// Assert
	if len(row) != numCols {
		t.Fatalf("expected %d cells, got %d", numCols, len(row))
	}
	if row[0] != "deny" || row[1] != "exact" {
		t.Fatalf("type/kind cells wrong: %q %q", row[0], row[1])
	}
	if row[2] != "ads.example.com" {
		t.Fatalf("domain cell wrong: %q", row[2])
	}
	if row[4] != glyphEnabled {
		t.Fatalf("enabled glyph wrong: %q", row[4])
	}
	if row[5] != "2" {
		t.Fatalf("groups count wrong: %q", row[5])
	}
}

func TestParseGroupsSkipsBlanksAndNonNumbers(t *testing.T) {
	// Arrange / Act
	got := parseGroups(" 0, 2 , x, ,3")

	// Assert
	want := []int{0, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestFocusSetsLoadingAndBumpsEpoch(t *testing.T) {
	// Arrange
	m := newTestModel()
	startEpoch := m.epoch

	// Act
	cmd := m.Focus()

	// Assert
	if !m.loading {
		t.Fatalf("focus should set loading")
	}
	if m.epoch != startEpoch+1 {
		t.Fatalf("focus should bump epoch: got %d want %d", m.epoch, startEpoch+1)
	}
	if cmd == nil {
		t.Fatalf("focus should return a fetch command")
	}
}

func TestDomainsMsgPopulatesRowsAndClearsLoading(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.loading = true

	// Act
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})

	// Assert
	if m.loading {
		t.Fatalf("result should clear loading")
	}
	if len(m.domains) != 3 {
		t.Fatalf("expected 3 domains stored, got %d", len(m.domains))
	}
	if len(m.visible) != 3 {
		t.Fatalf("default filter is all; expected 3 visible, got %d", len(m.visible))
	}
}

func TestDomainsMsgErrorSetsBanner(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.loading = true

	// Act
	_, _ = m.Update(domainsMsg{epoch: m.epoch, err: errDomainRequired})

	// Assert
	if m.err == nil {
		t.Fatalf("error result should set the banner")
	}
	if m.loading {
		t.Fatalf("error result should clear loading")
	}
}

func TestUpdateIgnoresStaleDomainsResult(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 9
	m.loading = true

	// Act
	_, _ = m.Update(domainsMsg{epoch: 8, domains: sampleDomains()})

	// Assert
	if len(m.domains) != 0 {
		t.Fatalf("stale result should not populate domains")
	}
	if !m.loading {
		t.Fatalf("stale result should not clear loading")
	}
}

func TestAddKeyOpensForm(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act
	_, _ = m.Update(keyPress("a"))

	// Assert
	if m.form == nil {
		t.Fatalf("'a' should open the add form")
	}
	if m.form.editing {
		t.Fatalf("add form should not be in editing mode")
	}
}

func TestSubmitFormReturnsCmdAndClosesForm(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(keyPress("a"))
	m.form.domain.SetValue("newdomain.example.com")

	// Act
	_, cmd := m.Update(keyPress("enter"))

	// Assert
	if m.form != nil {
		t.Fatalf("submitting should close the form")
	}
	if cmd == nil {
		t.Fatalf("submitting a valid form should return a mutation command")
	}
}

func TestSubmitBlankFormKeepsFormAndSetsError(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(keyPress("a"))

	// Act
	_, _ = m.Update(keyPress("enter"))

	// Assert
	if m.form == nil {
		t.Fatalf("blank submit should keep the form open")
	}
	if m.err == nil {
		t.Fatalf("blank submit should set an error")
	}
}

func TestDeleteKeyOpensConfirmDialog(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})

	// Act
	_, _ = m.Update(keyPress("x"))

	// Assert
	if !m.confirm.Active {
		t.Fatalf("'x' should open the confirm dialog")
	}
	if m.pending == nil {
		t.Fatalf("'x' should record the pending delete target")
	}
}

func TestConfirmingDeleteReturnsCmdAndHidesDialog(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})
	_, _ = m.Update(keyPress("x"))

	// Act
	_, cmd := m.Update(keyPress("y"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("confirming should hide the dialog")
	}
	if m.pending != nil {
		t.Fatalf("confirming should clear the pending target")
	}
	if cmd == nil {
		t.Fatalf("confirming should return a delete command")
	}
}

func TestCancelingDeleteHidesDialogWithoutCmd(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})
	_, _ = m.Update(keyPress("x"))

	// Act
	_, cmd := m.Update(keyPress("n"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("cancelling should hide the dialog")
	}
	if cmd != nil {
		t.Fatalf("cancelling should not return a command")
	}
}

func TestFilterKeyCyclesVisibleSet(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})
	if len(m.visible) != 3 {
		t.Fatalf("precondition: expected 3 visible under 'all', got %d", len(m.visible))
	}

	// Act: all -> allow-exact
	_, _ = m.Update(keyPress("f"))

	// Assert
	if m.filter != filterAllowExact {
		t.Fatalf("filter should advance to allow-exact, got %v", m.filter)
	}
	if len(m.visible) != 1 {
		t.Fatalf("allow-exact should show 1 domain, got %d", len(m.visible))
	}
}

func TestToggleReturnsMutationCmd(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(domainsMsg{epoch: m.epoch, domains: sampleDomains()})

	// Act
	_, cmd := m.Update(keyPress("t"))

	// Assert
	if cmd == nil {
		t.Fatalf("toggle should return a mutation command")
	}
}

func TestMutationSuccessTriggersRefetch(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 4

	// Act
	_, cmd := m.Update(mutationMsg{epoch: 4, err: nil})

	// Assert
	if m.epoch != 5 {
		t.Fatalf("successful mutation should bump epoch for refetch, got %d", m.epoch)
	}
	if !m.loading {
		t.Fatalf("refetch should set loading")
	}
	if cmd == nil {
		t.Fatalf("successful mutation should return a refetch command")
	}
}

func TestSetSizeDoesNotPanicAtTinySizes(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act / Assert (no panic)
	m.SetSize(1, 1)
	m.SetSize(0, 0)
	_ = m.View()
}

var _ core.Screen = (*Model)(nil)
var _ tea.Model = (*Model)(nil)
