package pihole

import (
	"context"
	"errors"
	"net/http"
)

// loginRequest is the body of POST /api/auth.
type loginRequest struct {
	Password string `json:"password"`
}

// login authenticates with the stored password and stores
// the returned session ID. It does not use do() so it cannot recurse on 401.
// The password and SID are never logged.
func (c *Client) login(ctx context.Context) error {
	if c.isClosed() {
		return ErrClosed
	}
	reqBody := loginRequest{Password: c.password}

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

// ErrClosed is returned by every request on a Client after Close.
var ErrClosed = errors.New("pihole: client closed")

// Close logs out and retires the client: later requests fail with ErrClosed
// instead of transparently logging in again, so a request still in flight
// on a dropped client can't open an orphan session. It is idempotent: a
// concurrent or later Close waits for the first one and sends no DELETE.
func (c *Client) Close(ctx context.Context) error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
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
