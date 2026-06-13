package pihole

import (
	"encoding/json"
	"fmt"
	"io"
)

// APIError represents a decoded FTL error envelope returned on a non-2xx
// response. The FTL API returns errors in the shape:
//
//	{ "error": { "key": "...", "message": "...", "hint": "..." }, "took": 0.0 }
type APIError struct {
	Status  int    // HTTP status code
	Key     string // FTL error key, e.g. "unauthorized"
	Message string // Human-readable message
	Hint    string // Optional remediation hint
}

func (e *APIError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("pihole: %s (status %d, key %q): %s [hint: %s]",
			e.Message, e.Status, e.Key, e.Message, e.Hint)
	}
	return fmt.Sprintf("pihole: %s (status %d, key %q)", e.Message, e.Status, e.Key)
}

// AuthError indicates an authentication or authorization failure that could
// not be transparently recovered (e.g. bad password, exhausted retries).
type AuthError struct {
	Status  int
	Message string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("pihole: authentication failed (status %d): %s", e.Status, e.Message)
}

// NetworkError wraps a transport-level failure (DNS, connection refused, TLS,
// timeout) so callers can distinguish it from an API-level error.
type NetworkError struct {
	Op  string
	Err error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("pihole: network error during %s: %v", e.Op, e.Err)
}

func (e *NetworkError) Unwrap() error { return e.Err }

// ftlErrorEnvelope is the wire shape of an FTL error response.
type ftlErrorEnvelope struct {
	Error struct {
		Key     string `json:"key"`
		Message string `json:"message"`
		Hint    string `json:"hint"`
	} `json:"error"`
	Took float64 `json:"took"`
}

// decodeAPIError reads an error-envelope response body and produces an
// *APIError. If the body cannot be decoded, a best-effort APIError is still
// returned carrying the status code so callers always get a typed error.
func decodeAPIError(status int, body io.Reader) *APIError {
	var env ftlErrorEnvelope
	apiErr := &APIError{Status: status}
	data, err := io.ReadAll(body)
	if err != nil {
		apiErr.Message = fmt.Sprintf("failed to read error body: %v", err)
		return apiErr
	}
	if len(data) == 0 {
		apiErr.Message = "empty error response"
		return apiErr
	}
	if err := json.Unmarshal(data, &env); err != nil {
		apiErr.Message = string(data)
		return apiErr
	}
	apiErr.Key = env.Error.Key
	apiErr.Message = env.Error.Message
	apiErr.Hint = env.Error.Hint
	if apiErr.Message == "" {
		apiErr.Message = string(data)
	}
	return apiErr
}
