package pihole

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"sync"
	"time"
)

const (
	memberTimeout  = 2 * time.Second
	memberCooldown = 30 * time.Second
)

// Member is one named Pi-hole instance of a Fleet.
type Member struct {
	Name   string
	Client *Client
}

// Fleet fans read calls out to every member and merges the results. A member
// that fails is skipped for memberCooldown, then retried.
type Fleet struct {
	members []Member
	failed  map[string]error // instances whose client couldn't be built

	mu   sync.Mutex
	down map[string]time.Time // last failure time
	now  func() time.Time
}

// NewFleet returns a Fleet over members. failed lists instances whose client
// couldn't be built; they are always reported by Unreachable.
func NewFleet(members []Member, failed map[string]error) *Fleet {
	return &Fleet{
		members: members,
		failed:  failed,
		down:    map[string]time.Time{},
		now:     time.Now,
	}
}

// Names returns the member names in config order.
func (g *Fleet) Names() []string {
	out := make([]string, len(g.members))
	for i, m := range g.members {
		out[i] = m.Name
	}
	return out
}

// Members returns a copy of the members in config order.
func (g *Fleet) Members() []Member { return slices.Clone(g.members) }

// Client returns the named member's client, or nil.
func (g *Fleet) Client(name string) *Client {
	for _, m := range g.members {
		if m.Name == name {
			return m.Client
		}
	}
	return nil
}

// Len counts the members plus the instances that failed to build.
func (g *Fleet) Len() int { return len(g.members) + len(g.failed) }

// Unreachable returns, sorted, the instances that failed to build and the
// members that failed within the cooldown.
func (g *Fleet) Unreachable() []string {
	out := make([]string, 0, len(g.failed))
	for name := range g.failed {
		out = append(out, name)
	}
	g.mu.Lock()
	for name := range g.down {
		if g.cooling(name) {
			out = append(out, name)
		}
	}
	g.mu.Unlock()
	slices.Sort(out)
	return out
}

// cooling reports whether name failed within the cooldown. Caller holds mu.
func (g *Fleet) cooling(name string) bool {
	t, ok := g.down[name]
	return ok && g.now().Sub(t) < memberCooldown
}

// recovered clears name's failure mark unless a call newer than start set
// it. Caller holds mu.
func (g *Fleet) recovered(name string, start time.Time) {
	if t, ok := g.down[name]; ok && t.Before(start) {
		delete(g.down, name)
	}
}

// fanOut calls every member not in cooldown concurrently, each with its own
// deadline. It returns the successful results and their member names in
// config order, and errors only when no member answered.
func fanOut[T any](
	g *Fleet,
	ctx context.Context,
	call func(context.Context, *Client) (T, error),
) ([]T, []string, error) {
	g.mu.Lock()
	start := g.now()
	var live []Member
	for _, m := range g.members {
		if !g.cooling(m.Name) {
			live = append(live, m)
		}
	}
	g.mu.Unlock()

	res := make([]T, len(live))
	errs := make([]error, len(live))
	var wg sync.WaitGroup
	for i, m := range live {
		wg.Go(func() {
			cctx, cancel := context.WithTimeout(ctx, memberTimeout)
			defer cancel()
			res[i], errs[i] = call(cctx, m.Client)
		})
	}
	wg.Wait()
	// The caller gave up; that says nothing about member health.
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	var out []T
	var names []string
	g.mu.Lock()
	for i, m := range live {
		if errs[i] != nil {
			g.down[m.Name] = g.now()
			errs[i] = fmt.Errorf("%s: %w", m.Name, errs[i])
			continue
		}
		g.recovered(m.Name, start)
		out = append(out, res[i])
		names = append(names, m.Name)
	}
	g.mu.Unlock()
	if len(out) == 0 {
		if err := errors.Join(errs...); err != nil {
			return nil, nil, fmt.Errorf("all instances unreachable: %w", err)
		}
		return nil, nil, errors.New("all instances unreachable")
	}
	return out, names, nil
}

// flip adapts a Client method expression to fanOut's call order.
func flip[T any](
	f func(*Client, context.Context) (T, error),
) func(context.Context, *Client) (T, error) {
	return func(ctx context.Context, c *Client) (T, error) { return f(c, ctx) }
}

// Summary merges GET /api/stats/summary across members.
func (g *Fleet) Summary(ctx context.Context) (Summary, error) {
	in, _, err := fanOut(g, ctx, flip((*Client).Summary))
	if err != nil {
		return Summary{}, err
	}
	return mergeSummary(in), nil
}

// History merges GET /api/history across members.
func (g *Fleet) History(ctx context.Context) (History, error) {
	in, _, err := fanOut(g, ctx, flip((*Client).History))
	if err != nil {
		return History{}, err
	}
	return mergeHistory(in), nil
}

// QueryTypes merges GET /api/stats/query_types across members.
func (g *Fleet) QueryTypes(ctx context.Context) (QueryTypes, error) {
	in, _, err := fanOut(g, ctx, flip((*Client).QueryTypes))
	if err != nil {
		return QueryTypes{}, err
	}
	return mergeQueryTypes(in), nil
}

// Upstreams merges GET /api/stats/upstreams across members.
func (g *Fleet) Upstreams(ctx context.Context) (Upstreams, error) {
	in, _, err := fanOut(g, ctx, flip((*Client).Upstreams))
	if err != nil {
		return Upstreams{}, err
	}
	return mergeUpstreams(in), nil
}

// TopDomains over-fetches count*3 per member, then merges and trims to count.
func (g *Fleet) TopDomains(
	ctx context.Context,
	blocked bool,
	count int,
) (TopDomains, error) {
	in, _, err := fanOut(g, ctx,
		func(ctx context.Context, c *Client) (TopDomains, error) {
			return c.TopDomains(ctx, blocked, count*3)
		})
	if err != nil {
		return TopDomains{}, err
	}
	return mergeTopDomains(in, count), nil
}

// TopClients over-fetches count*3 per member, then merges and trims to count.
func (g *Fleet) TopClients(
	ctx context.Context,
	blocked bool,
	count int,
) (TopClients, error) {
	in, _, err := fanOut(g, ctx,
		func(ctx context.Context, c *Client) (TopClients, error) {
			return c.TopClients(ctx, blocked, count*3)
		})
	if err != nil {
		return TopClients{}, err
	}
	return mergeTopClients(in, count), nil
}

// Queries sends f to every member, tags each row with its instance, and keeps
// the newest *f.Length rows (all rows when Length is unset or 0).
func (g *Fleet) Queries(
	ctx context.Context,
	f QueryFilter,
) (QueriesPage, error) {
	in, names, err := fanOut(g, ctx,
		func(ctx context.Context, c *Client) (QueriesPage, error) {
			return c.Queries(ctx, f)
		})
	if err != nil {
		return QueriesPage{}, err
	}
	length := math.MaxInt
	if f.Length != nil && *f.Length > 0 {
		length = *f.Length
	}
	return mergeQueries(in, names, length), nil
}

// Blocking merges GET /api/dns/blocking across members. Blocking reports
// whether every reachable member blocks; Total counts every instance, so
// On < Total while Blocking means some are unreachable.
func (g *Fleet) Blocking(ctx context.Context) (BlockingStatus, error) {
	in, _, err := fanOut(g, ctx, flip((*Client).Blocking))
	if err != nil {
		return BlockingStatus{}, err
	}
	out := mergeBlocking(in)
	out.Total = g.Len()
	return out, nil
}

// SetBlocking posts the same state and timer to every member, including those
// in cooldown, since writes are explicit user actions. The error names each
// member that failed.
func (g *Fleet) SetBlocking(
	ctx context.Context,
	blocking bool,
	timer *float64,
) error {
	g.mu.Lock()
	start := g.now()
	g.mu.Unlock()
	errs := make([]error, len(g.members))
	var wg sync.WaitGroup
	for i, m := range g.members {
		wg.Go(func() {
			cctx, cancel := context.WithTimeout(ctx, memberTimeout)
			defer cancel()
			if err := m.Client.SetBlocking(cctx, blocking, timer); err != nil {
				errs[i] = fmt.Errorf(
					"blocking not changed on %s: %w", m.Name, err)
			}
		})
	}
	wg.Wait()
	if ctx.Err() == nil {
		g.mu.Lock()
		for i, m := range g.members {
			if errs[i] == nil {
				g.recovered(m.Name, start)
			}
		}
		g.mu.Unlock()
	}
	for _, name := range slices.Sorted(maps.Keys(g.failed)) {
		errs = append(errs, fmt.Errorf(
			"blocking not changed on %s: %w", name, g.failed[name]))
	}
	return errors.Join(errs...)
}
