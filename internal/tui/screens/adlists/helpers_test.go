package adlists

import (
	"errors"
	"testing"

	"github.com/z19r/tihole/internal/pihole"
)

// --- pure helpers -----------------------------------------------------------

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
	for _, i := range []int{1, 2} { // flexible columns
		if widths[i] < minFlexCol {
			t.Fatalf("flexible column %d below minimum: got %d", i, widths[i])
		}
	}
}

func TestStatusLabelAndToken(t *testing.T) {
	cases := []struct {
		status int
		label  string
		token  string
	}{
		{0, "unknown", tokenNeutral},
		{1, "ok", tokenAllow},
		{2, "warn", tokenWarn},
		{3, "error", tokenBlock},
		{99, "unknown", tokenNeutral},
	}
	for _, c := range cases {
		// Act
		gotLabel := statusLabel(c.status)
		gotToken := statusToken(c.status)
		// Assert
		if gotLabel != c.label {
			t.Fatalf("statusLabel(%d)=%q, want %q", c.status, gotLabel, c.label)
		}
		if gotToken != c.token {
			t.Fatalf("statusToken(%d)=%q, want %q", c.status, gotToken, c.token)
		}
	}
}

func TestTypeTokenClassification(t *testing.T) {
	if typeToken("block") != tokenBlock {
		t.Fatalf("block should map to tokenBlock")
	}
	if typeToken("allow") != tokenAllow {
		t.Fatalf("allow should map to tokenAllow")
	}
	if typeToken("") != tokenNeutral {
		t.Fatalf("empty should map to tokenNeutral")
	}
}

func TestListToRowMapsFieldsInOrder(t *testing.T) {
	// Arrange
	widths := computeColumnWidths(120)
	l := pihole.List{
		Address: "https://ads.example.com/list.txt",
		Comment: "ad servers",
		Type:    "block",
		Enabled: true,
		Number:  1234,
		Status:  1,
	}

	// Act
	row := listToRow(l, widths)

	// Assert
	if len(row) != numCols {
		t.Fatalf("expected %d cells, got %d", numCols, len(row))
	}
	if row[0] != "block" {
		t.Fatalf("type cell wrong: %q", row[0])
	}
	if row[3] != enabledGlyph {
		t.Fatalf("enabled glyph wrong: %q", row[3])
	}
	if row[4] != "1234" {
		t.Fatalf("domains cell wrong: %q", row[4])
	}
	if row[5] != "ok" {
		t.Fatalf("status cell wrong: %q", row[5])
	}
}

func TestListToRowDisabledGlyph(t *testing.T) {
	// Arrange
	widths := computeColumnWidths(120)
	l := pihole.List{Address: "a", Enabled: false}

	// Act
	row := listToRow(l, widths)

	// Assert
	if row[3] != disabledGlyph {
		t.Fatalf("disabled glyph wrong: %q", row[3])
	}
}

func TestParseGroupsSkipsInvalid(t *testing.T) {
	// Arrange / Act
	got := parseGroups(" 0 , 2 ,x, ,3")

	// Assert
	want := []int{0, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("parseGroups length: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseGroups[%d]=%d want %d", i, got[i], want[i])
		}
	}
}

func TestGroupsOrDefaultFallsBackToZero(t *testing.T) {
	// Act
	got := groupsOrDefault("   ")

	// Assert
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("expected default group [0], got %v", got)
	}
}

func TestGroupsCSVRoundTrip(t *testing.T) {
	// Act
	csv := groupsCSV([]int{0, 3, 5})

	// Assert
	if csv != "0,3,5" {
		t.Fatalf("groupsCSV wrong: %q", csv)
	}
}

func TestTruncateAppendsEllipsis(t *testing.T) {
	// Act
	out := truncate("verylongaddress", 6)

	// Assert
	if out != "veryl"+ellipsis {
		t.Fatalf("unexpected truncation: %q", out)
	}
}

func TestGravitySummary(t *testing.T) {
	if gravitySummary(false, nil) != "running gravity update…" {
		t.Fatalf("running summary wrong")
	}
	if gravitySummary(true, nil) != "gravity update complete" {
		t.Fatalf("done summary wrong")
	}
	if got := gravitySummary(true, errors.New("boom")); got != "gravity failed: boom" {
		t.Fatalf("error summary wrong: %q", got)
	}
}
