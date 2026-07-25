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
		Refresh: map[string]int{
			"dashboard": 5,
			"querylog":  2,
		},
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
	if got.Refresh["dashboard"] != 5 {
		t.Fatalf("refresh not preserved: %+v", got.Refresh)
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
