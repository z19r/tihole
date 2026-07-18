package pihole

import (
	"context"
	"net/http"
	"net/url"
)

// Group is a single FTL group used to scope domains, clients, and lists.
type Group struct {
	Name         string `json:"name"`
	Comment      string `json:"comment"`
	Enabled      bool   `json:"enabled"`
	ID           int    `json:"id"`
	DateAdded    int64  `json:"date_added"`
	DateModified int64  `json:"date_modified"`
}

// GroupRef identifies a group for batch deletion.
type GroupRef struct {
	Item string `json:"item"`
}

// groupsEnvelope wraps the list/mutation response for groups.
type groupsEnvelope struct {
	Groups []Group `json:"groups"`
	Took   float64 `json:"took"`
}

// groupCreateRequest is the body of a group create (name in body).
type groupCreateRequest struct {
	Name    string `json:"name"`
	Comment string `json:"comment"`
	Enabled bool   `json:"enabled"`
}

// groupUpdateRequest is the body of a group update (name in path).
type groupUpdateRequest struct {
	Comment string `json:"comment"`
	Enabled bool   `json:"enabled"`
}

// Groups fetches GET /api/groups — every configured group.
func (c *Client) Groups(ctx context.Context) ([]Group, error) {
	var out groupsEnvelope
	if err := c.do(ctx, http.MethodGet, "/groups", nil, &out); err != nil {
		return nil, err
	}
	return out.Groups, nil
}

// AddGroup creates a group via POST /api/groups. The new group is enabled.
func (c *Client) AddGroup(
	ctx context.Context,
	name, comment string,
) (Group, error) {
	body := groupCreateRequest{Name: name, Comment: comment, Enabled: true}
	var out groupsEnvelope
	if err := c.do(ctx, http.MethodPost, "/groups", body, &out); err != nil {
		return Group{}, err
	}
	if len(out.Groups) > 0 {
		return out.Groups[0], nil
	}
	return Group{}, nil
}

// UpdateGroup updates a group via PUT /api/groups/{name}.
func (c *Client) UpdateGroup(
	ctx context.Context,
	name, comment string,
	enabled bool,
) (Group, error) {
	body := groupUpdateRequest{Comment: comment, Enabled: enabled}
	path := "/groups/" + url.PathEscape(name)
	var out groupsEnvelope
	if err := c.do(ctx, http.MethodPut, path, body, &out); err != nil {
		return Group{}, err
	}
	if len(out.Groups) > 0 {
		return out.Groups[0], nil
	}
	return Group{}, nil
}

// DeleteGroup removes a group via DELETE /api/groups/{name}.
func (c *Client) DeleteGroup(ctx context.Context, name string) error {
	return c.do(
		ctx,
		http.MethodDelete,
		"/groups/"+url.PathEscape(name),
		nil,
		nil,
	)
}

// BatchDeleteGroups removes multiple groups via POST /api/groups:batchDelete.
func (c *Client) BatchDeleteGroups(
	ctx context.Context,
	items []GroupRef,
) error {
	return c.do(ctx, http.MethodPost, "/groups:batchDelete", items, nil)
}
