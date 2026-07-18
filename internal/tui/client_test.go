package tui

import (
	"strings"
	"testing"

	"github.com/zackkitzmiller/tihole/internal/config"
)

func testConfig() *config.Config {
	verify := true
	return &config.Config{
		Active: "home",
		Instances: []config.Instance{
			{
				Name:      "home",
				URL:       "http://192.168.1.2",
				Password:  "pw",
				VerifyTLS: &verify,
			},
			{
				Name:        "envonly",
				URL:         "http://10.0.0.2",
				PasswordEnv: "TIHOLE_TEST_UNSET_PW",
			},
		},
	}
}

func TestClientForBuildsWithoutAuthenticating(t *testing.T) {
	// Arrange
	cfg := testConfig()

	// Act: clientFor must NOT perform network auth, so it returns quickly with
	// a
	// usable client even though nothing is listening.
	c, err := clientFor(cfg, "home")

	// Assert
	if err != nil {
		t.Fatalf("clientFor should not error for a valid instance, got %v", err)
	}
	if c == nil {
		t.Fatal("clientFor returned a nil client")
	}
}

func TestClientForUnknownInstance(t *testing.T) {
	// Act
	_, err := clientFor(testConfig(), "nope")

	// Assert
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error, got %v", err)
	}
}

func TestClientForUnresolvablePassword(t *testing.T) {
	// Act: password_env names an unset variable.
	_, err := clientFor(testConfig(), "envonly")

	// Assert
	if err == nil {
		t.Fatal("expected an error when password_env is unset")
	}
}

func TestFallbackClientNeverNil(t *testing.T) {
	cfg := testConfig()

	// Known instance → client from its URL.
	if got := fallbackClient(cfg, "home"); got == nil {
		t.Fatal("fallbackClient(known) returned nil")
	}
	// Unknown instance → still a usable (empty) client, never nil.
	if got := fallbackClient(cfg, "nope"); got == nil {
		t.Fatal("fallbackClient(unknown) returned nil")
	}
}
