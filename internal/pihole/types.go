package pihole

import "encoding/json"

// Session is the object returned inside the `session` key of an /api/auth
// response.
type Session struct {
	Valid    bool   `json:"valid"`
	SID      string `json:"sid"`
	CSRF     string `json:"csrf"`
	Validity int    `json:"validity"`
	Message  string `json:"message"`
	TOTP     bool   `json:"totp"`
}

// sessionEnvelope wraps a session object as returned by the auth endpoints.
type sessionEnvelope struct {
	Session Session `json:"session"`
	Took    float64 `json:"took"`
}

// Summary is the response of GET /api/stats/summary. The FTL summary is a
// nested object; the fields most consumers need are modeled explicitly and the
// full payload is retained in Raw for anything not covered here.
type Summary struct {
	Queries struct {
		Total          int     `json:"total"`
		Blocked        int     `json:"blocked"`
		PercentBlocked float64 `json:"percent_blocked"`
		UniqueDomains  int     `json:"unique_domains"`
		Forwarded      int     `json:"forwarded"`
		Cached         int     `json:"cached"`
		Frequency      float64 `json:"frequency"`
	} `json:"queries"`
	Clients struct {
		Active int `json:"active"`
		Total  int `json:"total"`
	} `json:"clients"`
	Gravity struct {
		DomainsBeingBlocked int `json:"domains_being_blocked"`
		LastUpdate          int `json:"last_update"`
	} `json:"gravity"`
	Took float64         `json:"took"`
	Raw  json.RawMessage `json:"-"`
}

// UpstreamServer is a single upstream entry in an /api/stats/upstreams
// response.
type UpstreamServer struct {
	IP         string `json:"ip"`
	Name       string `json:"name"`
	Port       int    `json:"port"`
	Count      int    `json:"count"`
	Statistics struct {
		Response float64 `json:"response"`
		Variance float64 `json:"variance"`
	} `json:"statistics"`
}

// Upstreams is the response of GET /api/stats/upstreams.
type Upstreams struct {
	Upstreams        []UpstreamServer `json:"upstreams"`
	ForwardedQueries int              `json:"forwarded_queries"`
	TotalQueries     int              `json:"total_queries"`
	Took             float64          `json:"took"`
}

// QueryTypes is the response of GET /api/stats/query_types. The `types` map is
// keyed by query type name (e.g. "A", "AAAA") with a count value.
type QueryTypes struct {
	Types map[string]int `json:"types"`
	Took  float64        `json:"took"`
}

// TopDomains is the response of GET /api/stats/top_domains.
type TopDomains struct {
	Domains []struct {
		Domain string `json:"domain"`
		Count  int    `json:"count"`
	} `json:"domains"`
	TotalQueries   int     `json:"total_queries"`
	BlockedQueries int     `json:"blocked_queries"`
	Took           float64 `json:"took"`
}

// TopClients is the response of GET /api/stats/top_clients.
type TopClients struct {
	Clients []struct {
		IP    string `json:"ip"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	} `json:"clients"`
	TotalQueries   int     `json:"total_queries"`
	BlockedQueries int     `json:"blocked_queries"`
	Took           float64 `json:"took"`
}

// HistoryPoint is a single timestamped bucket in an /api/history response.
type HistoryPoint struct {
	Timestamp int `json:"timestamp"`
	Total     int `json:"total"`
	Cached    int `json:"cached"`
	Blocked   int `json:"blocked"`
	Forwarded int `json:"forwarded"`
}

// History is the response of GET /api/history.
type History struct {
	History []HistoryPoint `json:"history"`
	Took    float64        `json:"took"`
}

// HistoryClientPoint is a single bucket in an /api/history/clients response,
// where per-client counts are keyed by client identifier.
type HistoryClientPoint struct {
	Timestamp int            `json:"timestamp"`
	Data      map[string]int `json:"data"`
}

// HistoryClients is the response of GET /api/history/clients.
type HistoryClients struct {
	Clients map[string]struct {
		Name  string `json:"name"`
		Total int    `json:"total"`
	} `json:"clients"`
	History []HistoryClientPoint `json:"history"`
	Took    float64              `json:"took"`
}

// Query is a single row in an /api/queries response.
type Query struct {
	ID       int64           `json:"id"`
	Time     float64         `json:"time"`
	Type     string          `json:"type"`
	Domain   string          `json:"domain"`
	CNAME    string          `json:"cname"`
	Status   string          `json:"status"`
	Client   QueryClient     `json:"client"`
	DNSSEC   string          `json:"dnssec"`
	Reply    QueryReply      `json:"reply"`
	ListID   *int64          `json:"list_id"`
	Upstream string          `json:"upstream"`
	Raw      json.RawMessage `json:"-"`
}

// QueryClient is the client sub-object of a query row.
type QueryClient struct {
	IP   string `json:"ip"`
	Name string `json:"name"`
}

// QueryReply is the reply sub-object of a query row.
type QueryReply struct {
	Type string  `json:"type"`
	Time float64 `json:"time"`
}

// QueriesPage is the response of GET /api/queries, including cursor-pagination
// metadata.
type QueriesPage struct {
	Queries         []Query `json:"queries"`
	Cursor          *int64  `json:"cursor"`
	RecordsFiltered int     `json:"recordsFiltered"`
	RecordsTotal    int     `json:"recordsTotal"`
	DraftCursor     *int64  `json:"draft_cursor"`
	Took            float64 `json:"took"`
}

// BlockingStatus is the response of GET /api/dns/blocking. Timer is the number
// of seconds remaining until blocking auto-toggles back, or nil when there is
// no active timer.
//
// FTL v6 reports the state as a STATUS STRING on this GET ("enabled",
// "disabled", "failure", "unknown") — not a boolean — even though the matching
// POST body takes a boolean. Status preserves the raw value; Blocking is the
// convenience boolean (true only for "enabled") the rest of the app reads.
type BlockingStatus struct {
	Blocking bool
	Status   string
	Timer    *float64
	Took     float64
}

// blockingWire is the FTL v6 wire shape of GET /api/dns/blocking.
type blockingWire struct {
	Blocking string   `json:"blocking"`
	Timer    *float64 `json:"timer"`
	Took     float64  `json:"took"`
}

// UnmarshalJSON maps FTL's status string into the convenience boolean while
// keeping the raw status available for display.
func (b *BlockingStatus) UnmarshalJSON(data []byte) error {
	var w blockingWire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	b.Status = w.Blocking
	b.Blocking = w.Blocking == "enabled"
	b.Timer = w.Timer
	b.Took = w.Took
	return nil
}
