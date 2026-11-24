# Combined stats across Pi-holes — design

Date: 2026-11-23 · Status: shipped in v1.4.0 (2026-11-24)

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
  "combined view needs two or more instances" when there are fewer than two.
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

- `fetchSummary`, `fetchHistory`, `fetchBreakdown` and `fetchTop` read
  through `ctx.Source()`: the fleet in combined mode, otherwise the active
  client.
- `fetchSystem` (CPU/mem/temp/messages) and the hot-temp `PatchConfig` stay
  on the active client. A failed hot-temp PATCH shows its own warning line
  ("hot-temperature threshold not changed: …") until the next refocus; it
  never marks the active instance unreachable.
- A partial result (`Fleet.Unreachable()` non-empty) shows a one-line warning,
  "secondary unreachable", in the existing error/banner area. The data still
  renders.
- When `GravityMismatch` is set, the Gravity tile shows a `≠` marker. The
  tile's value is the max.

#### Per-instance rows for metrics that cannot be combined (approved 2026-11-23)

CPU, memory, temperature, load/uptime and diagnostics messages are
per-machine, and gravity can differ between members. In combined mode the
Health strip is replaced by one labelled row per configured instance:

```
 Pi-holes        CPU    Mem    Temp   Load · Up      Gravity    Health
 ▸ primary       12%    41%    48°C   3% · 4d 2h     1,204,331  ✓ clear
 ● secondary      9%    37%    n/a    2% · 11d       1,204,331  ⚠ 2 issues
 ○ basement       —      —      —     unreachable
   and 2 more…
```

- **Rows:** config order, with the active instance first and marked `▸`
  (other reachable rows `●`, unreachable `○`), so the mark survives without
  colour. Plain text, no animated gauges.
- **Unreachable or failed-at-build instances** (`Fleet.Unreachable()`) get a
  dimmed `unreachable` row with `—` cells.
- **Cap:** at most **4** instance rows. If more exist, the last line reads
  `and N more…`, where N is the hidden count.
- **Gravity:** a member whose gravity differs from the most common value is
  shown in the warning colour, which ties in with the tile's `≠`.
- **Fetch (revised 2026-11-24 by the audit fixes):** each member is asked
  for each thing once per round.
  - Gravity comes from the round's summary fetch: `Fleet.SummaryByMember`
    returns the merged summary (same rules and errors as `Fleet.Summary`,
    which now calls it) plus each answering member's own summary.
  - `fetchInstances` probes the other members (System, SensorsInfo,
    Messages, concurrently, 2s per-member deadline) on the slow 5s cadence.
    The active instance is skipped with `pihole.ErrSkip` (skipped without
    touching its health or cooldown) and its row mirrors the dashboard's own
    `fetchSystem` result.
  - Members in cooldown are not called; they render as unreachable. Results
    arrive as one message keyed by instance name, guarded by the epoch.
- **Narrow widths:** while the name column would be under 12 columns,
  drop Gravity, then Load · Up, Temp, Mem and CPU, in that order. Health is
  never dropped. After that, names truncate (to `…` at worst) and lines are
  clipped to the panel instead of wrapping.
- **Single mode:** the strip is unchanged, with its animated gauges.

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
  sets them: `Total` counts every instance, including unreachable ones and
  ones that failed to build; `Blocking` means every reachable member blocks;
  and `Timer` is the shortest remaining disable timer among the members that
  are off. When `Blocking` holds but `On < Total`, the pill reads `ON x/N ⚠`
  in the warn colour.
- `Fleet.SetBlocking(ctx, blocking, timer) error` fans the POST out to every
  member with the same timer. It returns an error naming each member that
  failed, for example `blocking not changed on secondary: <err>`, even when
  the others succeeded.
- `components.BlockState` gains `On, Total int`; `Total == 0` means single
  mode. The pill renders `ON 2/2` (calm), `OFF 0/2` (alarm, with countdown
  from `MinTimer`), or `PARTIAL 1/2` (warn colour) when members disagree. The
  pill never renders a bare `ON` while any member is off. Total counts every
  instance, so with a member unreachable and the rest on it reads `ON 1/2 ⚠`
  (warn colour); `d` then disables.
- `toggleBlocking`, `setBlockingFor`, the palette blocking commands, the
  blocking poll, and the Blocking screen's fetch/apply all go through the
  group in combined mode. The toggle's desired state is "not every reachable
  member blocks": partial → enable all. A member that fails a write shows "blocking not
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

## Polling cost (revised 2026-11-24)

The dashboard ticks four times a second, but:

- **One round in flight.** A tick while the previous round's replies are
  still outstanding starts nothing (`inFlight` counts them; a refocus, page
  switch or `r` resets it, and stale-epoch replies are ignored). A slow
  Pi-hole therefore slows the dashboard down instead of piling up requests.
- **Two cadences.** Summary and the active instance's system stats refresh
  on every free tick. History, query types, upstreams, top lists and the
  per-instance probe refresh every 5s (`slowEvery`); a refocus or `r` forces
  them into the next round.
- **No duplicate calls in combined mode.** Each member gets one
  `/stats/summary` per round and, on slow rounds, one probe; the active
  instance's `/info/system` is fetched once.

Each member keeps its own HTTP client and session, so combined costs N
independent connections, acceptable for the expected N ≤ 3.

## Turning it on (added 2026-11-24)

Adding the **second** instance in Settings → Connections (`a`, or
duplicating with `D`) asks **Combine dashboards?** "Show stats from both
Pi-holes together by default? You can switch anytime with m."

- `y`/enter saves `combined: true` and announces the config change, so the
  app switches to combined view immediately. `n`/esc changes nothing.
- It is asked only when the instance count goes from one to two and combined
  view is not already on. Edits and later additions don't ask.
- While any Settings confirm dialog is open the screen captures input, so
  global keys such as `m` or `d` cannot act behind it.

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
- Dashboard cadence: a tick during a round starts nothing; slow panels only
  every 5s; refocus resets the round; per-member request counts are exactly
  one summary and one system call per round in combined mode.
- Settings: the combine prompt appears on the 2nd add only, yes persists and
  emits, no changes nothing, and it captures input.

## Left to the builder

- Exact banner wording and column widths, following existing style.
- The warning line reads `Unreachable()` once per render, so it never repeats. It
  should render one line listing the failed names.
- Doc updates (done): `docs/keys.md` (the `m` key), README feature list, and
  `docs/architecture.md` (Fleet, Source, dashboard polling).
