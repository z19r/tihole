package pihole

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// HostRecord is a local A/AAAA host entry stored in the dns.hosts config
// element as the string "IP domain".
type HostRecord struct {
	IP     string
	Domain string
}

// CNAMERecord is a local CNAME entry stored in the dns.cnameRecords config
// element as the string "domain,target[,ttl]".
type CNAMERecord struct {
	Domain string
	Target string
	TTL    int
}

// parseHostRecord splits an "IP domain" config string into a HostRecord. The
// separator is whitespace; the domain is the remainder after the first field.
func parseHostRecord(s string) HostRecord {
	fields := strings.Fields(s)
	switch len(fields) {
	case 0:
		return HostRecord{}
	case 1:
		return HostRecord{IP: fields[0]}
	default:
		return HostRecord{IP: fields[0], Domain: strings.Join(fields[1:], " ")}
	}
}

// parseCNAMERecord splits a "domain,target[,ttl]" config string into a
// CNAMERecord. A malformed or absent ttl segment yields TTL 0.
func parseCNAMERecord(s string) CNAMERecord {
	parts := strings.Split(s, ",")
	rec := CNAMERecord{}
	if len(parts) > 0 {
		rec.Domain = parts[0]
	}
	if len(parts) > 1 {
		rec.Target = parts[1]
	}
	if len(parts) > 2 {
		if ttl, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
			rec.TTL = ttl
		}
	}
	return rec
}

// hostValue renders a HostRecord's config-string value ("IP domain").
func hostValue(ip, domain string) string {
	return ip + " " + domain
}

// cnameValue renders a CNAMERecord's config-string value. A non-positive ttl is
// omitted, yielding "domain,target".
func cnameValue(domain, target string, ttl int) string {
	if ttl > 0 {
		return fmt.Sprintf("%s,%s,%d", domain, target, ttl)
	}
	return domain + "," + target
}

// HostRecords fetches the dns.hosts config element and parses each "IP domain"
// entry into a HostRecord.
func (c *Client) HostRecords(ctx context.Context) ([]HostRecord, error) {
	values, err := c.stringArrayElement(ctx, "dns.hosts")
	if err != nil {
		return nil, err
	}
	records := make([]HostRecord, 0, len(values))
	for _, v := range values {
		records = append(records, parseHostRecord(v))
	}
	return records, nil
}

// AddHostRecord appends an A/AAAA host entry via
// PUT /api/config/dns/hosts/{IP domain}.
func (c *Client) AddHostRecord(ctx context.Context, ip, domain string) error {
	return c.AddConfigItem(ctx, "dns.hosts", hostValue(ip, domain))
}

// DeleteHostRecord removes an A/AAAA host entry via
// DELETE /api/config/dns/hosts/{IP domain}.
func (c *Client) DeleteHostRecord(ctx context.Context, ip, domain string) error {
	return c.DeleteConfigItem(ctx, "dns.hosts", hostValue(ip, domain))
}

// CNAMERecords fetches the dns.cnameRecords config element and parses each
// "domain,target[,ttl]" entry into a CNAMERecord.
func (c *Client) CNAMERecords(ctx context.Context) ([]CNAMERecord, error) {
	values, err := c.stringArrayElement(ctx, "dns.cnameRecords")
	if err != nil {
		return nil, err
	}
	records := make([]CNAMERecord, 0, len(values))
	for _, v := range values {
		records = append(records, parseCNAMERecord(v))
	}
	return records, nil
}

// AddCNAMERecord appends a CNAME entry via
// PUT /api/config/dns/cnameRecords/{value}. A non-positive ttl is omitted.
func (c *Client) AddCNAMERecord(ctx context.Context, domain, target string, ttl int) error {
	return c.AddConfigItem(ctx, "dns.cnameRecords", cnameValue(domain, target, ttl))
}

// DeleteCNAMERecord removes a CNAME entry via
// DELETE /api/config/dns/cnameRecords/{value}.
func (c *Client) DeleteCNAMERecord(ctx context.Context, domain, target string, ttl int) error {
	return c.DeleteConfigItem(ctx, "dns.cnameRecords", cnameValue(domain, target, ttl))
}

// stringArrayElement fetches a config element whose value is a JSON array of
// strings, following the same dotted-element addressing as ConfigElement. The
// element's value lives at config[<last segment>] in the response subtree.
func (c *Client) stringArrayElement(ctx context.Context, element string) ([]string, error) {
	path := "/config/" + escapeElement(element)
	var out stringArrayEnvelope
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.value(element), nil
}

// stringArrayEnvelope decodes a config element response where the addressed
// value is a []string. FTL nests the value under the `config` key mirroring the
// element's dotted path (e.g. {"config":{"dns":{"hosts":[...]}}}).
type stringArrayEnvelope struct {
	Config map[string]any `json:"config"`
	Took   float64        `json:"took"`
}

// value walks the nested `config` object along the element's dotted path and
// returns the leaf as a []string, ignoring any non-string entries.
func (e stringArrayEnvelope) value(element string) []string {
	var node any = e.Config
	for _, seg := range strings.Split(element, ".") {
		m, ok := node.(map[string]any)
		if !ok {
			return nil
		}
		node = m[seg]
	}
	arr, ok := node.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
