package groups

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
	"github.com/z19r/tihole/internal/tui/core"
)

var _ core.Screen = (*Model)(nil)
var _ tea.Model = (*Model)(nil)

func newTestModel() *Model {
	th := theme.DeepNight()
	api := pihole.New("http://x", "pw")
	m := New(&core.AppContext{API: api, Theme: th, Keys: core.DefaultKeyMap()})
	m.SetSize(80, 24)
	return m
}

func keyPress(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	default:
		r := []rune(s)[0]
		return tea.KeyPressMsg{Code: r, Text: s}
	}
}

// --- pure helpers ---

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

func TestComputeColumnWidthsClampsFlexibleColumns(t *testing.T) {
	// Arrange
	total := 8 // absurdly small

	// Act
	widths := computeColumnWidths(total)

	// Assert
	for _, i := range []int{0, 1} { // flexible columns
		if widths[i] < minFlexCol {
			t.Fatalf("flexible column %d below minimum: got %d", i, widths[i])
		}
	}
}

func TestGroupToRowMapsFieldsInOrder(t *testing.T) {
	// Arrange
	widths := computeColumnWidths(120)
	g := pihole.Group{Name: "kids", Comment: "phones", Enabled: true, ID: 3}

	// Act
	row := groupToRow(g, widths)

	// Assert
	if len(row) != numCols {
		t.Fatalf("expected %d cells, got %d", numCols, len(row))
	}
	if row[0] != "kids" {
		t.Fatalf("name cell wrong: %q", row[0])
	}
	if row[1] != "phones" {
		t.Fatalf("comment cell wrong: %q", row[1])
	}
	if row[2] != enabledGlyph {
		t.Fatalf("enabled cell should be filled glyph, got %q", row[2])
	}
	if row[3] != "3" {
		t.Fatalf("id cell wrong: %q", row[3])
	}
}

func TestGroupToRowDisabledGlyph(t *testing.T) {
	// Arrange
	widths := computeColumnWidths(120)
	g := pihole.Group{Name: "off", Enabled: false, ID: 1}

	// Act
	row := groupToRow(g, widths)

	// Assert
	if row[2] != disabledGlyph {
		t.Fatalf("disabled group should render hollow glyph, got %q", row[2])
	}
}

func TestTruncateAppendsEllipsisWhenTooLong(t *testing.T) {
	// Arrange / Act
	out := truncate("verylonggroupname", 8)

	// Assert
	if out != "verylon"+ellipsis {
		t.Fatalf("unexpected truncation: %q", out)
	}
	if len([]rune(out)) != 8 {
		t.Fatalf("truncated width should equal 8, got %d", len([]rune(out)))
	}
}

// --- lifecycle / Update ---

func TestFocusSetsLoadingAndBumpsEpoch(t *testing.T) {
	// Arrange
	m := newTestModel()
	before := m.epoch

	// Act
	cmd := m.Focus()

	// Assert
	if !m.focused {
		t.Fatalf("Focus should mark the screen focused")
	}
	if !m.loading {
		t.Fatalf("Focus should set loading")
	}
	if m.epoch != before+1 {
		t.Fatalf("Focus should bump epoch: before=%d after=%d", before, m.epoch)
	}
	if cmd == nil {
		t.Fatalf("Focus should return a fetch command")
	}
}

func TestGroupsMsgPopulatesRowsAndClearsLoading(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 4
	m.loading = true
	gs := []pihole.Group{
		{Name: "default", Enabled: true, ID: 0},
		{Name: "kids", Enabled: false, ID: 1},
	}

	// Act
	_, _ = m.Update(groupsMsg{epoch: 4, groups: gs})

	// Assert
	if m.loading {
		t.Fatalf("groupsMsg should clear loading")
	}
	if len(m.groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(m.groups))
	}
	if len(m.table.Rows()) != 2 {
		t.Fatalf("table should have 2 rows, got %d", len(m.table.Rows()))
	}
}

func TestGroupsMsgErrorSetsBanner(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 2
	m.loading = true

	// Act
	_, _ = m.Update(groupsMsg{epoch: 2, err: errBoom})

	// Assert
	if m.err == nil {
		t.Fatalf("error result should set the banner error")
	}
	if m.loading {
		t.Fatalf("error result should clear loading")
	}
}

func TestUpdateIgnoresStaleGroupsResult(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 9
	m.loading = true

	// Act
	_, _ = m.Update(groupsMsg{epoch: 8, groups: []pihole.Group{{Name: "x"}}})

	// Assert
	if len(m.groups) != 0 {
		t.Fatalf("stale groups result should not be applied")
	}
	if !m.loading {
		t.Fatalf("stale result should not clear loading")
	}
}

func TestMutationSuccessTriggersRefetch(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 3

	// Act
	_, cmd := m.Update(mutationMsg{epoch: 3})

	// Assert
	if cmd == nil {
		t.Fatalf("successful mutation should trigger a refetch command")
	}
	if !m.loading {
		t.Fatalf("successful mutation should re-enter loading")
	}
}

func TestMutationErrorSetsBanner(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 3
	m.loading = true

	// Act
	_, cmd := m.Update(mutationMsg{epoch: 3, err: errBoom})

	// Assert
	if m.err == nil {
		t.Fatalf("mutation error should set the banner")
	}
	if cmd != nil {
		t.Fatalf("mutation error should not trigger a refetch")
	}
}

func TestMutationIgnoresStaleEpoch(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 5

	// Act
	_, cmd := m.Update(mutationMsg{epoch: 4})

	// Assert
	if cmd != nil {
		t.Fatalf("stale mutation should produce no command")
	}
}

// --- key handling ---

func TestAddKeyOpensForm(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act
	_, cmd := m.Update(keyPress("a"))

	// Assert
	if !m.form.active {
		t.Fatalf("'a' should open the add form")
	}
	if m.form.editing {
		t.Fatalf("add form should not be in editing mode")
	}
	if cmd == nil {
		t.Fatalf("opening the form should return a focus command")
	}
}

func TestFormSubmitReturnsCmdAndClosesForm(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.form.openAdd(m.w)
	m.form.name.SetValue("newgroup")

	// Act
	_, cmd := m.Update(keyPress("enter"))

	// Assert
	if m.form.active {
		t.Fatalf("submitting should close the form")
	}
	if cmd == nil {
		t.Fatalf("submitting a valid form should return a mutation command")
	}
}

func TestFormSubmitBlankNameIsNoop(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.form.openAdd(m.w)

	// Act
	_, cmd := m.Update(keyPress("enter"))

	// Assert
	if !m.form.active {
		t.Fatalf("blank-name submit should keep the form open")
	}
	if cmd != nil {
		t.Fatalf("blank-name submit should not return a command")
	}
}

func TestFormEscCancels(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.form.openAdd(m.w)

	// Act
	_, _ = m.Update(keyPress("esc"))

	// Assert
	if m.form.active {
		t.Fatalf("esc should cancel the form")
	}
}

func TestDeleteKeyOpensConfirm(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.groups = []pihole.Group{{Name: "kids", ID: 1}}
	m.syncRows()

	// Act
	_, _ = m.Update(keyPress("x"))

	// Assert
	if !m.confirm.Active {
		t.Fatalf("'x' should open the confirm dialog")
	}
	if m.pendingDelete != "kids" {
		t.Fatalf(
			"pending delete should be the selected group, got %q",
			m.pendingDelete,
		)
	}
}

func TestConfirmReturnsDeleteCmdAndHidesDialog(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.groups = []pihole.Group{{Name: "kids", ID: 1}}
	m.syncRows()
	_, _ = m.Update(keyPress("x"))

	// Act
	_, cmd := m.Update(keyPress("y"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("confirming should hide the dialog")
	}
	if cmd == nil {
		t.Fatalf("confirming should return a delete command")
	}
	if m.pendingDelete != "" {
		t.Fatalf("pending delete should be cleared after confirm")
	}
}

func TestConfirmCancelKeepsGroup(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.groups = []pihole.Group{{Name: "kids", ID: 1}}
	m.syncRows()
	_, _ = m.Update(keyPress("x"))

	// Act
	_, cmd := m.Update(keyPress("n"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("'n' should cancel the confirm dialog")
	}
	if cmd != nil {
		t.Fatalf("cancelling delete should not return a command")
	}
}

func TestToggleReturnsUpdateCmd(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.groups = []pihole.Group{{Name: "kids", Enabled: true, ID: 1}}
	m.syncRows()

	// Act
	_, cmd := m.Update(keyPress("space"))

	// Assert
	if cmd == nil {
		t.Fatalf("space on a selected group should return an update command")
	}
}

func TestToggleNoSelectionIsNoop(t *testing.T) {
	// Arrange
	m := newTestModel() // empty list

	// Act
	_, cmd := m.Update(keyPress("t"))

	// Assert
	if cmd != nil {
		t.Fatalf("toggle with no selection should be a no-op")
	}
}

func TestRefreshBumpsEpochAndLoads(t *testing.T) {
	// Arrange
	m := newTestModel()
	before := m.epoch

	// Act
	_, cmd := m.Update(keyPress("r"))

	// Assert
	if m.epoch != before+1 {
		t.Fatalf("refresh should bump epoch")
	}
	if !m.loading {
		t.Fatalf("refresh should set loading")
	}
	if cmd == nil {
		t.Fatalf("refresh should return a fetch command")
	}
}

func TestSetSizeSafeAtTinySizes(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act / Assert — must not panic
	m.SetSize(1, 1)
	m.SetSize(0, 0)
	_ = m.View()
}

func TestEditKeyOpensPrefilledForm(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.groups = []pihole.Group{
		{Name: "kids", Comment: "phones", Enabled: false, ID: 1},
	}
	m.syncRows()

	// Act
	_, _ = m.Update(keyPress("e"))

	// Assert
	if !m.form.active || !m.form.editing {
		t.Fatalf("'e' should open the edit form in editing mode")
	}
	if m.form.origName != "kids" {
		t.Fatalf(
			"edit form should key on original name, got %q",
			m.form.origName,
		)
	}
	if m.form.enabled {
		t.Fatalf("edit form should carry the group's enabled state")
	}
}

// errBoom is a reusable sentinel error for banner tests.
var errBoom = errTest("boom")

type errTest string

func (e errTest) Error() string { return string(e) }
