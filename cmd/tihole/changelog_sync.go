package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// changelogSection is one Keep-a-Changelog category (Added, Fixed, …)
// with its bullet list. The JSON tags produce the exact shape the site's
// Releases.jsx component consumes: { name, bullets }.
type changelogSection struct {
	Name    string   `json:"name"`
	Bullets []string `json:"bullets"`
}

// changelogEntry is a single released version. Field order is deliberate:
// the generated JSON matches site/src/version.js house style
// ({ ver, date, sections }).
type changelogEntry struct {
	Ver      string             `json:"ver"`
	Date     string             `json:"date"`
	Sections []changelogSection `json:"sections"`
}

// runChangelogSync parses the repo-root CHANGELOG.md and regenerates
// site/src/changelog.js as a browser global consumed by the marketing site.
func runChangelogSync() error {
	root, err := findRepoRoot()
	if err != nil {
		return err
	}

	changelogPath := filepath.Join(root, "CHANGELOG.md")
	raw, err := os.ReadFile(changelogPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", changelogPath, err)
	}

	entries := parseChangelog(string(raw))
	js := renderChangelogJS(entries)

	outPath := filepath.Join(root, "site", "src", "changelog.js")
	if err := os.WriteFile(outPath, []byte(js), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	fmt.Printf(
		"changelog-sync: wrote %d release(s) to %s\n",
		len(entries),
		outPath,
	)
	return nil
}

// findRepoRoot walks up from the working directory until it finds a
// directory containing go.mod, so the command works from anywhere.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found above %q", dir)
		}
		dir = parent
	}
}

// parseChangelog turns Keep-a-Changelog markdown into ordered release
// entries. The `## [Unreleased]` section is skipped, as are link-reference
// footers. Section order within an entry is preserved as written.
func parseChangelog(content string) []changelogEntry {
	var entries []changelogEntry
	var cur *changelogEntry
	var sec *changelogSection

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimRight(line, " \t\r")

		if ver, date, ok := parseVersionHeading(trimmed); ok {
			// Flush any in-progress entry before starting a new one.
			cur, sec = flushEntry(&entries, cur)
			if strings.EqualFold(ver, "unreleased") {
				cur = nil
				continue
			}
			cur = &changelogEntry{Ver: ver, Date: date}
			continue
		}

		if cur == nil {
			continue
		}

		if name, ok := parseSectionHeading(trimmed); ok {
			cur.Sections = append(cur.Sections, changelogSection{Name: name})
			sec = &cur.Sections[len(cur.Sections)-1]
			continue
		}

		if bullet, ok := parseBullet(trimmed); ok && sec != nil {
			sec.Bullets = append(sec.Bullets, bullet)
			continue
		}

		// A link-reference footer (`[x]: url`) at column 0 ends the entry.
		if isLinkReference(trimmed) {
			cur, sec = flushEntry(&entries, cur)
			continue
		}

		// Continuation of the previous bullet: only indented wrapped lines
		// (leading whitespace) fold into the prior bullet.
		indented := line != "" &&
			(line[0] == ' ' || line[0] == '\t')
		if sec != nil && len(sec.Bullets) > 0 && indented &&
			strings.TrimSpace(trimmed) != "" {
			last := len(sec.Bullets) - 1
			sec.Bullets[last] += " " + strings.TrimSpace(trimmed)
		}
	}

	flushEntry(&entries, cur)
	return entries
}

// flushEntry appends cur to entries if it carries at least one section,
// and resets the current section pointer.
func flushEntry(
	entries *[]changelogEntry,
	cur *changelogEntry,
) (*changelogEntry, *changelogSection) {
	if cur != nil && len(cur.Sections) > 0 {
		*entries = append(*entries, *cur)
	}
	return nil, nil
}

// parseVersionHeading matches `## [X.Y.Z] - DATE` and `## [Unreleased]`.
func parseVersionHeading(line string) (ver, date string, ok bool) {
	if !strings.HasPrefix(line, "## [") {
		return "", "", false
	}
	rest := line[len("## ["):]
	end := strings.Index(rest, "]")
	if end < 0 {
		return "", "", false
	}
	ver = rest[:end]
	after := strings.TrimSpace(rest[end+1:])
	after = strings.TrimPrefix(after, "-")
	date = strings.TrimSpace(after)
	return ver, date, true
}

// parseSectionHeading matches `### Added` etc., normalizing to the
// lowercase names Releases.jsx expects. Anything starting with "breaking"
// collapses to the "breaking" bucket.
func parseSectionHeading(line string) (name string, ok bool) {
	if !strings.HasPrefix(line, "### ") {
		return "", false
	}
	raw := strings.TrimSpace(line[len("### "):])
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "breaking") {
		return "breaking", true
	}
	// Take the first word so "Fixed" / "Security" map cleanly even if the
	// heading carries trailing prose.
	fields := strings.FieldsFunc(lower, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '—' || r == '-'
	})
	if len(fields) == 0 {
		return "", false
	}
	return fields[0], true
}

// isLinkReference matches a Markdown link-reference definition like
// `[Unreleased]: https://…` used in the CHANGELOG footer.
func isLinkReference(line string) bool {
	if !strings.HasPrefix(line, "[") {
		return false
	}
	close := strings.Index(line, "]:")
	return close > 0
}

// parseBullet matches `- text` or `* text` list items.
func parseBullet(line string) (text string, ok bool) {
	t := strings.TrimSpace(line)
	if strings.HasPrefix(t, "- ") {
		return strings.TrimSpace(t[2:]), true
	}
	if strings.HasPrefix(t, "* ") {
		return strings.TrimSpace(t[2:]), true
	}
	return "", false
}

// renderChangelogJS emits the browser-global file. It assigns
// window.TIHOLE_CHANGELOG (house style, matching version.js /
// metadata.js), which Releases.jsx consumes.
func renderChangelogJS(entries []changelogEntry) string {
	if entries == nil {
		entries = []changelogEntry{}
	}
	body, _ := json.MarshalIndent(entries, "", "  ")

	var b strings.Builder
	b.WriteString(
		"// Auto-generated by `tihole changelog-sync` — do not edit.\n",
	)
	b.WriteString("// Source of truth: repo-root CHANGELOG.md.\n")
	b.WriteString("window.TIHOLE_CHANGELOG = ")
	b.Write(body)
	b.WriteString(";\n")
	return b.String()
}
