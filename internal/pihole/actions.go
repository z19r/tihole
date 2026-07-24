package pihole

import (
	"context"
	"net/http"
)

// RestartDNS restarts the FTL DNS resolver via POST /api/action/restartdns.
// This is a destructive action and may return a 403 APIError unless the FTL
// webserver has allow_destructive enabled.
func (c *Client) RestartDNS(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/action/restartdns", nil, nil)
}

// FlushLogs flushes the FTL query log via POST /api/action/flush/logs. This is
// a destructive action and may return a 403 APIError unless allow_destructive
// is enabled.
func (c *Client) FlushLogs(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/action/flush/logs", nil, nil)
}

// FlushNetwork flushes FTL's network device table via
// POST /api/action/flush/network. This is a destructive action and may return a
// 403 APIError unless allow_destructive is enabled.
func (c *Client) FlushNetwork(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/action/flush/network", nil, nil)
}
