package pihole

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// DiagnosisMessage is a single FTL diagnosis/notification message from
// GET /api/info/messages.
type DiagnosisMessage struct {
	ID        int     `json:"id"`
	Timestamp float64 `json:"timestamp"`
	Type      string  `json:"type"`
	Plain     string  `json:"plain"`
	HTML      string  `json:"html"`
	URL       string  `json:"url"`
}

// DNSLogLine is a single line of the FTL DNS resolver log.
type DNSLogLine struct {
	Timestamp float64 `json:"timestamp"`
	Message   string  `json:"message"`
	PRIO      string  `json:"prio"`
}

// DNSLogPage is the response of GET /api/info/logs/dns, carrying a batch of log
// lines and the cursor to request the next batch.
type DNSLogPage struct {
	Log    []DNSLogLine `json:"log"`
	NextID int          `json:"nextID"`
}

// Info fetches GET /api/info/{section} for section in
// ftl|host|system|version|metrics|sensors|database|messages. The section object
// is returned as an open-ended tree (its top-level key mirrors the section
// name).
func (c *Client) Info(ctx context.Context, section string) (map[string]any, error) {
	path := "/info/" + section
	var raw map[string]any
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	if inner, ok := raw[section].(map[string]any); ok {
		return inner, nil
	}
	delete(raw, "took")
	return raw, nil
}

// messageCountEnvelope is the response of GET /api/info/messages/count.
type messageCountEnvelope struct {
	Count int     `json:"count"`
	Took  float64 `json:"took"`
}

// MessageCount fetches GET /api/info/messages/count — the number of active
// diagnosis messages.
func (c *Client) MessageCount(ctx context.Context) (int, error) {
	var out messageCountEnvelope
	if err := c.do(ctx, http.MethodGet, "/info/messages/count", nil, &out); err != nil {
		return 0, err
	}
	return out.Count, nil
}

// messagesEnvelope is the response of GET /api/info/messages.
type messagesEnvelope struct {
	Messages []DiagnosisMessage `json:"messages"`
	Took     float64            `json:"took"`
}

// Messages fetches GET /api/info/messages — the active diagnosis messages.
func (c *Client) Messages(ctx context.Context) ([]DiagnosisMessage, error) {
	var out messagesEnvelope
	if err := c.do(ctx, http.MethodGet, "/info/messages", nil, &out); err != nil {
		return nil, err
	}
	return out.Messages, nil
}

// DeleteMessages dismisses diagnosis messages via
// DELETE /api/info/messages/{ids} with ids comma-joined into a single path
// segment. A nil or empty slice is a no-op.
func (c *Client) DeleteMessages(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	path := "/info/messages/" + strings.Join(parts, ",")
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// DNSLog fetches GET /api/info/logs/dns for the live resolver log tail. When
// nextID is positive it is sent as the ?nextID cursor; otherwise the parameter
// is omitted to request from the beginning of the available window.
func (c *Client) DNSLog(ctx context.Context, nextID int) (DNSLogPage, error) {
	path := "/info/logs/dns"
	if nextID > 0 {
		path = fmt.Sprintf("%s?nextID=%d", path, nextID)
	}
	var out DNSLogPage
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return DNSLogPage{}, err
	}
	return out, nil
}
