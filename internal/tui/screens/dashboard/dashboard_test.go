package dashboard

import (
	"strings"
	"testing"
)

func TestSparkline_EmptyReturnsEmpty(t *testing.T) {
	// Arrange
	var values []float64

	// Act
	got := sparkline(values, 10)

	// Assert
	if got != "" {
		t.Fatalf("expected empty string for empty input, got %q", got)
	}
}

func TestSparkline_NonPositiveWidthReturnsEmpty(t *testing.T) {
	// Arrange
	values := []float64{1, 2, 3}

	// Act
	got := sparkline(values, 0)

	// Assert
	if got != "" {
		t.Fatalf("expected empty string for zero width, got %q", got)
	}
}

func TestSparkline_SingleValueRendersFlatBaseline(t *testing.T) {
	// Arrange
	values := []float64{42}

	// Act
	got := sparkline(values, 5)

	// Assert
	if got != "▁" {
		t.Fatalf("expected single lowest rune for one value, got %q", got)
	}
}

func TestSparkline_AllEqualRendersLowestRunes(t *testing.T) {
	// Arrange
	values := []float64{7, 7, 7, 7}

	// Act
	got := sparkline(values, 4)

	// Assert
	if got != "▁▁▁▁" {
		t.Fatalf("expected flat baseline for equal values, got %q", got)
	}
}

func TestSparkline_MapsExtremesToLowestAndHighest(t *testing.T) {
	// Arrange
	values := []float64{0, 10}

	// Act
	got := sparkline(values, 2)

	// Assert
	runes := []rune(got)
	if len(runes) != 2 {
		t.Fatalf("expected 2 runes, got %d (%q)", len(runes), got)
	}
	if runes[0] != '▁' {
		t.Errorf("expected min value to map to ▁, got %q", string(runes[0]))
	}
	if runes[1] != '█' {
		t.Errorf("expected max value to map to █, got %q", string(runes[1]))
	}
}

func TestSparkline_ResamplesWiderInputToWidth(t *testing.T) {
	// Arrange
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8}

	// Act
	got := sparkline(values, 4)

	// Assert
	if n := len([]rune(got)); n != 4 {
		t.Fatalf("expected 4 columns after resampling, got %d", n)
	}
}

func TestSparklineInt_MatchesFloatVariant(t *testing.T) {
	// Arrange
	ints := []int{1, 5, 3}

	// Act
	got := sparklineInt(ints, 3)

	// Assert
	if n := len([]rune(got)); n != 3 {
		t.Fatalf("expected 3 columns, got %d", n)
	}
}

func TestRuneForFraction_ClampsOutOfRange(t *testing.T) {
	// Arrange / Act / Assert
	if r := runeForFraction(-1); r != '▁' {
		t.Errorf("expected clamp low to ▁, got %q", string(r))
	}
	if r := runeForFraction(2); r != '█' {
		t.Errorf("expected clamp high to █, got %q", string(r))
	}
}

func TestMiniBar_ZeroFractionIsAllEmpty(t *testing.T) {
	// Arrange / Act
	got := miniBar(0, 4)

	// Assert
	if got != strings.Repeat(string(barEmpty), 4) {
		t.Fatalf("expected all-empty bar, got %q", got)
	}
}

func TestMiniBar_FullFractionIsAllFilled(t *testing.T) {
	// Arrange / Act
	got := miniBar(1, 4)

	// Assert
	if got != strings.Repeat(string(barFilled), 4) {
		t.Fatalf("expected all-filled bar, got %q", got)
	}
}

func TestMiniBar_HalfFractionFillsHalf(t *testing.T) {
	// Arrange / Act
	got := miniBar(0.5, 10)

	// Assert
	want := strings.Repeat(string(barFilled), 5) + strings.Repeat(string(barEmpty), 5)
	if got != want {
		t.Fatalf("expected half-filled bar %q, got %q", want, got)
	}
}

func TestMiniBar_ClampsOverAndUnderflow(t *testing.T) {
	// Arrange / Act / Assert
	if got := miniBar(2, 3); got != strings.Repeat(string(barFilled), 3) {
		t.Errorf("expected overflow clamp to full, got %q", got)
	}
	if got := miniBar(-1, 3); got != strings.Repeat(string(barEmpty), 3) {
		t.Errorf("expected underflow clamp to empty, got %q", got)
	}
}

func TestMiniBar_NonPositiveWidthReturnsEmpty(t *testing.T) {
	// Arrange / Act / Assert
	if got := miniBar(0.5, 0); got != "" {
		t.Fatalf("expected empty string for zero width, got %q", got)
	}
}

func TestFormatCount_InsertsThousandsSeparators(t *testing.T) {
	// Arrange
	cases := map[int]string{
		0:       "0",
		42:      "42",
		999:     "999",
		1000:    "1,000",
		12345:   "12,345",
		1000000: "1,000,000",
		-12345:  "-12,345",
	}

	// Act / Assert
	for in, want := range cases {
		if got := formatCount(in); got != want {
			t.Errorf("formatCount(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatPercent_OneDecimalWithSign(t *testing.T) {
	// Arrange / Act / Assert
	if got := formatPercent(42.17); got != "42.2%" {
		t.Errorf("expected rounded 42.2%%, got %q", got)
	}
	if got := formatPercent(0); got != "0.0%" {
		t.Errorf("expected 0.0%%, got %q", got)
	}
	if got := formatPercent(100); got != "100.0%" {
		t.Errorf("expected 100.0%%, got %q", got)
	}
}

func TestShouldHandleTick_StaleEpochIsIgnored(t *testing.T) {
	// Arrange
	focused, tickEpoch, current := true, 1, 2

	// Act
	got := shouldHandleTick(focused, tickEpoch, current)

	// Assert
	if got {
		t.Fatal("stale epoch tick should not be handled")
	}
}

func TestShouldHandleTick_CurrentEpochWhileFocusedIsHandled(t *testing.T) {
	// Arrange
	focused, tickEpoch, current := true, 3, 3

	// Act
	got := shouldHandleTick(focused, tickEpoch, current)

	// Assert
	if !got {
		t.Fatal("matching epoch while focused should be handled")
	}
}

func TestShouldHandleTick_UnfocusedIsIgnored(t *testing.T) {
	// Arrange
	focused, tickEpoch, current := false, 3, 3

	// Act
	got := shouldHandleTick(focused, tickEpoch, current)

	// Assert
	if got {
		t.Fatal("tick while blurred should not be handled")
	}
}

func TestUpdate_StaleTickReturnsNoCommand(t *testing.T) {
	// Arrange
	m := &Model{focused: true, epoch: 5}

	// Act
	_, cmd := m.Update(tickMsg{epoch: 4})

	// Assert
	if cmd != nil {
		t.Fatal("expected no command for stale tick epoch")
	}
}

func TestUpdate_StaleResultIsDiscarded(t *testing.T) {
	// Arrange
	m := &Model{focused: true, epoch: 5}

	// Act
	m.Update(summaryMsg{epoch: 4, err: errStub("boom")})

	// Assert
	if m.errSummary != "" {
		t.Fatalf("stale summary result should be ignored, got err %q", m.errSummary)
	}
	if m.loaded {
		t.Fatal("stale result should not mark model as loaded")
	}
}

func TestUpdate_CurrentResultStoresError(t *testing.T) {
	// Arrange
	m := &Model{focused: true, epoch: 5}

	// Act
	m.Update(summaryMsg{epoch: 5, err: errStub("down")})

	// Assert
	if m.errSummary != "down" {
		t.Fatalf("expected stored error %q, got %q", "down", m.errSummary)
	}
	if !m.loaded {
		t.Fatal("a delivered result should mark the model loaded")
	}
}

func TestSortedTypes_OrdersByDescendingCount(t *testing.T) {
	// Arrange
	types := map[string]int{"A": 3, "AAAA": 10, "PTR": 1}

	// Act
	got := sortedTypes(types)

	// Assert
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got[0].label != "AAAA" || got[0].count != 10 {
		t.Errorf("expected AAAA first, got %+v", got[0])
	}
	if got[2].label != "PTR" {
		t.Errorf("expected PTR last, got %+v", got[2])
	}
}

func TestTruncate_ShortensWithEllipsis(t *testing.T) {
	// Arrange / Act / Assert
	if got := truncate("example.com", 5); got != "exam…" {
		t.Errorf("expected ellipsis truncation, got %q", got)
	}
	if got := truncate("short", 10); got != "short" {
		t.Errorf("expected unchanged string, got %q", got)
	}
	if got := truncate("x", 0); got != "" {
		t.Errorf("expected empty for zero width, got %q", got)
	}
}

func TestPadHelpers_PadToWidth(t *testing.T) {
	// Arrange / Act / Assert
	if got := padRight("ab", 5); got != "ab   " {
		t.Errorf("padRight = %q", got)
	}
	if got := padLeft("ab", 5); got != "   ab" {
		t.Errorf("padLeft = %q", got)
	}
}

// errStub is a tiny error implementation for table-free tests.
type errStub string

func (e errStub) Error() string { return string(e) }
