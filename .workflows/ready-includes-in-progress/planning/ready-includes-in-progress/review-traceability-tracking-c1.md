---
status: complete
created: 2026-06-03
cycle: 1
phase: Traceability Review
topic: Ready Includes In-Progress
---

# Review Tracking: Ready Includes In-Progress - Traceability

## Result: CLEAN

No findings. The plan is a faithful, complete translation of the specification in both
directions. Every specification requirement, decision, edge case, code-surface change, test
inventory item, and acceptance criterion is covered by at least one task, and every piece of
task content traces back to a specific part of the specification — no invented scope.

## Findings

(none)

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
| Sort ordering — resume-first, ready-view-only | Task 1-2 (all subsections) |
| Scope keyed on `f.Ready` (not literal command); both `ready` and `list --ready` float | Task 1-2 Solution + Context |
| Float persists under narrowing filters (no special-case guard) | Task 1-2 Do + Edge Cases |
| Precise promise: top *unblocked* in-flight | Task 1-2 Edge Cases + Context |
| Within-band tiebreak `priority ASC, created ASC`; "most-recently-started" rejected | Task 1-2 Context |
| Stats — blocked count derivation `(Open + InProgress) − Ready` (required fix) | Task 1-3 |
| Stats — ready count tracks new semantics; two comment refreshes (verbatim strings) | Task 1-3 Do |
| `--status` composes (open→unstarted, in_progress→resumptions, terminal→empty) | Task 1-1 Do + AC + Edge Cases |
| `--count` (LIMIT after resume-first ORDER BY) | Task 1-2 Do + AC + Edge Cases |
| Affected code surface: query_helpers.go / list.go / stats.go | Tasks 1-1 / 1-2 / 1-3 |
| "No changes required" (state machine, flags, formatters, cache) | Honoured — no tasks invent work there |
| Test Impact — MUST change (4 files) | query_helpers_test.go, ready_test.go, blocked_test.go → 1-1; stats_test.go → 1-3 |
| Test Impact — KEEP no change | Named as KEEP in 1-1/1-2/1-3 Tests fields |
| Test Impact — New tests to ADD (5) | All 5 mapped (see below) |
| Acceptance Criteria #1–#10 | All 10 mapped (see below) |

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
- AC #10 → Task 1-1 AC + KEPT "excludes parent with in_progress children"

**New tests to ADD mapping** — all 5 traced:
1. Resume-first ordering (in_progress worse priority) → Task 1-2 `"it floats unblocked in_progress to the top of ready"`
2. Unblocked in_progress leaf in ready; blocked in_progress in blocked → Task 1-1 `"it partitions an in_progress task into exactly one of ready/blocked"`
3. Stats counts with in_progress present → Task 1-3 `"it derives a non-negative blocked count when ready exceeds open"`
4. `ready --status open` / `--status in_progress` composition → Task 1-1 `"it returns only unstarted work for ready --status open"` / `"it returns only resumptions for ready --status in_progress"`
5. `list --ready` floats identically to `ready` → Task 1-2 `"it floats in_progress identically for list --ready"`

**Items intentionally not made into tasks (correct):**
- *"Accepted consequence" (resumption-heavy ready)* — informational only; no behaviour or test. Correctly omitted.
- *Presentation — no change; Quiet mode and empty results* — spec explicitly states "pre-existing paths, no new work." The only behavioural effect (an unblocked `in_progress` task now appearing in `tick ready --quiet`) is a direct rendering consequence of Task 1-1's gate change through the untouched `fc.Quiet` path (verified at `internal/cli/list.go:220`); the spec's Test Impact inventory lists no quiet/empty test. No traceable code change to author. Correctly omitted.

### Direction 2: Plan → Specification (fidelity / anti-hallucination)

Every task's content traces back to the specification:

- **Task 1-1** — gate flip in both `ReadyConditions()`/`BlockedConditions()`, untouched inverse machinery, `--status` composition, partition, force-started/start-cascade/done-cancelled edge cases, and the `list_filter_test.go` comment refresh all map to spec sections "Ready & Blocked Definitions", "Consequences", "Filters/--count/Presentation", "Affected Code Surface (query_helpers.go)", and "Test Impact". The structural assertions ("ReadyConditions() returns 4 elements; BlockedConditions() returns 2 elements") were verified against `internal/cli/query_helpers.go` (4-element ready slice; 2-element blocked slice) — accurate, derived from the spec's "inverse machinery untouched" guarantee, not invented.
- **Task 1-2** — conditional `ORDER BY` keyed on `f.Ready`, the `handleReady`→`RunList` dispatch detail, worse-priority discriminating fixture, zero-in_progress no-op, `list --ready` parity, `--count 1`, narrowing-filter persistence, and "blocked gains no float" all map to spec "Sort Ordering — Resume-First, Ready-View-Only", "Filters/--count/Presentation", and "Affected Code Surface (list.go)".
- **Task 1-3** — blocked derivation, both comment-refresh strings (verbatim with spec), the explicit rejection of a `BlockedWhereClause()` helper, the fixture re-derivation (Ready=3, Blocked=2, `tick-bbb111` becomes a ready leaf), and the non-negative test all map to spec "Stats Counts" and "Test Impact".

**Verified, not hallucinated:**
- Task 1-1 names three `query_helpers_test.go` subtests to update; the spec text names two. Verified in code: the `t.status = 'open'` literal is asserted in **three** subtests (`query_helpers_test.go` lines 41, 60, 115 — `"ReadyConditions returns status open plus all four conditions"`, `"BlockedCondition returns open AND negation of ready subconditions"`, `"BlockedConditions contains no SQL literals beyond status check"`). Updating all three is *required* for the spec-mandated literal flip to compile/pass. This is faithful completion of the spec instruction "the gate becomes `t.status IN ('open','in_progress')`; both assertions update" applied to every site the literal actually appears — not invented scope.
- All existing test names referenced as KEEP/rewrite were verified to exist at the cited line numbers (`ready_test.go:204`, `blocked_test.go:126`, `blocked_test.go:400`, `stats_test.go:74`, `ready_test.go:82`/`167`/`304`, `blocked_test.go:208`).
- Task 1-1 correctly adopts spec **option (b)** for the `blocked_test.go` rewrite, matching the spec's stated preference ("prefer (b) here to keep this test focused on the blocked side").

No plan content lacks a specification anchor.
