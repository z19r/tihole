package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

func validConfig() *Config {
	return &Config{
		Active: "home",
		Theme:  "omarchy",
		Instances: []Instance{
			{
				Name:      "home",
				URL:       "https://192.168.1.2",
				Password:  "secret",
				VerifyTLS: boolPtr(false),
			},
		},
	}
}

func TestDefaultPathEndsWithTiholeConfig(t *testing.T) {
	// Arrange / Act
	got, err := DefaultPath()

	// Assert
	if err != nil {
		t.Fatalf("DefaultPath returned error: %v", err)
	}
	want := filepath.Join("tihole", "config.yaml")
	if !strings.HasSuffix(got, want) {
		t.Fatalf("DefaultPath() = %q, want suffix %q", got, want)
	}
}

func TestSaveThenLoadRoundTripsConfig(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	orig := validConfig()

	// Act
	if err := Save(path, orig); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// Assert
	if got.Active != orig.Active || got.Theme != orig.Theme {
		t.Fatalf("round-trip mismatch: got %+v want %+v", got, orig)
	}
	if len(got.Instances) != 1 || got.Instances[0].Name != "home" {
		t.Fatalf("instances not preserved: %+v", got.Instances)
	}
	if got.Instances[0].VerifyTLSValue() != false {
		t.Fatalf(
			"verify_tls not preserved, got %v",
			got.Instances[0].VerifyTLSValue(),
		)
	}
}

func TestSaveEnforces0600FileMode(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Act
	if err := Save(path, validConfig()); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// Assert
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("file mode = %o, want 600", perm)
	}
}

func TestSaveTightensExistingLoosePermissions(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("stale"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	// Act
	if err := Save(path, validConfig()); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// Assert
	info, _ := os.Stat(path)
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("file mode = %o, want 600", perm)
	}
}

func TestSaveNilConfigReturnsError(t *testing.T) {
	// Act
	err := Save(filepath.Join(t.TempDir(), "c.yaml"), nil)

	// Assert
	if err == nil {
		t.Fatal("expected error saving nil config")
	}
}

func TestSaveErrorsWhenParentIsAFile(t *testing.T) {
	// Arrange: make a regular file, then try to treat it as a parent dir.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	path := filepath.Join(blocker, "config.yaml")

	// Act
	err := Save(path, validConfig())

	// Assert
	if err == nil {
		t.Fatal("expected error when parent path is a file")
	}
}

func TestLoadMissingFileReturnsError(t *testing.T) {
	// Arrange / Act
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))

	// Assert
	if err == nil {
		t.Fatal("expected error loading missing file, got nil")
	}
}

func TestLoadInvalidYAMLReturnsError(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("active: [unterminated"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Act
	_, err := Load(path)

	// Assert
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestSaveFailureLeavesOriginalIntact(t *testing.T) {
	// Arrange: a valid file in a directory that then turns read-only, so the
	// temp file can't be created.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := Save(path, validConfig()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	before, _ := os.ReadFile(path)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	changed := validConfig()
	changed.Theme = "other"

	// Act
	err := Save(path, changed)

	// Assert
	if err == nil {
		t.Skip("directory still writable (running as root?)")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatalf("original modified:\n%s\nwant:\n%s", after, before)
	}
	if _, err := Load(path); err != nil {
		t.Fatalf("original no longer loads: %v", err)
	}
}

func TestSaveUsesUniqueTempFile(t *testing.T) {
	// Arrange: a leftover at a fixed temp name must not block or race saves.
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.Mkdir(path+".tmp", 0o700); err != nil {
		t.Fatal(err)
	}

	// Act
	err := Save(path, validConfig())

	// Assert
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 2 {
		t.Fatalf("temp file left behind: %v", entries)
	}
}

func TestSavePreservesCommentsAndUnknownKeys(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "config.yaml")
	yml := "# my pi-holes\nactive: home # main one\nextra: keep-me\n" +
		"theme: old\ninstances:\n  - name: home\n    url: http://pi.hole\n" +
		"    password: x\n"
	if err := os.WriteFile(path, []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	c.Theme = ""
	c.Combined = true

	// Act
	if err := Save(path, c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Assert
	data, _ := os.ReadFile(path)
	out := string(data)
	for _, want := range []string{
		"# my pi-holes", "# main one", "extra: keep-me", "combined: true",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("saved file missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "theme") {
		t.Fatalf("cleared theme should be removed:\n%s", out)
	}
	got, err := Load(path)
	if err != nil || !got.Combined || got.Instances[0].URL != "http://pi.hole" {
		t.Fatalf("reload mismatch: %+v err=%v", got, err)
	}
}

func TestSaveThroughSymlinkUpdatesTarget(t *testing.T) {
	// Arrange: config.yaml links into a dotfiles dir.
	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles", "config.yaml")
	if err := Save(target, validConfig()); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "config.yaml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	changed := validConfig()
	changed.Theme = "other"

	// Act
	if err := Save(link, changed); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Assert
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link replaced: %v %v", fi.Mode(), err)
	}
	got, err := Load(target)
	if err != nil || got.Theme != "other" {
		t.Fatalf("target not updated: %+v %v", got, err)
	}
}
