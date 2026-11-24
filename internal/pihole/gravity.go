package pihole

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// UpdateGravity triggers POST /api/action/gravity and streams the chunked
// text/plain progress output line by line, invoking onLine for each line until
// the stream closes. Unlike other endpoints this response is NOT JSON and NOT
// SSE — it is a raw chunked text stream. On a 401 it re-authenticates once and
// retries. Streaming stops (returning ctx.Err()) if ctx is cancelled.
func (c *Client) UpdateGravity(ctx context.Context, onLine func(string)) error {
	return c.streamGravity(ctx, onLine, true)
}

func (c *Client) streamGravity(
	ctx context.Context,
	onLine func(string),
	allowRetry bool,
) error {
	if c.isClosed() {
		return ErrClosed
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.apiURL+"/action/gravity",
		nil,
	)
	if err != nil {
		return &NetworkError{Op: "POST /action/gravity", Err: err}
	}
	req.Header.Set("Accept", "text/plain")
	sentSID := c.sessionID()
	if sentSID != "" {
		req.Header.Set("X-FTL-SID", sentSID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &NetworkError{Op: "POST /action/gravity", Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized && allowRetry {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if err := c.reauth(ctx, sentSID); err != nil {
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
			onLine(cleanGravityLine(scanner.Text()))
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

// cleanGravityLine reduces a raw gravity.sh line to what a terminal would
// finally show: the text after the last carriage return (progress lines are
// overwritten with "\r\e[K"), without colour codes, tabs as spaces. Passing
// a raw "\r" on would move the TUI's cursor to column 0 and corrupt the
// screen.
func cleanGravityLine(line string) string {
	if i := strings.LastIndexByte(strings.TrimRight(line, "\r"), '\r'); i >= 0 {
		line = line[i+1:]
	}
	line = ansi.Strip(strings.TrimRight(line, "\r"))
	return strings.ReplaceAll(line, "\t", "    ")
}
