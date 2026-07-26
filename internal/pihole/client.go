package pihole

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Client is a PiHole v6 FTL REST API client for a single instance. It manages
// a session (X-FTL-SID header auth) and transparently re-authenticates on a
// 401. It is safe for concurrent use.
type Client struct {
	baseURL string // host root, e.g. "https://pi.hole" (no trailing /api)
	apiURL  string // baseURL + "/api"

	httpClient *http.Client

	password string
	totp     *int

	mu  sync.Mutex // guards sid
	sid string

	authMu sync.Mutex // single-flights re-authentication (see reauth)
}

// Option configures a Client at construction time.
type Option func(*Client)

// WithInsecureTLS enables or disables TLS certificate verification. Self-signed
// certificates are common on PiHole installs, so pass true to skip
// verification.
func WithInsecureTLS(insecure bool) Option {
	return func(c *Client) {
		if !insecure {
			return
		}
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			}, //nolint:gosec // opt-in per instance
		}
		c.httpClient = &http.Client{
			Transport: transport,
			Timeout:   c.httpClient.Timeout,
		}
	}
}

// WithTOTP sets a TOTP code to send during login (for web passwords with 2FA).
func WithTOTP(code int) Option {
	return func(c *Client) { c.totp = &code }
}

// WithHTTPClient overrides the underlying *http.Client (e.g. for tests or
// custom transports).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// New constructs a Client. baseURL is the instance host root (with scheme),
// e.g. "https://192.168.1.2"; a trailing "/api" is appended automatically and
// any existing trailing slash or "/api" suffix is normalized away.
func New(baseURL, password string, opts ...Option) *Client {
	root := strings.TrimRight(baseURL, "/")
	root = strings.TrimSuffix(root, "/api")
	c := &Client{
		baseURL:    root,
		apiURL:     root + "/api",
		httpClient: &http.Client{},
		password:   password,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SID returns the current session ID (empty when not authenticated). Exposed
// for tests and diagnostics; callers must not log it.
func (c *Client) sessionID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sid
}

func (c *Client) setSID(sid string) {
	c.mu.Lock()
	c.sid = sid
	c.mu.Unlock()
}

// do performs an authenticated request against path (which must start with "/",
// relative to the /api root), marshaling body as JSON when non-nil and
// decoding a 2xx response into out when non-nil. On a 401 it transparently logs
// in once and retries. Non-2xx responses are decoded into an *APIError.
func (c *Client) do(
	ctx context.Context,
	method, path string,
	body, out any,
) error {
	if err := c.doOnce(ctx, method, path, body, out, true, true); err != nil {
		return err
	}
	return nil
}

// reauth re-authenticates at most once across concurrent callers. staleSID is
// the session the failed request sent (empty if none). Holding authMu, if the
// stored SID has already advanced past staleSID another goroutine re-logged in
// while we waited, so we skip a redundant login — this prevents a thundering
// herd of parallel widget fetches from each opening a fresh FTL session and
// exhausting the server's session seats.
func (c *Client) reauth(ctx context.Context, staleSID string) error {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	if cur := c.sessionID(); cur != "" && cur != staleSID {
		return nil
	}
	return c.login(ctx)
}

// doOnce issues a single request. When allowRetry is true and the response is
// 401, it re-authenticates (single-flight) and retries exactly once. sendAuth
// controls whether the stored X-FTL-SID is attached — login itself passes false
// so it never sends a stale session and never mutates shared state.
func (c *Client) doOnce(
	ctx context.Context,
	method, path string,
	body, out any,
	allowRetry, sendAuth bool,
) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("pihole: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.apiURL+path, reqBody)
	if err != nil {
		return &NetworkError{Op: method + " " + path, Err: err}
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	// Header auth: send X-FTL-SID when we have a session. No CSRF token is
	// required for header-based auth. Capture the SID we send so re-auth can
	// tell
	// whether another goroutine already refreshed it.
	sentSID := ""
	if sendAuth {
		if sid := c.sessionID(); sid != "" {
			sentSID = sid
			req.Header.Set("X-FTL-SID", sid)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &NetworkError{Op: method + " " + path, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized && allowRetry {
		// Drain and close before re-auth so the connection can be reused.
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if err := c.reauth(ctx, sentSID); err != nil {
			return err
		}
		return c.doOnce(ctx, method, path, body, out, false, sendAuth)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeAPIError(resp.StatusCode, resp.Body)
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf(
			"pihole: decode %s %s response: %w",
			method,
			path,
			err,
		)
	}
	return nil
}
