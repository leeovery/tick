---
status: complete
created: 2026-06-03
cycle: 2
phase: Traceability Review
topic: Ready Includes In-Progress
---

# Review Tracking: Ready Includes In-Progress - Traceability

## Result: CLEAN

No findings. The plan remains a faithful, complete translation of the specification in
both directions after the cycle-1 fixes. Every specification requirement, decision, edge
case, code-surface change, test-inventory item, and acceptance criterion is covered by at
least one task, and every piece of task content traces back to a specific part of the
specification — no invented scope. The two cycle-1 edits introduced **no traceability
drift**.

## Findings

(none)

---

## Cycle-1 Fix Verification

Both cycle-1 edits were re-checked for traceability drift. Neither added scope; both are
spec-grounded.

### Edit 1 — Task 1-1 Tests entry for the `list_filter_test.go` stale-comment refresh

`phase-1-tasks.md` Task 1-1 Tests section now lists
`"contradictory filters return empty result no error"` (KEEP, list_filter_test.go),
matching the Do step that refreshes the stale `// ... (ready only applies to open tasks)`
comment. Traces to spec **Test Impact → "Tests that stay valid — KEEP"** (`list_filter_test.go`
`--status` filter tests) and to the **Filters/--count/Presentation** rule that
`--status done` composes to an empty intersection (`status IN (open,in_progress) AND
status = done` always false), plus **AC #8**. Verified against live source:
`list_filter_test.go:385` still carries the stale comment and `:393` runs
`runList(..., "--status", "done", "--ready")` asserting an empty result. The entry
describes a KEEP test plus a comment refresh — no new behaviour, no invented scope.
**No drift.**

### Edit 2 — Task 1-3 stats_test anchor + `tick-aaa222`-stays-blocked rationale

`phase-1-tasks.md` Task 1-3 Do step (and the tick task tick-6e6a9c) now anchor on
"the `workflow["ready"]`/`workflow["blocked"]` assertion block and the fixture-setup
comments at the top of the subtest" instead of the imprecise `(line ~74)`, and make the
`tick-aaa222`-stays-blocked rationale explicit ("its blocker `tick-bbb111` is in_progress
(not done/cancelled), so widening the gate does not unblock it"). Both trace to spec
**Test Impact → stats_test.go** (the `tick-bbb111`-becomes-ready-leaf re-derivation,
Ready = 3 / Blocked = 2) and the **Governing invariant** (partition over the live set).
Verified arithmetically against the live fixture (`stats_test.go:74`): Open 4 (aaa111,
aaa222, ccc111, ccc222) + InProgress 1 (bbb111) − Ready 3 (aaa111, ccc222, bbb111) =
Blocked 2 (aaa222, ccc111). `tick-aaa222` is `BlockedBy: ["tick-bbb111"]` and bbb111 is
`StatusInProgress` (unclosed), so the gate widening leaves aaa222 blocked — exactly as the
added rationale states. The anchor change is a pointer-precision improvement (no spec
content); the rationale is a faithful derivation of the spec-stated Blocked = 2, not a new
requirement. **No drift.**

---

## Analysis Record

### Direction 1: Specification → Plan (completeness)

Every spec element is represented in the plan:

| Spec element | Plan coverage |
|--------------|---------------|
| Overview / Problem / Goal / Actor model / Governing invariant | Task 1-1 Problem + Context |
| Ready definition (conditions 1–4; only gate widens) | Task 1-1 Do + Context |
| Blocked definition (De Morgan inverse over shared gate) | Task 1-1 Solution + Context |
| Consequences: force-started blocked; leaf gate symmetric; done/cancelled in neither | Task 1-1 Edge Cases |
| Sort ordering — resume-first, ready-view-only (all subsections) | Task 1-2 |
| Scope keyed on `f.Ready` (not literal command); `ready` and `list --ready` float | Task 1-2 Solution + Context |
| Float persists under narrowing filters (no special-case guard) | Task 1-2 Do + Edge Cases |
| Precise promise: top *unblocked* in-flight | Task 1-2 Edge Cases + Context |
| Within-band tiebreak `priority ASC, created ASC`; "most-recently-started" rejected | Task 1-2 Context |
| Stats — blocked derivation `(Open + InProgress) − Ready` (required fix) | Task 1-3 |
| Stats — ready count tracks new semantics; two comment refreshes (verbatim strings) | Task 1-3 Do |
| `--status` composes (open→unstarted, in_progress→resumptions, terminal→empty) | Task 1-1 Do + AC + Edge Cases |
| `--count` (LIMIT after resume-first ORDER BY) | Task 1-2 Do + AC + Edge Cases |
| Presentation — no change; Quiet mode and empty results (pre-existing paths) | Correctly omitted (no code change; spec states "no new work") |
| Affected code surface: query_helpers.go / list.go / stats.go | Tasks 1-1 / 1-2 / 1-3 |
| "No changes required" (state machine, flags, formatters, cache) | Honoured — no tasks invent work there |
| Test Impact — MUST change (4 files) | query_helpers_test.go, ready_test.go, blocked_test.go → 1-1; stats_test.go → 1-3 |
| Test Impact — KEEP no change | Named as KEEP in 1-1/1-2/1-3 Tests fields (incl. list_filter_test.go via cycle-1 edit) |
| Test Impact — New tests to ADD (5) | All 5 mapped across 1-1/1-2/1-3 |
| Acceptance Criteria #1–#10 | All 10 tagged in task ACs |
| Out of Scope (multi-actor, pretty cue, most-recently-started) | Correctly excluded; flagged as out-of-scope in Context |
| Accepted consequence (resumption-heavy ready) | Informational only; correctly omitted |

**Acceptance Criteria mapping** — all 10 traced:
- AC #1 → Task 1-1 AC + Tests
- AC #2 → Task 1-1 AC + Task 1-2 AC (no-regression)
- AC #3 → Task 1-1 AC + Tests
- AC #4 → Task 1-1 AC + Tests (partition)
- AC #5 → Task 1-2 AC
- AC #6 → Task 1-2 AC
- AC #7 → Task 1-2 AC
- AC #8 → Task 1-1 AC
- AC #9 → Task 1-3 AC
- AC #10 → Task 1-1 AC + KEPT `"it excludes parent with in_progress children"`

**New tests to ADD mapping** — all 5 traced:
1. Resume-first ordering (in_progress worse priority) → Task 1-2 `"it floats unblocked in_progress to the top of ready"`
2. Unblocked in_progress leaf in ready; blocked in_progress in blocked → Task 1-1 `"it partitions an in_progress task into exactly one of ready/blocked"`
3. Stats counts with in_progress present → Task 1-3 `"it derives a non-negative blocked count when ready exceeds open"`
4. `ready --status open` / `--status in_progress` composition → Task 1-1 `"it returns only unstarted work for ready --status open"` / `"it returns only resumptions for ready --status in_progress"`
5. `list --ready` floats identically to `ready` → Task 1-2 `"it floats in_progress identically for list --ready"`

### Direction 2: Plan → Specification (fidelity / anti-hallucination)

Every task's content traces back to the specification; load-bearing structural claims were
re-verified against live source:

- **Task 1-1** — gate flip in both `ReadyConditions()`/`BlockedConditions()` (verified
  `query_helpers.go:53,77` still hold the `t.status = 'open'` literal — the correct
  pre-feature target), untouched inverse machinery, `--status` composition, partition,
  force-started / start-cascade / done-cancelled edge cases, and the `list_filter_test.go`
  comment refresh all map to spec "Ready & Blocked Definitions", "Consequences",
  "Filters/--count/Presentation", "Affected Code Surface (query_helpers.go)", and
  "Test Impact". The three `query_helpers_test.go` literal sites named for update were
  re-verified to exist (`:41`, `:60`, `:115`) — the spec text names two, but the literal
  appears at three sites and all three must change for the spec-mandated flip to pass;
  faithful completion, not invented scope. Structural assertions ("ReadyConditions() = 4
  elements; BlockedConditions() = 2") trace to the spec's "inverse machinery untouched"
  guarantee.
- **Task 1-2** — conditional `ORDER BY` keyed on `f.Ready`, `handleReady`→`RunList`
  dispatch, worse-priority discriminating fixture, zero-in_progress no-op, `list --ready`
  parity, `--count 1`, narrowing-filter persistence, and "blocked gains no float" all map
  to spec "Sort Ordering — Resume-First, Ready-View-Only", "Filters/--count/Presentation",
  and "Affected Code Surface (list.go)".
- **Task 1-3** — blocked derivation, both comment-refresh strings (verbatim with spec,
  verified `stats.go:78,84` carry the pre-feature comments and `:85` the `Open - Ready`
  derivation), the explicit rejection of a `BlockedWhereClause()` helper, the fixture
  re-derivation (Ready 3 / Blocked 2, `tick-bbb111` becomes a ready leaf,
  `tick-aaa222` stays blocked), and the non-negative test all map to spec "Stats Counts"
  and "Test Impact". The fixture re-derivation was re-verified arithmetically against the
  live `stats_test.go` fixture.

No plan content lacks a specification anchor. No new scope introduced by the cycle-1 fixes.

### Note (out of traceability scope — not a finding)

Tasks 1-1 (tick-fe7e70) and 1-2 (tick-2f0d2a) fold their test instructions into the Do
section ("ADD…/Rewrite…/KEEP…") rather than carrying a standalone "Tests:" section in
the tick record; Task 1-3 (tick-6e6a9c) and the full `phase-1-tasks.md` for all three
carry explicit Tests sections. This is a task-template-completeness matter (the integrity
gate's domain, passed in cycle 1), not a traceability matter — the test *content* is
present and every test traces to the spec's Test Impact inventory. Recorded for visibility
only; no traceability action.
