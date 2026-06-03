# Specification: Ready Includes In-Progress

## Specification

## Overview

### Problem

`tick ready` answers "what should I act on right now?" Today both `ready` and `blocked` gate on `status = 'open'`. An `in_progress` task is therefore in *neither* view — invisible to the very command used to resume interrupted work. Start a task, get interrupted, run `tick ready`, and it points you at *new* open work instead of the started-but-dangling task. The same `status = 'open'` gate also drops a started-but-stuck task out of `blocked`. The real defect is **double invisibility**: in-progress work falls off both actionability views.

### Goal

`ready` means "everything actionable right now" = unblocked `open` **+** `in_progress`. A started task is the most ready thing there is; resuming interrupted work becomes the natural default. `blocked` stays the strict complement so no live task ever falls through the cracks.

### Actor model (decided premise)

tick is single-actor with no ownership concept — a task carries no assignee. `in_progress` means "the task *I* started and got pulled off," not "someone else claimed it." Surfacing it in `ready` means "resume this." The "someone else took it, hands off" reading would require an assignee field tick doesn't have; excluding all `in_progress` to dodge collisions can't tell *my* work from *theirs*, so it would break the single-actor resumption case. Multi-actor claiming is a separate future feature (see Out of Scope).

### Governing invariant

`ready ⊎ blocked = all live tasks`, where live = `status IN ('open','in_progress')`. Every live task is in **exactly one** of `ready`/`blocked` — never both, never neither. `done` and `cancelled` are in neither. `ready` and `blocked` share one identical status gate; `blocked` is the literal De Morgan inverse of `ready`'s `NOT EXISTS` conditions. This invariant is what makes the change small (flip one shared literal per side) and guarantees nothing is dropped.

---

## Ready & Blocked Definitions

### Ready

A task is **ready** when **all** of the following hold:

1. `status IN ('open', 'in_progress')` — the changed gate (was `status = 'open'`).
2. No unclosed blocker — no incomplete task in its blocked-by dependencies.
3. No open or in-progress child — the **leaf gate**: any child with `status IN ('open','in_progress')` disqualifies it.
4. No dependency-blocked ancestor — walking the parent chain, no ancestor is itself blocked by an unclosed dependency.

Conditions 2–4 are the existing `NOT EXISTS` conditions, unchanged. Only the status gate widens.

### Blocked

A task is **blocked** when it is live (`status IN ('open','in_progress')`) **and** at least one of the ready `NOT EXISTS` conditions fails:

- has an unclosed blocker, **OR**
- has an open or in-progress child, **OR**
- has a dependency-blocked ancestor.

`blocked` is the literal De Morgan inverse of `ready` over the same status gate. The inverse/negation machinery is untouched — only the shared status literal changes on each side.

### Consequences that fall out of the definitions

- **Force-started blocked task.** Starting is not gated by blockers (the `start` transition constrains only `from: open`; there is no blocker check in the state machine). So an `open` task with unclosed blockers can be force-started into `in_progress`, becoming blocked-and-in-progress. It shows in `tick blocked` only, **never** `tick ready`. Starting a task does not clear its block — a task that was blocked stays blocked.
- **Leaf gate is symmetric under in-progress.** An `in_progress` parent that exists only because the start-cascade drove it up the ancestor chain does **not** surface in `ready` — it has an `in_progress` child, so the leaf gate excludes it; it surfaces in `blocked` instead. Only the leaf surfaces. Parent and child **never co-occur in `ready`**, regardless of whether rows are `open` or `in_progress`.
- **`done`/`cancelled`** remain in neither view.

---

## Sort Ordering — Resume-First, Ready-View-Only

`in_progress` tasks float to the top of `ready` as a band; within the band — and within the `open` tasks beneath it — the existing `priority ASC, created ASC` ordering holds. This makes resumption the default: `--count 1` naturally returns in-flight work first.

### Scope: the ready *filter* (`f.Ready`), not a literal command

The float applies whenever the ready filter is active — set identically by `tick ready` (a literal alias dispatching as `--ready` into the same `RunList`) and `tick list --ready`. Both are the same ready view, so both float. Plain `tick list` and `tick list --blocked` never set the ready filter and keep the current neutral `priority ASC, created ASC` ordering, unchanged. "Ready-only" means *ready-view-only*, keyed on `f.Ready`.

Rationale: resume-first is a property of `ready`'s "what now?" intent. `tick list` is a neutral browse view where silently floating started work would be surprising; `tick blocked` gains nothing from it.

### Precise promise: "top **unblocked** in-flight work"

Resume-first applies only to *actionable* (unblocked) in-flight tasks. An `in_progress` task that is itself blocked — by a direct blocker or by a blocked ancestor — is not in `ready` at all; by the partition invariant it lives in `tick blocked`. So `--count 1` returns the top **unblocked** in-flight task; force-starting a blocked task does not float it into `ready`.

### Within-band tiebreak

`priority ASC, created ASC` — the existing clause, applied within the in-progress band and within the open tasks beneath it. A "most-recently-started first" variant via transition history is deliberately not adopted (gold-plating — see Out of Scope).

### Accepted consequence

A user with several `in_progress` tasks sees a resumption-heavy `ready`. This is treated as desirable (nudges finishing work-in-progress before pulling new work), not a defect.

---

## Stats Counts

### Blocked count derivation (required fix)

`stats` currently computes `Blocked = Open − Ready`. This is only correct while `ready ⊆ open`. Once `ready` counts `in_progress`, `Ready` can exceed `Open` and the blocked count goes wrong (e.g. 5 open / 3 ready + 4 in_progress / 3 ready → `Ready = 6`, `Open = 5`, `Blocked = −1`).

The derivation must move to the live set, following the partition invariant:

```
Blocked = (Open + InProgress) − Ready
```

The **arithmetic route is canonical** — it reuses counts `stats` already gathers and is exactly how blocked is derived today (`Open − Ready`); the partition invariant guarantees its correctness. A direct count via a new blocked WHERE clause is *not* adopted: it would add net-new query-helper surface (`query_helpers.go` exposes `ReadyWhereClause()` but no blocked counterpart) and is only more robust if the shared-gate invariant were ever broken. This fix is **not optional** — it's a correctness consequence of the feature.

### Ready count tracks the new semantics (intended)

The stats ready count uses `ReadyWhereClause`, so it includes `in_progress` automatically — correct, because the stats "ready" number must mean the same as `tick ready`. An `in_progress` task counted in both `InProgress` and `Ready` is fine — two lenses (status breakdown vs actionability), exactly as an open-ready task is already counted in both `Open` and `Ready`. The stale comment at `stats.go:78` should be refreshed to reflect the new semantics.

---

## Filters, `--count`, and Presentation

### `--status` composes cleanly (no new work)

`--status` is already a valid `ready` flag (inherited from `list`). It intersects with the widened gate:

- `tick ready --status open` → `status IN (open,in_progress) AND status = open` → **unstarted ready work**. This is the canonical "I only want new work" query and supersedes any earlier `tick list --status open` suggestion (which wouldn't apply the blocker/leaf/ancestor filtering).
- `tick ready --status in_progress` → **resumptions only**.

**Terminal statuses compose to empty (accepted).** `tick ready --status done` and `--status cancelled` become `status IN (open,in_progress) AND status = <terminal>` — always false, returning a silent empty list. Accepted as-is: consistent with filter semantics everywhere (an empty intersection returns empty; tick doesn't reject contradictory filter combinations). No special validation.

### `--count`

No special handling. `LIMIT` applies after the resume-first `ORDER BY`, so `--count 1` returns the top **unblocked** in-flight task (blocked-but-started work is in `tick blocked`, not `ready`).

### Presentation — no change

In-progress tasks are not visually distinguished beyond what already exists. The two signals that answer "which are resumptions?" — the `status` value and the top-of-list position — are already present in every format:

- For an agent reading toon/JSON (the primary consumers), it's fully machine-distinguishable with zero change, keyed off the `status` field.
- For a human on `pretty`, the existing Status column already reads `in_progress`.

Sectioning ("In progress" / "Ready to start" headers) is explicitly **rejected** — it's noise and a parsing hazard for the machine formats. A pretty-only visual cue is a possible trivial future polish, out of scope now (see Out of Scope).

---

## Affected Code Surface

These are implementation shapes captured from the discussion — concrete enough to plan against, not prescriptive line-by-line.

### `internal/cli/query_helpers.go` — the status gate

The shared `t.status = 'open'` literal in **both** `ReadyConditions()` and `BlockedConditions()` becomes `t.status IN ('open','in_progress')`. The `negateNotExists` / inverse machinery that derives `BlockedConditions()` from the ready conditions is **untouched** — the symmetry is exactly why this is a one-line-per-side change. `ReadyWhereClause()` (consumed by `stats`) picks up the new gate automatically.

### `internal/cli/list.go` — conditional `ORDER BY` in `buildListQuery`

The `ORDER BY` becomes conditional on the ready filter. When `f.Ready`, prepend a status-priority term — e.g. `ORDER BY (t.status = 'in_progress') DESC, t.priority ASC, t.created ASC`; otherwise the current `t.priority ASC, t.created ASC` clause is unchanged. Keyed on `f.Ready`, so it applies to both `tick ready` and `tick list --ready` and to neither `tick list` nor `tick list --blocked`.

### `internal/cli/stats.go` — blocked derivation + comment

Change the blocked count from `Open − Ready` to `(Open + InProgress) − Ready`. Refresh the stale comment at `stats.go:78` to describe the new ready/blocked semantics.

### No changes required

- State machine — confirmed: `start` constrains only `from: open`, no blocker guard; blockers stay a query-time concept. No transition logic changes.
- Flag registry / `commandFlags` — no new flags added; `--status` and `--count` already valid on `ready`.
- Formatters — no presentation change in any format.
- Cache schema — no schema change; queries only.

---

## Test Impact

The SQL diff is small; the **test-update surface is the larger part of the work** and is sized here rather than discovered at implementation. Inventory verified against the test files.

### Tests asserting the OLD semantics — MUST change

- **`query_helpers_test.go`** — `"ReadyConditions returns status open plus all four conditions"` and `"BlockedConditions contains no SQL literals beyond status check"` both assert `conditions[0] == "t.status = 'open'"`. The gate becomes `t.status IN ('open','in_progress')`; both assertions update.
- **`ready_test.go`** — `"it excludes in_progress tasks"` (line ~204) **inverts**: an unblocked `in_progress` leaf must now *appear* in `ready`. Rewrite, don't delete.
- **`blocked_test.go`** — `"it excludes in_progress tasks from output"` (line ~126, rationale "only open") is now misleading: a lone *unblocked* `in_progress` task is still absent from `blocked`, but because it's *ready*, not because `in_progress` is excluded. Update the rationale; the assertion as written may still pass for the wrong reason.
- **`stats_test.go`** — `"it counts ready and blocked tasks correctly"` (line ~74) encodes `in_progress => neither ready nor blocked (not open)` with Ready=2/Blocked=2. Under the new semantics the unblocked `in_progress` task becomes ready, so expected counts change; this test exercises the `Blocked = (Open + InProgress) − Ready` derivation.

### Tests that stay valid — KEEP, no change

- **`ready_test.go`** — `"excludes task with in_progress blocker"`, `"excludes parent with in_progress children"` (leaf/blocker rules unchanged).
- **`blocked_test.go`** — blocked-by-open/in_progress dep, parent with open/in_progress children, blocked-ancestor cases.
- **`list_filter_test.go`** — `--status open/in_progress/done/cancelled` filter tests; `commandFlags` drift test (no new flags added).
- **`stats_test.go`** formatting tests (run with `InProgress=0`, so semantics don't bite).

### New tests to ADD

- Resume-first ordering on `ready` with mixed `in_progress`/`open` (float above open regardless of priority; within band `priority, created`).
- An unblocked `in_progress` leaf appears in `ready`; a *blocked* `in_progress` task appears in `blocked`.
- `stats` counts with `in_progress` ready/blocked tasks present (exercises the new derivation).
- `tick ready --status open` (unstarted ready) and `--status in_progress` (resumptions only) composition.
- `tick list --ready` floats `in_progress` identically to `tick ready` (locks the `f.Ready` scope decision).

---

## Acceptance Criteria

1. An unblocked `in_progress` leaf task appears in `tick ready`.
2. An unblocked `open` leaf task still appears in `tick ready` (no regression).
3. A `blocked` `in_progress` task (unclosed blocker, or blocked ancestor) appears in `tick blocked` and **never** in `tick ready`.
4. Every live task (`open` or `in_progress`) appears in exactly one of `tick ready` / `tick blocked`; `done`/`cancelled` appear in neither.
5. In `tick ready` (and `tick list --ready`), `in_progress` tasks sort above all `open` tasks; within each band, `priority ASC, created ASC` holds.
6. `tick list` and `tick list --blocked` retain the existing `priority ASC, created ASC` ordering, unchanged.
7. `tick ready --count 1` returns the top unblocked in-flight task when one exists, otherwise the top unblocked open task.
8. `tick ready --status open` returns unstarted ready work; `tick ready --status in_progress` returns resumptions only; `--status done`/`--status cancelled` return empty.
9. `tick stats` blocked count equals `(Open + InProgress) − Ready` and is never negative; the ready count includes unblocked `in_progress` tasks.
10. An `in_progress` parent that exists only via the start-cascade does not appear in `tick ready`; only its leaf does.

## Out of Scope (Future Work)

- **Multi-actor / assignee model.** Add an assignee field and make `ready` exclude tasks assigned to *others* — the precise, collision-safe rule. Revisit only when multi-actor claiming is pursued; do **not** approximate it now by excluding all `in_progress`.
- **Pretty-only visual cue** for `in_progress` rows. Trivial future polish if a human-ergonomics need appears; the machine formats must stay untouched.
- **"Most-recently-started first"** ordering within the `in_progress` band via transition history. A possible refinement, deliberately not adopted (gold-plating).

---

## Working Notes
