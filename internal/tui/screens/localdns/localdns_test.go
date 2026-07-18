package localdns

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
	"github.com/z19r/tihole/internal/tui/core"
)

// newTestModel builds a Local DNS model with a non-nil client. No login is
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

func sampleHosts() []pihole.HostRecord {
	return []pihole.HostRecord{
		{IP: "192.0.2.10", Domain: "host.lan"},
		{IP: "::1", Domain: "local.lan"},
	}
}

func sampleCNAMEs() []pihole.CNAMERecord {
	return []pihole.CNAMERecord{
		{Domain: "alias.lan", Target: "host.lan", TTL: 0},
		{Domain: "www.lan", Target: "host.lan", TTL: 300},
	}
}

func TestComputeHostColumnWidthsFitsWithinTotal(t *testing.T) {
	// Arrange
	total := 120

	// Act
	widths := computeHostColumnWidths(total)

	// Assert
	if len(widths) != hostNumCols {
		t.Fatalf("expected %d columns, got %d", hostNumCols, len(widths))
	}
	sum := hostNumCols * cellPad
	for _, w := range widths {
		sum += w
	}
	if sum > total {
		t.Fatalf("columns overflow total width: sum=%d total=%d", sum, total)
	}
}

func TestHostToRowMapsFieldsInOrder(t *testing.T) {
	// Arrange
	widths := computeHostColumnWidths(120)
	r := pihole.HostRecord{IP: "192.0.2.10", Domain: "host.lan"}

	// Act
	row := hostToRow(r, widths)

	// Assert
	if len(row) != hostNumCols {
		t.Fatalf("expected %d cells, got %d", hostNumCols, len(row))
	}
	if row[0] != "192.0.2.10" || row[1] != "host.lan" {
		t.Fatalf("cells wrong: %q %q", row[0], row[1])
	}
}

func TestCNAMEToRowBlanksZeroTTL(t *testing.T) {
	// Arrange
	widths := computeCNAMEColumnWidths(120)

	// Act
	zero := cnameToRow(
		pihole.CNAMERecord{Domain: "a.lan", Target: "b.lan", TTL: 0},
		widths,
	)
	set := cnameToRow(
		pihole.CNAMERecord{Domain: "a.lan", Target: "b.lan", TTL: 300},
		widths,
	)

	// Assert
	if zero[2] != "" {
		t.Fatalf("zero ttl should render blank, got %q", zero[2])
	}
	if set[2] != "300" {
		t.Fatalf("ttl 300 should render, got %q", set[2])
	}
}

func TestParseTTLOmitsNonPositive(t *testing.T) {
	// Arrange / Act / Assert
	if got := parseTTL(""); got != 0 {
		t.Fatalf("blank ttl should be 0, got %d", got)
	}
	if got := parseTTL("x"); got != 0 {
		t.Fatalf("non-numeric ttl should be 0, got %d", got)
	}
	if got := parseTTL("-5"); got != 0 {
		t.Fatalf("negative ttl should be 0, got %d", got)
	}
	if got := parseTTL("300"); got != 300 {
		t.Fatalf("valid ttl should parse, got %d", got)
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
		t.Fatalf(
			"focus should bump epoch: got %d want %d",
			m.epoch,
			startEpoch+1,
		)
	}
	if cmd == nil {
		t.Fatalf("focus should return a fetch command")
	}
}

func TestHostRecordsMsgPopulatesAndClearsLoading(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.loading = true

	// Act
	_, _ = m.Update(hostRecordsMsg{epoch: m.epoch, records: sampleHosts()})

	// Assert
	if m.loading {
		t.Fatalf("result should clear loading")
	}
	if len(m.hosts) != 2 {
		t.Fatalf("expected 2 hosts stored, got %d", len(m.hosts))
	}
}

func TestCNAMERecordsMsgPopulatesAndClearsLoading(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.loading = true

	// Act
	_, _ = m.Update(cnameRecordsMsg{epoch: m.epoch, records: sampleCNAMEs()})

	// Assert
	if m.loading {
		t.Fatalf("result should clear loading")
	}
	if len(m.cnames) != 2 {
		t.Fatalf("expected 2 cnames stored, got %d", len(m.cnames))
	}
}

func TestHostRecordsMsgErrorSetsBanner(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.loading = true

	// Act
	_, _ = m.Update(hostRecordsMsg{epoch: m.epoch, err: errRecordRequired})

	// Assert
	if m.err == nil {
		t.Fatalf("error result should set the banner")
	}
	if m.loading {
		t.Fatalf("error result should clear loading")
	}
}

func TestUpdateIgnoresStaleResult(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 9
	m.loading = true

	// Act
	_, _ = m.Update(hostRecordsMsg{epoch: 8, records: sampleHosts()})

	// Assert
	if len(m.hosts) != 0 {
		t.Fatalf("stale result should not populate hosts")
	}
	if !m.loading {
		t.Fatalf("stale result should not clear loading")
	}
}

func TestFilterToggleSwitchesKind(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(hostRecordsMsg{epoch: m.epoch, records: sampleHosts()})
	_, _ = m.Update(cnameRecordsMsg{epoch: m.epoch, records: sampleCNAMEs()})
	if m.kind != kindHosts {
		t.Fatalf("precondition: default kind should be hosts")
	}
	if m.activeCount() != 2 {
		t.Fatalf("precondition: 2 hosts active, got %d", m.activeCount())
	}

	// Act
	_, _ = m.Update(keyPress("f"))

	// Assert
	if m.kind != kindCNAMEs {
		t.Fatalf("f should switch to cnames, got %v", m.kind)
	}
	if m.activeCount() != 2 {
		t.Fatalf("cnames active count should be 2, got %d", m.activeCount())
	}
}

func TestAddKeyOpensFormWithKindFields(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act: hosts kind
	_, _ = m.Update(keyPress("a"))

	// Assert
	if m.form == nil {
		t.Fatalf("'a' should open the add form")
	}
	if len(m.form.inputs) != 2 {
		t.Fatalf(
			"host add form should have 2 fields, got %d",
			len(m.form.inputs),
		)
	}

	// Act: switch to cnames, open form
	_, _ = m.Update(keyPress("esc"))
	_, _ = m.Update(keyPress("f"))
	_, _ = m.Update(keyPress("a"))

	// Assert
	if m.form == nil || m.form.kind != kindCNAMEs {
		t.Fatalf("cname add form should open in cname kind")
	}
	if len(m.form.inputs) != 3 {
		t.Fatalf(
			"cname add form should have 3 fields, got %d",
			len(m.form.inputs),
		)
	}
}

func TestSubmitHostFormReturnsCmdAndClosesForm(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(keyPress("a"))
	m.form.inputs[0].SetValue("192.0.2.20")
	m.form.inputs[1].SetValue("new.lan")

	// Act
	_, cmd := m.Update(keyPress("enter"))

	// Assert
	if m.form != nil {
		t.Fatalf("submitting should close the form")
	}
	if cmd == nil {
		t.Fatalf(
			"submitting a valid host form should return a mutation command",
		)
	}
}

func TestSubmitCNAMEFormReturnsCmdAndClosesForm(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(keyPress("f")) // switch to cnames
	_, _ = m.Update(keyPress("a"))
	m.form.inputs[0].SetValue("alias.lan")
	m.form.inputs[1].SetValue("host.lan")
	// leave ttl blank -> omitted

	// Act
	_, cmd := m.Update(keyPress("enter"))

	// Assert
	if m.form != nil {
		t.Fatalf("submitting should close the form")
	}
	if cmd == nil {
		t.Fatalf(
			"submitting a valid cname form should return a mutation command",
		)
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

func TestDeleteKeyOpensConfirmForActiveKind(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(hostRecordsMsg{epoch: m.epoch, records: sampleHosts()})

	// Act
	_, _ = m.Update(keyPress("x"))

	// Assert
	if !m.confirm.Active {
		t.Fatalf("'x' should open the confirm dialog")
	}
	if m.pendingHost == nil {
		t.Fatalf("'x' should record a pending host delete target")
	}
	if m.pendingCNAME != nil {
		t.Fatalf("'x' on hosts kind should not set a cname target")
	}
}

func TestConfirmingHostDeleteReturnsCmd(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(hostRecordsMsg{epoch: m.epoch, records: sampleHosts()})
	_, _ = m.Update(keyPress("x"))

	// Act
	_, cmd := m.Update(keyPress("y"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("confirming should hide the dialog")
	}
	if m.pendingHost != nil {
		t.Fatalf("confirming should clear the pending target")
	}
	if cmd == nil {
		t.Fatalf("confirming should return a delete command")
	}
}

func TestConfirmingCNAMEDeleteReturnsCmd(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(cnameRecordsMsg{epoch: m.epoch, records: sampleCNAMEs()})
	_, _ = m.Update(keyPress("f")) // switch to cnames
	_, _ = m.Update(keyPress("x"))
	if m.pendingCNAME == nil {
		t.Fatalf("'x' on cnames kind should set a cname target")
	}

	// Act
	_, cmd := m.Update(keyPress("y"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("confirming should hide the dialog")
	}
	if cmd == nil {
		t.Fatalf("confirming should return a delete command")
	}
}

func TestCancelingDeleteHidesDialogWithoutCmd(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(hostRecordsMsg{epoch: m.epoch, records: sampleHosts()})
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

func TestMutationSuccessTriggersRefetch(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 4

	// Act
	_, cmd := m.Update(mutationMsg{epoch: 4, err: nil})

	// Assert
	if m.epoch != 5 {
		t.Fatalf(
			"successful mutation should bump epoch for refetch, got %d",
			m.epoch,
		)
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
