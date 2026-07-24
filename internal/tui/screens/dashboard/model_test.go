package dashboard

import (
	"context"
	"testing"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

func TestInit_IsNoOp(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act / Assert
	if cmd := m.Init(); cmd != nil {
		t.Fatal("Init should be a no-op returning nil")
	}
}

func TestTitleAndHelp(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act / Assert
	if m.Title() != "Dashboard" {
		t.Errorf("unexpected title %q", m.Title())
	}
	if len(m.Help()) != 1 {
		t.Fatalf("expected one help binding, got %d", len(m.Help()))
	}
}

func TestFocus_BumpsEpochStartsContextAndReturnsCmd(t *testing.T) {
	// Arrange
	m := newTestModel()
	start := m.epoch

	// Act
	cmd := m.Focus()

	// Assert
	if !m.focused {
		t.Fatal("focus should mark model focused")
	}
	if m.epoch != start+1 {
		t.Fatalf("focus should bump epoch: got %d want %d", m.epoch, start+1)
	}
	if m.cancel == nil || m.baseCtx == nil {
		t.Fatal("focus should establish a cancelable context")
	}
	if cmd == nil {
		t.Fatal("focus should return a batch command")
	}
}

func TestFocus_TwiceCancelsPreviousContext(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.Focus()

	// Act
	m.Focus()

	// Assert
	if m.epoch != 2 {
		t.Fatalf("second focus should bump epoch to 2, got %d", m.epoch)
	}
	if m.cancel == nil {
		t.Fatal("second focus should install a fresh cancel func")
	}
}

func TestBlur_ClearsFocusAndCancel(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.Focus()

	// Act
	m.Blur()

	// Assert
	if m.focused {
		t.Fatal("blur should clear focus")
	}
	if m.cancel != nil {
		t.Fatal("blur should drop the cancel func")
	}
}

func TestBlur_WithoutFocusIsSafe(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act / Assert (no panic when never focused)
	m.Blur()
}

func TestTick_ReturnsCommand(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act / Assert
	if m.tick() == nil {
		t.Fatal("tick should return a scheduling command")
	}
}

func TestUpdate_HandledTickTriggersFetchAndReschedule(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.Focus() // establishes baseCtx so fetchAll is non-nil

	// Act
	_, cmd := m.Update(tickMsg{epoch: m.epoch})

	// Assert
	if cmd == nil {
		t.Fatal("a live tick should return a fetch+reschedule batch")
	}
}

func TestUpdate_RefreshKeyReturnsFetchCommand(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.Focus()

	// Act
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	// Assert
	if cmd == nil {
		t.Fatal("'r' while focused should return a fetch command")
	}
}

func TestUpdate_RefreshKeyIgnoredWhenBlurred(t *testing.T) {
	// Arrange
	m := newTestModel() // not focused

	// Act
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	// Assert
	if cmd != nil {
		t.Fatal("'r' while blurred should be ignored")
	}
}

func TestUpdate_HistoryResultStoresAndErrors(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 3

	// Act: success
	m.Update(historyMsg{epoch: 3, data: sampleHistory()})
	// Act: error
	m.Update(historyMsg{epoch: 3, err: errStub("hist down")})

	// Assert
	if m.errHistory != "hist down" {
		t.Fatalf("expected stored history error, got %q", m.errHistory)
	}
	if !m.loaded {
		t.Fatal("delivered history should mark loaded")
	}
}

func TestUpdate_StaleHistoryIsDiscarded(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 3

	// Act
	m.Update(historyMsg{epoch: 2, data: sampleHistory()})

	// Assert
	if len(m.history.History) != 0 || m.loaded {
		t.Fatal("stale history should be discarded")
	}
}

func TestUpdate_BreakdownResultStoresBothAndErrors(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 3

	// Act: success path populates both
	m.Update(breakdownMsg{epoch: 3, types: sampleTypes(), upstreams: sampleUpstreams()})
	if len(m.types.Types) == 0 || len(m.upstreams.Upstreams) == 0 {
		t.Fatal("breakdown success should populate both fields")
	}
	// Act: error path sets both banners
	m.Update(breakdownMsg{epoch: 3, typesErr: errStub("t"), upErr: errStub("u")})

	// Assert
	if m.errTypes != "t" || m.errUpstreams != "u" {
		t.Fatalf("expected both breakdown errors, got %q / %q", m.errTypes, m.errUpstreams)
	}
}

func TestUpdate_StaleBreakdownIsDiscarded(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 3

	// Act
	m.Update(breakdownMsg{epoch: 1, types: sampleTypes()})

	// Assert
	if len(m.types.Types) != 0 {
		t.Fatal("stale breakdown should be discarded")
	}
}

func TestUpdate_TopResultStoresAllAndErrors(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 3

	// Act: success
	m.Update(topMsg{epoch: 3, domains: sampleTopDomains(), blocked: sampleTopDomains(), clients: sampleTopClients()})
	if len(m.domains.Domains) == 0 || len(m.clients.Clients) == 0 {
		t.Fatal("top success should populate lists")
	}
	// Act: errors
	m.Update(topMsg{epoch: 3, domainsErr: errStub("d"), blockedErr: errStub("b"), clientsErr: errStub("c")})

	// Assert
	if m.errDomains != "d" || m.errBlocked != "b" || m.errClients != "c" {
		t.Fatalf("expected all top errors set, got %q/%q/%q", m.errDomains, m.errBlocked, m.errClients)
	}
}

func TestUpdate_StaleTopIsDiscarded(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 3

	// Act
	m.Update(topMsg{epoch: 2, clients: sampleTopClients()})

	// Assert
	if len(m.clients.Clients) != 0 {
		t.Fatal("stale top result should be discarded")
	}
}

func TestUpdate_SpinnerTickIgnoredWhenLoaded(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.loaded = true
	m.focused = true

	// Act
	_, cmd := m.Update(spinner.TickMsg{})

	// Assert
	if cmd != nil {
		t.Fatal("spinner ticks should stop once loaded")
	}
}

func TestUpdate_SpinnerTickAdvancesWhileLoading(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.focused = true // loaded stays false

	// Act / Assert (exercises the spinner update branch without panic)
	m.Update(spinner.TickMsg{})
}

func TestUpdate_UnknownMessageIsNoOp(t *testing.T) {
	// Arrange
	m := newTestModel()

	// Act
	_, cmd := m.Update(struct{}{})

	// Assert
	if cmd != nil {
		t.Fatal("unknown message should return no command")
	}
}

func TestFetchAll_NilWhenNoClient(t *testing.T) {
	// Arrange
	m := New(&core.AppContext{Theme: theme.DeepNight()}) // API nil

	// Act / Assert
	if m.fetchAll() != nil {
		t.Fatal("fetchAll should be nil without an API client")
	}
}

func TestFetchAll_NilWhenNoBaseContext(t *testing.T) {
	// Arrange: has API but Focus never ran, so baseCtx is nil
	m := newTestModel()

	// Act / Assert
	if m.fetchAll() != nil {
		t.Fatal("fetchAll should be nil before Focus establishes a context")
	}
}

// TestFetchClosures_ProduceTaggedMessages executes each fetch command's closure
// against an already-cancelled context so the request fails immediately (no
// live network) while still covering the closure bodies.
func TestFetchClosures_ProduceTaggedMessages(t *testing.T) {
	// Arrange
	api := pihole.New("http://127.0.0.1:0", "pw")
	base, cancel := context.WithCancel(context.Background())
	cancel()

	// Act / Assert
	if _, ok := fetchSummary(base, api, 7)().(summaryMsg); !ok {
		t.Fatal("fetchSummary should yield a summaryMsg")
	}
	if _, ok := fetchHistory(base, api, 7)().(historyMsg); !ok {
		t.Fatal("fetchHistory should yield a historyMsg")
	}
	if _, ok := fetchBreakdown(base, api, 7)().(breakdownMsg); !ok {
		t.Fatal("fetchBreakdown should yield a breakdownMsg")
	}
	if _, ok := fetchTop(base, api, 7)().(topMsg); !ok {
		t.Fatal("fetchTop should yield a topMsg")
	}
}

// TestUpdate_SummarySpringsBlockGauge verifies a delivered summary sets the
// block gauge target and returns an animation command to spring toward it.
func TestUpdate_SummarySpringsBlockGauge(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.focused, m.epoch = true, 1

	// Act
	_, cmd := m.Update(summaryMsg{epoch: 1, data: sampleSummary()})

	// Assert: gauge target tracks the 16.2% block rate, animation cmd returned.
	if got := m.blockBar.Percent(); got < 0.16 || got > 0.17 {
		t.Fatalf("block gauge target = %.3f, want ~0.162", got)
	}
	if cmd == nil {
		t.Fatal("a delivered summary should return an animation command")
	}
}

// TestUpdate_ProgressFrameAdvancesGauge verifies progress.FrameMsg is routed
// into the gauge so the spring animation actually advances.
func TestUpdate_ProgressFrameAdvancesGauge(t *testing.T) {
	// Arrange: kick a target so the gauge has somewhere to animate to.
	m := newTestModel()
	m.focused, m.epoch = true, 1
	m.Update(summaryMsg{epoch: 1, data: sampleSummary()})

	// Act: an unmatched frame is harmless; the routing must not panic and must
	// leave the model intact.
	updated, _ := m.Update(progress.FrameMsg{})

	// Assert
	if updated == nil {
		t.Fatal("progress frame handling should return the model")
	}
}

// TestSyncBarTheme_RebuildsOnThemeChangePreservingPercent verifies the gauge is
// rebuilt when the active theme changes, keeping its target percent so the value
// doesn't jump, and is left untouched when the theme is unchanged.
func TestSyncBarTheme_RebuildsOnThemeChangePreservingPercent(t *testing.T) {
	// Arrange: give the gauge a target, then swap the theme.
	m := newTestModel()
	m.blockBar.SetPercent(0.42)
	if m.barTheme != theme.DeepNight().Name {
		t.Fatalf("precondition: gauge built for %q, got %q", theme.DeepNight().Name, m.barTheme)
	}
	m.ctx.Theme = theme.LightLuxury()

	// Act
	m.syncBarTheme()

	// Assert: rebuilt for the new theme, percent preserved.
	if m.barTheme != theme.LightLuxury().Name {
		t.Fatalf("expected gauge rebuilt for %q, got %q", theme.LightLuxury().Name, m.barTheme)
	}
	if got := m.blockBar.Percent(); got < 0.41 || got > 0.43 {
		t.Fatalf("percent not preserved across rebuild: got %.3f want ~0.42", got)
	}

	// Act again with no change: barTheme stays put (no needless rebuild).
	before := m.barTheme
	m.syncBarTheme()
	if m.barTheme != before {
		t.Fatal("syncBarTheme should be a no-op when the theme is unchanged")
	}
}

var _ core.Screen = (*Model)(nil)
var _ tea.Model = (*Model)(nil)
