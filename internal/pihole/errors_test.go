package pihole

import (
	"errors"
	"strings"
	"testing"
)

func TestDecodeAPIError(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantKey     string
		wantMessage string
		wantHint    string
	}{
		{
			name:        "full envelope decodes all fields",
			status:      401,
			body:        `{"error":{"key":"unauthorized","message":"Unauthorized","hint":"Use an app password"},"took":0.01}`,
			wantKey:     "unauthorized",
			wantMessage: "Unauthorized",
			wantHint:    "Use an app password",
		},
		{
			name:        "envelope without hint",
			status:      400,
			body:        `{"error":{"key":"bad_request","message":"Bad request"},"took":0.0}`,
			wantKey:     "bad_request",
			wantMessage: "Bad request",
			wantHint:    "",
		},
		{
			name:        "non-json body falls back to raw message",
			status:      500,
			body:        `internal server error`,
			wantMessage: "internal server error",
		},
		{
			name:        "empty body still carries status",
			status:      503,
			body:        ``,
			wantMessage: "empty error response",
		},
		{
			// FTL answers a bad-password login with 401 and a session object,
			// NOT an {"error":...} envelope. We must never splatter that raw
			// JSON (it's noisy and can carry session fields) — surface a clean,
			// status-derived message instead.
			name:        "json without error envelope yields clean status message",
			status:      401,
			body:        `{"session":{"valid":false,"totp":false,"sid":null,"validity":-1}}`,
			wantMessage: "invalid credentials or expired session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			r := strings.NewReader(tt.body)

			// Act
			got := decodeAPIError(tt.status, r)

			// Assert
			if got.Status != tt.status {
				t.Errorf("Status = %d, want %d", got.Status, tt.status)
			}
			if got.Key != tt.wantKey {
				t.Errorf("Key = %q, want %q", got.Key, tt.wantKey)
			}
			if got.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", got.Message, tt.wantMessage)
			}
			if got.Hint != tt.wantHint {
				t.Errorf("Hint = %q, want %q", got.Hint, tt.wantHint)
			}
			// A decoded error must never carry a raw JSON body through to the UI.
			if strings.HasPrefix(strings.TrimSpace(tt.body), "{") && strings.Contains(got.Message, "{") {
				t.Errorf("Message leaks raw JSON body: %q", got.Message)
			}
		})
	}
}

func TestAPIErrorMessageIncludesHint(t *testing.T) {
	// Arrange
	e := &APIError{Status: 403, Key: "forbidden", Message: "denied", Hint: "set allow_destructive"}

	// Act
	msg := e.Error()

	// Assert
	if !strings.Contains(msg, "set allow_destructive") {
		t.Errorf("Error() = %q, want it to contain the hint", msg)
	}
}

func TestNetworkErrorUnwrap(t *testing.T) {
	// Arrange
	inner := errors.New("connection refused")
	ne := &NetworkError{Op: "GET /auth", Err: inner}

	// Act & Assert
	if !errors.Is(ne, inner) {
		t.Errorf("errors.Is did not unwrap to inner error")
	}
}

func TestAuthErrorMessage(t *testing.T) {
	// Arrange
	e := &AuthError{Status: 401, Message: "invalid password"}

	// Act & Assert
	if !strings.Contains(e.Error(), "invalid password") {
		t.Errorf("Error() = %q, want it to contain the message", e.Error())
	}
}
