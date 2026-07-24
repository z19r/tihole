package system

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

// newTestModel builds a System model with an offline client (no login/network).
func newTestModel() *Model {
	th := theme.DeepNight()
	ctx := &core.AppContext{
		Theme: th,
		Keys:  core.DefaultKeyMap(),
		API:   pihole.New("http://x", "pw"),
	}
	m := New(ctx)
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

func TestFocusStartsLoadingBumpsEpochAndFetchesDefaultTab(t *testing.T) {
	// Arrange
	m := newTestModel()
	before := m.epoch

	// Act
	cmd := m.Focus()

	// Assert
	if !m.focused {
		t.Fatalf("Focus should mark the screen focused")
	}
	if m.epoch != before+1 {
		t.Fatalf("Focus should bump epoch: before=%d after=%d", before, m.epoch)
	}
	if m.activeTab != tabInfo {
		t.Fatalf("default tab should be Info, got %v", m.activeTab)
	}
	if !m.loading {
		t.Fatalf("Focus should set loading for the default (Info) tab")
	}
	if cmd == nil {
		t.Fatalf("Focus should return a fetch command for the default tab")
	}
}

func TestSwitchTabChangesActiveTabAndTriggersLoad(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.focused = true
	before := m.epoch

	// Act
	_, cmd := m.switchTab(tabMessages)

	// Assert
	if m.activeTab != tabMessages {
		t.Fatalf("switchTab should change the active tab, got %v", m.activeTab)
	}
	if m.epoch != before+1 {
		t.Fatalf("switchTab should bump epoch")
	}
	if cmd == nil {
		t.Fatalf("switchTab should trigger the new tab's load command")
	}
}

func TestSwitchTabToLogResetsCursor(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.logLines = []string{"old"}
	m.nextID = 99

	// Act
	_, _ = m.switchTab(tabLog)

	// Assert
	if m.nextID != 0 {
		t.Fatalf("entering Log tab should reset nextID to 0, got %d", m.nextID)
	}
	if len(m.logLines) != 0 {
		t.Fatalf("entering Log tab should clear buffered lines")
	}
}

func TestSwitchTabSameTabIsNoop(t *testing.T) {
	// Arrange
	m := newTestModel()
	before := m.epoch

	// Act
	_, cmd := m.switchTab(tabInfo)

	// Assert
	if cmd != nil {
		t.Fatalf("switching to the current tab should be a no-op")
	}
	if m.epoch != before {
		t.Fatalf("no-op switch should not bump epoch")
	}
}

func TestInfoMsgPopulatesStateAndClearsLoading(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 4
	m.loading = true
	sections := []infoSection{{name: "ftl", rows: []kv{{"version", "6.0"}}}}

	// Act
	_, _ = m.Update(infoMsg{epoch: 4, sections: sections})

	// Assert
	if m.loading {
		t.Fatalf("infoMsg should clear loading")
	}
	if len(m.info) != 1 || m.info[0].name != "ftl" {
		t.Fatalf("infoMsg should populate info sections: %+v", m.info)
	}
}

func TestMessagesMsgPopulatesStateAndClearsLoading(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 2
	m.loading = true
	msgs := []pihole.DiagnosisMessage{{ID: 1, Type: "DNSMASQ_WARN", Plain: "warn"}}

	// Act
	_, _ = m.Update(messagesMsg{epoch: 2, messages: msgs, count: 1})

	// Assert
	if m.loading {
		t.Fatalf("messagesMsg should clear loading")
	}
	if len(m.messages) != 1 || m.msgCount != 1 {
		t.Fatalf("messagesMsg should populate messages+count: %+v count=%d", m.messages, m.msgCount)
	}
}

func TestNetworkMsgPopulatesStateAndClearsLoading(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 3
	m.loading = true
	devices := []pihole.NetworkDevice{{ID: 7, HWAddr: "aa:bb:cc:dd:ee:ff"}}

	// Act
	_, _ = m.Update(networkMsg{epoch: 3, devices: devices})

	// Assert
	if m.loading {
		t.Fatalf("networkMsg should clear loading")
	}
	if len(m.devices) != 1 || m.devices[0].ID != 7 {
		t.Fatalf("networkMsg should populate devices: %+v", m.devices)
	}
}

func TestDNSLogMsgAppendsLinesAndAdvancesCursor(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.activeTab = tabLog
	m.epoch = 6
	page := pihole.DNSLogPage{
		Log: []pihole.DNSLogLine{
			{Timestamp: 1_700_000_000, Message: "query A example.com"},
			{Timestamp: 1_700_000_001, Message: "reply example.com is 1.2.3.4"},
		},
		NextID: 50,
	}

	// Act
	_, _ = m.Update(dnsLogMsg{epoch: 6, page: page})

	// Assert
	if len(m.logLines) != 2 {
		t.Fatalf("dnsLogMsg should append its lines, got %d", len(m.logLines))
	}
	if m.nextID != 50 {
		t.Fatalf("dnsLogMsg should advance nextID to 50, got %d", m.nextID)
	}
}

func TestDNSLogMsgAppendsAcrossPolls(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.activeTab = tabLog
	m.epoch = 1
	m.logLines = []string{"existing"}

	// Act
	_, _ = m.Update(dnsLogMsg{epoch: 1, page: pihole.DNSLogPage{
		Log:    []pihole.DNSLogLine{{Timestamp: 1, Message: "new"}},
		NextID: 5,
	}})

	// Assert
	if len(m.logLines) != 2 {
		t.Fatalf("second poll should append, not replace: %v", m.logLines)
	}
}

func TestErrorMsgSetsBanner(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 5
	m.loading = true

	// Act
	_, _ = m.Update(messagesMsg{epoch: 5, err: &pihole.APIError{Status: 500, Message: "boom"}})

	// Assert
	if m.err == nil {
		t.Fatalf("an errored fetch should set the banner error")
	}
	if m.loading {
		t.Fatalf("an errored fetch should clear loading")
	}
}

func TestUpdateIgnoresStaleFetchResult(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 9
	m.loading = true

	// Act
	_, _ = m.Update(networkMsg{epoch: 8, devices: []pihole.NetworkDevice{{ID: 1}}})

	// Assert
	if len(m.devices) != 0 {
		t.Fatalf("stale-epoch result should be dropped")
	}
	if !m.loading {
		t.Fatalf("stale result should not clear loading")
	}
}

func TestLogTickReschedulesOnlyWhenFocusedAndOnLogTab(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 3

	cases := []struct {
		name    string
		focused bool
		active  tab
		want    bool
	}{
		{"focused on log", true, tabLog, true},
		{"unfocused on log", false, tabLog, false},
		{"focused on other tab", true, tabInfo, false},
	}

	for _, tc := range cases {
		m.focused = tc.focused
		m.activeTab = tc.active

		// Act
		_, cmd := m.Update(tickMsg{epoch: 3})

		// Assert
		if (cmd != nil) != tc.want {
			t.Fatalf("%s: reschedule=%v, want %v", tc.name, cmd != nil, tc.want)
		}
	}
}

func TestLogTickIgnoresStaleEpoch(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.focused = true
	m.activeTab = tabLog
	m.epoch = 7

	// Act
	_, cmd := m.Update(tickMsg{epoch: 6})

	// Assert
	if cmd != nil {
		t.Fatalf("a stale-epoch tick should not reschedule")
	}
}

func TestMessagesDeleteArmsConfirmThenRunsDelete(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.activeTab = tabMessages
	m.messages = []pihole.DiagnosisMessage{{ID: 11, Type: "WARN", Plain: "disk low"}}
	m.syncMsgRows()

	// Act: x arms the confirm dialog.
	_, _ = m.Update(keyPress("x"))

	// Assert
	if !m.confirm.Active {
		t.Fatalf("x on a message should open the confirm dialog")
	}
	if m.pendingOp == nil {
		t.Fatalf("x should arm a pending delete op")
	}
	if !m.pendingReload {
		t.Fatalf("a message delete should reload on success")
	}

	// Act: y confirms and dispatches the delete command.
	_, cmd := m.Update(keyPress("y"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("confirming should close the dialog")
	}
	if cmd == nil {
		t.Fatalf("confirming should dispatch the delete command")
	}
}

func TestNetworkDeleteArmsConfirmThenRunsDelete(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.activeTab = tabNetwork
	m.devices = []pihole.NetworkDevice{{
		ID:     42,
		HWAddr: "aa:bb:cc:dd:ee:ff",
		IPs:    []pihole.NetworkAddress{{IP: "10.0.0.5", Name: "nas"}},
	}}
	m.syncNetRows()

	// Act
	_, _ = m.Update(keyPress("x"))

	// Assert
	if !m.confirm.Active || m.pendingOp == nil {
		t.Fatalf("x on a device should arm the delete confirm")
	}

	// Act
	_, cmd := m.Update(keyPress("y"))

	// Assert
	if cmd == nil {
		t.Fatalf("confirming a device delete should dispatch the command")
	}
}

func TestActionsEnterArmsConfirmThenRunsAction(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.activeTab = tabActions
	m.actionCursor = 0

	// Act
	_, _ = m.Update(keyPress("enter"))

	// Assert
	if !m.confirm.Active {
		t.Fatalf("enter on an action should open the confirm dialog")
	}
	if !m.confirm.Danger {
		t.Fatalf("destructive actions should use the danger confirm")
	}
	if m.pendingReload {
		t.Fatalf("actions should not trigger a tab reload")
	}

	// Act
	_, cmd := m.Update(keyPress("y"))

	// Assert
	if cmd == nil {
		t.Fatalf("confirming an action should dispatch the action command")
	}
}

func TestActionsCursorMovesWithinBounds(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.activeTab = tabActions

	// Act: up at the top stays clamped.
	_, _ = m.Update(keyPress("k"))
	if m.actionCursor != 0 {
		t.Fatalf("cursor should clamp at the top, got %d", m.actionCursor)
	}

	// Act: down advances.
	_, _ = m.Update(keyPress("j"))
	if m.actionCursor != 1 {
		t.Fatalf("down should advance the cursor, got %d", m.actionCursor)
	}
}

func TestTabKeySwitchesToNextTab(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.focused = true

	// Act
	_, cmd := m.Update(keyPress("tab"))

	// Assert
	if m.activeTab != tabMessages {
		t.Fatalf("tab should advance to the next tab, got %v", m.activeTab)
	}
	if cmd == nil {
		t.Fatalf("tab switch should load the new tab")
	}
}

func TestBlurCancelsAndClearsPending(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.focused = true
	m.confirm = m.confirm.Show("x", "y", true)
	m.pendingOp = func(ctx context.Context, api *pihole.Client) error { return nil }

	// Act
	m.Blur()

	// Assert
	if m.pendingOp != nil {
		t.Fatalf("Blur should clear any armed pending op")
	}

	// Assert
	if m.focused {
		t.Fatalf("Blur should clear focus")
	}
	if m.confirm.Active {
		t.Fatalf("Blur should hide the confirm dialog")
	}
}

func TestFlattenInfoKeepsScalarsSortedAndSkipsNested(t *testing.T) {
	// Arrange
	data := map[string]any{
		"version":  "6.0.1",
		"uptime":   float64(1200),
		"dnssec":   true,
		"nested":   map[string]any{"a": 1},
		"list":     []any{1, 2},
		"fraction": 1.5,
	}

	// Act
	rows := flattenInfo(data)

	// Assert
	if len(rows) != 4 {
		t.Fatalf("nested map/slice should be skipped, got %d rows: %+v", len(rows), rows)
	}
	// Sorted by key: dnssec, fraction, uptime, version.
	if rows[0].key != "dnssec" || rows[0].val != "true" {
		t.Fatalf("first row wrong: %+v", rows[0])
	}
	if rows[1].key != "fraction" || rows[1].val != "1.5" {
		t.Fatalf("fractional float should keep decimals: %+v", rows[1])
	}
	if rows[2].key != "uptime" || rows[2].val != "1200" {
		t.Fatalf("integral float should drop decimals: %+v", rows[2])
	}
	if rows[3].key != "version" || rows[3].val != "6.0.1" {
		t.Fatalf("string row wrong: %+v", rows[3])
	}
}

func TestComputeMsgWidthsFitsWithinTotal(t *testing.T) {
	// Arrange
	total := 120

	// Act
	widths := computeMsgWidths(total)

	// Assert
	if len(widths) != len(msgColumnTitles) {
		t.Fatalf("expected %d columns, got %d", len(msgColumnTitles), len(widths))
	}
	sum := len(msgColumnTitles) * cellPad
	for _, w := range widths {
		sum += w
	}
	if sum > total {
		t.Fatalf("columns overflow total: sum=%d total=%d", sum, total)
	}
}

func TestSetSizeSafeAtTinySizes(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act / Assert: must not panic at degenerate sizes.
	m.SetSize(0, 0)
	m.SetSize(1, 1)
	m.SetSize(-5, -5)
}

func TestDestructiveHintAugmentsForbiddenError(t *testing.T) {
	// Arrange
	err := &pihole.APIError{Status: 403, Key: "forbidden", Message: "destructive action disabled"}

	// Act
	got := destructiveHint(err)

	// Assert
	if got == err {
		t.Fatalf("a 403 destructive error should be augmented")
	}
	if got == nil {
		t.Fatalf("augmented error should be non-nil")
	}
}

func TestDestructiveHintLeavesOtherErrorsUnchanged(t *testing.T) {
	// Arrange
	err := &pihole.APIError{Status: 404, Key: "not_found", Message: "missing"}

	// Act
	got := destructiveHint(err)

	// Assert
	if got != err {
		t.Fatalf("a non-destructive error should be returned unchanged")
	}
}

func TestTabNextPrevWrap(t *testing.T) {
	// Arrange / Act / Assert
	if tabActions.next() != tabInfo {
		t.Fatalf("next past the last tab should wrap to the first")
	}
	if tabInfo.prev() != tabActions {
		t.Fatalf("prev before the first tab should wrap to the last")
	}
}

var _ core.Screen = (*Model)(nil)
var _ tea.Model = (*Model)(nil)
