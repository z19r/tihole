package pihole

import (
	"context"
	"net/http"
	"net/url"
)

// ListType is the allow/block classification of a subscribed list (adlist).
type ListType string

const (
	// ListAllow is an allow-list (whitelist) subscription.
	ListAllow ListType = "allow"
	// ListBlock is a block-list (blocklist) subscription.
	ListBlock ListType = "block"
)

// List is a subscribed adlist as returned by FTL.
type List struct {
	Address        string `json:"address"`
	Comment        string `json:"comment"`
	Type           string `json:"type"`
	Groups         []int  `json:"groups"`
	Enabled        bool   `json:"enabled"`
	ID             int    `json:"id"`
	Number         int    `json:"number"`
	InvalidDomains int    `json:"invalid_domains"`
	Status         int    `json:"status"`
	DateAdded      int64  `json:"date_added"`
	DateModified   int64  `json:"date_modified"`
	DateUpdated    int64  `json:"date_updated"`
}

// ListRef identifies a list for batch deletion.
type ListRef struct {
	Item string   `json:"item"`
	Type ListType `json:"type"`
}

// listsEnvelope wraps the list/mutation response for adlists.
type listsEnvelope struct {
	Lists []List  `json:"lists"`
	Took  float64 `json:"took"`
}

// listCreateRequest is the body of a list create (address in body).
type listCreateRequest struct {
	Address string   `json:"address"`
	Type    ListType `json:"type"`
	Comment string   `json:"comment"`
	Groups  []int    `json:"groups"`
	Enabled bool     `json:"enabled"`
}

// listUpdateRequest is the body of a list update (address in path).
type listUpdateRequest struct {
	Type    ListType `json:"type"`
	Comment string   `json:"comment"`
	Groups  []int    `json:"groups"`
	Enabled bool     `json:"enabled"`
}

// Lists fetches GET /api/lists?type={listType}.
func (c *Client) Lists(ctx context.Context, listType ListType) ([]List, error) {
	path := "/lists?type=" + url.QueryEscape(string(listType))
	var out listsEnvelope
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Lists, nil
}

// AddList creates a list via POST /api/lists. The new list is enabled.
func (c *Client) AddList(
	ctx context.Context,
	address string,
	listType ListType,
	comment string,
	groups []int,
) (List, error) {
	body := listCreateRequest{
		Address: address,
		Type:    listType,
		Comment: comment,
		Groups:  groups,
		Enabled: true,
	}
	var out listsEnvelope
	if err := c.do(ctx, http.MethodPost, "/lists", body, &out); err != nil {
		return List{}, err
	}
	if len(out.Lists) > 0 {
		return out.Lists[0], nil
	}
	return List{}, nil
}

// UpdateList updates a list via PUT /api/lists/{address}?type={listType}.
func (c *Client) UpdateList(
	ctx context.Context,
	address string,
	listType ListType,
	comment string,
	groups []int,
	enabled bool,
) (List, error) {
	body := listUpdateRequest{
		Type:    listType,
		Comment: comment,
		Groups:  groups,
		Enabled: enabled,
	}
	path := "/lists/" + url.PathEscape(
		address,
	) + "?type=" + url.QueryEscape(
		string(listType),
	)
	var out listsEnvelope
	if err := c.do(ctx, http.MethodPut, path, body, &out); err != nil {
		return List{}, err
	}
	if len(out.Lists) > 0 {
		return out.Lists[0], nil
	}
	return List{}, nil
}

// DeleteList removes a list via DELETE /api/lists/{address}?type={listType}.
func (c *Client) DeleteList(
	ctx context.Context,
	address string,
	listType ListType,
) error {
	path := "/lists/" + url.PathEscape(
		address,
	) + "?type=" + url.QueryEscape(
		string(listType),
	)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// BatchDeleteLists removes multiple lists via POST /api/lists:batchDelete.
func (c *Client) BatchDeleteLists(ctx context.Context, items []ListRef) error {
	return c.do(ctx, http.MethodPost, "/lists:batchDelete", items, nil)
}
