package pihole

import (
	"cmp"
	"slices"
	"sort"
	"strconv"
)

func mergeSummary(in []Summary) Summary {
	var out Summary
	for i, s := range in {
		q, o := s.Queries, &out.Queries
		o.Total += q.Total
		o.Blocked += q.Blocked
		o.Forwarded += q.Forwarded
		o.Cached += q.Cached
		o.Frequency += q.Frequency
		o.UniqueDomains = max(o.UniqueDomains, q.UniqueDomains)
		out.Clients.Active = max(out.Clients.Active, s.Clients.Active)
		out.Clients.Total = max(out.Clients.Total, s.Clients.Total)
		g := s.Gravity
		if i == 0 {
			out.Gravity = g
			continue
		}
		if g.DomainsBeingBlocked != out.Gravity.DomainsBeingBlocked {
			out.GravityMismatch = true
		}
		out.Gravity.DomainsBeingBlocked = max(
			out.Gravity.DomainsBeingBlocked, g.DomainsBeingBlocked)
		out.Gravity.LastUpdate = min(out.Gravity.LastUpdate, g.LastUpdate)
	}
	if out.Queries.Total > 0 {
		out.Queries.PercentBlocked =
			float64(out.Queries.Blocked) / float64(out.Queries.Total) * 100
	}
	return out
}

func mergeHistory(in []History) History {
	by := map[int]*HistoryPoint{}
	var out History
	for _, h := range in {
		for _, p := range h.History {
			m := by[p.Timestamp]
			if m == nil {
				out.History = append(out.History,
					HistoryPoint{Timestamp: p.Timestamp})
				m = &HistoryPoint{Timestamp: p.Timestamp}
				by[p.Timestamp] = m
			}
			m.Total += p.Total
			m.Cached += p.Cached
			m.Blocked += p.Blocked
			m.Forwarded += p.Forwarded
		}
	}
	for i := range out.History {
		out.History[i] = *by[out.History[i].Timestamp]
	}
	slices.SortFunc(out.History, func(a, b HistoryPoint) int {
		return cmp.Compare(a.Timestamp, b.Timestamp)
	})
	return out
}

func mergeQueryTypes(in []QueryTypes) QueryTypes {
	out := QueryTypes{Types: map[string]int{}}
	for _, q := range in {
		for k, v := range q.Types {
			out.Types[k] += v
		}
	}
	return out
}

func mergeUpstreams(in []Upstreams) Upstreams {
	var out Upstreams
	idx := map[string]int{}
	for _, u := range in {
		out.ForwardedQueries += u.ForwardedQueries
		out.TotalQueries += u.TotalQueries
		for _, s := range u.Upstreams {
			key := s.IP + ":" + strconv.Itoa(s.Port)
			i, ok := idx[key]
			if !ok {
				idx[key] = len(out.Upstreams)
				s.Statistics.Response *= float64(s.Count)
				s.Statistics.Variance *= float64(s.Count)
				out.Upstreams = append(out.Upstreams, s)
				continue
			}
			m := &out.Upstreams[i]
			c := float64(s.Count)
			m.Count += s.Count
			m.Statistics.Response += s.Statistics.Response * c
			m.Statistics.Variance += s.Statistics.Variance * c
		}
	}
	for i := range out.Upstreams {
		m := &out.Upstreams[i]
		if m.Count == 0 {
			m.Statistics.Response, m.Statistics.Variance = 0, 0
			continue
		}
		m.Statistics.Response /= float64(m.Count)
		m.Statistics.Variance /= float64(m.Count)
	}
	return out
}

// mergeTopDomains sums counts by domain, sorts and trims to count.
// shortcut: top-N merge over-fetches 3x, exact only if each member returns
// everything
func mergeTopDomains(in []TopDomains, count int) TopDomains {
	var out TopDomains
	idx := map[string]int{}
	for _, t := range in {
		out.TotalQueries += t.TotalQueries
		out.BlockedQueries += t.BlockedQueries
		for _, d := range t.Domains {
			if i, ok := idx[d.Domain]; ok {
				out.Domains[i].Count += d.Count
				continue
			}
			idx[d.Domain] = len(out.Domains)
			out.Domains = append(out.Domains, d)
		}
	}
	sort.SliceStable(out.Domains, func(i, j int) bool {
		a, b := out.Domains[i], out.Domains[j]
		return a.Count > b.Count || a.Count == b.Count && a.Domain < b.Domain
	})
	if len(out.Domains) > count {
		out.Domains = out.Domains[:count]
	}
	return out
}

func mergeTopClients(in []TopClients, count int) TopClients {
	var out TopClients
	idx := map[string]int{}
	for _, t := range in {
		out.TotalQueries += t.TotalQueries
		out.BlockedQueries += t.BlockedQueries
		for _, c := range t.Clients {
			if i, ok := idx[c.IP]; ok {
				out.Clients[i].Count += c.Count
				if out.Clients[i].Name == "" {
					out.Clients[i].Name = c.Name
				}
				continue
			}
			idx[c.IP] = len(out.Clients)
			out.Clients = append(out.Clients, c)
		}
	}
	sort.SliceStable(out.Clients, func(i, j int) bool {
		a, b := out.Clients[i], out.Clients[j]
		return a.Count > b.Count || a.Count == b.Count && a.IP < b.IP
	})
	if len(out.Clients) > count {
		out.Clients = out.Clients[:count]
	}
	return out
}

// mergeQueries tags in[i] rows with names[i]; Cursor/DraftCursor stay nil.
func mergeQueries(in []QueriesPage, names []string, length int) QueriesPage {
	var out QueriesPage
	for i, p := range in {
		out.RecordsFiltered += p.RecordsFiltered
		out.RecordsTotal += p.RecordsTotal
		for _, q := range p.Queries {
			q.Instance = names[i]
			out.Queries = append(out.Queries, q)
		}
	}
	sort.SliceStable(out.Queries, func(i, j int) bool {
		return out.Queries[i].Time > out.Queries[j].Time
	})
	if len(out.Queries) > length {
		out.Queries = out.Queries[:length]
	}
	return out
}

// mergeBlocking reports blocking only when every member blocks; Timer is the
// soonest one among members that have one.
func mergeBlocking(in []BlockingStatus) BlockingStatus {
	out := BlockingStatus{Total: len(in)}
	for _, b := range in {
		if b.Blocking {
			out.On++
		}
		if b.Timer != nil && (out.Timer == nil || *b.Timer < *out.Timer) {
			t := *b.Timer
			out.Timer = &t
		}
	}
	out.Blocking = out.Total > 0 && out.On == out.Total
	if out.Blocking {
		out.Timer = nil
	}
	return out
}
