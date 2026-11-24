# Combined Stats Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** A global `m` toggle shows dashboard stats, the query log and
blocking state/actions merged across every configured Pi-hole.

**Architecture:** `pihole.Fleet` fans read calls out to one `*Client` per
instance and merges them into the existing response types with pure merge
functions. `core.Source` is the 9-method interface both `*Client` and
`*Fleet` satisfy. `AppContext.Source()` returns the group when combined,
otherwise the active client. Screens swap `m.ctx.API` for `m.ctx.Source()`
only on combined-capable calls.

**Tech Stack:** Go 1.26, Bubble Tea v2 (`charm.land/*/v2`), `net/http`,
`httptest`, `gopkg.in/yaml` (existing).

**Spec:** `docs/superpowers/specs/2026-11-23-combined-stats-design.md`

**Status:** complete. Tasks 1–9 shipped in #34, Task 10 in #35, Task 11
in #36; released in v1.4.0 (2026-11-24).

## Global Constraints

- 80-column lines: run `just fmt` before each commit; `just lint` must pass.
- No new dependencies.
- Combined includes **every** configured instance; no subset selection.
- In combined mode, writes other than blocking on/off go to `ctx.API` (the
  active instance), never to the group.
- Per-member deadline is **2s**. The failure cooldown is **30s**. Top-N
  over-fetch is **3×count**.
- `m` with fewer than 2 instances: no state change; set the root error banner
  to `combined view needs two or more instances`.
- Commits: imperative subject, no AI attribution lines.

## Review Focus

1. **A member's password env var is unset at startup.** Expect: that member is
   skipped, listed as unreachable, and the rest still merge (Task 4 test
   `TestBuildFleetSkipsUnbuildableMember`).
2. **Instances edited in Settings while combined is on.** Expect: the group is
   rebuilt, the `Combined` flag survives the save (`cloneConfig` must copy
   it), and dropping to one instance turns combined off in the UI (Task 3 test
   `TestCloneConfigKeepsCombined`, Task 4 test
   `TestInstancesChangeRebuildsFleet`).
3. **History buckets that don't line up across members** (one Pi-hole
   restarted mid-bucket or has a shorter history). Expect: buckets are summed
   by exact timestamp and unmatched buckets are kept, not dropped (Task 1
   `TestMergeHistoryKeepsUnmatchedBuckets`).
4. **The same client IP shows a name on one Pi-hole and an empty name on the
   other.** Expect: the merged top-clients row keeps the non-empty name (Task
   1 `TestMergeTopClientsKeepsName`).
5. **Every member down.** Expect: an error, never an all-zero "success" that
   paints the dashboard as 0 queries (Task 2 `TestFleetAllFailErrors`).

---

## File Structure

| File | Responsibility |
|---|---|
| `internal/pihole/types.go` | add `Query.Instance`, `Summary.GravityMismatch`, `BlockingStatus.On/Total` |
| `internal/pihole/merge.go` (new) | pure merge funcs |
| `internal/pihole/merge_test.go` (new) | merge table tests |
| `internal/pihole/fleet.go` (new) | `Fleet`, fan-out, deadline, cooldown, `Unreachable` |
| `internal/pihole/fleet_test.go` (new) | fan-out tests with `httptest` |
| `internal/config/config.go` | `Combined bool` |
| `internal/tui/core/context.go` | `Source` interface, `Fleet`, `Combined`, `Source()` |
| `internal/tui/core/keys.go` | `ToggleCombined` binding |
| `internal/tui/screens/settings/connections.go` | `cloneConfig` copies `Combined` |
| `internal/tui/client.go` | `buildFleet` |
| `internal/tui/app.go` | toggle, group lifecycle, label, blocking via `Source()` |
| `internal/tui/components/blocking.go`, `statusbar.go` | counts in the pill |
| `internal/tui/screens/dashboard/*.go` | read via `Source()`, warning, gravity `≠`, system title |
| `internal/tui/screens/querylog/*.go` | read via `Source()`, Pi-hole column, footer |
| `internal/tui/screens/blocking/blocking.go` | read/write via `Source()`, heading |
| `docs/keys.md`, `README.md`, `docs/architecture.md` | docs |

---

### Task 1: Merge functions and type fields

**Files:**
- Modify: `internal/pihole/types.go` (Summary, Query, BlockingStatus)
- Create: `internal/pihole/merge.go`, `internal/pihole/merge_test.go`

**Interfaces:**
- Produces (all unexported, used by Task 2):
  - `mergeSummary(in []Summary) Summary`
  - `mergeHistory(in []History) History`
  - `mergeQueryTypes(in []QueryTypes) QueryTypes`
  - `mergeUpstreams(in []Upstreams) Upstreams`
  - `mergeTopDomains(in []TopDomains, count int) TopDomains`
  - `mergeTopClients(in []TopClients, count int) TopClients`
  - `mergeQueries(in []QueriesPage, names []string, length int) QueriesPage`
    (`names[i]` tags `in[i]` rows)
  - `mergeBlocking(in []BlockingStatus) BlockingStatus`
- Produces (exported fields, `json:"-"`): `Summary.GravityMismatch bool`,
  `Query.Instance string`, `BlockingStatus.On int`,
  `BlockingStatus.Total int`.

- [x] **Step 1: Write failing tests in `merge_test.go`**, one test per rule
  in the spec's merge table. Required tests and key assertions:
  - `TestMergeSummarySumsAndRecomputesPercent`: totals 100/10 and 300/90 →
    Total 400, Blocked 100, PercentBlocked 25.0; Frequency summed.
  - `TestMergeSummaryZeroTotal`: all zero → PercentBlocked 0 (no NaN).
  - `TestMergeSummaryMaxFields`: Clients.Active 5 and 7 → 7; UniqueDomains
    max; Gravity 1000 and 1200 → 1200 with `GravityMismatch` true; equal
    gravity → false; LastUpdate min.
  - `TestMergeHistorySumsByTimestamp`: two members, same 3 timestamps → per
    field sums, ascending order.
  - `TestMergeHistoryKeepsUnmatchedBuckets`: timestamps {0,600} and
    {600,1200} → 3 buckets 0/600/1200, 600 summed.
  - `TestMergeQueryTypesSumsKeys`.
  - `TestMergeUpstreamsWeighted`: same `IP:Port` with counts 1 and 3,
    Response 4 and 8 → Count 4, Response 7.0; a distinct upstream is kept;
    Forwarded/TotalQueries summed.
  - `TestMergeTopDomainsSumsSortsTrims`: counts merge by Domain, sort desc,
    ties by Domain asc, trim to `count`; totals summed.
  - `TestMergeTopClientsKeepsName`: IP `10.0.0.5` named "" then "tv" → "tv".
  - `TestMergeQueriesOrderTagTrim`: two pages, times interleave, length 3 →
    3 rows newest first, each `Instance` set from `names`; RecordsFiltered
    and RecordsTotal summed; Cursor and DraftCursor nil.
  - `TestMergeBlocking`: {on}, {off, Timer 30}, {off, Timer 10} → On 1,
    Total 3, Blocking false, Timer 10. All on → Blocking true, Timer nil.

- [x] **Step 2: Run** `go test ./internal/pihole -run Merge` → FAIL
  (undefined).

- [x] **Step 3: Implement `merge.go` and add the type fields.** Ties in
  top-N: count desc, then key asc. `mergeQueries` sorts by `Time` desc with
  `sort.SliceStable`. Put the shortcut comment on `mergeTopDomains`:
  `// shortcut: top-N merge over-fetches 3x, exact only if each member returns everything`
  (wrap at 80 columns).

- [x] **Step 4: Run** `go test ./internal/pihole` → PASS.

- [x] **Step 5: Commit** `just fmt && just lint`, then commit
  `feat(pihole): merge rules for combined stats`.

---

### Task 2: `pihole.Fleet` fan-out

**Files:**
- Create: `internal/pihole/fleet.go`, `internal/pihole/fleet_test.go`

**Interfaces:**
- Consumes: Task 1 merge funcs.
- Produces:
  ```go
  type Member struct { Name string; Client *Client }
  func NewFleet(members []Member, failed map[string]error) *Fleet
  func (g *Fleet) Names() []string           // config order
  func (g *Fleet) Len() int                  // members + failed
  func (g *Fleet) Unreachable() []string     // sorted
  func (g *Fleet) Summary(ctx) (Summary, error)
  func (g *Fleet) History(ctx) (History, error)
  func (g *Fleet) QueryTypes(ctx) (QueryTypes, error)
  func (g *Fleet) Upstreams(ctx) (Upstreams, error)
  func (g *Fleet) TopDomains(ctx, blocked bool, count int) (TopDomains, error)
  func (g *Fleet) TopClients(ctx, blocked bool, count int) (TopClients, error)
  func (g *Fleet) Queries(ctx, f QueryFilter) (QueriesPage, error)
  func (g *Fleet) Blocking(ctx) (BlockingStatus, error)
  func (g *Fleet) SetBlocking(ctx, blocking bool, timer *float64) error
  ```
  `failed` holds instances whose client couldn't be built. They are always in
  `Unreachable()` and count in `Len()`.

- [x] **Step 1: Write failing tests** using an `httptest.Server` per member
  that serves `/api/auth`, `/api/stats/summary` and `/api/dns/blocking`. Copy
  the server setup from existing `internal/pihole/*_test.go` client tests.
  - `TestFleetSummaryMerges`: two members → summed Total.
  - `TestFleetPartialFailure`: one server returns 500 → result from the other,
    no error, `Unreachable()` == ["b"].
  - `TestFleetAllFailErrors`: both 500 → non-nil error.
  - `TestFleetSlowMemberDeadline`: member b sleeps 3s → the call returns in
    under 2.5s with a's data, and b is unreachable.
  - `TestFleetCooldownSkipsThenRetries`: inject `g.now` as a fake clock;
    after b fails, a second call within 30s makes no request to b (count hits
    on b's server). After advancing the clock 31s, b is called again, and on
    success it leaves `Unreachable()`.
  - `TestFleetTopDomainsOverfetch`: the server sees `count=30` when called
    with count 10.
  - `TestFleetQueriesTagsInstance`.
  - `TestFleetSetBlockingPartialError`: b fails the POST → error message
    contains `blocking not changed on b`; a received the POST with the same
    timer.
  - `TestFleetFailedMemberListed`: `NewFleet(nil-ish members, {"c": err})`
    → `Unreachable()` contains "c" and `Len()` counts it.

- [x] **Step 2: Run** `go test ./internal/pihole -run Fleet` → FAIL.

- [x] **Step 3: Implement `fleet.go`.** Fields: `members []Member`,
  `failed map[string]error`, `mu sync.Mutex`, `down map[string]time.Time`
  (failure time), `now func() time.Time` (default `time.Now`). Constants
  `memberTimeout = 2 * time.Second`, `memberCooldown = 30 * time.Second`.
  - One generic helper does the fan-out for every read:
    `func fanOut[T any](g *Fleet, ctx context.Context, call func(context.Context, *Client) (T, error)) ([]T, []string, error)`.
    It returns the successful results and their member names in config order.
  - It skips members in cooldown, runs a goroutine per member with
    `context.WithTimeout(ctx, memberTimeout)`, and records failure/success in
    `down`. It errors with `all instances unreachable` when no result arrives.
  - `SetBlocking` does not skip cooldown members, because writes are explicit
    user actions. It joins per-member errors with `errors.Join`.

- [x] **Step 4: Run** `go test -race ./internal/pihole` → PASS.

- [x] **Step 5: Commit** `feat(pihole): fan out reads across a group of instances`.

---

### Task 3: Config flag, `core.Source`, key binding

**Files:**
- Modify: `internal/config/config.go` (`Combined bool \`yaml:"combined,omitempty"\``)
- Modify: `internal/tui/core/context.go`, `internal/tui/core/keys.go`
- Modify: `internal/tui/screens/settings/connections.go` (`cloneConfig`)
- Test: `internal/config/*_test.go`, `internal/tui/core/*_test.go`,
  `internal/tui/screens/settings/settings_test.go`

**Interfaces:**
- Produces:
  ```go
  // core.Source: the combined-capable calls; *pihole.Client and *pihole.Fleet satisfy it.
  type Source interface {
      Summary(context.Context) (pihole.Summary, error)
      History(context.Context) (pihole.History, error)
      QueryTypes(context.Context) (pihole.QueryTypes, error)
      Upstreams(context.Context) (pihole.Upstreams, error)
      TopDomains(context.Context, bool, int) (pihole.TopDomains, error)
      TopClients(context.Context, bool, int) (pihole.TopClients, error)
      Queries(context.Context, pihole.QueryFilter) (pihole.QueriesPage, error)
      Blocking(context.Context) (pihole.BlockingStatus, error)
      SetBlocking(context.Context, bool, *float64) error
  }
  // AppContext gains: Fleet *pihole.Fleet; Combined bool
  func (c *AppContext) Source() Source        // Fleet if Combined && Fleet != nil, else API
  func (c *AppContext) CombinedActive() bool  // Combined && Fleet != nil && Fleet.Len() >= 2
  // KeyMap gains ToggleCombined: keys "m", help "m" / "combined"; in ShortHelp after ToggleBlock and in FullHelp's second group
  ```
  `Source()` uses `CombinedActive()`.

- [x] **Step 1: Write failing tests.**
  - `TestCombinedRoundTrip` (config): Save then Load keeps `Combined: true`.
  - `TestSourcePicksGroup` (core): `Combined` false → the API; true with a
    2-member group → the group; true with nil Fleet → the API. Add
    `var _ Source = (*pihole.Client)(nil)` and `(*pihole.Fleet)(nil)`
    compile-time checks in `context.go`.
  - `TestCloneConfigKeepsCombined` (settings).
  - `TestToggleCombinedBinding` (core): `DefaultKeyMap().ToggleCombined`
    matches "m".

- [x] **Step 2: Run** `go test ./internal/config ./internal/tui/core ./internal/tui/screens/settings` → FAIL.

- [x] **Step 3: Implement.**

- [x] **Step 4: Run** the same packages → PASS. Also check that no screen
  already binds plain `m`: `grep -rn '"m"' internal/tui --include=*.go`
  should show nothing except keys.go.

- [x] **Step 5: Commit** `feat(tui): combined flag, Source seam and m binding`.

---

### Task 4: Root wiring and the blocking pill

**Files:**
- Modify: `internal/tui/client.go`, `internal/tui/app.go`
- Modify: `internal/tui/components/blocking.go`, `internal/tui/components/statusbar.go`
- Test: `internal/tui/*_test.go`, `internal/tui/components/*_test.go`

**Interfaces:**
- Consumes: Tasks 2–3.
- Produces:
  - `func buildFleet(cfg *config.Config, active string, activeClient *pihole.Client) *pihole.Fleet`
    (reuses `activeClient` for the active instance; `clientFor` errors go
    into `failed`).
  - `components.BlockState` gains `On, Total int`.
  - `StatusBar.Instance` is set by the root to `all (N)` (N = `Fleet.Len()`)
    when `CombinedActive()` and the active page is `PageDashboard`,
    `PageQueryLog` or `PageBlocking`; otherwise `ctx.InstanceName`.

- [x] **Step 1: Write failing tests.**
  - `TestBuildFleetSkipsUnbuildableMember`: an instance with
    `password_env` unset → in `Unreachable()`, and `Len()` is 2.
  - `TestToggleCombinedPersists`: 2 instances, press `m` →
    `ctx.Combined` true, the config file on disk has `combined: true`, and the
    returned cmd is non-nil (refocus + blocking fetch).
  - `TestToggleCombinedNeedsTwo`: 1 instance → `ctx.Combined` stays false,
    and `m.err` == `combined view needs two or more instances`.
  - `TestInstancesChangeRebuildsFleet`: send `InstancesChangedMsg` with 3
    instances → `ctx.Fleet.Len()` == 3.
  - `TestStatusLabelPerPage`: combined, dashboard → the label contains
    `all (2)`; on adlists → the active name.
  - `TestToggleBlockingPartialEnablesAll`: the block state is
    `On 1, Total 2` → `toggleBlocking`'s desired value is `true`. Expose
    `func desiredBlocking(b components.BlockState) bool` and test it.
  - Pill (components): `Total 0` renders as today. `On 2 Total 2` →
    contains `ON 2/2`. `On 0 Total 2` with countdown → `OFF 0/2 0:30`.
    `On 1 Total 2` → `PARTIAL 1/2`, styled with `th.Warn` background.

- [x] **Step 2: Run** `go test ./internal/tui/...` → FAIL.

- [x] **Step 3: Implement.**
  - `New`: `ctx.Combined = cfg.Combined`; `ctx.Fleet = buildFleet(...)`.
  - `switchInstance` and `applyInstancesChange` rebuild the group.
  - `fetchBlocking`, `toggleBlocking`, `setBlockingFor` and the blocking poll
    take `core.Source` (`m.ctx.Source()`).
  - `blockingResultMsg` copies `On/Total` into `BlockState`.
  - The toggle handler in `handleKey` (global set, next to `ToggleBlock`):
    flip `ctx.Combined` and `cfg.Combined`, then `config.Save`. On a save
    error set `m.err` and keep the flipped state. Then
    `tea.Batch(m.screens[m.active].Focus(), fetchBlocking(m.ctx.Source()))`.
  - A palette command "Toggle combined view" runs the same function.

- [x] **Step 4: Run** `go test -race ./internal/tui/...` → PASS.

- [x] **Step 5: Commit** `feat(tui): toggle combined view and span blocking across instances`.

---

### Task 5: Dashboard reads through `Source()`

**Files:**
- Modify: `internal/tui/screens/dashboard/dashboard.go` (`fetchAll`,
  `fetchSummary`, `fetchHistory`, `fetchBreakdown`, `fetchTop`),
  `render.go`
- Test: `internal/tui/screens/dashboard/dashboard_test.go`, `render_test.go`

**Interfaces:**
- Consumes: `core.Source`, `ctx.Source()`, `ctx.CombinedActive()`,
  `ctx.Fleet.Unreachable()`, `Summary.GravityMismatch`.
- The four fetch funcs take `src core.Source` instead of `*pihole.Client`.
  `fetchSystem` and the hot-temp `PatchConfig` keep `ctx.API`.

- [x] **Step 1: Write failing tests.**
  - `TestFetchAllUsesSource`: a fake `core.Source` with a fixed Summary
    (Total 4242) and combined on → after running the batch, the model's summary
    Total is 4242.
  - `TestUnreachableWarningRenders`: combined, the group has failed member
    "secondary" → `View()` contains `secondary unreachable`. With two failed
    → `secondary, tertiary unreachable`.
  - `TestGravityMismatchMarker`: `GravityMismatch` true → the gravity tile
    contains `≠`.
  - `TestSystemPanelNamesActiveWhenCombined`: combined → the system/health
    panel title contains `ctx.InstanceName`; single → unchanged.

- [x] **Step 2: Run** `go test ./internal/tui/screens/dashboard` → FAIL.

- [x] **Step 3: Implement.** The warning goes in the same line/area as the
  existing summary error (`render.go:85`) and uses `th.Warn`. The hot-temp
  `PatchConfig` path is unchanged.

- [x] **Step 4: Run** → PASS.

- [x] **Step 5: Commit** `feat(dashboard): combined stats across instances`.

---

### Task 6: Query log reads through `Source()`

**Files:**
- Modify: `internal/tui/screens/querylog/querylog.go` (`fetch`,
  `renderFooter`, detail rows near line 539), `columns.go`
  (`columnTitles`, `queryToRow`, `styledRows`)
- Test: `internal/tui/screens/querylog/*_test.go`

**Interfaces:**
- Consumes: `ctx.Source()`, `ctx.CombinedActive()`, `Query.Instance`.
- `queryToRow(q, widths, combined bool)` and
  `tableColumns(widths, combined bool)`: when `combined`, column 6 has the
  title `Pi-hole` and shows `q.Instance`.

- [x] **Step 1: Write failing tests.**
  - `TestCombinedColumnShowsInstance`: combined → header `Pi-hole`, and the
    row's last cell == `secondary`.
  - `TestSingleColumnShowsUpstream`: unchanged behavior.
  - `TestFooterShowsWriteTarget`: combined → the footer contains
    `→ primary`.
  - `TestDetailShowsInstance`: combined → the detail lists `Pi-hole` and
    `Upstream`.
  - `TestBlockActionTargetsActive`: combined, press `b` on a row from
    `secondary` → the `AddDomain` call goes to `ctx.API` (assert with an
    `httptest` server for ctx.API; the group's servers see no POST).

- [x] **Step 2: Run** → FAIL. **Step 3: Implement.** **Step 4: Run** → PASS.

- [x] **Step 5: Commit** `feat(querylog): merged log with answering instance`.

---

### Task 7: Blocking screen through `Source()`

**Files:**
- Modify: `internal/tui/screens/blocking/blocking.go` (`fetch`, `apply`,
  `View` heading/status)
- Test: `internal/tui/screens/blocking/*_test.go`

- [x] **Step 1: Write failing tests.**
  - `TestBlockingCombinedHeading`: combined with Fleet.Len 2 → the view
    contains `all (2)`.
  - `TestBlockingCombinedPartialStatus`: status `On 1 Total 2` → the view
    contains `1/2`.
  - `TestBlockingApplyUsesSource`: a fake Source records `SetBlocking(false, 30)`.

- [x] **Step 2: Run** → FAIL. **Step 3: Implement** (`fetch`/`apply` call
  `m.ctx.Source()`). **Step 4: Run** → PASS.

- [x] **Step 5: Commit** `feat(blocking): act on every instance in combined view`.

---

### Task 8: Docs and final verification

**Files:** `docs/keys.md`, `README.md`, `docs/architecture.md`

- [x] **Step 1:** In `docs/keys.md`, add `m` (toggle combined view) to the
  global table. In the README feature list, add a line about the combined
  view for redundant Pi-holes. In `docs/architecture.md`, add a short
  `pihole.Fleet` / `core.Source` paragraph.
- [x] **Step 2: Run** `just fmt && just lint && go test -race ./...` → all
  pass, with no `gofmt`/`golines` diffs.
- [x] **Step 3:** Manual check with `go run .` against two configured
  instances, one of them reachable. Press `m`: the status bar shows
  `all (2)`, the warning names the unreachable one, the pill shows a count,
  and `m` again restores single mode.
- [x] **Step 4: Commit** `docs: combined view`. Then push and open the PR,
  with no AI attribution.

---

### Task 9: Per-instance health rows on the combined dashboard

**Spec:** "Per-instance rows for metrics that cannot be combined" under
Dashboard.

**Files:**
- Modify: `internal/pihole/fleet.go` (+ `fleet_test.go`)
- Modify: `internal/tui/screens/dashboard/dashboard.go`, `health.go`
- Create: `internal/tui/screens/dashboard/instances.go`,
  `instances_test.go`

**Interfaces:**
- Produces in `pihole`:
  `func EachMember[T any](ctx context.Context, f *Fleet, call func(context.Context, *Client) (T, error)) map[string]MemberResult[T]`
  with `type MemberResult[T any] struct { Value T; Err error }`. It honors
  cooldown (members in cooldown get `Err` set to `ErrUnreachable`, a new
  exported sentinel, and are not called), uses the 2s per-member deadline,
  updates cooldown like `fanOut`, and ignores caller cancellation for the
  cooldown bookkeeping. Failed-at-build members appear with their build
  error.
- Dashboard: `type instanceHealth struct { name string; active bool; system
  pihole.SystemInfo; sensors pihole.SensorsInfo; messages int; gravity int;
  err error }` and `instancesMsg{epoch int; rows []instanceHealth}`. Rows are
  in config order with the active instance moved first.
- `func (m *Model) renderInstances() string` is used instead of
  `renderHealth()` when `ctx.CombinedActive()`.

- [x] **Step 1: Failing tests.**
  - `TestEachMemberSkipsCooldownAndReportsFailed` (pihole).
  - `TestInstancesRowsLabelled`: 2 members over httptest → the view
    contains the header `Pi-holes`, both names, `CPU`, the active marked
    `▸` first (others `●`, unreachable `○`).
  - `TestInstancesUnreachableRow`: a failed-at-build member → its row
    contains `unreachable` and `—`.
  - `TestInstancesOverflow`: 6 instances → exactly 4 instance rows plus
    `and 2 more…`.
  - `TestInstancesGravityOddOneOut`: gravity 100,100,90 → the 90 row is
    rendered with the warn style (compare against
    `th.WarnStyle().Render(...)`).
  - `TestSingleModeKeepsHealthStrip`: not combined → `renderHealth` output
    unchanged (gauges present, no `Pi-holes` header).
- [x] **Step 2:** RED run. **Step 3:** implement. Columns are truncated to
  width; while the name column would be under 12
  columns drop Gravity, Load · Up, Temp, Mem, then CPU; Health is never
  dropped; then names truncate (`TestInstancesNarrowKeepsHealth`). **Step 4:** `just fmt && just lint && go test -race ./...`.
- [x] **Step 5: Commit** `feat(dashboard): per-instance health rows in combined view`.

### Task 10: Dashboard polling and shared fan-out (audit follow-up, #35)

**Spec:** "Polling cost" and the revised **Fetch** bullet under
"Per-instance rows".

**Files:** `internal/pihole/fleet.go` (+ tests),
`internal/tui/screens/dashboard/dashboard.go`, `instances.go` (+ tests).

**Interfaces:**
- `func (g *Fleet) SummaryByMember(ctx) (Summary, map[string]Summary, error)`;
  `Fleet.Summary` calls it.
- `var ErrSkip`: an `EachMember` call returning it skips that member without
  touching its cooldown or health.
- Dashboard: `inFlight int`, `lastSlow time.Time`, `slowEvery = 5s`,
  `fetchAll(force bool)`, `errPatch string`.

- [x] **Step 1: Failing tests:** a tick during a round issues nothing;
  slow panels only every 5s; refocus mid-round starts a fresh round
  (`TestRefocusMidRoundStartsFreshRound`); per-member request counts are one
  `/stats/summary` and one `/info/system` per round in combined mode; a
  failed hot-temp PATCH doesn't mark the active row unreachable.
- [x] **Step 2–4:** RED, implement, `just fmt && just lint && go test -race ./...`.
- [x] **Step 5: Commits** `fix(dashboard): one fetch round in flight and slow
  panels every 5s`, `fix(dashboard): fetch each member's summary and system
  once per round`, `test(dashboard): pin the in-flight reset on refocus`,
  `fix(dashboard): keep the hot-temp patch error apart from system health`.

### Task 11: Ask to combine when the second instance is added (#36)

**Spec:** "Turning it on".

**Files:** `internal/tui/screens/settings/connections.go`, `settings.go`,
`combined_prompt_test.go`, `README.md`.

**Interfaces:** `Model.askCombined bool`; `handleCombinedKey`;
`CapturesInput()` also returns true while `confirm.Active`.

- [x] **Step 1: Failing tests:** `TestAddingSecondInstanceAsksAboutCombined`,
  `TestCombinedPromptYesSavesAndEmits`, `TestCombinedPromptNoLeavesItOff`,
  `TestAddingThirdInstanceDoesNotAsk`, `TestCombinedPromptCapturesInput`.
- [x] **Step 2–4:** RED, implement, `just fmt && just lint && go test -race ./...`.
- [x] **Step 5: Commit** `feat(settings): ask to combine dashboards when the
  second instance is added`.
