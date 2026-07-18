package dashboard

import (
	"testing"

	"github.com/z19r/tihole/internal/pihole"
)

func TestHumanUptime(t *testing.T) {
	cases := []struct {
		name    string
		seconds int64
		want    string
	}{
		{"sub-minute reads zero", 42, "0m"},
		{"negative reads zero", -10, "0m"},
		{"minutes only", 25 * 60, "25m"},
		{"hours and minutes", 3*3600 + 12*60, "3h 12m"},
		{"days and hours", 4*86400 + 9*3600, "4d 9h"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got := humanUptime(tc.seconds)
			// Assert
			if got != tc.want {
				t.Errorf(
					"humanUptime(%d) = %q, want %q",
					tc.seconds,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestHumanKiB(t *testing.T) {
	cases := []struct {
		name string
		kib  int64
		want string
	}{
		{"kibibytes", 512, "512K"},
		{"mebibytes rounds", 263808, "258M"},
		{"gibibytes with decimal", 3886904, "3.7G"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got := humanKiB(tc.kib)
			// Assert
			if got != tc.want {
				t.Errorf("humanKiB(%d) = %q, want %q", tc.kib, got, tc.want)
			}
		})
	}
}

func TestMemLabelIncludesTotalsWhenKnown(t *testing.T) {
	// Arrange
	s := pihole.SystemInfo{
		MemUsedPercent: 6.787,
		MemUsedKiB:     263808,
		MemTotalKiB:    3886904,
	}

	// Act
	got := memLabel(s)

	// Assert
	want := "6.8% · 258M/3.7G"
	if got != want {
		t.Errorf("memLabel = %q, want %q", got, want)
	}
}

func TestMemLabelPercentOnlyWhenTotalsUnknown(t *testing.T) {
	// Arrange: a host that reports a percent but no absolute figures.
	s := pihole.SystemInfo{MemUsedPercent: 40.0}

	// Act
	got := memLabel(s)

	// Assert
	if got != "40.0%" {
		t.Errorf("memLabel = %q, want %q", got, "40.0%")
	}
}

func TestPluralIssues(t *testing.T) {
	// Arrange / Act / Assert
	if got := pluralIssues(1); got != "⚠ 1 issue" {
		t.Errorf("pluralIssues(1) = %q, want %q", got, "⚠ 1 issue")
	}
	if got := pluralIssues(3); got != "⚠ 3 issues" {
		t.Errorf("pluralIssues(3) = %q, want %q", got, "⚠ 3 issues")
	}
}
