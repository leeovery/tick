---
status: complete
created: 2026-06-03
cycle: 3
phase: Plan Integrity Review
topic: Ready Includes In-Progress
---

# Review Tracking: Ready Includes In-Progress - Integrity

## Summary

Cycle-3 re-review of the single-phase, three-task plan after the cycle-2 fix was
applied. **No new findings — the plan is clean and implementation-ready.**

### Cycle-2 fix verification (sound and complete)

The cycle-2 finding was that the required **Tests** field was absent from the
tick descriptions of Task 1-1 (tick-fe7e70) and Task 1-2 (tick-2f0d2a), leaving
them below template and inconsistent with Task 1-3 (tick-6e6a9c).

Verified via `tick show` against the canonical store:

- **Task 1-1 (tick-fe7e70)** now carries a full `Tests:` section between
  `Edge Cases:` and `Context:`, with all 10 entries — including the cycle-1-added
  `"contradictory filters return empty result no error"` (list_filter_test.go)
  entry — rendered verbatim from `phase-1-tasks.md` lines 38-47.
- **Task 1-2 (tick-2f0d2a)** now carries a full `Tests:` section in the same
  position, with all 6 entries matching `phase-1-tasks.md` lines 94-99.
- **Task 1-3 (tick-6e6a9c)** already carried its `Tests:` section (from cycle-1
  fix #2); it is unchanged and correct.

All three tick descriptions are now **template-complete** — each runs
Problem → Solution → Outcome → Do → Acceptance Criteria → Tests → Edge Cases →
Context → Spec Reference, satisfying every required field in `task-design.md`.

**Mutual consistency** confirmed: the rewrite/KEEP/new distinctions in the Tests
sections agree across tasks and with the detail file. E.g. Task 1-1's
`"it includes unblocked in_progress leaf"` (rewritten from
`"it excludes in_progress tasks"`) and `"it includes blocked in_progress task"`
(rewritten, option b) are stated identically in the tick store and
`phase-1-tasks.md`; Task 1-2's `"it orders by priority ASC then created ASC"`
KEEP entries (ready_test.go + blocked_test.go) and Task 1-3's
`"it maintains stats count consistency with blocked ancestors"` KEEP entry all
reference real, located subtests. The tick store and `phase-1-tasks.md` are now
fully in sync on every task's Tests section.

### Full integrity re-pass (all criteria)

- **Task Template Compliance**: PASS — all three tasks carry all required fields
  in both the tick store and the detail file; every field is substantive. The
  cycle-2 gap is closed.
- **Vertical Slicing**: PASS — each task is a complete, independently verifiable
  behaviour increment on its own seam (query_helpers.go status gate, list.go
  conditional ORDER BY, stats.go blocked derivation); none is a horizontal layer.
- **Phase Structure**: PASS — the single-phase decision is explicitly justified
  by the partition invariant `ready ⊎ blocked = all live tasks`; splitting would
  leave a broken negative-blocked-count intermediate. Ten phase acceptance
  criteria map to the spec AC.
- **Dependencies and Ordering**: PASS — authored in dependency order (1-1 widens
  the gate → 1-2 floats in_progress → 1-3 fixes the count); `tick list --parent
  <phase>` returns them in exactly that creation order. Natural ordering produces
  the correct sequence; no convergence point and no cross-phase edge, so explicit
  `blocked_by` edges are correctly not required. No circular dependencies.
- **Task Self-Containment**: PASS — each task's Context block pulls the governing
  spec decisions forward; an implementer needs nothing beyond the task itself.
- **Scope and Granularity**: PASS — each task is one TDD cycle (a one-line-per-side
  SQL change plus its test impact); none is boilerplate or oversized.
- **Acceptance Criteria Quality**: PASS — criteria are concrete and pass/fail,
  each tagged to a spec AC number; fixture re-derivations are spelled out
  numerically (Ready 3 / Blocked 2 = (Open 4 + InProgress 1) − Ready 3).
- **External Dependencies**: N/A (feature, not epic).

### Load-bearing code claims re-verified against live source

- `query_helpers.go`: `ReadyConditions()` returns 4 elements with
  `` `t.status = 'open'` `` first (line 53); `BlockedConditions()` returns 2 with
  `` `t.status = 'open'` `` (line 77) and the 3-EXISTS De Morgan inverse
  (negateNotExists of the three ReadyNo*() helpers, lines 72-74). The
  one-line-per-side flip described by Task 1-1 is accurate.
- `list.go`: `buildListQuery` emits `ORDER BY t.priority ASC, t.created ASC`
  (line 311) before the `LIMIT ?` append (lines 313-315), inside the `f.Ready`
  context (line 266); `--status` composes as `t.status = ?` AND-ed (line 275).
  Task 1-2's conditional-ORDER-BY plan is accurate.
- `stats.go`: derives `stats.Blocked = stats.Open - stats.Ready` (line 85) with
  the two stale comments at lines 78 ("Ready count: open, no unclosed blockers,
  no open children.") and 84 ("Blocked count: open AND NOT ready..."); ready
  count uses `ReadyWhereClause()` (line 79). Task 1-3's derivation change and
  comment refreshes are accurate.
- Cited test references all resolve: `list_filter_test.go:384-385` (stale
  comment + "contradictory filters" subtest), `stats_test.go:74` ("it counts
  ready and blocked tasks correctly") with the `tick-bbb111` fixture comment at
  line 78 and the `workflow["ready"]`/`workflow["blocked"]` assertions at lines
  104-109 (currently expecting 2/2 — exactly the lines Task 1-3 re-derives to
  3/2), `blocked_test.go:208` and `ready_test.go:304` ("it orders by priority
  ASC then created ASC"), `blocked_test.go:400` ("it maintains stats count
  consistency with blocked ancestors"), `list_show_test.go:143` (plain-list
  ordering assertion).

## Findings

None. The plan meets structural-quality and implementation-readiness standards.
The cycle-2 fix is verified sound and complete; all three tick descriptions are
template-complete and mutually consistent; no remaining integrity issues.
