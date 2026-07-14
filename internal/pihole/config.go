package pihole

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// configEnvelope wraps the config tree returned by the config endpoints. FTL
// returns the tree (or an element subtree) under the `config` key.
type configEnvelope struct {
	Config map[string]any `json:"config"`
	Took   float64        `json:"took"`
}

// escapeElement URL-escapes each dotted segment of a config element path (e.g.
// "dns.upstreams") and rejoins them with "/", producing the FTL path suffix.
func escapeElement(element string) string {
	segments := strings.Split(element, ".")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}

// Config fetches GET /api/config — the full configuration tree. When detailed
// is true, FTL annotates each leaf with metadata; otherwise raw values are
// returned. The decoded `config` object is returned as an open-ended tree.
func (c *Client) Config(
	ctx context.Context,
	detailed bool,
) (map[string]any, error) {
	path := fmt.Sprintf("/config?detailed=%t", detailed)
	var out configEnvelope
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Config, nil
}

// ConfigElement fetches GET /api/config/{element} — a single subtree addressed
// by a dotted path such as "dns.upstreams". Each path segment is escaped.
func (c *Client) ConfigElement(
	ctx context.Context,
	element string,
) (map[string]any, error) {
	path := "/config/" + escapeElement(element)
	var out configEnvelope
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Config, nil
}

// configPatchRequest is the body of PATCH /api/config.
type configPatchRequest struct {
	Config map[string]any `json:"config"`
}

// PatchConfig applies a partial config update via PATCH /api/config. The patch
// is sent under the `config` key. When restart is true FTL restarts affected
// subsystems to apply the change immediately.
func (c *Client) PatchConfig(
	ctx context.Context,
	patch map[string]any,
	restart bool,
) error {
	path := fmt.Sprintf("/config?restart=%t", restart)
	body := configPatchRequest{Config: patch}
	return c.do(ctx, http.MethodPatch, path, body, nil)
}

// AddConfigItem appends value to an array-valued config element via
// PUT /api/config/{element}/{value}. Both the element path and the value are
// escaped as path segments.
func (c *Client) AddConfigItem(
	ctx context.Context,
	element, value string,
) error {
	path := "/config/" + escapeElement(element) + "/" + url.PathEscape(value)
	return c.do(ctx, http.MethodPut, path, nil, nil)
}

// DeleteConfigItem removes value from an array-valued config element via
// DELETE /api/config/{element}/{value}.
func (c *Client) DeleteConfigItem(
	ctx context.Context,
	element, value string,
) error {
	path := "/config/" + escapeElement(element) + "/" + url.PathEscape(value)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}
