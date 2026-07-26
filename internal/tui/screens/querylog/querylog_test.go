package querylog

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
	"github.com/z19r/tihole/internal/tui/core"
)

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
	total := 10 // absurdly small

	// Act
	widths := computeColumnWidths(total)

	// Assert
	for _, i := range []int{1, 2, 5} { // flexible columns
		if widths[i] < minFlexCol {
			t.Fatalf("flexible column %d below minimum: got %d", i, widths[i])
		}
	}
}

func TestTruncateAppendsEllipsisWhenTooLong(t *testing.T) {
	// Arrange
	in := "verylongdomain.example.com"

	// Act
	out := truncate(in, 10)

	// Assert
	if out != "verylongd"+ellipsis {
		t.Fatalf("unexpected truncation: %q", out)
	}
	if len([]rune(out)) != 10 {
		t.Fatalf("truncated width should equal 10, got %d", len([]rune(out)))
	}
}

func TestTruncateLeavesShortStringUntouched(t *testing.T) {
	// Arrange / Act
	out := truncate("short", 20)

	// Assert
	if out != "short" {
		t.Fatalf("short string should be unchanged, got %q", out)
	}
}

func TestQueryToRowMapsFieldsInOrder(t *testing.T) {
	// Arrange
	widths := computeColumnWidths(120)
	q := pihole.Query{
		Time:     1_700_000_000,
		Type:     "A",
		Domain:   "ads.example.com",
		Status:   "GRAVITY",
		Upstream: "",
		Client:   pihole.QueryClient{IP: "10.0.0.5", Name: "laptop"},
	}

	// Act
	row := queryToRow(q, widths)

	// Assert
	if len(row) != numCols {
		t.Fatalf("expected %d cells, got %d", numCols, len(row))
	}
	if row[1] != "ads.example.com" {
		t.Fatalf("domain cell wrong: %q", row[1])
	}
	if row[2] != "laptop" {
		t.Fatalf("client cell should prefer name: %q", row[2])
	}
	if row[3] != "A" {
		t.Fatalf("type cell wrong: %q", row[3])
	}
	if row[4] != "GRAVITY" {
		t.Fatalf("status cell wrong: %q", row[4])
	}
	if row[5] != "-" {
		t.Fatalf("empty upstream should render dash, got %q", row[5])
	}
}

func TestQueryToRowFallsBackToClientIP(t *testing.T) {
	// Arrange
	widths := computeColumnWidths(120)
	q := pihole.Query{Client: pihole.QueryClient{IP: "10.0.0.9", Name: ""}}

	// Act
	row := queryToRow(q, widths)

	// Assert
	if row[2] != "10.0.0.9" {
		t.Fatalf("client cell should fall back to IP, got %q", row[2])
	}
}

func TestStatusTokenClassification(t *testing.T) {
	cases := map[string]string{
		"GRAVITY":             tokenBlock,
		"DENYLIST":            tokenBlock,
		"REGEX_CNAME":         tokenBlock,
		"EXTERNAL_BLOCKED_IP": tokenBlock,
		"SPECIAL_DOMAIN":      tokenBlock,
		"FORWARDED":           tokenAllow,
		"CACHE":               tokenAllow,
		"RETRIED":             tokenAllow,
		"OK":                  tokenAllow,
		"IN_PROGRESS":         tokenNeutral,
		"":                    tokenNeutral,
	}
	for status, want := range cases {
		// Act
		got := statusToken(status)
		// Assert
		if got != want {
			t.Fatalf("statusToken(%q)=%q, want %q", status, got, want)
		}
	}
}

func TestBuildFilterSetsOnlyProvidedFields(t *testing.T) {
	// Arrange / Act
	f := buildFilter("  example.com  ", defaultPage, nil)

	// Assert
	if f.Domain == nil || *f.Domain != "example.com" {
		t.Fatalf("domain not trimmed/set: %+v", f.Domain)
	}
	if f.Length == nil || *f.Length != defaultPage {
		t.Fatalf("length not set: %+v", f.Length)
	}
	if f.Cursor != nil {
		t.Fatalf("cursor should be nil for a fresh page")
	}
}

func TestBuildFilterOmitsEmptyDomain(t *testing.T) {
	// Arrange / Act
	f := buildFilter("   ", defaultPage, nil)

	// Assert
	if f.Domain != nil {
		t.Fatalf("blank domain should be omitted, got %v", *f.Domain)
	}
}

func TestBuildFilterCarriesCursor(t *testing.T) {
	// Arrange
	cur := int64(42)

	// Act
	f := buildFilter("", defaultPage, &cur)

	// Assert
	if f.Cursor == nil || *f.Cursor != 42 {
		t.Fatalf("cursor not carried through: %+v", f.Cursor)
	}
}

func newTestModel() *Model {
	th := theme.DeepNight()
	return New(&core.AppContext{Theme: th, Keys: core.DefaultKeyMap()})
}

func TestUpdateIgnoresStaleTickEpoch(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.focused = true
	m.epoch = 5

	// Act
	_, cmd := m.Update(tickMsg{epoch: 4})

	// Assert
	if cmd != nil {
		t.Fatalf("stale tick epoch should produce no command")
	}
}

func TestUpdateTickReturnsNoCmdWhileSearching(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.focused = true
	m.searching = true
	m.epoch = 3

	// Act
	_, cmd := m.Update(tickMsg{epoch: 3})

	// Assert
	if cmd != nil {
		t.Fatalf("tick while searching should produce no command")
	}
}

func TestUpdateTickReturnsNoCmdWhenUnfocused(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.focused = false
	m.epoch = 1

	// Act
	_, cmd := m.Update(tickMsg{epoch: 1})

	// Assert
	if cmd != nil {
		t.Fatalf("tick while unfocused should produce no command")
	}
}

func TestUpdateLiveTickSchedulesWork(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.focused = true
	m.searching = false
	m.epoch = 7

	// Act
	_, cmd := m.Update(tickMsg{epoch: 7})

	// Assert
	if cmd == nil {
		t.Fatalf("matching focused tick should schedule fetch+tick")
	}
}

func TestUpdateIgnoresStaleQueriesResult(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.epoch = 9
	m.loading = true

	// Act
	_, _ = m.Update(
		queriesMsg{epoch: 8, page: pihole.QueriesPage{RecordsTotal: 100}},
	)

	// Assert
	if m.total == 100 {
		t.Fatalf("stale queries result should not be applied")
	}
	if !m.loading {
		t.Fatalf("stale result should not clear loading")
	}
}

var _ core.Screen = (*Model)(nil)
var _ tea.Model = (*Model)(nil)
