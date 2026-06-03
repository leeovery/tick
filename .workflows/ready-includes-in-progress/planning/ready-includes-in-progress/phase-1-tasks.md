---
phase: 1
phase_name: Ready Includes In-Progress
total: 3
---

## ready-includes-in-progress-1-1 | approved

### Task 1-1: Widen the shared ready/blocked status gate to live tasks

**Problem**: Both `tick ready` and `tick blocked` gate on `t.status = 'open'`, so an `in_progress` task falls into *neither* view — invisible to the very command used to resume interrupted work, and silently dropped from `blocked` even when it is genuinely stuck. This is the core "double invisibility" defect. The fix is to widen the single shared status gate to the live set so `ready ⊎ blocked = all live tasks` holds.

**Solution**: Change the one shared SQL literal `t.status = 'open'` to `t.status IN ('open','in_progress')` in **both** `ReadyConditions()` and `BlockedConditions()` in `internal/cli/query_helpers.go`. Leave the `negateNotExists`/De Morgan inverse machinery and the three `ReadyNo*()` `NOT EXISTS` helpers completely untouched — the symmetry is exactly why this is a one-line-per-side change. `ReadyWhereClause()` (consumed by `stats`) and `buildListQuery` (consumed by `ready`/`blocked`/`list --ready`/`list --blocked`) pick up the new gate automatically.

**Outcome**: An unblocked `in_progress` leaf appears in `tick ready`; a `blocked` `in_progress` task (unclosed blocker or blocked ancestor) appears in `tick blocked` and never in `tick ready`; `done`/`cancelled` remain in neither; every live task is in exactly one view. `--status` composes cleanly with the widened gate (`--status open` → unstarted-only, `--status in_progress` → resumptions-only, `--status done`/`cancelled` → empty).

**Do**:
- In `internal/cli/query_helpers.go`, in `ReadyConditions()`, change the first slice element from `` `t.status = 'open'` `` to `` `t.status IN ('open', 'in_progress')` ``.
- In the same file, in `BlockedConditions()`, change the first slice element of the returned slice from `` `t.status = 'open'` `` to `` `t.status IN ('open', 'in_progress')` ``. Do NOT touch `negateNotExists`, `ReadyNoUnclosedBlockers()`, `ReadyNoOpenChildren()`, `ReadyNoBlockedAncestor()`, or the `parts`/`strings.Join` disjunction construction.
- Update `internal/cli/query_helpers_test.go`: in `"ReadyConditions returns status open plus all four conditions"` change the `conditions[0]` assertion to expect `` `t.status IN ('open', 'in_progress')` ``; in `"BlockedConditions contains no SQL literals beyond status check"` and `"BlockedCondition returns open AND negation of ready subconditions"` change the `conditions[0]` assertion likewise. Keep the `len()`, `EXISTS`-count (exactly 3), `ancestors`, and `negateNotExists`-derivation assertions unchanged (the inverse machinery is unchanged).
- Rewrite `internal/cli/ready_test.go` subtest `"it excludes in_progress tasks"` (currently asserts a lone `in_progress` task is absent from `ready`): it must now INVERT — an unblocked `in_progress` leaf MUST appear in `ready`. Rename the subtest to reflect this (e.g. `"it includes unblocked in_progress leaf"`) and assert `tick-aaa111` IS present. Do not delete it.
- Rewrite `internal/cli/blocked_test.go` subtest `"it excludes in_progress tasks from output"` per spec **option (b)**: give the `in_progress` task an unclosed blocker (a separate open blocker task) so it is genuinely blocked, and assert it APPEARS in `blocked` (proving "blocked is blocked regardless of status"). Rename to reflect the new intent (e.g. `"it includes blocked in_progress task"`).
- ADD to `internal/cli/ready_test.go`: a partition test — an unblocked `in_progress` leaf appears in `ready`; ADD to `internal/cli/blocked_test.go` (or keep within the rewritten subtest): a blocked `in_progress` task appears in `blocked` and is absent from `ready`. Together these assert the partition for the `in_progress` case.
- ADD to `internal/cli/ready_test.go`: `tick ready --status open` returns unstarted ready work only (the unblocked `open` leaf, NOT the unblocked `in_progress` leaf), and `tick ready --status in_progress` returns resumptions only (the unblocked `in_progress` leaf, NOT the unblocked `open` leaf).
- Refresh the stale inline comment in `internal/cli/list_filter_test.go` subtest `"contradictory filters return empty result no error"` (line ~385): the comment `// --status done + --ready is contradictory (ready only applies to open tasks)` is now stale ("ready only applies to open tasks" is false). Update the comment to explain the empty intersection (`status IN (open,in_progress) AND status = done` is always false). The assertion (empty result) stays valid — do not change it.

**Acceptance Criteria**:
- [ ] An unblocked `in_progress` leaf task appears in `tick ready`. (Spec AC #1)
- [ ] An unblocked `open` leaf task still appears in `tick ready` (existing `ready_test.go` happy-path tests still pass — no regression). (Spec AC #2)
- [ ] A `blocked` `in_progress` task (unclosed blocker, or blocked ancestor) appears in `tick blocked` and never in `tick ready`. (Spec AC #3)
- [ ] Every live task (`open` or `in_progress`) appears in exactly one of `tick ready` / `tick blocked`; `done`/`cancelled` appear in neither. (Spec AC #4)
- [ ] `tick ready --status open` returns unstarted ready work; `tick ready --status in_progress` returns resumptions only; `--status done`/`--status cancelled` return an empty list. (Spec AC #8)
- [ ] An `in_progress` parent that exists only via the start-cascade (it has an `in_progress` child) does not appear in `tick ready`; only its leaf surfaces — disqualified by the unchanged leaf gate `ReadyNoOpenChildren()`. (Spec AC #10)
- [ ] `query_helpers_test.go` literal assertions updated; the `EXISTS`-count and `negateNotExists`-derivation assertions remain green (inverse machinery untouched).
- [ ] `ReadyConditions()` still returns 4 elements; `BlockedConditions()` still returns 2 elements.

**Tests**:
- `"it includes unblocked in_progress leaf"` (rewritten from `"it excludes in_progress tasks"`) — an unblocked `in_progress` leaf now appears in `tick ready`.
- `"it includes blocked in_progress task"` (rewritten from `blocked_test.go` `"it excludes in_progress tasks from output"`, option b) — an `in_progress` task with an unclosed blocker appears in `tick blocked`.
- `"it partitions an in_progress task into exactly one of ready/blocked"` (new) — an unblocked `in_progress` leaf is in `ready` and absent from `blocked`; a blocked `in_progress` task is in `blocked` and absent from `ready`.
- `"it returns only unstarted work for ready --status open"` (new) — `tick ready --status open` excludes the unblocked `in_progress` leaf, includes the unblocked `open` leaf.
- `"it returns only resumptions for ready --status in_progress"` (new) — `tick ready --status in_progress` includes the unblocked `in_progress` leaf, excludes the unblocked `open` leaf.
- `"it excludes parent with in_progress children"` (KEEP, ready_test.go) — leaf gate still excludes a start-cascade `in_progress` parent (AC #10).
- `"it excludes task with in_progress blocker"` (KEEP, ready_test.go) — blocker rule unchanged.
- `"ReadyConditions returns status open plus all four conditions"` (updated literal) — asserts `conditions[0] == "t.status IN ('open', 'in_progress')"`.
- `"BlockedConditions contains no SQL literals beyond status check"` (updated literal, KEEP `EXISTS`-count == 3) — gate literal updated, inverse machinery unchanged.
- `"contradictory filters return empty result no error"` (KEEP assertion, list_filter_test.go) — `--status done --ready` still returns an empty result; only the stale inline comment is refreshed to explain the now-empty intersection (`status IN (open,in_progress) AND status = done` is always false). Confirm it stays green.

**Edge Cases**:
- **Force-started blocked task.** Starting is not gated by blockers (the `start` transition constrains only `from: open`; no blocker check). An `open` task with unclosed blockers can be force-started into `in_progress`, becoming blocked-and-in-progress. It must show in `tick blocked` only, NEVER `tick ready` — verified by the rewritten `blocked_test.go` subtest (option b).
- **`in_progress` parent via start-cascade.** A parent driven to `in_progress` only because the start-cascade walked up the ancestor chain has an `in_progress` child, so `ReadyNoOpenChildren()` (which already matches `status IN ('open','in_progress')`) excludes it from `ready`; it surfaces in `blocked`. Parent and child never co-occur in `ready`. Covered by the KEPT `"it excludes parent with in_progress children"` test.
- **`done`/`cancelled` in neither.** The widened gate is `IN ('open','in_progress')`, so terminal statuses are excluded from both views (existing `ready_test.go`/`blocked_test.go` done/cancelled exclusion tests stay green).
- **`--status done`/`--status cancelled` compose to empty.** `status IN (open,in_progress) AND status = <terminal>` is always false; returns a silent empty list. No validation rejection — consistent with filter semantics everywhere.

**Context**:
> Governing invariant (spec): `ready ⊎ blocked = all live tasks`, where live = `status IN ('open','in_progress')`. Every live task is in exactly one of `ready`/`blocked`. `blocked` is the literal De Morgan inverse of `ready` over the same status gate; the inverse/negation machinery is untouched — only the shared status literal changes on each side. This is what makes the change a one-line-per-side flip and guarantees nothing is dropped.
>
> Ready definition (spec): `status IN ('open','in_progress')` AND no unclosed blocker AND no open-or-in_progress child (leaf gate — `child.status IN ('open','in_progress')`) AND no dependency-blocked ancestor. Conditions 2–4 are the existing `NOT EXISTS` conditions, unchanged. Only the status gate widens.
>
> `--status` composition (spec): `tick ready --status open` → `status IN (open,in_progress) AND status = open` → unstarted ready work (the canonical "I only want new work" query). `tick ready --status in_progress` → resumptions only. Terminal statuses compose to empty (accepted, no special validation).
>
> Actor model (spec, decided premise): tick is single-actor with no assignee. `in_progress` means "the task I started and got pulled off," so surfacing it in `ready` means "resume this." Excluding all `in_progress` to dodge multi-actor collisions would break the single-actor resumption case; multi-actor claiming is out of scope.

**Spec Reference**: `.workflows/ready-includes-in-progress/specification/ready-includes-in-progress/specification.md` — sections "Ready & Blocked Definitions", "Consequences that fall out of the definitions", "Filters, --count, and Presentation" (`--status` composition), "Affected Code Surface" (query_helpers.go), "Test Impact" (query_helpers_test.go / ready_test.go / blocked_test.go / list_filter_test.go), Acceptance Criteria #1, #2, #3, #4, #8, #10.

## ready-includes-in-progress-1-2 | approved

### Task 1-2: Float unblocked in-progress to the top of the ready view

**Problem**: With the widened gate (Task 1-1), `in_progress` tasks now appear in `ready` but sort among `open` tasks by `priority ASC, created ASC` — so a started-but-dangling task can sink below new open work, defeating the "resume interrupted work as the default" goal. `tick ready --count 1` should naturally surface in-flight work first.

**Solution**: Make the `ORDER BY` in `buildListQuery` (`internal/cli/list.go`) conditional on the ready filter `f.Ready`. When `f.Ready` is set, prepend a status-band term so the clause becomes `ORDER BY (t.status = 'in_progress') DESC, t.priority ASC, t.created ASC`; otherwise emit the current `ORDER BY t.priority ASC, t.created ASC` unchanged. Keying on `f.Ready` (not a literal command) covers both `tick ready` (which dispatches as `--ready` into `RunList` via `handleReady`) and `tick list --ready`, and excludes both plain `tick list` and `tick list --blocked`.

**Outcome**: In `tick ready` and `tick list --ready`, `in_progress` tasks sort above all `open` tasks as a band; within each band the existing `priority ASC, created ASC` tiebreak holds. `tick ready --count 1` returns the top unblocked in-flight task when one exists, otherwise the top unblocked open task. With zero `in_progress` rows the band term is uniformly false and a no-op, so ordering is byte-identical to today. Plain `tick list` and `tick list --blocked` ordering is unchanged.

**Do**:
- In `internal/cli/list.go`, in `buildListQuery`, replace the unconditional `query += " ORDER BY t.priority ASC, t.created ASC"` with a conditional: when `f.Ready` is true, append `" ORDER BY (t.status = 'in_progress') DESC, t.priority ASC, t.created ASC"`; otherwise append the current `" ORDER BY t.priority ASC, t.created ASC"`. Place this before the `--count`/`LIMIT` append so `LIMIT` applies after the resume-first ordering.
- Do NOT add any guard for narrowing filters (`--parent`, `--tag`, `--type`, `--priority`, `--status`): the float is keyed solely on `f.Ready` and applies regardless of additional narrowing.
- ADD to `internal/cli/ready_test.go`: a resume-first ordering test with mixed `in_progress`/`open` where the `in_progress` task has a WORSE priority than an `open` task (e.g. `in_progress` priority 3, `open` priority 0). Assert the `in_progress` row sorts FIRST despite the worse priority — this discriminates the band term from the priority term (it would fail if the band were absent, since the open task has the better priority).
- ADD to `internal/cli/ready_test.go` (or a list-focused test using `runList`): a zero-`in_progress` ready list whose ordering is byte-identical to the current `priority ASC, created ASC` (the existing `"it orders by priority ASC then created ASC"` test already exercises this with only open tasks — confirm it stays green; optionally add an explicit no-regression assertion).
- ADD a test that `tick list --ready` floats `in_progress` identically to `tick ready` — use `runList(t, dir, "--ready", ...)` with the same mixed fixture as the resume-first test and assert the same top-of-list `in_progress` row, locking the `f.Ready` scope decision.
- KEEP `internal/cli/list_show_test.go` / list ordering tests for plain `tick list` and the `blocked_test.go` `"it orders by priority ASC then created ASC"` test unchanged — they must still assert the neutral `priority ASC, created ASC` ordering.

**Acceptance Criteria**:
- [ ] In `tick ready` (and `tick list --ready`), `in_progress` tasks sort above all `open` tasks; within each band, `priority ASC, created ASC` holds. (Spec AC #5)
- [ ] An `in_progress` task with a worse priority still sorts above a better-priority `open` task (proves the band term is distinct from the priority term). (Spec AC #5)
- [ ] `tick list` and `tick list --blocked` retain the existing `priority ASC, created ASC` ordering, unchanged. (Spec AC #6)
- [ ] A ready list with zero `in_progress` rows produces ordering byte-identical to the current `priority ASC, created ASC` (no regression). (Spec AC #6, basis for the no-regression criterion)
- [ ] `tick ready --count 1` returns the top unblocked in-flight task when one exists, otherwise the top unblocked open task (`LIMIT` applies after the resume-first `ORDER BY`). (Spec AC #7)
- [ ] `tick list --ready` floats `in_progress` identically to `tick ready` (the `f.Ready` scope decision is locked).
- [ ] The float persists under narrowing filters (`--parent`, `--tag`, `--type`, `--priority`, `--status`) — no special-case guard added.

**Tests**:
- `"it floats unblocked in_progress to the top of ready"` (new) — `in_progress` priority 3 sorts above `open` priority 0; within each band `priority ASC, created ASC` holds.
- `"it floats in_progress identically for list --ready"` (new, via `runList`) — `tick list --ready` produces the same in-progress-first ordering as `tick ready`.
- `"it returns the top unblocked in-flight task with --count 1"` (new) — `tick ready --count 1` returns the floated `in_progress` task; with zero in-progress, returns the top unblocked open task.
- `"it orders by priority ASC then created ASC"` (KEEP, ready_test.go with only open tasks) — zero-`in_progress` ordering is byte-identical (no regression).
- `"it orders by priority ASC then created ASC"` (KEEP, blocked_test.go) — `blocked` ordering unchanged (no float).
- Plain `tick list` ordering test (KEEP, list_show_test.go) — unchanged neutral ordering.

**Edge Cases**:
- **`in_progress` with worse priority.** The fixture must give the `in_progress` task a worse priority than an `open` task so the test proves the band term sorts it first, not the priority term passing incidentally. (Spec "Test Impact" explicitly calls this out.)
- **Zero `in_progress` rows.** `(t.status = 'in_progress') DESC` is uniformly false and a no-op; ordering must be byte-identical to the current `priority ASC, created ASC`. This is the basis for the no-regression criteria.
- **Float persists under narrowing filters.** Any query with `f.Ready` set floats `in_progress` regardless of `--parent`/`--tag`/`--type`/`--priority`/`--status`; no special-case guard for narrowed browses.
- **`--count 1` returns top *unblocked* in-flight.** A force-started blocked `in_progress` task is in `tick blocked`, not `ready` (Task 1-1), so it is never floated into `ready`; `--count 1` returns the top unblocked in-flight task, or the top unblocked open task if none.
- **`tick blocked` gains no float.** `f.Ready` is false for `--blocked`, so the band term is never emitted there.

**Context**:
> Sort ordering — resume-first, ready-view-only (spec): `in_progress` tasks float to the top of `ready` as a band; within the band — and within the `open` tasks beneath it — the existing `priority ASC, created ASC` ordering holds. This makes resumption the default: `--count 1` naturally returns in-flight work first.
>
> Scope (spec): the float applies whenever the ready filter `f.Ready` is active — set identically by `tick ready` (a literal alias dispatching as `--ready` into the same `RunList`) and `tick list --ready`. Both float. Plain `tick list` and `tick list --blocked` never set `f.Ready` and keep the neutral ordering. "Ready-only" means ready-view-only, keyed on `f.Ready`, with no special-case guard for narrowed browses.
>
> Implementation shape (spec, Affected Code Surface): `ORDER BY (t.status = 'in_progress') DESC, t.priority ASC, t.created ASC` when `f.Ready`, otherwise `t.priority ASC, t.created ASC`. When the result set contains zero `in_progress` rows the band term is uniformly false and a no-op, so ordering is byte-identical to today — the basis for the no-regression criteria (AC #2, #6).
>
> Precise promise (spec): resume-first applies only to *actionable* (unblocked) in-flight tasks. A blocked `in_progress` task is not in `ready` at all (it's in `blocked`), so `--count 1` returns the top unblocked in-flight task; force-starting a blocked task does not float it into `ready`.
>
> Within-band tiebreak (spec): `priority ASC, created ASC` — the existing clause. A "most-recently-started first" variant is deliberately NOT adopted (gold-plating, out of scope).

**Spec Reference**: `.workflows/ready-includes-in-progress/specification/ready-includes-in-progress/specification.md` — sections "Sort Ordering — Resume-First, Ready-View-Only" (all subsections), "Filters, --count, and Presentation" (`--count`), "Affected Code Surface" (list.go conditional ORDER BY), "Test Impact" (new ordering tests), Acceptance Criteria #5, #6, #7.

## ready-includes-in-progress-1-3 | approved

### Task 1-3: Correct the stats blocked-count derivation to the live set

**Problem**: `tick stats` computes `Blocked = Open − Ready`, which is only correct while `ready ⊆ open`. Once `ready` counts `in_progress` (Task 1-1), `Ready` can exceed `Open` and the blocked count goes wrong — even negative (e.g. 5 open / 3 ready + 4 in_progress / 3 ready → `Ready = 6`, `Open = 5`, `Blocked = −1`). This is a correctness consequence of the feature, not optional.

**Solution**: In `internal/cli/stats.go`, change the blocked derivation from `stats.Blocked = stats.Open - stats.Ready` to `stats.Blocked = (stats.Open + stats.InProgress) - stats.Ready`, following the partition invariant `Blocked = (Open + InProgress) − Ready`. This is the canonical arithmetic route — it reuses counts `stats` already gathers (`Open`, `InProgress`, `Ready`) and is exactly how blocked is derived today, just over the live set. The ready count is already correct: it uses `ReadyWhereClause()`, which picks up the widened gate from Task 1-1 automatically, so `in_progress` ready tasks are counted. Refresh the two stale inline comments.

**Outcome**: `tick stats` blocked count equals `(Open + InProgress) − Ready` and is never negative; the ready count includes unblocked `in_progress` tasks. An `in_progress` task counted in both `InProgress` and `Ready` is correct (two lenses: status breakdown vs actionability), exactly as an open-ready task is already counted in both `Open` and `Ready`.

**Do**:
- In `internal/cli/stats.go`, change `stats.Blocked = stats.Open - stats.Ready` to `stats.Blocked = (stats.Open + stats.InProgress) - stats.Ready`.
- Refresh the ready-count comment (currently `// Ready count: open, no unclosed blockers, no open children.`) to describe the new semantics, e.g. `// Ready count: open or in_progress, no unclosed blockers, no open/in-progress children, no blocked ancestor.` (it was already incomplete pre-feature, omitting the ancestor condition).
- Refresh the blocked-count comment (currently `// Blocked count: open AND NOT ready (derived from ready).`) to e.g. `// Blocked count: live (open or in_progress) AND NOT ready, derived as (Open + InProgress) − Ready.`
- Do NOT add a new blocked WHERE clause / `BlockedWhereClause()` helper — the arithmetic route is canonical; a direct count is explicitly NOT adopted (it would add net-new query-helper surface).
- Update `internal/cli/stats_test.go` subtest `"it counts ready and blocked tasks correctly"` (the `workflow["ready"]`/`workflow["blocked"]` assertion block and the fixture-setup comments at the top of the subtest): under the new semantics `tick-bbb111` (in_progress, no blockers, no children) becomes a READY leaf. Re-derive expected counts against the fixture: Ready = 3 (`tick-aaa111` open ready leaf, `tick-ccc222` open ready child leaf, `tick-bbb111` in_progress ready leaf); Blocked = 2 (`tick-aaa222` blocked by its in_progress blocker `tick-bbb111`, which is unclosed; `tick-ccc111` has open child) = `(Open 4 + InProgress 1) − Ready 3`. Change `workflow["ready"]` expected from `2` to `3`; `workflow["blocked"]` stays `2` but update its inline derivation comment. Correct the inline fixture comment for `tick-bbb111` (it is now a ready leaf, not "neither ready nor blocked"). Note: `tick-aaa222` stays blocked because its blocker `tick-bbb111` is in_progress (not done/cancelled), so widening the gate does not unblock it.
- KEEP the `stats_test.go` formatting tests (`"it formats stats in TOON format"`, `"it formats stats in Pretty format..."`, `"it formats stats in JSON format..."`) unchanged — they run with `InProgress=0` so the new semantics do not bite.

**Acceptance Criteria**:
- [ ] `tick stats` blocked count equals `(Open + InProgress) − Ready` and is never negative. (Spec AC #9)
- [ ] The stats ready count includes unblocked `in_progress` tasks (via `ReadyWhereClause()`, which inherits the Task 1-1 gate). (Spec AC #9)
- [ ] An `in_progress` task with no blockers and no children is counted in both `InProgress` and `Ready` (two lenses), and is NOT double-counted into `Blocked`.
- [ ] The `stats_test.go` `"it counts ready and blocked tasks correctly"` test passes with Ready = 3, Blocked = 2 against the existing fixture, exercising the `(Open + InProgress) − Ready` derivation.
- [ ] Both inline comments (ready-count and blocked-count) refreshed to describe the live-set semantics.
- [ ] Stats formatting tests (TOON/Pretty/JSON, `InProgress=0`) remain green.

**Tests**:
- `"it counts ready and blocked tasks correctly"` (updated, stats_test.go) — with the existing 6-task fixture, Ready = 3 and Blocked = 2; `tick-bbb111` (in_progress, no blockers, no children) is now a ready leaf.
- `"it derives a non-negative blocked count when ready exceeds open"` (new) — a fixture with several unblocked `in_progress` ready tasks so `Ready > Open`; assert `Blocked == (Open + InProgress) − Ready` and `Blocked >= 0` (regression-proofs the old `Open − Ready` which would go negative).
- `"it maintains stats count consistency with blocked ancestors"` (KEEP/verify, blocked_test.go) — this test asserts `statsReady + statsBlocked == open` for an all-open fixture; with `InProgress = 0` the new derivation reduces to `Open − Ready`, so it stays green. Confirm it still passes (no in_progress tasks in its fixture).
- Stats formatting tests (KEEP) — `InProgress=0`, unaffected.

**Edge Cases**:
- **Blocked count never negative.** The partition invariant `ready ⊎ blocked = live` guarantees `(Open + InProgress) − Ready >= 0`. The new test with `Ready > Open` proves the old `Open − Ready` formula would have gone negative and the new one does not.
- **`in_progress` counted in both `InProgress` and `Ready`.** This is correct and intended — two lenses (status breakdown vs actionability), exactly as an open-ready task is already counted in both `Open` and `Ready`. It must NOT inflate `Blocked` (the arithmetic subtracts the full `Ready`, including the in_progress portion, from the full live set).
- **`InProgress = 0` reduces to the old formula.** When there are no in-progress tasks, `(Open + InProgress) − Ready == Open − Ready`, so all pre-existing stats tests and the consistency check in `blocked_test.go` stay green.

**Context**:
> Blocked count derivation (spec, required fix): `stats` currently computes `Blocked = Open − Ready`, correct only while `ready ⊆ open`. Once `ready` counts `in_progress`, `Ready` can exceed `Open` and the blocked count goes wrong (e.g. 5 open / 3 ready + 4 in_progress / 3 ready → `Ready = 6`, `Open = 5`, `Blocked = −1`). The derivation must move to the live set: `Blocked = (Open + InProgress) − Ready`. This fix is NOT optional — it is a correctness consequence of the feature.
>
> Arithmetic route is canonical (spec): it reuses counts `stats` already gathers and is exactly how blocked is derived today (`Open − Ready`); the partition invariant guarantees its correctness. A direct count via a new blocked WHERE clause is NOT adopted — it would add net-new query-helper surface (`query_helpers.go` exposes `ReadyWhereClause()` but no blocked counterpart) and is only more robust if the shared-gate invariant were ever broken.
>
> Ready count tracks the new semantics (spec, intended): the stats ready count uses `ReadyWhereClause`, so it includes `in_progress` automatically — correct, because the stats "ready" number must mean the same as `tick ready`. An `in_progress` task counted in both `InProgress` and `Ready` is fine — two lenses, exactly as an open-ready task is already counted in both `Open` and `Ready`.
>
> Fixture re-derivation (spec, Test Impact): in `stats_test.go` `"it counts ready and blocked tasks correctly"`, `tick-bbb111` (in_progress, no blockers, no children) becomes a ready leaf, so Ready = 3 (adds bbb111) and Blocked = 2 (aaa222, ccc111) — now derived as `(Open 4 + InProgress 1) − Ready 3 = 2`. The inline fixture comment for `tick-bbb111` must be corrected (now a ready leaf, not "neither").

**Spec Reference**: `.workflows/ready-includes-in-progress/specification/ready-includes-in-progress/specification.md` — sections "Stats Counts" (Blocked count derivation, Ready count tracks the new semantics), "Affected Code Surface" (stats.go blocked derivation + comments), "Test Impact" (stats_test.go), Acceptance Criteria #9.
