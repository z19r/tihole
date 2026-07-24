package pihole

import (
	"context"
	"net/http"
)

// loginRequest is the body of POST /api/auth.
type loginRequest struct {
	Password string `json:"password"`
	TOTP     *int   `json:"totp,omitempty"`
}

// login authenticates with the stored password (and optional TOTP) and stores
// the returned session ID. It does not use do() so it cannot recurse on 401.
// The password and SID are never logged.
func (c *Client) login(ctx context.Context) error {
	reqBody := loginRequest{Password: c.password, TOTP: c.totp}

	var env sessionEnvelope
	// The login request is sent unauthenticated (sendAuth=false) so it never
	// carries a stale SID and never mutates the shared session — concurrent
	// authenticated requests keep working while we log in.
	if err := c.doOnce(ctx, http.MethodPost, "/auth", reqBody, &env, false, false); err != nil {
		if apiErr, ok := err.(*APIError); ok {
			return &AuthError{Status: apiErr.Status, Message: apiErr.Message}
		}
		return err
	}
	if !env.Session.Valid || env.Session.SID == "" {
		msg := env.Session.Message
		if msg == "" {
			msg = "login did not return a valid session"
		}
		return &AuthError{Status: http.StatusUnauthorized, Message: msg}
	}
	c.setSID(env.Session.SID)
	return nil
}

// Login authenticates and stores a session. Callers may invoke this eagerly,
// though do() will also log in transparently on demand.
func (c *Client) Login(ctx context.Context) error {
	return c.login(ctx)
}

// SessionValid checks the current session via GET /api/auth.
func (c *Client) SessionValid(ctx context.Context) (Session, error) {
	var env sessionEnvelope
	if err := c.do(ctx, http.MethodGet, "/auth", nil, &env); err != nil {
		return Session{}, err
	}
	return env.Session, nil
}

// Logout ends the current session via DELETE /api/auth and clears the stored
// SID. It is a no-op when there is no active session.
func (c *Client) Logout(ctx context.Context) error {
	if c.sessionID() == "" {
		return nil
	}
	if err := c.do(ctx, http.MethodDelete, "/auth", nil, nil); err != nil {
		return err
	}
	c.setSID("")
	return nil
}
