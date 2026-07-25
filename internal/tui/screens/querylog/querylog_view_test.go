package querylog

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

// newModel builds a query-log model with a non-nil (offline) client and a real
// content area. No login is performed and no network command is executed by
// these tests — assertions are on state transitions and rendered output.
func newModel() *Model {
	th := theme.DeepNight()
	api := pihole.New("http://x", "pw")
	m := New(&core.AppContext{API: api, Theme: th, Keys: core.DefaultKeyMap()})
	m.SetSize(120, 30)
	return m
}

// keyPress builds a KeyPressMsg whose String() matches the given keystroke.
func keyPress(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEsc}
	default:
		r := []rune(s)[0]
		return tea.KeyPressMsg{Code: r, Text: s}
	}
}

func sampleQueries() []pihole.Query {
	return []pihole.Query{
		{
			Time:     1_700_000_000,
			Type:     "A",
			Domain:   "ads.example.com",
			Status:   "GRAVITY",
			Client:   pihole.QueryClient{IP: "10.0.0.5", Name: "laptop"},
			Upstream: "",
			Reply:    pihole.QueryReply{Type: "IP", Time: 0.012},
		},
		{
			Time:     1_700_000_042,
			Type:     "AAAA",
			Domain:   "good.example.org",
			Status:   "FORWARDED",
			Client:   pihole.QueryClient{IP: "10.0.0.9"},
			Upstream: "1.1.1.1#53",
			CNAME:    "cdn.example.net",
			DNSSEC:   "SECURE",
			Reply:    pihole.QueryReply{Type: "CNAME", Time: 0.003},
		},
	}
}

func TestInitReturnsNoCmd(t *testing.T) {
	m := newModel()
	if cmd := m.Init(); cmd != nil {
		t.Fatalf("Init should return nil until focus")
	}
}

func TestTitleIsQueryLog(t *testing.T) {
	if got := newModel().Title(); got != "Query Log" {
		t.Fatalf("unexpected title: %q", got)
	}
}

func TestHelpReturnsBindings(t *testing.T) {
	if got := len(newModel().Help()); got != 5 {
		t.Fatalf("expected 5 help bindings, got %d", got)
	}
}

func TestFocusSetsLoadingBumpsEpochAndReturnsCmd(t *testing.T) {
	// Arrange
	m := newModel()
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
	if !m.loading {
		t.Fatalf("Focus should set loading")
	}
	if m.err != nil {
		t.Fatalf("Focus should clear any prior error")
	}
	if cmd == nil {
		t.Fatalf("Focus should return a fetch/tick command")
	}
}

func TestBlurClearsFocusAndCancelsInflight(t *testing.T) {
	// Arrange: Focus starts a fetch which stores a cancel func.
	m := newModel()
	m.Focus()
	if m.cancel == nil {
		t.Fatalf("precondition: Focus should arm an in-flight cancel")
	}

	// Act
	m.Blur()

	// Assert
	if m.focused {
		t.Fatalf("Blur should clear focus")
	}
	if m.searching {
		t.Fatalf("Blur should clear searching")
	}
	if m.cancel != nil {
		t.Fatalf("Blur should cancel and clear the in-flight request")
	}
}

func TestQueriesMsgPopulatesStateAndClearsLoading(t *testing.T) {
	// Arrange
	m := newModel()
	m.epoch = 3
	m.loading = true
	page := pihole.QueriesPage{
		Queries:         sampleQueries(),
		RecordsFiltered: 2,
		RecordsTotal:    500,
	}

	// Act
	_, _ = m.Update(queriesMsg{epoch: 3, page: page})

	// Assert
	if m.loading {
		t.Fatalf("queriesMsg should clear loading")
	}
	if len(m.queries) != 2 {
		t.Fatalf("queriesMsg should store queries, got %d", len(m.queries))
	}
	if m.filtered != 2 || m.total != 500 {
		t.Fatalf("queriesMsg should carry counts: filtered=%d total=%d", m.filtered, m.total)
	}
	if m.err != nil {
		t.Fatalf("a successful result should clear the error")
	}
}

func TestQueriesMsgErrorSetsBanner(t *testing.T) {
	// Arrange
	m := newModel()
	m.epoch = 4
	m.loading = true

	// Act
	_, _ = m.Update(queriesMsg{epoch: 4, err: errors.New("boom")})

	// Assert
	if m.err == nil {
		t.Fatalf("an errored fetch should set the banner")
	}
	if m.loading {
		t.Fatalf("an errored fetch should clear loading")
	}
	if len(m.queries) != 0 {
		t.Fatalf("an errored fetch should not populate rows")
	}
}

func TestSpinnerTickAdvancesOnlyWhileLoading(t *testing.T) {
	// Arrange: use this model's own spinner tick so the ID matches.
	m := newModel()
	msg := m.spinner.Tick()

	// Act / Assert: ignored when not loading.
	m.loading = false
	if _, cmd := m.Update(msg); cmd != nil {
		t.Fatalf("spinner tick should be ignored when not loading")
	}

	// Act / Assert: advances when loading.
	m.loading = true
	if _, cmd := m.Update(msg); cmd == nil {
		t.Fatalf("spinner tick should reschedule while loading")
	}
}

func TestTickSchedulesNextPoll(t *testing.T) {
	// Arrange
	m := newModel()

	// Act
	cmd := m.tick()

	// Assert
	if cmd == nil {
		t.Fatalf("tick should return a scheduled poll command")
	}
}

func TestSlashOpensSearchSeededWithFilter(t *testing.T) {
	// Arrange
	m := newModel()
	m.filterDomain = "seed.example.com"

	// Act
	_, cmd := m.Update(keyPress("/"))

	// Assert
	if !m.searching {
		t.Fatalf("'/' should enter search mode")
	}
	if m.search.Value() != "seed.example.com" {
		t.Fatalf("search box should seed with the active filter, got %q", m.search.Value())
	}
	if cmd == nil {
		t.Fatalf("'/' should return the input focus command")
	}
}

func TestSearchEnterAppliesFilterAndRefetches(t *testing.T) {
	// Arrange
	m := newModel()
	_, _ = m.Update(keyPress("/"))
	m.search.SetValue("  blocked.example.com  ")
	before := m.epoch

	// Act
	_, cmd := m.Update(keyPress("enter"))

	// Assert
	if m.filterDomain != "blocked.example.com" {
		t.Fatalf("enter should trim+apply the filter, got %q", m.filterDomain)
	}
	if m.searching {
		t.Fatalf("enter should leave search mode")
	}
	if m.epoch != before+1 {
		t.Fatalf("applying a filter should bump the epoch for a refetch")
	}
	if !m.loading {
		t.Fatalf("applying a filter should set loading")
	}
	if cmd == nil {
		t.Fatalf("applying a filter should return a refetch command")
	}
}

func TestSearchEscRestoresPriorFilter(t *testing.T) {
	// Arrange
	m := newModel()
	m.filterDomain = "keep.example.com"
	_, _ = m.Update(keyPress("/"))
	m.search.SetValue("throwaway")

	// Act
	_, cmd := m.Update(keyPress("esc"))

	// Assert
	if m.searching {
		t.Fatalf("esc should leave search mode")
	}
	if m.filterDomain != "keep.example.com" {
		t.Fatalf("esc should not change the applied filter")
	}
	if m.search.Value() != "keep.example.com" {
		t.Fatalf("esc should reset the box to the applied filter, got %q", m.search.Value())
	}
	if cmd != nil {
		t.Fatalf("esc out of search should not issue a command")
	}
}

func TestSearchTypingEditsBox(t *testing.T) {
	// Arrange
	m := newModel()
	_, _ = m.Update(keyPress("/"))

	// Act
	_, _ = m.Update(keyPress("x"))

	// Assert
	if !strings.Contains(m.search.Value(), "x") {
		t.Fatalf("typing in search mode should edit the box, got %q", m.search.Value())
	}
}

func TestEnterOpensDetailForCursorRow(t *testing.T) {
	// Arrange
	m := newModel()
	_, _ = m.Update(queriesMsg{epoch: m.epoch, page: pihole.QueriesPage{Queries: sampleQueries()}})

	// Act
	_, _ = m.Update(keyPress("enter"))

	// Assert
	if m.detail == nil {
		t.Fatalf("enter on a row should open the detail pane")
	}
	if m.detail.Domain != "ads.example.com" {
		t.Fatalf("detail should target the cursor row, got %q", m.detail.Domain)
	}
}

func TestDetailEscCloses(t *testing.T) {
	// Arrange
	m := newModel()
	q := sampleQueries()[0]
	m.detail = &q

	// Act
	_, _ = m.Update(keyPress("esc"))

	// Assert
	if m.detail != nil {
		t.Fatalf("esc should close the detail pane")
	}
}

func TestNavigationKeyPassesToTableWithoutPanic(t *testing.T) {
	// Arrange
	m := newModel()
	_, _ = m.Update(queriesMsg{epoch: m.epoch, page: pihole.QueriesPage{Queries: sampleQueries()}})

	// Act / Assert (no panic, routes to table)
	_, _ = m.Update(keyPress("j"))
	_, _ = m.Update(keyPress("k"))
}

func TestViewRendersTableWithHeaderAndFooter(t *testing.T) {
	// Arrange
	m := newModel()
	m.focused = true
	_, _ = m.Update(queriesMsg{epoch: m.epoch, page: pihole.QueriesPage{
		Queries:         sampleQueries(),
		RecordsFiltered: 2,
		RecordsTotal:    500,
	}})

	// Act
	out := m.View().Content

	// Assert
	if out == "" {
		t.Fatalf("View should render non-empty output")
	}
	for _, want := range []string{"Query Log", "all domains", "total", "navigate"} {
		if !strings.Contains(out, want) {
			t.Fatalf("View missing %q:\n%s", want, out)
		}
	}
}

func TestViewShowsFilterChipWhenFiltering(t *testing.T) {
	// Arrange
	m := newModel()
	m.filterDomain = "ads.example.com"
	_, _ = m.Update(queriesMsg{epoch: m.epoch, page: pihole.QueriesPage{Queries: sampleQueries()}})

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "domain: ads.example.com") {
		t.Fatalf("View should show the active domain chip:\n%s", out)
	}
}

func TestViewLoadingStateShowsSpinnerText(t *testing.T) {
	// Arrange
	m := newModel()
	m.loading = true

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "loading queries") {
		t.Fatalf("empty+loading should render a loading indicator:\n%s", out)
	}
}

func TestViewEmptyStateShowsNoMatches(t *testing.T) {
	// Arrange
	m := newModel()
	m.loading = false

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "no queries match") {
		t.Fatalf("empty+idle should render the no-match message:\n%s", out)
	}
}

func TestViewSearchingShowsInput(t *testing.T) {
	// Arrange
	m := newModel()
	_, _ = m.Update(keyPress("/"))

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "domain") { // placeholder / prompt visible
		t.Fatalf("searching should render the input line:\n%s", out)
	}
}

func TestViewErrorBannerRendered(t *testing.T) {
	// Arrange
	m := newModel()
	m.err = errors.New("connection refused")

	// Act
	out := m.View().Content

	// Assert
	if !strings.Contains(out, "error:") {
		t.Fatalf("a set error should render the banner:\n%s", out)
	}
}

func TestViewDetailPaneRendered(t *testing.T) {
	// Arrange
	m := newModel()
	q := sampleQueries()[1] // has CNAME, upstream, DNSSEC
	m.detail = &q

	// Act
	out := m.View().Content

	// Assert
	for _, want := range []string{"Query detail", "Domain", "Upstream", "DNSSEC", "esc"} {
		if !strings.Contains(out, want) {
			t.Fatalf("detail pane missing %q:\n%s", want, out)
		}
	}
}

func TestSetSizeSafeAtTinySizes(t *testing.T) {
	// Arrange
	m := newModel()

	// Act / Assert: must not panic at degenerate sizes.
	m.SetSize(0, 0)
	m.SetSize(1, 1)
	m.SetSize(-5, -5)
	_ = m.View()
}

func TestFormatQueryTimeRendersClock(t *testing.T) {
	// Arrange / Act
	got := formatQueryTime(1_700_000_000)

	// Assert: HH:MM:SS shape.
	if len(got) != 8 || strings.Count(got, ":") != 2 {
		t.Fatalf("expected HH:MM:SS, got %q", got)
	}
}

func TestClientLabelPrefersNameThenIP(t *testing.T) {
	if got := clientLabel(pihole.QueryClient{IP: "10.0.0.1", Name: "box"}); got != "box" {
		t.Fatalf("should prefer name, got %q", got)
	}
	if got := clientLabel(pihole.QueryClient{IP: "10.0.0.1", Name: "  "}); got != "10.0.0.1" {
		t.Fatalf("blank name should fall back to IP, got %q", got)
	}
}

func TestClientDetailFormatsNameAndIP(t *testing.T) {
	if got := clientDetail(pihole.QueryClient{IP: "10.0.0.1", Name: "box"}); got != "box (10.0.0.1)" {
		t.Fatalf("named client detail wrong, got %q", got)
	}
	if got := clientDetail(pihole.QueryClient{IP: "10.0.0.1"}); got != "10.0.0.1" {
		t.Fatalf("nameless client detail should be bare IP, got %q", got)
	}
}

func TestOrDashReplacesBlank(t *testing.T) {
	if got := orDash("   "); got != "-" {
		t.Fatalf("blank should become a dash, got %q", got)
	}
	if got := orDash("x"); got != "x" {
		t.Fatalf("non-blank should pass through, got %q", got)
	}
}

func TestMaxMinInt(t *testing.T) {
	if maxInt(3, 7) != 7 || maxInt(9, 2) != 9 {
		t.Fatalf("maxInt wrong")
	}
	if minInt(3, 7) != 3 || minInt(9, 2) != 2 {
		t.Fatalf("minInt wrong")
	}
}

func TestStyleForTokenRendersText(t *testing.T) {
	th := theme.DeepNight()
	for _, tok := range []string{tokenBlock, tokenAllow, tokenNeutral, "unknown"} {
		if out := styleForToken(th, tok).Render("cell"); !strings.Contains(out, "cell") {
			t.Fatalf("styleForToken(%q) should render its text, got %q", tok, out)
		}
	}
}

func TestStyledRowsColorsAndPreservesShape(t *testing.T) {
	// Arrange
	th := theme.DeepNight()
	widths := computeColumnWidths(120)
	queries := sampleQueries()

	// Act
	rows := styledRows(th, queries, widths)

	// Assert
	if len(rows) != len(queries) {
		t.Fatalf("styledRows should be 1:1 with queries, got %d", len(rows))
	}
	for i, r := range rows {
		if len(r) != numCols {
			t.Fatalf("row %d should have %d cells, got %d", i, numCols, len(r))
		}
	}
}

func TestColumnWidthsFromRoundTrips(t *testing.T) {
	// Arrange
	widths := computeColumnWidths(120)
	cols := tableColumns(widths)

	// Act
	got := columnWidthsFrom(cols)

	// Assert
	if len(got) != len(widths) {
		t.Fatalf("width count mismatch: got %d want %d", len(got), len(widths))
	}
	for i := range widths {
		if got[i] != widths[i] {
			t.Fatalf("column %d width mismatch: got %d want %d", i, got[i], widths[i])
		}
	}
}
