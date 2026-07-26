package clients

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
	return New(
		&core.AppContext{API: api, Theme: th, Keys: core.DefaultKeyMap()},
	)
}

// key builds a KeyPressMsg for a single-character or named keystroke. Named
// keys carry a Code; printable characters carry Text (which String() echoes).
func press(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	default:
		return tea.KeyPressMsg{Text: s}
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

func TestComputeColumnWidthsClampsFlexibleColumns(t *testing.T) {
	// Arrange
	total := 8 // absurdly small

	// Act
	widths := computeColumnWidths(total)

	// Assert
	for _, i := range []int{0, 1, 2} { // flexible columns
		if widths[i] < minFlexCol {
			t.Fatalf("flexible column %d below minimum: got %d", i, widths[i])
		}
	}
}

func TestClientToRowMapsFieldsInOrder(t *testing.T) {
	// Arrange
	widths := computeColumnWidths(120)
	c := pihole.ClientEntry{
		Client:  "10.0.0.5",
		Name:    "laptop",
		Comment: "work machine",
		Groups:  []int{0, 2, 3},
	}

	// Act
	row := clientToRow(c, widths)

	// Assert
	if len(row) != numCols {
		t.Fatalf("expected %d cells, got %d", numCols, len(row))
	}
	if row[0] != "10.0.0.5" {
		t.Fatalf("client cell wrong: %q", row[0])
	}
	if row[1] != "laptop" {
		t.Fatalf("name cell wrong: %q", row[1])
	}
	if row[2] != "work machine" {
		t.Fatalf("comment cell wrong: %q", row[2])
	}
	if row[3] != "3" {
		t.Fatalf("groups count cell wrong: %q", row[3])
	}
}

func TestClientToRowRendersDashForEmptyFields(t *testing.T) {
	// Arrange
	widths := computeColumnWidths(120)
	c := pihole.ClientEntry{Client: "10.0.0.9"}

	// Act
	row := clientToRow(c, widths)

	// Assert
	if row[1] != "-" {
		t.Fatalf("empty name should render dash, got %q", row[1])
	}
	if row[2] != "-" {
		t.Fatalf("empty comment should render dash, got %q", row[2])
	}
	if row[3] != "0" {
		t.Fatalf("empty groups should count zero, got %q", row[3])
	}
}

func TestParseGroupsSplitsAndTrims(t *testing.T) {
	// Arrange
	in := "1, 2,3"

	// Act
	got, err := parseGroups(in)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestParseGroupsEmptyYieldsEmptySlice(t *testing.T) {
	// Arrange / Act
	got, err := parseGroups("   ")

	// Assert
	if err != nil {
		t.Fatalf("blank input should not error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("blank input should yield empty slice, got %v", got)
	}
}

func TestParseGroupsRejectsNonInteger(t *testing.T) {
	// Arrange / Act
	_, err := parseGroups("1,two,3")

	// Assert
	if err == nil {
		t.Fatalf("non-integer token should return an error")
	}
}

func TestGroupsToStringRoundTrips(t *testing.T) {
	// Arrange / Act
	s := groupsToString([]int{0, 1})

	// Assert
	if s != "0, 1" {
		t.Fatalf("unexpected groups string: %q", s)
	}
}

func TestFirstAddressTakesLeadingToken(t *testing.T) {
	// Arrange / Act
	got := firstAddress("10.0.0.5, 10.0.0.6")

	// Assert
	if got != "10.0.0.5" {
		t.Fatalf("expected first address, got %q", got)
	}
}

func TestFocusSetsLoadingAndBumpsEpoch(t *testing.T) {
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
		t.Fatalf("Focus should bump epoch: before=%d after=%d", before, m.epoch)
	}
	if cmd == nil {
		t.Fatalf("Focus should return a fetch command")
	}
}

func TestClientsMsgPopulatesRowsAndClearsLoading(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.SetSize(100, 20)
	m.epoch = 4
	m.loading = true

	// Act
	_, _ = m.Update(clientsMsg{epoch: 4, clients: []pihole.ClientEntry{
		{Client: "10.0.0.1", Name: "one"},
		{Client: "10.0.0.2", Name: "two"},
	}})

	// Assert
	if m.loading {
		t.Fatalf("clientsMsg should clear loading")
	}
	if len(m.clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(m.clients))
	}
	if len(m.table.Rows()) != 2 {
		t.Fatalf("expected 2 table rows, got %d", len(m.table.Rows()))
	}
}

func TestClientsMsgErrorSetsBanner(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 2
	m.loading = true

	// Act
	_, _ = m.Update(clientsMsg{epoch: 2, err: errors.New("boom")})

	// Assert
	if m.err == nil {
		t.Fatalf("error result should set the banner error")
	}
	if m.loading {
		t.Fatalf("error result should clear loading")
	}
}

func TestUpdateIgnoresStaleClientsResult(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 9
	m.loading = true

	// Act
	_, _ = m.Update(
		clientsMsg{epoch: 8, clients: []pihole.ClientEntry{{Client: "x"}}},
	)

	// Assert
	if len(m.clients) != 0 {
		t.Fatalf("stale clients result should not be applied")
	}
	if !m.loading {
		t.Fatalf("stale result should not clear loading")
	}
}

func TestKeyAOpensAddForm(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act
	_, cmd := m.Update(press("a"))

	// Assert
	if !m.form.active {
		t.Fatalf("`a` should open the add form")
	}
	if m.form.mode != formAdd {
		t.Fatalf("`a` should select add mode")
	}
	if cmd == nil {
		t.Fatalf("opening the form should return a focus command")
	}
}

func TestSubmitFormReturnsCmdAndClosesForm(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.openAddForm()
	m.form.client.SetValue("10.0.0.42")
	m.form.comment.SetValue("printer")
	m.form.groups.SetValue("0, 1")

	// Act
	_, cmd := m.Update(press("enter"))

	// Assert
	if m.form.active {
		t.Fatalf("submit should close the form")
	}
	if cmd == nil {
		t.Fatalf("submit should return a mutation command")
	}
}

func TestSubmitFormRejectsEmptyClient(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.openAddForm()

	// Act
	_, cmd := m.Update(press("enter"))

	// Assert
	if !m.form.active {
		t.Fatalf("invalid submit should keep the form open")
	}
	if m.form.err == nil {
		t.Fatalf("empty client should set a form error")
	}
	if cmd != nil {
		t.Fatalf("invalid submit should not issue a mutation command")
	}
}

func TestKeyXOpensDeleteConfirm(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.SetSize(100, 20)
	m.clients = []pihole.ClientEntry{{Client: "10.0.0.7"}}
	m.syncRows()

	// Act
	_, _ = m.Update(press("x"))

	// Assert
	if !m.confirm.Active {
		t.Fatalf("`x` should open the confirm dialog")
	}
	if m.pendingDelete != "10.0.0.7" {
		t.Fatalf(
			"pending delete should hold the selected client: %q",
			m.pendingDelete,
		)
	}
}

func TestConfirmYesReturnsDeleteCmdAndHidesDialog(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.confirm = m.confirm.Show("Delete client?", "10.0.0.7", true)
	m.pendingDelete = "10.0.0.7"

	// Act
	_, cmd := m.Update(press("y"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("confirming should hide the dialog")
	}
	if cmd == nil {
		t.Fatalf("confirming should return a delete command")
	}
	if m.pendingDelete != "" {
		t.Fatalf("pending delete should be cleared")
	}
}

func TestConfirmCancelHidesDialogWithoutCmd(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.confirm = m.confirm.Show("Delete client?", "10.0.0.7", true)
	m.pendingDelete = "10.0.0.7"

	// Act
	_, cmd := m.Update(press("n"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("cancel should hide the dialog")
	}
	if cmd != nil {
		t.Fatalf("cancel should not issue a command")
	}
}

func TestKeyShiftSOpensSuggestions(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act
	_, cmd := m.Update(press("S"))

	// Assert
	if !m.suggest.active {
		t.Fatalf("`S` should open the suggestions pane")
	}
	if !m.suggest.loading {
		t.Fatalf("`S` should mark the pane loading")
	}
	if cmd == nil {
		t.Fatalf("`S` should return a fetch command")
	}
}

func TestSuggestionsMsgFillsPane(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 3
	m.suggest.active = true
	m.suggest.loading = true

	// Act
	_, _ = m.Update(
		suggestionsMsg{epoch: 3, suggestions: []pihole.ClientSuggestion{
			{
				Addresses: "10.0.0.5",
				Name:      "phone",
				MACVendor: "Apple",
				LastQuery: 1_700_000_000,
			},
		}},
	)

	// Assert
	if m.suggest.loading {
		t.Fatalf("suggestionsMsg should clear the pane loading flag")
	}
	if len(m.suggest.suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(m.suggest.suggestions))
	}
}

func TestSuggestionEnterPrefillsAddForm(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.suggest.active = true
	m.suggest.suggestions = []pihole.ClientSuggestion{
		{Addresses: "10.0.0.5, 10.0.0.6"},
	}
	m.suggest.cursor = 0

	// Act
	_, _ = m.Update(press("enter"))

	// Assert
	if m.suggest.active {
		t.Fatalf("selecting a suggestion should close the pane")
	}
	if !m.form.active || m.form.mode != formAdd {
		t.Fatalf("selecting a suggestion should open the add form")
	}
	if m.form.client.Value() != "10.0.0.5" {
		t.Fatalf(
			"add form should be pre-filled with the address, got %q",
			m.form.client.Value(),
		)
	}
}

func TestEditFormPreFillsAndFixesClient(t *testing.T) {
	// Arrange
	m := newTestModel()
	c := pihole.ClientEntry{
		Client:  "10.0.0.8",
		Comment: "nas",
		Groups:  []int{0, 1},
	}

	// Act
	m.openEditForm(c)

	// Assert
	if m.form.mode != formEdit {
		t.Fatalf("openEditForm should select edit mode")
	}
	if m.form.client.Value() != "10.0.0.8" {
		t.Fatalf("edit form should carry the client id")
	}
	if m.form.groups.Value() != "0, 1" {
		t.Fatalf(
			"edit form should pre-fill groups, got %q",
			m.form.groups.Value(),
		)
	}
	// In edit mode the client field is not focusable.
	for _, f := range m.form.fields() {
		if f == &m.form.client {
			t.Fatalf("client field should not be focusable in edit mode")
		}
	}
}

func TestSetSizeSafeAtTinySizes(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act / Assert — must not panic
	m.SetSize(0, 0)
	m.SetSize(1, 1)
	m.SetSize(3, 2)

	if m.w != 3 || m.h != 2 {
		t.Fatalf("SetSize should record dimensions: w=%d h=%d", m.w, m.h)
	}
}

func TestReloadBumpsEpochAndLoads(t *testing.T) {
	// Arrange
	m := newTestModel()
	before := m.epoch

	// Act
	cmd := m.reload()

	// Assert
	if m.epoch != before+1 {
		t.Fatalf("reload should bump epoch")
	}
	if !m.loading {
		t.Fatalf("reload should set loading")
	}
	if cmd == nil {
		t.Fatalf("reload should return a fetch command")
	}
}

var _ core.Screen = (*Model)(nil)
var _ tea.Model = (*Model)(nil)
