package pihole

import (
	"context"
	"fmt"
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

// HistoryClients fetches GET /api/history/clients?N — per-client over-time
// buckets for the top n clients. n <= 0 omits the parameter (server default).
func (c *Client) HistoryClients(
	ctx context.Context,
	n int,
) (HistoryClients, error) {
	var out HistoryClients
	path := "/history/clients"
	if n > 0 {
		path += "?N=" + fmt.Sprintf("%d", n)
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return HistoryClients{}, err
	}
	return out, nil
}
