# Combined stats across Pi-holes — design

Date: 2026-11-23 · Status: awaiting approval

## Intent

The user runs several Pi-holes on one network **for redundancy**. A device
sends a given DNS query to one of them, so any single Pi-hole shows only part
of the traffic. The read-only views should be able to show all configured
Pi-holes as one network.

## Goals

- One global toggle, `m`, switches the read-only views between **combined**
  (every configured instance) and **single** (the active instance).
- Combined covers the **dashboard** stats panels and the **query log**.
- Merged numbers are correct: counts add up, ratios are recomputed from the
  merged totals, and nothing that overlaps between Pi-holes is double-counted.
- If one Pi-hole is down, the view still renders from the rest and says which
  instance it is missing.

## Non-goals

- Combined write actions (block a domain on all Pi-holes, sync adlists,
  groups or clients). That is a separate "manage as one" feature. The one
  deliberate exception is **blocking on/off** (see *Blocking in combined
  mode*).
- Combined Clients, System, Adlists, Domains, Groups, Local DNS or Settings
  screens. Clients is client *management* (CRUD) and System is
  per-machine. These keep using the active instance.
- Choosing a subset of instances. Combined means every configured instance.

## Anti-goals (failure even if it "works")

- A write action silently landing on a Pi-hole the user didn't expect. In
  combined mode, writes still go to the active instance (`s`), and the UI says
  so.
- Numbers that look precise but are wrong (averaged percentages, summed
  overlapping client counts, summed gravity).
- One unreachable Pi-hole blanking the combined view.

## Approaches considered

1. **`pihole.Fleet` with merge functions (picked).** A new type in
   `internal/pihole` holds one `*Client` per instance and exposes merged
   versions of only the read calls the views use: `Summary`, `History`,
   `QueryTypes`, `Upstreams`, `TopDomains`, `TopClients`, `Queries`, plus
   `Blocking`/`SetBlocking`. It fans
   out concurrently and returns the existing response types, so screens change
   only where they pick a source. The merge rules are pure functions, unit
   tested in one place.
2. **An interface both `Client` and `Fleet` satisfy.** This was rejected:
   `Client` has about 70 methods, most of them writes with no combined
   meaning, so the interface is large for little gain and hides from screens
   whether a write is going to one or many. (The 9-method `core.Source` below
   is not this approach: it covers only the combined-capable calls, so every
   other write still goes through `ctx.API` explicitly.)
3. **Merge inside each screen.** This was rejected because the fan-out and the
   partial-failure handling get duplicated, and the merge rules end up mixed
   into the rendering code.

## Design

### Package `internal/pihole`: `fleet.go`

```go
type Member struct {
    Name   string
    Client *Client
}

type Fleet struct{ members []Member }

func NewFleet(members []Member) *Fleet

// Unreachable names members that failed their most recent call (or are in
// cooldown), sorted. Screens read it at render time for the warning line.
func (g *Fleet) Unreachable() []string
```

Merged read methods keep `*Client`'s exact signatures (`(T, error)`), so the
`Fleet` and the `Client` both satisfy one small read interface, `core.Source`
(the 9 methods below). A call returns an error only when **every** member
fails; partial failures show up in `Unreachable()`. Fan-out uses one goroutine
per member under the caller's context. The merge functions are unexported pure
funcs (`mergeSummary(...)` and so on) and are tested directly.

### Merge rules

| Data | Rule |
|---|---|
| Summary.Queries.Total/Blocked/Forwarded/Cached/Frequency | sum |
| Summary.Queries.PercentBlocked | Blocked / Total × 100 from the summed totals (0 if Total is 0) |
| Summary.Queries.UniqueDomains | max (a lower bound: sets overlap and the API gives no set) |
| Summary.Clients.Active/Total | max (devices usually query both Pi-holes; summing double-counts) |
| Summary.Gravity.DomainsBeingBlocked | max; set a `GravityMismatch` flag on the result if the members differ |
| Summary.Gravity.LastUpdate | min (the oldest, so a stale member is visible) |
| History | sum the points by `Timestamp`; output sorted by time |
| QueryTypes.Types | sum per key |
| Upstreams.Upstreams | key by `IP:Port`; sum `Count`; `Response` is the count-weighted mean; `Variance` is the count-weighted mean (an approximation) |
| Upstreams.Forwarded/TotalQueries | sum |
| TopDomains / TopClients | ask each member for `3×count`, sum by `Domain` / `IP` (keeping the first non-empty `Name`), sort descending (ties by key), trim to `count`; sum the totals |
| Queries (one page) | ask each member for `length`, tag every row with `Query.Instance`, merge by `Time` descending, trim to `length`; `RecordsFiltered`/`RecordsTotal` summed; `Cursor`/`DraftCursor` nil |

The over-fetch in Top* is a deliberate approximation: an item that ranks just
outside one member's top list can be under-counted. Mark it with
`shortcut: top-N merge over-fetches 3x, exact only if each member returns
everything`.

### Types

- `Query` gains `Instance string` with `json:"-"`. It is empty for
  single-instance results.
- `Summary` gains `GravityMismatch bool` with `json:"-"`.

### App wiring (`internal/tui`)

- `core.AppContext` gains `Fleet *pihole.Fleet` and `Combined bool`.
- The root builds `Fleet` from `cfg.Instances` with the existing `clientFor`.
  It rebuilds `Fleet` in `applyInstancesChange`. The active instance's
  `*Client` is reused in the group so its session is shared, not duplicated.
- An instance whose client can't be built (for example, a missing password
  env var) is left out of the group and reported as failed.
- `m` (a new `ToggleCombined` binding in `core.DefaultKeyMap`, help text
  "combined") flips `Combined`, persists it (below) and re-focuses the current
  screen so it refetches with a new epoch. It is a no-op with a toast/banner
  "only one instance configured" when there are fewer than two instances.
- **Persistence:** `config.Config` gains `Combined bool` (yaml `combined`),
  saved with the existing `config.Save`, so a redundant setup opens combined
  every time.
- **Status bar (decided by the designer):** the root picks the label per
  page. On Dashboard, Query Log and Blocking in combined mode it reads
  `all (N)`. On every other page it shows the active instance name as today.
  The root keeps a small set of combined-capable `core.PageID`s; screens are
  not changed.
  The blocking indicator and `d`: see *Blocking in combined mode*.

### Dashboard

- `fetchSummary`, `fetchHistory`, `fetchBreakdown` and `fetchTop` take a source
  chosen at `fetchAll` time: the group when `ctx.Combined`, otherwise the
  client. A small function-typed seam per fetch is enough; no interface.
- `fetchSystem` (CPU/mem/temp/messages) and the hot-temp `PatchConfig` stay on
  the active client. The System/health panel title shows the active instance
  name in combined mode.
- A partial result (`Fleet.Unreachable()` non-empty) shows a one-line warning, "secondary unreachable", in the
  existing error/banner area. The data still renders.
- When `GravityMismatch` is set, the Gravity tile shows a `≠` marker. The
  tile's value is the max.

### Query log

- `fetch` uses `Fleet.Queries` when combined. In combined mode the
  table's last column shows **Pi-hole** (the answering instance) instead of
  Upstream, so narrow terminals keep the same column budget. The row detail
  view still shows the upstream and adds the instance.
- Block/allow on a row (`AddDomain`) goes to the active instance, as today.
  The footer hint says `→ <active>` in combined mode.
- Pagination: the screen only loads the newest page today (cursor nil), so
  merging one page per member matches current behavior.

## Blocking in combined mode (decided by the user)

With redundant Pi-holes, disabling blocking on one doesn't disable it on the
network. So in combined mode blocking state and blocking actions span every
member, on every page.

- `pihole.BlockingStatus` gains `On, Total int` (`json:"-"`; `Total == 0`
  means a single instance). `Fleet.Blocking(ctx) (BlockingStatus, error)`
  sets them, `Blocking = On == Total`, and `Timer` is the shortest remaining
  disable timer among the members that are off.
- `Fleet.SetBlocking(ctx, blocking, timer) error` fans the POST out to every
  member with the same timer. It returns an error naming each member that
  failed, for example `blocking not changed on secondary: <err>`, even when
  the others succeeded.
- `components.BlockState` gains `On, Total int`; `Total == 0` means single
  mode. The pill renders `ON 2/2` (calm), `OFF 0/2` (alarm, with countdown
  from `MinTimer`), or `PARTIAL 1/2` (warn colour) when members disagree. The
  pill never renders a bare `ON` while any member is off.
- `toggleBlocking`, `setBlockingFor`, the palette blocking commands, the
  blocking poll, and the Blocking screen's fetch/apply all go through the
  group in combined mode. The toggle's desired state is `!(On == Total)`:
  partial → enable all. A member that fails a write shows "blocking not
  changed on secondary", and the refreshed state shows the real per-member
  count.
- The Blocking screen's heading reads "all (N)" in combined mode.
- `fetchBlocking` on instance switch (`s`) still runs. In combined mode the
  pill comes from the group, so `s` does not change it.

## Slow or unreachable members (decided by the designer)

Each member call in a fan-out gets its own deadline of 2s, inside the screen's
existing 8s `fetchTimeout`. A member that fails is skipped for 30s. It is
still listed by `Unreachable()`, and retried after the
cooldown. One blackholed Pi-hole therefore costs at most a 2s lag every 30s,
instead of an 8s lag on every poll. The cooldown state lives in `Fleet`
behind a mutex and is tested with an injectable clock.

## Polling cost

The dashboard polls four times a second. Combined multiplies requests by N
instances. Each member has its own HTTP client and session, so this is N
independent connections. That is acceptable for the expected N ≤ 3. There is
no further throttling beyond the failure cooldown above.

## Testing

- Table tests for every merge function in `pihole/fleet_test.go`: sums, the
  ratio recompute, the max/min fields, gravity mismatch, history alignment,
  upstream weighting, top-N tie order, and query merge order and tagging.
- A fan-out test with an `httptest` server per member, including one failing
  member → listed by `Unreachable()`, data from the rest; all failing → error.
- Dashboard and query log model tests: combined toggle → the group is used,
  the Pi-hole column replaces Upstream, the write target stays active, and a partial
  warning renders.
- Blocking: merge counts and timers, partial write failure, toggle desired
  state from PARTIAL, pill rendering for ON/OFF/PARTIAL with counts.
- Root: `m` toggles and persists; it is a no-op with one instance; the group
  is rebuilt on instance edits.

## Left to the builder

- Exact banner wording and column widths, following existing style.
- The warning line reads `Unreachable()` once per render, so it never repeats. It
  should render one line listing the failed names.
- Doc updates: `docs/keys.md` (the `m` key), README feature list, and
  `docs/architecture.md` (Fleet).
