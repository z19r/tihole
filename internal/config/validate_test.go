package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAcceptsValidConfig(t *testing.T) {
	// Arrange
	c := validConfig()

	// Act
	err := Validate(c)

	// Assert
	if err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestValidateRejectsMissingActive(t *testing.T) {
	// Arrange
	c := validConfig()
	c.Active = ""

	// Act
	err := Validate(c)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "active") {
		t.Fatalf("expected active error, got: %v", err)
	}
}

func TestValidateRejectsActiveNotMatchingInstance(t *testing.T) {
	// Arrange
	c := validConfig()
	c.Active = "ghost"

	// Act
	err := Validate(c)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected active-mismatch error, got: %v", err)
	}
}

func TestValidateRejectsDuplicateNames(t *testing.T) {
	// Arrange
	c := validConfig()
	c.Instances = append(
		c.Instances,
		Instance{Name: "home", URL: "https://10.0.0.2"},
	)

	// Act
	err := Validate(c)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate-name error, got: %v", err)
	}
}

func TestValidateRejectsEmptyName(t *testing.T) {
	// Arrange
	c := validConfig()
	c.Instances[0].Name = ""

	// Act
	err := Validate(c)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected empty-name error, got: %v", err)
	}
}

func TestValidateRejectsBadURL(t *testing.T) {
	// Arrange
	c := validConfig()
	c.Instances[0].URL = "ftp://192.168.1.2"

	// Act
	err := Validate(c)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "http") {
		t.Fatalf("expected url-scheme error, got: %v", err)
	}
}

func TestValidateRejectsEmptyURL(t *testing.T) {
	// Arrange
	c := validConfig()
	c.Instances[0].URL = ""

	// Act
	err := Validate(c)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "url is required") {
		t.Fatalf("expected empty-url error, got: %v", err)
	}
}

func TestValidateRejectsEmptyInstances(t *testing.T) {
	// Arrange
	c := &Config{Active: "home"}

	// Act
	err := Validate(c)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "no instances") {
		t.Fatalf("expected no-instances error, got: %v", err)
	}
}

func TestValidateNilConfig(t *testing.T) {
	// Act
	err := Validate(nil)

	// Assert
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestValidateAcceptsActiveWithStraySpaces(t *testing.T) {
	// Arrange
	c := &Config{
		Active:    " home ",
		Instances: []Instance{{Name: "home ", URL: "http://pi.hole"}},
	}

	// Act / Assert
	if err := Validate(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadTrimsActiveAndNames(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "config.yaml")
	yml := "active: ' home '\ninstances:\n" +
		"  - name: ' home'\n    url: http://pi.hole\n    password: x\n"
	if err := os.WriteFile(path, []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}

	// Act
	c, err := Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Active != "home" || c.Instances[0].Name != "home" {
		t.Fatalf(
			"not trimmed: active=%q name=%q",
			c.Active,
			c.Instances[0].Name,
		)
	}
}
