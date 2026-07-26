package main

import (
	"strings"
	"testing"
)

func TestParseChangelogExtractsVersionsAndSections(t *testing.T) {
	// Arrange
	md := `# Changelog

## [Unreleased]

### Added

- something not yet released

## [0.2.0] - 2026-08-01

### Added

- new dashboard gauge
- another add

### Fixed

- a nasty bug

## [0.1.0] - 2026-07-25

### Added

- first cut

[Unreleased]: https://example.com/compare/v0.2.0...HEAD
[0.2.0]: https://example.com/releases/tag/v0.2.0
`

	// Act
	entries := parseChangelog(md)

	// Assert
	if len(entries) != 2 {
		t.Fatalf("expected 2 released entries, got %d", len(entries))
	}
	if entries[0].Ver != "0.2.0" || entries[0].Date != "2026-08-01" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
	if len(entries[0].Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(entries[0].Sections))
	}
	if entries[0].Sections[0].Name != "added" {
		t.Fatalf("expected first section 'added', got %q",
			entries[0].Sections[0].Name)
	}
	if len(entries[0].Sections[0].Bullets) != 2 {
		t.Fatalf("expected 2 bullets, got %d",
			len(entries[0].Sections[0].Bullets))
	}
	if entries[0].Sections[1].Name != "fixed" {
		t.Fatalf("expected second section 'fixed', got %q",
			entries[0].Sections[1].Name)
	}
	if entries[1].Ver != "0.1.0" {
		t.Fatalf("expected second entry 0.1.0, got %q", entries[1].Ver)
	}
}

func TestParseChangelogFoldsWrappedBulletsAndDropsFooter(t *testing.T) {
	// Arrange
	md := `## [0.1.0] - 2026-07-25

### Added

- a bullet that wraps
  onto a second line
- a standalone bullet

[Unreleased]: https://example.com/compare/v0.1.0...HEAD
[0.1.0]: https://example.com/releases/tag/v0.1.0
`

	// Act
	entries := parseChangelog(md)

	// Assert
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	bullets := entries[0].Sections[0].Bullets
	if len(bullets) != 2 {
		t.Fatalf("expected 2 bullets, got %d: %#v", len(bullets), bullets)
	}
	if bullets[0] != "a bullet that wraps onto a second line" {
		t.Fatalf("wrapped bullet not folded: %q", bullets[0])
	}
	if strings.Contains(bullets[1], "http") ||
		strings.Contains(bullets[1], "[0.1.0]") {
		t.Fatalf("footer leaked into bullet: %q", bullets[1])
	}
}

func TestParseChangelogSkipsUnreleased(t *testing.T) {
	// Arrange
	md := `## [Unreleased]

### Added

- pending work
`

	// Act
	entries := parseChangelog(md)

	// Assert
	if len(entries) != 0 {
		t.Fatalf("expected Unreleased to be skipped, got %d entries",
			len(entries))
	}
}

func TestParseChangelogNormalizesBreakingHeading(t *testing.T) {
	// Arrange
	md := `## [1.0.0] - 2026-09-01

### BREAKING — read this before upgrading

- removed the old flag
`

	// Act
	entries := parseChangelog(md)

	// Assert
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Sections[0].Name != "breaking" {
		t.Fatalf("expected 'breaking', got %q",
			entries[0].Sections[0].Name)
	}
}

func TestRenderChangelogJSAssignsGlobals(t *testing.T) {
	// Arrange
	entries := []changelogEntry{
		{
			Ver:  "0.1.0",
			Date: "2026-07-25",
			Sections: []changelogSection{
				{Name: "added", Bullets: []string{"first cut"}},
			},
		},
	}

	// Act
	js := renderChangelogJS(entries)

	// Assert
	if !strings.Contains(js, "window.TIHOLE_CHANGELOG = ") {
		t.Fatalf("missing TIHOLE_CHANGELOG assignment:\n%s", js)
	}
	if strings.Contains(js, "WHETSTONE") {
		t.Fatalf("legacy WHETSTONE alias should be gone:\n%s", js)
	}
	if !strings.Contains(js, `"ver": "0.1.0"`) {
		t.Fatalf("expected ver in output:\n%s", js)
	}
	if !strings.Contains(js, `"first cut"`) {
		t.Fatalf("expected bullet in output:\n%s", js)
	}
}

func TestRenderChangelogJSEmptyIsValidArray(t *testing.T) {
	// Arrange / Act
	js := renderChangelogJS(nil)

	// Assert
	if !strings.Contains(js, "window.TIHOLE_CHANGELOG = [];") {
		t.Fatalf("expected empty array literal, got:\n%s", js)
	}
}
