package config

import (
	"testing"
)

func TestResolvePasswordReturnsInlinePassword(t *testing.T) {
	// Arrange
	inst := Instance{Name: "home", Password: "inline-secret"}

	// Act
	got, err := inst.ResolvePassword()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "inline-secret" {
		t.Fatalf("got %q, want inline-secret", got)
	}
}

func TestResolvePasswordReadsFromEnvWhenSet(t *testing.T) {
	// Arrange
	t.Setenv("TIHOLE_TEST_PW", "env-secret")
	inst := Instance{Name: "office", PasswordEnv: "TIHOLE_TEST_PW"}

	// Act
	got, err := inst.ResolvePassword()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "env-secret" {
		t.Fatalf("got %q, want env-secret", got)
	}
}

func TestResolvePasswordErrorsWhenEnvUnset(t *testing.T) {
	// Arrange: ensure the var is empty in this test's env.
	t.Setenv("TIHOLE_TEST_PW_UNSET", "")
	inst := Instance{Name: "office", PasswordEnv: "TIHOLE_TEST_PW_UNSET"}

	// Act
	_, err := inst.ResolvePassword()

	// Assert
	if err == nil {
		t.Fatal("expected error when password_env is unset, got nil")
	}
}

func TestResolvePasswordErrorsWhenNeitherSet(t *testing.T) {
	// Arrange
	inst := Instance{Name: "home"}

	// Act
	_, err := inst.ResolvePassword()

	// Assert
	if err == nil {
		t.Fatal("expected error when no password source, got nil")
	}
}

func TestResolvePasswordPrefersInlineOverEnv(t *testing.T) {
	// Arrange
	t.Setenv("TIHOLE_TEST_PW2", "env-secret")
	inst := Instance{Name: "home", Password: "inline", PasswordEnv: "TIHOLE_TEST_PW2"}

	// Act
	got, _ := inst.ResolvePassword()

	// Assert
	if got != "inline" {
		t.Fatalf("got %q, want inline (inline should win)", got)
	}
}
