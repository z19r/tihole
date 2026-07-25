package pihole

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// QueryFilter holds the optional filter and cursor-pagination parameters for
// GET /api/queries. Only fields that are set (non-nil) are sent.
type QueryFilter struct {
	Length     *int    // number of records to return
	Start      *int    // offset into the result set
	Cursor     *int64  // pagination cursor from a prior page
	From       *int64  // unix timestamp lower bound
	Until      *int64  // unix timestamp upper bound
	Domain     *string // filter by domain
	ClientIP   *string // filter by client IP
	ClientName *string // filter by client name
	Upstream   *string // filter by upstream
	Type       *string // filter by query type
	Status     *string // filter by status
	Reply      *string // filter by reply type
	DNSSEC     *string // filter by DNSSEC status
	Disk       *bool   // include on-disk (long-term) database records
}

// values renders the filter as URL query parameters, emitting only set fields.
func (f QueryFilter) values() url.Values {
	v := url.Values{}
	if f.Length != nil {
		v.Set("length", strconv.Itoa(*f.Length))
	}
	if f.Start != nil {
		v.Set("start", strconv.Itoa(*f.Start))
	}
	if f.Cursor != nil {
		v.Set("cursor", strconv.FormatInt(*f.Cursor, 10))
	}
	if f.From != nil {
		v.Set("from", strconv.FormatInt(*f.From, 10))
	}
	if f.Until != nil {
		v.Set("until", strconv.FormatInt(*f.Until, 10))
	}
	if f.Domain != nil {
		v.Set("domain", *f.Domain)
	}
	if f.ClientIP != nil {
		v.Set("client_ip", *f.ClientIP)
	}
	if f.ClientName != nil {
		v.Set("client_name", *f.ClientName)
	}
	if f.Upstream != nil {
		v.Set("upstream", *f.Upstream)
	}
	if f.Type != nil {
		v.Set("type", *f.Type)
	}
	if f.Status != nil {
		v.Set("status", *f.Status)
	}
	if f.Reply != nil {
		v.Set("reply", *f.Reply)
	}
	if f.DNSSEC != nil {
		v.Set("dnssec", *f.DNSSEC)
	}
	if f.Disk != nil {
		v.Set("disk", strconv.FormatBool(*f.Disk))
	}
	return v
}

// Queries fetches GET /api/queries with cursor pagination. Only the filter
// fields that are set are included in the request.
func (c *Client) Queries(
	ctx context.Context,
	filter QueryFilter,
) (QueriesPage, error) {
	path := "/queries"
	if q := filter.values().Encode(); q != "" {
		path += "?" + q
	}
	var out QueriesPage
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return QueriesPage{}, err
	}
	return out, nil
}
