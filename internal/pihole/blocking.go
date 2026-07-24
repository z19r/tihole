package pihole

import (
	"context"
	"net/http"
)

// Blocking fetches GET /api/dns/blocking — the current blocking state and any
// active countdown timer.
func (c *Client) Blocking(ctx context.Context) (BlockingStatus, error) {
	var out BlockingStatus
	if err := c.do(ctx, http.MethodGet, "/dns/blocking", nil, &out); err != nil {
		return BlockingStatus{}, err
	}
	return out, nil
}

// blockingRequest is the body of POST /api/dns/blocking.
type blockingRequest struct {
	Blocking bool     `json:"blocking"`
	Timer    *float64 `json:"timer"`
}

// SetBlocking sets the blocking state via POST /api/dns/blocking. timer is the
// number of seconds after which blocking reverts, or nil for a permanent change.
func (c *Client) SetBlocking(ctx context.Context, blocking bool, timer *float64) error {
	reqBody := blockingRequest{Blocking: blocking, Timer: timer}
	return c.do(ctx, http.MethodPost, "/dns/blocking", reqBody, nil)
}
