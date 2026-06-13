package pihole

import (
	"context"
	"net/http"
	"net/url"
)

// Client entry (note: the transport Client is distinct — this models an FTL
// client record). It maps an IP, subnet, MAC, or hostname to group membership.
type ClientEntry struct {
	Client       string `json:"client"`
	Comment      string `json:"comment"`
	Groups       []int  `json:"groups"`
	ID           int    `json:"id"`
	Name         string `json:"name"`
	DateAdded    int64  `json:"date_added"`
	DateModified int64  `json:"date_modified"`
}

// ClientSuggestion is a candidate client (seen in queries but not yet
// configured) returned by GET /api/clients/_suggestions.
type ClientSuggestion struct {
	HWAddr    string `json:"hwaddr"`
	MACVendor string `json:"macVendor"`
	LastQuery int64  `json:"lastQuery"`
	Name      string `json:"name"`
	Addresses string `json:"addresses"`
}

// ClientRef identifies a client for batch deletion.
type ClientRef struct {
	Item string `json:"item"`
}

// clientsEnvelope wraps the list/mutation response for clients.
type clientsEnvelope struct {
	Clients []ClientEntry `json:"clients"`
	Took    float64       `json:"took"`
}

// suggestionsEnvelope wraps GET /api/clients/_suggestions.
type suggestionsEnvelope struct {
	Clients []ClientSuggestion `json:"clients"`
	Took    float64            `json:"took"`
}

// clientCreateRequest is the body of a client create (client id in body).
type clientCreateRequest struct {
	Client  string `json:"client"`
	Comment string `json:"comment"`
	Groups  []int  `json:"groups"`
}

// clientUpdateRequest is the body of a client update (client id in path).
type clientUpdateRequest struct {
	Comment string `json:"comment"`
	Groups  []int  `json:"groups"`
}

// Clients fetches GET /api/clients — every configured client.
func (c *Client) Clients(ctx context.Context) ([]ClientEntry, error) {
	var out clientsEnvelope
	if err := c.do(ctx, http.MethodGet, "/clients", nil, &out); err != nil {
		return nil, err
	}
	return out.Clients, nil
}

// AddClient creates a client via POST /api/clients.
func (c *Client) AddClient(ctx context.Context, client, comment string, groups []int) (ClientEntry, error) {
	body := clientCreateRequest{Client: client, Comment: comment, Groups: groups}
	var out clientsEnvelope
	if err := c.do(ctx, http.MethodPost, "/clients", body, &out); err != nil {
		return ClientEntry{}, err
	}
	if len(out.Clients) > 0 {
		return out.Clients[0], nil
	}
	return ClientEntry{}, nil
}

// UpdateClient updates a client via PUT /api/clients/{client}.
func (c *Client) UpdateClient(ctx context.Context, client, comment string, groups []int) (ClientEntry, error) {
	body := clientUpdateRequest{Comment: comment, Groups: groups}
	path := "/clients/" + url.PathEscape(client)
	var out clientsEnvelope
	if err := c.do(ctx, http.MethodPut, path, body, &out); err != nil {
		return ClientEntry{}, err
	}
	if len(out.Clients) > 0 {
		return out.Clients[0], nil
	}
	return ClientEntry{}, nil
}

// DeleteClient removes a client via DELETE /api/clients/{client}.
func (c *Client) DeleteClient(ctx context.Context, client string) error {
	return c.do(ctx, http.MethodDelete, "/clients/"+url.PathEscape(client), nil, nil)
}

// ClientSuggestions fetches GET /api/clients/_suggestions — clients seen in
// queries that are not yet configured.
func (c *Client) ClientSuggestions(ctx context.Context) ([]ClientSuggestion, error) {
	var out suggestionsEnvelope
	if err := c.do(ctx, http.MethodGet, "/clients/_suggestions", nil, &out); err != nil {
		return nil, err
	}
	return out.Clients, nil
}

// BatchDeleteClients removes multiple clients via POST /api/clients:batchDelete.
func (c *Client) BatchDeleteClients(ctx context.Context, items []ClientRef) error {
	return c.do(ctx, http.MethodPost, "/clients:batchDelete", items, nil)
}
