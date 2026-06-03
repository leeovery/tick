---
status: in-progress
created: 2026-06-03
cycle: 2
phase: Plan Integrity Review
topic: Ready Includes In-Progress
---

# Review Tracking: Ready Includes In-Progress - Integrity

## Summary

Cycle-2 re-review of the single-phase, three-task plan after the two cycle-1
fixes were applied. Both prior fixes were verified present and sound:

- **Cycle-1 fix #1** (Task 1-1 Tests-section entry for the `list_filter_test.go`
  stale-comment refresh): present in the detail file `phase-1-tasks.md` line 47.
  Verified the stale comment is at `list_filter_test.go:385` exactly as cited,
  and that the assertion (empty result) genuinely stays valid under the widened
  gate (`status IN (open,in_progress) AND status = done` is always false). Sound.
- **Cycle-1 fix #2** (Task 1-3 stats_test pointer re-anchored on the assertion
  block + explicit `tick-aaa222`-stays-blocked rationale): present in BOTH the
  tick task (tick-6e6a9c) and `phase-1-tasks.md` line 136. Verified the
  arithmetic against the live `stats_test.go` fixture (lines 83-87): Open 4,
  InProgress 1, so Ready = 3 (`aaa111`, `ccc222`, `bbb111`) and Blocked = 2
  (`aaa222`, `ccc111`) = `(Open 4 + InProgress 1) − Ready 3`. The
  `tick-aaa222`-stays-blocked rationale is correct: its blocker `tick-bbb111` is
  `in_progress` (unclosed, not done/cancelled), so widening the gate does not
  unblock it. Sound and complete.

All load-bearing code claims re-verified against live source:
`ReadyConditions()` returns 4 elements with `t.status = 'open'` first
(query_helpers.go:51-58); `BlockedConditions()` returns 2 with `t.status =
'open'` first and the 3-EXISTS De Morgan inverse (query_helpers.go:70-80);
`stats.go:85` derives `Blocked = Open - Ready` with the two stale comments at
lines 78 and 84; `buildListQuery` (list.go:262) emits `ORDER BY t.priority ASC,
t.created ASC` (line 311) before the `LIMIT ?` append (314), inside `f.Ready`
branching; `handleReady` (app.go:211) is the `--ready` alias dispatch; the
`--status` filter composes as `t.status = ?` AND-ed (list.go:274-276). All
cited subtest names exist (ready_test.go, blocked_test.go, query_helpers_test.go,
stats_test.go, list_show_test.go, list_filter_test.go). The specification file
exists.

Re-checking the remaining integrity criteria (vertical slicing, phase structure,
dependencies/ordering, self-containment, scope/granularity, acceptance-criteria
testability) surfaced no new issues — those all remain PASS as in cycle 1.

One new finding surfaced while comparing the canonical tick store against the
detail file: the **Tests** section — a required template field — is absent from
the tick descriptions of Tasks 1-1 and 1-2, even though it is present in the
detail file and in Task 1-3's tick description. This is a template-compliance
gap in the implementer-facing plan-of-record (the tick store, read via
`tick show` per the format's reading.md). It is the same class of issue cycle 1
raised (Tests-section completeness), partially un-propagated to the canonical
store. Minor severity (the Do sections carry the test instructions narratively),
but it leaves the two tasks below template and inconsistent with 1-3.

## Findings

### 1. Tasks 1-1 and 1-2 tick descriptions are missing the required Tests section

**Severity**: Minor
**Plan Reference**: Phase 1, Task 1-1 (tick-fe7e70) and Task 1-2 (tick-2f0d2a)
**Category**: Task Template Compliance (Tests is a required field)
**Change Type**: add-to-task

**Details**:
`task-design.md` lists **Tests** as a required task field ("At least one test
name; include edge cases, not just happy path"). The canonical, implementer-facing
plan-of-record is the tick store, which an implementer reads via `tick show
<id>` (per the tick format's reading.md). Task 1-3's tick description
(tick-6e6a9c) carries a full `Tests:` section — it was added during cycle-1 fix
#2 when that description was re-anchored. But Tasks 1-1 (tick-fe7e70) and 1-2
(tick-2f0d2a) have **no `Tests:` section at all** in their tick descriptions;
their sections run Problem → Solution → Outcome → Do → Acceptance Criteria →
Edge Cases → Context → Spec Reference, skipping Tests entirely. The detail file
`phase-1-tasks.md` does carry Tests sections for all three, so the gap is only
in the tick store.

Why it matters for implementation: an implementer working from the tick store
(the intended consumption path) gets the full Do/AC/Edge-Cases for 1-1 and 1-2
but loses the explicit Tests index — which carries load-bearing
rewrite/KEEP/new distinctions (e.g. `"it excludes in_progress tasks"` →
rewritten to `"it includes unblocked in_progress leaf"`; `"it excludes parent
with in_progress children"` is KEPT; the `list_filter_test.go` comment-refresh
test stays green). These are only partially recoverable from the Do prose. The
omission also makes 1-1/1-2 inconsistent with 1-3, which does carry the section.
Cycle 1's fix #1 added the missing Tests entry to the *detail file* for 1-1 but
did not propagate a Tests section into the 1-1/1-2 tick descriptions.

The fix is to add the `Tests:` section to the tick descriptions of tick-fe7e70
and tick-2f0d2a, copied verbatim from the corresponding Tests section already in
`phase-1-tasks.md` (including the cycle-1-added `list_filter_test.go` entry for
1-1), rendered in the plain-text style the tick descriptions use (matching how
1-3's Tests section is rendered in tick-6e6a9c). No detail-file change is needed
— `phase-1-tasks.md` is already correct.

**Current** (Task 1-1, tick-fe7e70 — the section ordering, abbreviated; there is
no `Tests:` section between `Edge Cases:` and `Context:`):
```
  Edge Cases:
  - Force-started blocked task: ...
  - in_progress parent via start-cascade: ...
  - done/cancelled in neither: ...
  - --status done/cancelled compose to empty: ...

  Context: Governing invariant — ...
```

**Proposed** (Task 1-1, tick-fe7e70 — insert a `Tests:` section between
`Edge Cases:` and `Context:`):
```
  Edge Cases:
  - Force-started blocked task: ...
  - in_progress parent via start-cascade: ...
  - done/cancelled in neither: ...
  - --status done/cancelled compose to empty: ...

  Tests:
  - 'it includes unblocked in_progress leaf' (rewritten from 'it excludes in_progress tasks') — an unblocked in_progress leaf now appears in tick ready.
  - 'it includes blocked in_progress task' (rewritten from blocked_test.go 'it excludes in_progress tasks from output', option b) — an in_progress task with an unclosed blocker appears in tick blocked.
  - 'it partitions an in_progress task into exactly one of ready/blocked' (new) — an unblocked in_progress leaf is in ready and absent from blocked; a blocked in_progress task is in blocked and absent from ready.
  - 'it returns only unstarted work for ready --status open' (new) — tick ready --status open excludes the unblocked in_progress leaf, includes the unblocked open leaf.
  - 'it returns only resumptions for ready --status in_progress' (new) — tick ready --status in_progress includes the unblocked in_progress leaf, excludes the unblocked open leaf.
  - 'it excludes parent with in_progress children' (KEEP, ready_test.go) — leaf gate still excludes a start-cascade in_progress parent (AC #10).
  - 'it excludes task with in_progress blocker' (KEEP, ready_test.go) — blocker rule unchanged.
  - 'ReadyConditions returns status open plus all four conditions' (updated literal) — asserts conditions[0] == "t.status IN ('open', 'in_progress')".
  - 'BlockedConditions contains no SQL literals beyond status check' (updated literal, KEEP EXISTS-count == 3) — gate literal updated, inverse machinery unchanged.
  - 'contradictory filters return empty result no error' (KEEP assertion, list_filter_test.go) — --status done --ready still returns an empty result; only the stale inline comment is refreshed to explain the now-empty intersection (status IN (open,in_progress) AND status = done is always false). Confirm it stays green.

  Context: Governing invariant — ...
```

**Resolution**: Pending
**Notes**: Also apply the analogous insertion to Task 1-2 (tick-2f0d2a): add a
`Tests:` section between its `Edge Cases:` and `Context:` sections, copied from
the Tests section already in `phase-1-tasks.md` lines 93-99:

```
  Tests:
  - 'it floats unblocked in_progress to the top of ready' (new) — in_progress priority 3 sorts above open priority 0; within each band priority ASC, created ASC holds.
  - 'it floats in_progress identically for list --ready' (new, via runList) — tick list --ready produces the same in-progress-first ordering as tick ready.
  - 'it returns the top unblocked in-flight task with --count 1' (new) — tick ready --count 1 returns the floated in_progress task; with zero in-progress, returns the top unblocked open task.
  - 'it orders by priority ASC then created ASC' (KEEP, ready_test.go with only open tasks) — zero-in_progress ordering is byte-identical (no regression).
  - 'it orders by priority ASC then created ASC' (KEEP, blocked_test.go) — blocked ordering unchanged (no float).
  - Plain tick list ordering test (KEEP, list_show_test.go) — unchanged neutral ordering.
```

The detail file `phase-1-tasks.md` already carries both Tests sections verbatim
and needs no change; this finding only brings the canonical tick store into
template compliance and into line with Task 1-3.

---
