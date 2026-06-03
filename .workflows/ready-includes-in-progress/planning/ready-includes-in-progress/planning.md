# Plan: Ready Includes In-Progress

## Phases

### Phase 1: Ready Includes In-Progress
status: approved
approved_at: 2026-06-03

**Goal**: Widen the `ready`/`blocked` partition to treat `in_progress` as live work, float unblocked in-progress tasks to the top of the ready view, and correct the `stats` blocked-count derivation — delivering the full "resume interrupted work" behaviour as one verifiable increment.

**Why this order**: This is a single, surgically-scoped feature with no meaningful intermediate checkpoint. The three code changes — the shared status-gate literal in `query_helpers.go` (`ReadyConditions`/`BlockedConditions`), the conditional `ORDER BY` keyed on `f.Ready` in `list.go` (`buildListQuery`), and the blocked-count derivation in `stats.go` — are bound together by a single partition invariant (`ready ⊎ blocked = all live tasks`). Splitting them would leave a broken intermediate state: flipping the gate without the stats fix produces a negative blocked count (spec example: Ready 6 / Open 5 → Blocked −1). The SQL diff is tiny; the bulk of the work is updating tests asserting the old semantics and adding tests for the new partition, ordering, and stats behaviour. No transition logic, flag registry, formatter, or cache-schema changes are required.

**Acceptance**:
- [ ] An unblocked `in_progress` leaf task appears in `tick ready`.
- [ ] An unblocked `open` leaf task still appears in `tick ready` (no regression).
- [ ] A `blocked` `in_progress` task (unclosed blocker, or blocked ancestor) appears in `tick blocked` and never in `tick ready`.
- [ ] Every live task (`open` or `in_progress`) appears in exactly one of `tick ready` / `tick blocked`; `done`/`cancelled` appear in neither.
- [ ] In `tick ready` (and `tick list --ready`), `in_progress` tasks sort above all `open` tasks; within each band, `priority ASC, created ASC` holds.
- [ ] `tick list` and `tick list --blocked` retain the existing `priority ASC, created ASC` ordering, unchanged.
- [ ] `tick ready --count 1` returns the top unblocked in-flight task when one exists, otherwise the top unblocked open task.
- [ ] `tick ready --status open` returns unstarted ready work; `tick ready --status in_progress` returns resumptions only; `--status done`/`--status cancelled` return empty.
- [ ] `tick stats` blocked count equals `(Open + InProgress) − Ready` and is never negative; the ready count includes unblocked `in_progress` tasks.
- [ ] An `in_progress` parent that exists only via the start-cascade does not appear in `tick ready`; only its leaf does.

#### Tasks
status: draft

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| ready-includes-in-progress-1-1 | Widen the shared ready/blocked status gate to live tasks | force-started blocked in-progress appears only in blocked never ready; in_progress parent via start-cascade excluded from ready by leaf gate (only leaf surfaces); done/cancelled in neither; --status open → unstarted-only, --status in_progress → resumptions-only, --status done/cancelled → empty |
| ready-includes-in-progress-1-2 | Float unblocked in-progress to the top of the ready view | in_progress with worse priority still sorts above better-priority open; zero in-progress rows → byte-identical ordering (no regression); float persists under narrowing filters; --count 1 returns top unblocked in-flight; plain list and list --blocked ordering unchanged |
| ready-includes-in-progress-1-3 | Correct the stats blocked-count derivation to the live set | blocked count never negative; in_progress counted in both InProgress and Ready is correct; ready count includes unblocked in_progress via ReadyWhereClause |
