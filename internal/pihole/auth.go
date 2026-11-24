package pihole

import (
	"context"
	"errors"
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
	if c.isClosed() {
		return ErrClosed
	}
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
	c.mu.Lock()
	if c.closed { // Close ran while we waited; don't keep the session
		c.mu.Unlock()
		c.deleteSession(ctx, env.Session.SID)
		return ErrClosed
	}
	c.sid = env.Session.SID
	c.mu.Unlock()
	return nil
}

// deleteSession best-effort ends sid without touching the stored session.
func (c *Client) deleteSession(ctx context.Context, sid string) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete, c.apiURL+"/auth", nil)
	if err != nil {
		return
	}
	req.Header.Set("X-FTL-SID", sid)
	if resp, err := c.httpClient.Do(req); err == nil {
		_ = resp.Body.Close()
	}
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

// ErrClosed is returned by every request on a Client after Close.
var ErrClosed = errors.New("pihole: client closed")

// Close logs out and retires the client: later requests fail with ErrClosed
// instead of transparently logging in again, so a request still in flight
// on a dropped client can't open an orphan session.
func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	c.closed = true
	sid := c.sid
	c.mu.Unlock()
	if sid == "" {
		return nil
	}
	err := c.doOnce(ctx, http.MethodDelete, "/auth", nil, nil, false, true)
	c.setSID("")
	return err
}

func (c *Client) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}
