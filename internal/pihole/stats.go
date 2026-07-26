package pihole

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Summary fetches GET /api/stats/summary.
func (c *Client) Summary(ctx context.Context) (Summary, error) {
	var out Summary
	if err := c.do(ctx, http.MethodGet, "/stats/summary", nil, &out); err != nil {
		return Summary{}, err
	}
	return out, nil
}

// Upstreams fetches GET /api/stats/upstreams.
func (c *Client) Upstreams(ctx context.Context) (Upstreams, error) {
	var out Upstreams
	if err := c.do(ctx, http.MethodGet, "/stats/upstreams", nil, &out); err != nil {
		return Upstreams{}, err
	}
	return out, nil
}

// QueryTypes fetches GET /api/stats/query_types.
func (c *Client) QueryTypes(ctx context.Context) (QueryTypes, error) {
	var out QueryTypes
	if err := c.do(ctx, http.MethodGet, "/stats/query_types", nil, &out); err != nil {
		return QueryTypes{}, err
	}
	return out, nil
}

// TopDomains fetches GET /api/stats/top_domains with the blocked and count
// query parameters.
func (c *Client) TopDomains(
	ctx context.Context,
	blocked bool,
	count int,
) (TopDomains, error) {
	var out TopDomains
	path := "/stats/top_domains" + topQuery(blocked, count)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return TopDomains{}, err
	}
	return out, nil
}

// TopClients fetches GET /api/stats/top_clients with the blocked and count
// query parameters.
func (c *Client) TopClients(
	ctx context.Context,
	blocked bool,
	count int,
) (TopClients, error) {
	var out TopClients
	path := "/stats/top_clients" + topQuery(blocked, count)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return TopClients{}, err
	}
	return out, nil
}

// RecentBlocked fetches GET /api/stats/recent_blocked. It returns the list of
// most-recently blocked domain names.
func (c *Client) RecentBlocked(
	ctx context.Context,
	count int,
) ([]string, error) {
	var out struct {
		Blocked []string `json:"blocked"`
		Took    float64  `json:"took"`
	}
	path := "/stats/recent_blocked"
	if count > 0 {
		path += "?count=" + fmt.Sprintf("%d", count)
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Blocked, nil
}

// topQuery builds the ?blocked=&count= query string shared by the top_* stats
// endpoints. count <= 0 omits the count parameter.
func topQuery(blocked bool, count int) string {
	v := url.Values{}
	v.Set("blocked", fmt.Sprintf("%t", blocked))
	if count > 0 {
		v.Set("count", fmt.Sprintf("%d", count))
	}
	return "?" + v.Encode()
}
