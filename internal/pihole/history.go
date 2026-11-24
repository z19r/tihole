package pihole

import (
	"context"
	"net/http"
)

// History fetches GET /api/history — queries-over-time buckets.
func (c *Client) History(ctx context.Context) (History, error) {
	var out History
	if err := c.do(ctx, http.MethodGet, "/history", nil, &out); err != nil {
		return History{}, err
	}
	return out, nil
}
