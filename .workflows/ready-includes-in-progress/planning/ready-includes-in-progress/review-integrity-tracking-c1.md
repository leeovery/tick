---
status: in-progress
created: 2026-06-03
cycle: 1
phase: Plan Integrity Review
topic: Ready Includes In-Progress
---

# Review Tracking: Ready Includes In-Progress - Integrity

## Summary

The plan is a single, surgically-scoped phase of three tasks, each touching a
distinct seam (query_helpers.go status gate, list.go conditional ORDER BY,
stats.go blocked derivation). Reviewed against all integrity criteria:

- **Task Template Compliance**: PASS — all three tasks carry Problem, Solution,
  Outcome, Do, Acceptance Criteria, Tests, Edge Cases, Context, and Spec
  Reference. Every field is substantive.
- **Vertical Slicing**: PASS — each task delivers a complete, independently
  verifiable behaviour increment on its own seam; none is a horizontal layer.
- **Phase Structure**: PASS — the single-phase decision is explicitly justified
  ("Why this order") by the partition invariant `ready ⊎ blocked = all live
  tasks`; splitting would leave a broken negative-blocked-count intermediate.
  Phase acceptance criteria are comprehensive (10 criteria mapping to spec AC).
- **Dependencies and Ordering**: PASS — tasks are authored in dependency order
  (1-1 widens the gate → 1-2 floats in_progress → 1-3 fixes the count), and
  `tick ready --parent <phase>` returns them in exactly that order. Natural
  (creation-date) ordering produces the correct execution sequence; there is no
  convergence point and no cross-phase edge, so per the review criteria explicit
  `blocked_by` edges are correctly NOT required. No circular dependencies.
- **Task Self-Containment**: PASS — each task pulls the governing spec decisions
  into its Context block; an implementer needs nothing beyond the task itself.
- **Scope and Granularity**: PASS — each task is one TDD cycle (a one-line-per-side
  SQL change plus its test impact); none is mechanical boilerplate or oversized.
- **Acceptance Criteria Quality**: PASS — criteria are concrete, pass/fail, and
  each is tagged to a spec AC number; fixture re-derivations are spelled out
  numerically.
- **External Dependencies**: N/A (feature, not epic).

I verified every load-bearing code claim against the live source:
`ReadyConditions()` returns 4 elements with `t.status = 'open'` first;
`BlockedConditions()` returns 2 with the De Morgan inverse machinery (3 EXISTS);
`stats.go` derives `Blocked = stats.Open - stats.Ready` at line 85;
`buildListQuery` emits `ORDER BY t.priority ASC, t.created ASC` before the
`LIMIT` append; `handleReady` is an alias dispatching `--ready` into `RunList`;
the `--status` filter composes as `t.status = ?` AND-ed with `ReadyConditions()`;
and the stats_test fixture re-derivation (Ready 3 / Blocked 2 = (Open 4 +
InProgress 1) − Ready 3) is arithmetically correct against the actual fixture.
All referenced test subtest names and line numbers (ready_test.go,
blocked_test.go, query_helpers_test.go, stats_test.go, list_filter_test.go)
exist as cited. The referenced specification file and all cited sections exist.

The findings below are minor polish only. None blocks implementation.

## Findings

### 1. Task 1-1 omits a Tests-section entry for the stale-comment refresh step in list_filter_test.go

**Severity**: Minor
**Plan Reference**: Phase 1, Task 1-1 (Widen the shared ready/blocked status gate to live tasks)
**Category**: Task Template Compliance (Tests completeness)
**Change Type**: add-to-task

**Details**:
The Do section instructs the implementer to refresh the stale inline comment in
`list_filter_test.go` subtest `"contradictory filters return empty result no
error"` (the `// ... (ready only applies to open tasks)` comment is now false),
and the Edge Cases section covers the `--status done/cancelled compose to empty`
behaviour. But the Tests section lists no entry naming this kept-and-still-green
subtest, even though every other touched/kept test (ready_test.go,
blocked_test.go, query_helpers_test.go) is named explicitly there. This is the
one Do-step whose verifying test is not surfaced in the Tests list, leaving a
small gap between "what I must touch" and "what test proves it." Adding the
entry keeps the Tests section a complete index of the task's test impact.

**Current**:
```markdown
- `"it excludes parent with in_progress children"` (KEEP, ready_test.go) — leaf gate still excludes a start-cascade `in_progress` parent (AC #10).
- `"it excludes task with in_progress blocker"` (KEEP, ready_test.go) — blocker rule unchanged.
- `"ReadyConditions returns status open plus all four conditions"` (updated literal) — asserts `conditions[0] == "t.status IN ('open', 'in_progress')"`.
- `"BlockedConditions contains no SQL literals beyond status check"` (updated literal, KEEP `EXISTS`-count == 3) — gate literal updated, inverse machinery unchanged.
```

**Proposed**:
```markdown
- `"it excludes parent with in_progress children"` (KEEP, ready_test.go) — leaf gate still excludes a start-cascade `in_progress` parent (AC #10).
- `"it excludes task with in_progress blocker"` (KEEP, ready_test.go) — blocker rule unchanged.
- `"ReadyConditions returns status open plus all four conditions"` (updated literal) — asserts `conditions[0] == "t.status IN ('open', 'in_progress')"`.
- `"BlockedConditions contains no SQL literals beyond status check"` (updated literal, KEEP `EXISTS`-count == 3) — gate literal updated, inverse machinery unchanged.
- `"contradictory filters return empty result no error"` (KEEP assertion, list_filter_test.go) — `--status done --ready` still returns an empty result; only the stale inline comment is refreshed to explain the now-empty intersection (`status IN (open,in_progress) AND status = done` is always false). Confirm it stays green.
```

**Resolution**: Fixed
**Notes**: Applied to the Tests section of Task 1-1 in phase-1-tasks.md. The tick task (tick-fe7e70) Do section already instructs refreshing this comment; this completes the Tests index in the permanent record.

---

### 2. Task 1-3 Do step cites stats_test "(line ~74)" but the count assertions live ~25 lines lower; line hint is imprecise

**Severity**: Minor
**Plan Reference**: Phase 1, Task 1-3 (Correct the stats blocked-count derivation to the live set)
**Category**: Acceptance Criteria Quality (precision of implementation pointer)
**Change Type**: update-task

**Details**:
The Do section locates the test to update as `"it counts ready and blocked tasks
correctly" (line ~74)`. Line 74 is the `t.Run(...)` opener; the actual lines the
task tells the implementer to change — `workflow["ready"]` expected `2`→`3`, the
`workflow["blocked"]` derivation comment, and the `tick-bbb111` fixture comment —
sit at roughly lines 99–112 (the assertions block and the fixture-setup comment
near line 78). The `~74` hint points at the subtest header rather than the edit
sites. This is harmless (the implementer will find them), but the line hint is
the one place in the otherwise-precise task that under-points. Either drop the
line number (the subtest name is an unambiguous anchor) or point at the
assertion block. Proposed: remove the bare `(~line 74)` and reference the
assertion block / fixture comment instead, since exact line numbers drift.

**Current**:
```markdown
- Update `internal/cli/stats_test.go` subtest `"it counts ready and blocked tasks correctly"` (line ~74): under the new semantics `tick-bbb111` (in_progress, no blockers, no children) becomes a READY leaf. Re-derive expected counts against the fixture: Ready = 3 (`tick-aaa111` open ready leaf, `tick-ccc222` open ready child leaf, `tick-bbb111` in_progress ready leaf); Blocked = 2 (`tick-aaa222` blocked by dep, `tick-ccc111` has open child) = `(Open 4 + InProgress 1) − Ready 3`. Change `workflow["ready"]` expected from `2` to `3` and `workflow["blocked"]` stays `2` but update its inline derivation comment. Correct the inline fixture comment for `tick-bbb111` (it is now a ready leaf, not "neither ready nor blocked").
```

**Proposed**:
```markdown
- Update `internal/cli/stats_test.go` subtest `"it counts ready and blocked tasks correctly"` (the `workflow["ready"]`/`workflow["blocked"]` assertion block and the fixture-setup comments at the top of the subtest): under the new semantics `tick-bbb111` (in_progress, no blockers, no children) becomes a READY leaf. Re-derive expected counts against the fixture: Ready = 3 (`tick-aaa111` open ready leaf, `tick-ccc222` open ready child leaf, `tick-bbb111` in_progress ready leaf); Blocked = 2 (`tick-aaa222` blocked by its in_progress blocker `tick-bbb111`, which is unclosed; `tick-ccc111` has open child) = `(Open 4 + InProgress 1) − Ready 3`. Change `workflow["ready"]` expected from `2` to `3`; `workflow["blocked"]` stays `2` but update its inline derivation comment. Correct the inline fixture comment for `tick-bbb111` (it is now a ready leaf, not "neither ready nor blocked"). Note: `tick-aaa222` stays blocked because its blocker `tick-bbb111` is in_progress (not done/cancelled), so widening the gate does not unblock it.
```

**Resolution**: Pending
**Notes**: The Proposed text also makes explicit *why* `tick-aaa222` stays blocked after the gate widens (its blocker is in_progress, hence still unclosed) — the original wording leaves the reader to confirm this themselves, and it is the one re-derivation step a careless implementer could get wrong.

---
