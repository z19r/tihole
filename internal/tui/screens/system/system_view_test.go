package system

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/pihole"
)

// renderedModel builds a focused, sized model ready for View() rendering.
func renderedModel() *Model {
	m := newTestModel()
	m.SetSize(120, 30)
	m.focused = true
	return m
}

// sampleMessages returns two diagnosis messages for table rendering.
func sampleMessages() []pihole.DiagnosisMessage {
	return []pihole.DiagnosisMessage{
		{
			ID:        1,
			Type:      "DNSMASQ_WARN",
			Plain:     "config warning",
			Timestamp: 1_700_000_000,
		},
		{
			ID:        2,
			Type:      "SUBNET",
			Plain:     "overlapping subnet detected",
			Timestamp: 1_700_000_042,
		},
	}
}

// sampleDevices returns two network devices for table rendering.
func sampleDevices() []pihole.NetworkDevice {
	return []pihole.NetworkDevice{
		{
			ID:         7,
			HWAddr:     "aa:bb:cc:dd:ee:ff",
			MACVendor:  "Acme",
			NumQueries: 12,
			LastQuery:  1_700_000_000,
			IPs:        []pihole.NetworkAddress{{IP: "10.0.0.5", Name: "nas"}},
		},
		{ID: 8, HWAddr: "11:22:33:44:55:66"},
	}
}

func TestInitReturnsNilCmd(t *testing.T) {
	if cmd := newTestModel().Init(); cmd != nil {
		t.Fatalf("Init should return nil until focus")
	}
}

func TestTitleIsSystem(t *testing.T) {
	if got := newTestModel().Title(); got != "System" {
		t.Fatalf("unexpected title: %q", got)
	}
}

func TestHelpReturnsBindings(t *testing.T) {
	if got := len(newTestModel().Help()); got != 5 {
		t.Fatalf("expected 5 help bindings, got %d", got)
	}
}

func TestViewInfoTabRendersSectionsAndTabBar(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.info = []infoSection{
		{name: "ftl", rows: []kv{{"version", "6.0.1"}, {"dnssec", "true"}}},
	}

	// Act
	out := m.View().Content

	// Assert
	if out == "" {
		t.Fatalf("View should render non-empty output")
	}
	for _, want := range []string{"System", "Info", "Messages", "Network", "Log tail", "Actions", "ftl", "version", "refresh"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Info view missing %q:\n%s", want, out)
		}
	}
}

func TestViewInfoLoadingAndEmptyStates(t *testing.T) {
	// Arrange: loading with no data.
	m := renderedModel()
	m.loading = true

	// Act / Assert
	if out := m.View().Content; !strings.Contains(out, "loading info") {
		t.Fatalf("empty+loading Info should show spinner text:\n%s", out)
	}

	// Arrange: idle with no data.
	m.loading = false
	if out := m.View().Content; !strings.Contains(out, "no info available") {
		t.Fatalf("empty+idle Info should show placeholder:\n%s", out)
	}
}

func TestViewMessagesTabRendersTable(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.activeTab = tabMessages
	m.messages = sampleMessages()
	m.syncMsgRows()

	// Act
	out := m.View().Content

	// Assert
	for _, want := range []string{"Messages", "DNSMASQ_WARN", "config warning"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Messages view missing %q:\n%s", want, out)
		}
	}
}

func TestViewMessagesLoadingAndEmptyStates(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.activeTab = tabMessages
	m.loading = true

	if out := m.View().Content; !strings.Contains(out, "loading messages") {
		t.Fatalf("empty+loading Messages should show spinner text:\n%s", out)
	}

	m.loading = false
	if out := m.View().Content; !strings.Contains(
		out,
		"no diagnosis messages",
	) {
		t.Fatalf("empty+idle Messages should show placeholder:\n%s", out)
	}
}

func TestViewNetworkTabRendersTable(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.activeTab = tabNetwork
	m.devices = sampleDevices()
	m.syncNetRows()

	// Act
	out := m.View().Content

	// Assert
	for _, want := range []string{"Network", "HWAddr", "nas", "10.0.0.5"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Network view missing %q:\n%s", want, out)
		}
	}
}

func TestViewNetworkLoadingAndEmptyStates(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.activeTab = tabNetwork
	m.loading = true

	if out := m.View().Content; !strings.Contains(out, "loading devices") {
		t.Fatalf("empty+loading Network should show spinner text:\n%s", out)
	}

	m.loading = false
	if out := m.View().Content; !strings.Contains(out, "no network devices") {
		t.Fatalf("empty+idle Network should show placeholder:\n%s", out)
	}
}

func TestViewLogTabRendersViewportAndStates(t *testing.T) {
	// Arrange: waiting state.
	m := renderedModel()
	m.activeTab = tabLog
	if out := m.View().Content; !strings.Contains(
		out,
		"waiting for log lines",
	) {
		t.Fatalf("empty+idle Log should show waiting text:\n%s", out)
	}

	// Arrange: loading state.
	m.loading = true
	if out := m.View().Content; !strings.Contains(out, "reading log") {
		t.Fatalf("empty+loading Log should show spinner text:\n%s", out)
	}

	// Arrange: populated viewport.
	m.loading = false
	m.appendLog(pihole.DNSLogPage{
		Log: []pihole.DNSLogLine{
			{Timestamp: 1_700_000_000, Message: "query A example.com"},
		},
		NextID: 5,
	})
	if out := m.View().Content; !strings.Contains(out, "example.com") {
		t.Fatalf("populated Log should render the tail:\n%s", out)
	}
}

func TestViewActionsTabRendersMenu(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.activeTab = tabActions
	m.actionCursor = 1

	// Act
	out := m.View().Content

	// Assert
	for _, want := range []string{"Destructive actions", "Restart DNS", "Flush logs", "Flush network", "allow_destructive"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Actions view missing %q:\n%s", want, out)
		}
	}
}

func TestViewConfirmDialogOverlaysBody(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.activeTab = tabActions
	m.confirm = m.confirm.Show("Restart DNS?", "Brief outage.", true)

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "Restart DNS?") {
		t.Fatalf("active confirm dialog should overlay the body:\n%s", out)
	}
}

func TestViewErrorBannerRendered(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.err = &pihole.APIError{Status: 500, Message: "connection refused"}

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "error:") {
		t.Fatalf("a set error should render the banner:\n%s", out)
	}
}

func TestViewSafeAtTinySizes(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act / Assert: must not panic while rendering degenerate sizes.
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {-5, -5}} {
		m.SetSize(sz[0], sz[1])
		for t2 := tab(0); t2 < tabCount; t2++ {
			m.activeTab = t2
			_ = m.View()
		}
	}
}

func TestViewRendersEveryTabWithoutPanic(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.info = []infoSection{{name: "ftl", rows: []kv{{"version", "6.0"}}}}
	m.messages = sampleMessages()
	m.syncMsgRows()
	m.devices = sampleDevices()
	m.syncNetRows()
	m.appendLog(
		pihole.DNSLogPage{
			Log:    []pihole.DNSLogLine{{Timestamp: 1, Message: "x"}},
			NextID: 2,
		},
	)

	// Act / Assert
	for t2 := tab(0); t2 < tabCount; t2++ {
		m.activeTab = t2
		if out := m.View().Content; out == "" {
			t.Fatalf("tab %v should render non-empty content", t2)
		}
	}
}

func TestMessagesDismissAllArmsConfirm(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.activeTab = tabMessages
	m.messages = sampleMessages()
	m.syncMsgRows()

	// Act
	_, _ = m.Update(keyPress("X"))

	// Assert
	if !m.confirm.Active {
		t.Fatalf("X should arm the dismiss-all confirm dialog")
	}
	if m.pendingOp == nil || !m.pendingReload {
		t.Fatalf("dismiss-all should arm a reloading pending op")
	}
	if !strings.Contains(m.confirm.Message, "2 messages") {
		t.Fatalf(
			"dismiss-all confirm should mention the count, got %q",
			m.confirm.Message,
		)
	}
}

func TestMessagesDismissAllNoopWhenEmpty(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.activeTab = tabMessages

	// Act
	_, _ = m.Update(keyPress("X"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("X with no messages should be a no-op")
	}
}

func TestConfirmCancelClearsPending(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.activeTab = tabActions
	_, _ = m.Update(keyPress("enter")) // arm confirm

	// Act
	_, cmd := m.Update(keyPress("n"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("n should dismiss the confirm dialog")
	}
	if m.pendingOp != nil {
		t.Fatalf("n should clear the pending op")
	}
	if cmd != nil {
		t.Fatalf("cancelling should not dispatch a command")
	}
}

func TestRefreshKeyBumpsEpochAndLoads(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.activeTab = tabInfo
	before := m.epoch

	// Act
	_, cmd := m.Update(keyPress("r"))

	// Assert
	if m.epoch != before+1 {
		t.Fatalf("r should bump the epoch for a refetch")
	}
	if cmd == nil {
		t.Fatalf("r should return the active tab's load command")
	}
}

func TestLeftKeySwitchesToPrevTab(t *testing.T) {
	// Arrange
	m := renderedModel()

	// Act
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})

	// Assert
	if m.activeTab != tabActions {
		t.Fatalf("left from Info should wrap to Actions, got %v", m.activeTab)
	}
}

func TestMutationReloadTriggersActiveLoad(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.activeTab = tabMessages
	m.epoch = 4

	// Act
	_, cmd := m.Update(mutationMsg{epoch: 4, reload: true})

	// Assert
	if cmd == nil {
		t.Fatalf(
			"a successful reloading mutation should refetch the active tab",
		)
	}
	if m.epoch != 5 {
		t.Fatalf("reloading mutation should bump the epoch, got %d", m.epoch)
	}
}

func TestMutationErrorSurfacesDestructiveHint(t *testing.T) {
	// Arrange
	m := renderedModel()
	m.epoch = 2
	m.loading = true

	// Act
	_, _ = m.Update(
		mutationMsg{
			epoch: 2,
			err: &pihole.APIError{
				Status:  403,
				Message: "destructive disabled",
			},
		},
	)

	// Assert
	if m.loading {
		t.Fatalf("an errored mutation should clear loading")
	}
	if m.err == nil || !strings.Contains(m.err.Error(), "allow_destructive") {
		t.Fatalf(
			"a 403 mutation error should carry the destructive hint, got %v",
			m.err,
		)
	}
}

func TestLabelReturnsNamesAndDefault(t *testing.T) {
	if tabMessages.label() != "Messages" {
		t.Fatalf("tabMessages label wrong")
	}
	if tab(99).label() != "Info" {
		t.Fatalf("unknown tab should default to Info label")
	}
}

func TestFirstAddressFallbacks(t *testing.T) {
	// No IPs.
	if ip, name := firstAddress(pihole.NetworkDevice{}); ip != "-" ||
		name != "-" {
		t.Fatalf("no-IP device should yield dashes, got %q/%q", ip, name)
	}
	// Blank fields.
	d := pihole.NetworkDevice{IPs: []pihole.NetworkAddress{{IP: "", Name: ""}}}
	if ip, name := firstAddress(d); ip != "-" || name != "-" {
		t.Fatalf("blank IP/name should yield dashes, got %q/%q", ip, name)
	}
	// Populated.
	d = pihole.NetworkDevice{
		IPs: []pihole.NetworkAddress{{IP: "10.0.0.1", Name: "box"}},
	}
	if ip, name := firstAddress(d); ip != "10.0.0.1" || name != "box" {
		t.Fatalf("populated address wrong, got %q/%q", ip, name)
	}
}

func TestTruncateEdgeCases(t *testing.T) {
	cases := []struct {
		in    string
		width int
		want  string
	}{
		{"hello", 0, ""},
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 1, ellipsis},
		{"hello", 3, "he" + ellipsis},
		{"héllo wörld", 4, "hél" + ellipsis},
	}
	for _, c := range cases {
		if got := truncate(c.in, c.width); got != c.want {
			t.Fatalf("truncate(%q,%d)=%q want %q", c.in, c.width, got, c.want)
		}
	}
}

func TestFormatUnixTimeZeroIsDash(t *testing.T) {
	if got := formatUnixTime(0); got != "-" {
		t.Fatalf("zero timestamp should render a dash, got %q", got)
	}
	if got := formatUnixTime(-1); got != "-" {
		t.Fatalf("negative timestamp should render a dash, got %q", got)
	}
	if got := formatUnixTime(1_700_000_000); len(got) != 8 {
		t.Fatalf("positive timestamp should render HH:MM:SS, got %q", got)
	}
}

func TestScalarStringVariants(t *testing.T) {
	if got := scalarString(1.5); got != "1.5" {
		t.Fatalf("fractional float wrong: %q", got)
	}
	if got := scalarString(float64(1200)); got != "1200" {
		t.Fatalf("integral float wrong: %q", got)
	}
	if got := scalarString(false); got != "false" {
		t.Fatalf("bool false wrong: %q", got)
	}
	if got := scalarString([]int{1}); got == "" {
		t.Fatalf("non-scalar should still stringify, got %q", got)
	}
}

func TestMinIntAndClampMin(t *testing.T) {
	if minInt(3, 7) != 3 || minInt(9, 2) != 2 {
		t.Fatalf("minInt wrong")
	}
	if clampMin(2, 5) != 5 || clampMin(9, 5) != 9 {
		t.Fatalf("clampMin wrong")
	}
}

func TestDestructiveHintNilPassThrough(t *testing.T) {
	if destructiveHint(nil) != nil {
		t.Fatalf("nil error should pass through as nil")
	}
}
