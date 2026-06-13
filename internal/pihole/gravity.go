package pihole

import (
	"bufio"
	"context"
	"io"
	"net/http"
)

// UpdateGravity triggers POST /api/action/gravity and streams the chunked
// text/plain progress output line by line, invoking onLine for each line until
// the stream closes. Unlike other endpoints this response is NOT JSON and NOT
// SSE — it is a raw chunked text stream. On a 401 it re-authenticates once and
// retries. Streaming stops (returning ctx.Err()) if ctx is cancelled.
func (c *Client) UpdateGravity(ctx context.Context, onLine func(string)) error {
	return c.streamGravity(ctx, onLine, true)
}

func (c *Client) streamGravity(ctx context.Context, onLine func(string), allowRetry bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL+"/action/gravity", nil)
	if err != nil {
		return &NetworkError{Op: "POST /action/gravity", Err: err}
	}
	req.Header.Set("Accept", "text/plain")
	if sid := c.sessionID(); sid != "" {
		req.Header.Set("X-FTL-SID", sid)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &NetworkError{Op: "POST /action/gravity", Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized && allowRetry {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if err := c.login(ctx); err != nil {
			return err
		}
		return c.streamGravity(ctx, onLine, false)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeAPIError(resp.StatusCode, resp.Body)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if onLine != nil {
			onLine(scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		// A cancelled context surfaces as a read error; prefer the ctx cause.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return &NetworkError{Op: "POST /action/gravity", Err: err}
	}
	return nil
}
