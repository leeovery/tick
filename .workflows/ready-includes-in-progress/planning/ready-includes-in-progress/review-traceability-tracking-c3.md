---
status: complete
created: 2026-06-03
cycle: 3
phase: Traceability Review
topic: Ready Includes In-Progress
---

# Review Tracking: Ready Includes In-Progress - Traceability

## Result: CLEAN

No findings. The plan remains a faithful, complete translation of the specification in
both directions after the cycle-2 fix. Every specification requirement, decision, edge
case, code-surface change, test-inventory item, and acceptance criterion is covered by at
least one task, and every piece of task content traces back to a specific part of the
specification — no invented scope. The cycle-2 edit (inserting the `Tests:` section into
the tick records of Tasks 1-1 and 1-2) introduced **no traceability drift**.

## Findings

(none)

---

## Cycle-2 Fix Verification

The cycle-2 fix was a single edit applied by the integrity gate (commit `0a950dd`,
"apply integrity finding (Tests section for 1-1/1-2)"). It was re-checked for traceability
drift in both directions.

### Edit — `Tests:` section inserted into tick records of Tasks 1-1 and 1-2

**What changed (verified against the commit diff):** The fix inserted a `Tests:` section
into the description field of `tick-fe7e70` (Task 1-1) and `tick-2f0d2a` (Task 1-2) in
`.tick/tasks.jsonl`, placed between `Edge Cases:` and `Context:`, copied verbatim from the
already-present Tests sections in `phase-1-tasks.md` (lines 37–47 for 1-1, lines 93–99 for
1-2). The commit touched only `.tick/tasks.jsonl` (the two task descriptions + their
`updated` timestamps), `manifest.json` (`review_cycle` 1→2), and the cycle-2 integrity
tracking file's status. It did **not** touch `phase-1-tasks.md` or `planning.md` — the
detail file already carried these Tests sections (verified in prior cycles), so the fix
only propagated existing, already-traced content into the tick mirror.

**Verified:** The Tests content now rendered by `tick show tick-fe7e70` and
`tick show tick-2f0d2a` is byte-identical to the corresponding `phase-1-tasks.md` Tests
sections. No test name, KEEP/rewrite/new annotation, or rationale was added, removed, or
altered relative to the detail file. The structural drift the cycle-2 traceability review
recorded as an out-of-scope note ("Tasks 1-1/1-2 fold test instructions into Do rather than
carrying a standalone Tests section in the tick record") is now resolved — all three tick
records carry explicit Tests sections matching the detail file.

**Every inserted test entry traces to the spec's Test Impact inventory and Acceptance
Criteria** (re-confirmed against the specification, not memory):

Task 1-1 (10 entries):
- `"it includes unblocked in_progress leaf"` (rewrite) → Test Impact "MUST change" `ready_test.go` inversion; AC #1.
- `"it includes blocked in_progress task"` (rewrite, option b) → Test Impact "MUST change" `blocked_test.go`, spec-preferred option (b); AC #3.
- `"it partitions an in_progress task into exactly one of ready/blocked"` (new) → New tests to ADD #2; AC #4.
- `"it returns only unstarted work for ready --status open"` (new) → New tests to ADD #4; AC #8.
- `"it returns only resumptions for ready --status in_progress"` (new) → New tests to ADD #4; AC #8.
- `"it excludes parent with in_progress children"` (KEEP) → Test Impact "KEEP" `ready_test.go`; AC #10.
- `"it excludes task with in_progress blocker"` (KEEP) → Test Impact "KEEP" `ready_test.go`.
- `"ReadyConditions returns status open plus all four conditions"` (updated literal) → Test Impact "MUST change" `query_helpers_test.go`.
- `"BlockedConditions contains no SQL literals beyond status check"` (updated literal) → Test Impact "MUST change" `query_helpers_test.go`.
- `"contradictory filters return empty result no error"` (KEEP) → Test Impact "KEEP" `list_filter_test.go`; `--status done` empty-intersection rule; AC #8.

Task 1-2 (6 entries):
- `"it floats unblocked in_progress to the top of ready"` (new) → New tests to ADD #1; AC #5.
- `"it floats in_progress identically for list --ready"` (new) → New tests to ADD #5.
- `"it returns the top unblocked in-flight task with --count 1"` (new) → "Filters/--count" + AC #7.
- `"it orders by priority ASC then created ASC"` (KEEP, ready_test.go) → no-regression basis; AC #6.
- `"it orders by priority ASC then created ASC"` (KEEP, blocked_test.go) → Test Impact "KEEP" `blocked_test.go`; AC #6.
- Plain `tick list` ordering test (KEEP, list_show_test.go) → AC #6.

No entry introduces a test, behaviour, or scope absent from the spec. **No drift.**

---

## Analysis Record

### Direction 1: Specification → Plan (completeness)

Every spec element remains represented in the plan:

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
| Test Impact — KEEP no change | Named as KEEP in 1-1/1-2/1-3 Tests fields (incl. list_filter_test.go) |
| Test Impact — New tests to ADD (5) | All 5 mapped across 1-1/1-2/1-3 |
| Acceptance Criteria #1–#10 | All 10 tagged in task ACs |
| Out of Scope (multi-actor, pretty cue, most-recently-started) | Correctly excluded; flagged as out-of-scope in Context |
| Accepted consequence (resumption-heavy ready) | Informational only; correctly omitted |

**Acceptance Criteria mapping** — all 10 traced:
- AC #1 → Task 1-1 AC + Tests
- AC #2 → Task 1-1 AC + Task 1-2 AC (no-regression)
- AC #3 → Task 1-1 AC + Tests
- AC #4 → Task 1-1 AC + Tests (partition)
- AC #5 → Task 1-2 AC + Tests
- AC #6 → Task 1-2 AC + Tests
- AC #7 → Task 1-2 AC + Tests
- AC #8 → Task 1-1 AC + Tests
- AC #9 → Task 1-3 AC
- AC #10 → Task 1-1 AC + KEPT `"it excludes parent with in_progress children"`

**New tests to ADD mapping** — all 5 traced:
1. Resume-first ordering (in_progress worse priority) → Task 1-2 `"it floats unblocked in_progress to the top of ready"`
2. Unblocked in_progress leaf in ready; blocked in_progress in blocked → Task 1-1 `"it partitions an in_progress task into exactly one of ready/blocked"`
3. Stats counts with in_progress present → Task 1-3 `"it derives a non-negative blocked count when ready exceeds open"`
4. `ready --status open` / `--status in_progress` composition → Task 1-1 `"it returns only unstarted work for ready --status open"` / `"it returns only resumptions for ready --status in_progress"`
5. `list --ready` floats identically to `ready` → Task 1-2 `"it floats in_progress identically for list --ready"`

### Direction 2: Plan → Specification (fidelity / anti-hallucination)

Every task's content continues to trace back to the specification; the cycle-2 Tests-section
insertion added no untraceable content (each entry mapped above).

- **Task 1-1** — gate flip in both `ReadyConditions()`/`BlockedConditions()`, untouched
  inverse machinery, `--status` composition, partition, force-started / start-cascade /
  done-cancelled edge cases, the `list_filter_test.go` comment refresh, and the now-explicit
  Tests section all map to spec "Ready & Blocked Definitions", "Consequences",
  "Filters/--count/Presentation", "Affected Code Surface (query_helpers.go)", and
  "Test Impact". Structural assertions ("ReadyConditions() = 4 elements; BlockedConditions()
  = 2") trace to the spec's "inverse machinery untouched" guarantee.
- **Task 1-2** — conditional `ORDER BY` keyed on `f.Ready`, `handleReady`→`RunList`
  dispatch, worse-priority discriminating fixture, zero-in_progress no-op, `list --ready`
  parity, `--count 1`, narrowing-filter persistence, "blocked gains no float", and the
  now-explicit Tests section all map to spec "Sort Ordering — Resume-First,
  Ready-View-Only", "Filters/--count/Presentation", and "Affected Code Surface (list.go)".
- **Task 1-3** — blocked derivation, both comment-refresh strings (verbatim with spec), the
  explicit rejection of a `BlockedWhereClause()` helper, the fixture re-derivation
  (Ready 3 / Blocked 2, `tick-bbb111` becomes a ready leaf, `tick-aaa222` stays blocked),
  and the non-negative test all map to spec "Stats Counts" and "Test Impact". (Unchanged by
  the cycle-2 fix.)

No plan content lacks a specification anchor. The cycle-2 fix introduced no new scope — it
mirrored already-traced detail-file Tests content into the tick records.
