package adlists

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
	"github.com/z19r/tihole/internal/tui/core"
)

func newTestModel() *Model {
	th := theme.DeepNight()
	api := pihole.New("http://x", "pw")
	m := New(&core.AppContext{API: api, Theme: th, Keys: core.DefaultKeyMap()})
	m.SetSize(120, 30)
	return m
}

func keyPress(s string) tea.KeyPressMsg {
	if s == "space" {
		return tea.KeyPressMsg{Code: tea.KeySpace}
	}
	return tea.KeyPressMsg{Text: s}
}

// --- lifecycle & Update -----------------------------------------------------

func TestFocusSetsLoadingBumpsEpochAndReturnsCmd(t *testing.T) {
	// Arrange
	m := newTestModel()
	before := m.epoch

	// Act
	cmd := m.Focus()

	// Assert
	if !m.loading {
		t.Fatalf("Focus should set loading")
	}
	if m.epoch != before+1 {
		t.Fatalf("Focus should bump epoch: got %d want %d", m.epoch, before+1)
	}
	if cmd == nil {
		t.Fatalf("Focus should return fetch/spinner cmds")
	}
}

func TestListsMsgPopulatesRowsAndClearsLoading(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 4
	m.loading = true

	// Act
	_, _ = m.Update(listsMsg{
		epoch: 4,
		block: []pihole.List{{Address: "a", Type: "block"}},
		allow: []pihole.List{
			{Address: "b", Type: "allow"},
			{Address: "c", Type: "allow"},
		},
	})

	// Assert
	if m.loading {
		t.Fatalf("listsMsg should clear loading")
	}
	if len(m.lists[pihole.ListBlock]) != 1 ||
		len(m.lists[pihole.ListAllow]) != 2 {
		t.Fatalf("lists not populated: %v", m.lists)
	}
}

func TestListsMsgErrorSetsBanner(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 2
	m.loading = true

	// Act
	_, _ = m.Update(listsMsg{epoch: 2, err: errors.New("network down")})

	// Assert
	if m.err == nil {
		t.Fatalf("error listsMsg should set banner")
	}
	if m.loading {
		t.Fatalf("error listsMsg should clear loading")
	}
}

func TestUpdateIgnoresStaleListsEpoch(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 9
	m.loading = true

	// Act
	_, _ = m.Update(listsMsg{epoch: 8, block: []pihole.List{{Address: "a"}}})

	// Assert
	if len(m.lists[pihole.ListBlock]) != 0 {
		t.Fatalf("stale listsMsg should not populate rows")
	}
	if !m.loading {
		t.Fatalf("stale result should not clear loading")
	}
}

func TestMutatedMsgSuccessTriggersRefetch(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 3

	// Act
	_, cmd := m.Update(mutatedMsg{epoch: 3})

	// Assert
	if cmd == nil {
		t.Fatalf("successful mutation should trigger a refetch cmd")
	}
	if !m.loading {
		t.Fatalf("successful mutation should set loading")
	}
}

func TestMutatedMsgErrorSetsBanner(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 3

	// Act
	_, cmd := m.Update(mutatedMsg{epoch: 3, err: errors.New("bad request")})

	// Assert
	if m.err == nil {
		t.Fatalf("mutation error should set banner")
	}
	if cmd != nil {
		t.Fatalf("mutation error should not refetch")
	}
}

// --- type toggle & table keys ----------------------------------------------

func TestToggleVisibleSwitchesType(t *testing.T) {
	// Arrange
	m := newTestModel()
	if m.visible != pihole.ListBlock {
		t.Fatalf("default visible should be block")
	}

	// Act
	_, _ = m.Update(keyPress("f"))

	// Assert
	if m.visible != pihole.ListAllow {
		t.Fatalf("f should switch visible to allow, got %q", m.visible)
	}
}

func TestRefreshBumpsEpochAndReturnsCmd(t *testing.T) {
	// Arrange
	m := newTestModel()
	before := m.epoch

	// Act
	_, cmd := m.Update(keyPress("r"))

	// Assert
	if m.epoch != before+1 {
		t.Fatalf("r should bump epoch")
	}
	if cmd == nil {
		t.Fatalf("r should return a fetch cmd")
	}
}

// --- add/edit form ----------------------------------------------------------

func TestAddOpensForm(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act
	_, _ = m.Update(keyPress("a"))

	// Assert
	if !m.form.active {
		t.Fatalf("a should open the add form")
	}
	if m.form.mode != formAdd {
		t.Fatalf("form should be in add mode")
	}
}

func TestSubmitAddReturnsCmdAndClosesForm(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(keyPress("a"))
	m.form.address.SetValue("https://example.com/list.txt")

	// Act
	_, cmd := m.Update(keyPress("enter"))

	// Assert
	if m.form.active {
		t.Fatalf("submit should close the form")
	}
	if cmd == nil {
		t.Fatalf("submit with an address should return an AddList cmd")
	}
}

func TestSubmitAddWithoutAddressSetsError(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(keyPress("a"))

	// Act
	_, cmd := m.Update(keyPress("enter"))

	// Assert
	if cmd != nil {
		t.Fatalf("empty address should not return a cmd")
	}
	if m.err == nil {
		t.Fatalf("empty address should set an error")
	}
}

func TestFormEscCancels(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(keyPress("a"))

	// Act
	_, _ = m.Update(keyPress("esc"))

	// Assert
	if m.form.active {
		t.Fatalf("esc should cancel the form")
	}
}

func TestEditPrefillsForm(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.lists[pihole.ListBlock] = []pihole.List{
		{
			Address: "https://a",
			Comment: "c",
			Type:    "block",
			Groups:  []int{0, 2},
			Enabled: true,
		},
	}
	m.syncRows()

	// Act
	_, _ = m.Update(keyPress("e"))

	// Assert
	if !m.form.active || m.form.mode != formEdit {
		t.Fatalf("e should open the edit form")
	}
	if m.form.address.Value() != "https://a" {
		t.Fatalf(
			"edit form should prefill address, got %q",
			m.form.address.Value(),
		)
	}
	if m.form.groups.Value() != "0,2" {
		t.Fatalf(
			"edit form should prefill groups, got %q",
			m.form.groups.Value(),
		)
	}
}

func TestFormTypeToggle(t *testing.T) {
	// Arrange
	m := newTestModel()
	f := m.form.openAdd(pihole.ListBlock)
	// advance focus to the type field.
	for f.current() != fieldType {
		f = f.advance(1)
	}

	// Act
	f = f.toggle()

	// Assert
	if f.listType != pihole.ListAllow {
		t.Fatalf("toggle should flip type to allow, got %q", f.listType)
	}
}

// --- delete confirm ---------------------------------------------------------

func TestDeleteOpensConfirm(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.lists[pihole.ListBlock] = []pihole.List{
		{Address: "https://gone", Type: "block"},
	}
	m.syncRows()

	// Act
	_, _ = m.Update(keyPress("x"))

	// Assert
	if !m.confirm.Active {
		t.Fatalf("x should open the confirm dialog")
	}
	if m.confirmKind != confirmDelete {
		t.Fatalf("confirm kind should be delete")
	}
	if m.pendingAddr != "https://gone" {
		t.Fatalf("pending address wrong: %q", m.pendingAddr)
	}
}

func TestConfirmYesReturnsDeleteCmd(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.lists[pihole.ListBlock] = []pihole.List{
		{Address: "https://gone", Type: "block"},
	}
	m.syncRows()
	_, _ = m.Update(keyPress("x"))

	// Act
	_, cmd := m.Update(keyPress("y"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("confirm should close on y")
	}
	if cmd == nil {
		t.Fatalf("y should return a DeleteList cmd")
	}
}

func TestConfirmNoCancels(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.lists[pihole.ListBlock] = []pihole.List{
		{Address: "https://gone", Type: "block"},
	}
	m.syncRows()
	_, _ = m.Update(keyPress("x"))

	// Act
	_, cmd := m.Update(keyPress("n"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("n should close the confirm")
	}
	if cmd != nil {
		t.Fatalf("n should not return a cmd")
	}
}

// --- toggle enabled ---------------------------------------------------------

func TestToggleEnabledReturnsCmd(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.lists[pihole.ListBlock] = []pihole.List{
		{Address: "https://a", Type: "block", Enabled: true},
	}
	m.syncRows()

	// Act
	_, cmd := m.Update(keyPress("space"))

	// Assert
	if cmd == nil {
		t.Fatalf("space should return an UpdateList cmd")
	}
}

// --- gravity streaming ------------------------------------------------------

func TestGravityKeyOpensConfirm(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act
	_, _ = m.Update(keyPress("g"))

	// Assert
	if !m.confirm.Active {
		t.Fatalf("g should open the gravity confirm")
	}
	if m.confirmKind != confirmGravity {
		t.Fatalf("confirm kind should be gravity")
	}
}

func TestConfirmGravityStartsStreaming(t *testing.T) {
	// Arrange
	m := newTestModel()
	_, _ = m.Update(keyPress("g"))

	// Act
	_, cmd := m.Update(keyPress("y"))

	// Assert
	if !m.gravityOpen || !m.gravityRunning {
		t.Fatalf(
			"confirming gravity should open the log pane and start streaming",
		)
	}
	if m.gravityChan == nil {
		t.Fatalf("gravity channel should be initialised")
	}
	if cmd == nil {
		t.Fatalf("confirming gravity should return the pump cmds")
	}
}

func TestGravityLineAppendsToBuffer(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 5
	m.gravityChan = make(chan string, 1)
	m.gravityRunning = true

	// Act
	_, cmd := m.Update(
		gravityLineMsg{epoch: 5, line: "[i] Building tree", ok: true},
	)

	// Assert
	if len(m.gravityLines) != 1 || m.gravityLines[0] != "[i] Building tree" {
		t.Fatalf("gravity line should be appended: %v", m.gravityLines)
	}
	if cmd == nil {
		t.Fatalf("a received line should re-issue the reader cmd")
	}
}

func TestGravityLineClosedChannelStopsPump(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 5
	m.gravityRunning = true

	// Act
	_, cmd := m.Update(gravityLineMsg{epoch: 5, ok: false})

	// Assert
	if cmd != nil {
		t.Fatalf("closed channel should not re-issue the reader")
	}
}

func TestGravityDoneSuccessMarksComplete(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 5
	m.gravityRunning = true

	// Act
	_, _ = m.Update(gravityDoneMsg{epoch: 5, err: nil})

	// Assert
	if m.gravityRunning {
		t.Fatalf("done should stop running")
	}
	if !m.gravityDone || m.gravityErr != nil {
		t.Fatalf("done with nil err should mark success")
	}
}

func TestGravityDoneErrorShowsError(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 5
	m.gravityRunning = true

	// Act
	_, _ = m.Update(gravityDoneMsg{epoch: 5, err: errors.New("gravity boom")})

	// Assert
	if m.gravityErr == nil {
		t.Fatalf("done with err should record the error")
	}
	if m.gravityRunning {
		t.Fatalf("done should stop running")
	}
}

func TestGravityLineStaleEpochDropped(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 6
	m.gravityChan = make(chan string, 1)

	// Act
	_, cmd := m.Update(gravityLineMsg{epoch: 5, line: "old", ok: true})

	// Assert
	if len(m.gravityLines) != 0 {
		t.Fatalf("stale gravity line should be dropped")
	}
	if cmd != nil {
		t.Fatalf("stale gravity line should not pump")
	}
}

func TestGravityEscClosesWhenFinished(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.gravityOpen = true
	m.gravityRunning = false
	m.gravityDone = true

	// Act
	_, _ = m.Update(keyPress("esc"))

	// Assert
	if m.gravityOpen {
		t.Fatalf("esc should close a finished gravity pane")
	}
}

func TestGravityEscIgnoredWhileRunning(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.gravityOpen = true
	m.gravityRunning = true

	// Act
	_, _ = m.Update(keyPress("esc"))

	// Assert
	if !m.gravityOpen {
		t.Fatalf("esc should not close a running gravity pane")
	}
}

// --- sizing & interface -----------------------------------------------------

func TestSetSizeSafeAtTinySizes(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act / Assert (must not panic)
	m.SetSize(1, 1)
	m.SetSize(0, 0)
	_ = m.View()
}

func TestViewRendersAtNormalSize(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.lists[pihole.ListBlock] = []pihole.List{
		{
			Address: "https://a",
			Type:    "block",
			Enabled: true,
			Status:  1,
			Number:  5,
		},
	}
	m.syncRows()

	// Act
	v := m.View()

	// Assert
	if v.Content == "" {
		t.Fatalf("View should render non-empty content")
	}
}

var _ core.Screen = (*Model)(nil)
var _ tea.Model = (*Model)(nil)
