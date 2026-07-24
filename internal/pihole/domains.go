package pihole

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// DomainType is the allow/deny classification of a managed domain.
type DomainType string

// DomainKind distinguishes exact-match from regex domains.
type DomainKind string

const (
	// DomainAllow is the allow-list domain type.
	DomainAllow DomainType = "allow"
	// DomainDeny is the deny-list domain type.
	DomainDeny DomainType = "deny"
	// KindExact matches a domain literally.
	KindExact DomainKind = "exact"
	// KindRegex matches domains by regular expression.
	KindRegex DomainKind = "regex"
)

// Domain is a single managed allow/deny domain entry as returned by FTL.
type Domain struct {
	Domain       string `json:"domain"`
	Unicode      string `json:"unicode"`
	Type         string `json:"type"`
	Kind         string `json:"kind"`
	Comment      string `json:"comment"`
	Groups       []int  `json:"groups"`
	Enabled      bool   `json:"enabled"`
	ID           int    `json:"id"`
	DateAdded    int64  `json:"date_added"`
	DateModified int64  `json:"date_modified"`
}

// DomainRef identifies a domain for batch deletion.
type DomainRef struct {
	Item string     `json:"item"`
	Type DomainType `json:"type"`
	Kind DomainKind `json:"kind"`
}

// domainsEnvelope wraps the list/mutation response for domains.
type domainsEnvelope struct {
	Domains []Domain `json:"domains"`
	Took    float64  `json:"took"`
}

// domainRequest is the body of a domain create/update.
type domainRequest struct {
	Comment string `json:"comment"`
	Groups  []int  `json:"groups"`
	Enabled bool   `json:"enabled"`
}

// domainPath builds the /domains/{type}/{kind}/{domain} path, URL-escaping the
// domain segment so regex patterns (which contain reserved characters) are
// transmitted safely.
func domainPath(dt DomainType, dk DomainKind, domain string) string {
	return fmt.Sprintf("/domains/%s/%s/%s", dt, dk, url.PathEscape(domain))
}

// Domains fetches GET /api/domains — every managed allow/deny domain.
func (c *Client) Domains(ctx context.Context) ([]Domain, error) {
	var out domainsEnvelope
	if err := c.do(ctx, http.MethodGet, "/domains", nil, &out); err != nil {
		return nil, err
	}
	return out.Domains, nil
}

// AddDomain creates a domain via POST /api/domains/{type}/{kind}/{domain}. The
// new entry is created enabled.
func (c *Client) AddDomain(ctx context.Context, dt DomainType, dk DomainKind, domain, comment string, groups []int) (Domain, error) {
	body := domainRequest{Comment: comment, Groups: groups, Enabled: true}
	var out domainsEnvelope
	if err := c.do(ctx, http.MethodPost, domainPath(dt, dk, domain), body, &out); err != nil {
		return Domain{}, err
	}
	if len(out.Domains) > 0 {
		return out.Domains[0], nil
	}
	return Domain{}, nil
}

// UpdateDomain updates a domain via PUT /api/domains/{type}/{kind}/{domain}.
func (c *Client) UpdateDomain(ctx context.Context, dt DomainType, dk DomainKind, domain, comment string, groups []int, enabled bool) (Domain, error) {
	body := domainRequest{Comment: comment, Groups: groups, Enabled: enabled}
	var out domainsEnvelope
	if err := c.do(ctx, http.MethodPut, domainPath(dt, dk, domain), body, &out); err != nil {
		return Domain{}, err
	}
	if len(out.Domains) > 0 {
		return out.Domains[0], nil
	}
	return Domain{}, nil
}

// DeleteDomain removes a domain via DELETE /api/domains/{type}/{kind}/{domain}.
func (c *Client) DeleteDomain(ctx context.Context, dt DomainType, dk DomainKind, domain string) error {
	return c.do(ctx, http.MethodDelete, domainPath(dt, dk, domain), nil, nil)
}

// BatchDeleteDomains removes multiple domains via POST /api/domains:batchDelete.
func (c *Client) BatchDeleteDomains(ctx context.Context, items []DomainRef) error {
	return c.do(ctx, http.MethodPost, "/domains:batchDelete", items, nil)
}
