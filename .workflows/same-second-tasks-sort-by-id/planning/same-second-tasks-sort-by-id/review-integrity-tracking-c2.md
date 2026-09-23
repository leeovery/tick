# Review Tracking: Same Second Tasks Sort By Id - Integrity

## Findings

### 1. The migration-batch criterion asserts a one-second tie the import does not guarantee

**Severity**: Important
**Plan Reference**: Phase 1 Acceptance (planning.md); Task same-second-tasks-sort-by-id-1-5 (tick-a21794): Problem, Acceptance Criteria (first criterion), Context
**Category**: Acceptance Criteria Quality
**Move**: settled
**Change Type**: update-task

**Problem**:
Task 1-5 tells the implementer to assert that a five-issue `tick migrate` run stores every task with one identical creation second. The import does not guarantee that. Each imported task is stamped by its own `time.Now()` inside its own `Store.Mutate` (`internal/migrate/store_creator.go:60-63`). A five-issue import takes about 46ms, and 9 of 300 measured runs crossed a second boundary. A test written to this criterion fails about one run in thirty, on CI and locally, with nothing wrong in the product: a batch that crosses a boundary still lists in import order, sorted by its creation seconds. The Phase 1 Acceptance states the same guarantee ("writes a batch with identical timestamps"), and Task 1-5's Problem says every such batch shares one second.

**Proposal**:
Keep the one-second tie, because §8's fixture constraints require it. A batch spread across two seconds lists in import order by `created` and proves nothing about the sequence. State the tie as a condition of the fixture, not as a result the import is asserted to produce. Task 1-3's same-second criterion already does this ("successive `tick create` calls that record the same creation second"). The Problem stops claiming every batch shares one second. The Context records the per-task stamping and the measurement, so the implementer knows the tie has to be arranged. How the test arranges it is left to them. The code and the measurement settle this. §8.1's "its timestamps are identical" describes the fixture, and the fixture keeps it.

**Current**:

*planning.md — Phase 1 Acceptance (one bullet):*

- [ ] A migration import whose source carries no creation times writes a batch with identical timestamps: listing it returns import order

*Task 1-5 (tick-a21794) — Problem:*

**Problem**: `tick migrate` is the in-process batch writer: one run creates every imported task back to back, so a source that carries no creation times of its own stamps the whole batch with one second. Finer timestamps would not have separated them — 100 in-process stamps taken with tick's own ID generation in the loop produced one distinct millisecond. Unless each imported task carries a sequence, such a batch reads back in plan-dependent order. Imports that do carry historical creation times must keep that chronology rather than have import order override it.

*Task 1-5 — Acceptance Criteria (first criterion):*

- [ ] A beads source of five issues carrying no `created_at` and one priority, imported by a single `tick migrate` run: the stored tasks record one identical creation second, carry sequences in import order, and `tick list` returns them in import order — a result ascending task-ID order could not produce (§1.3, §8.1)

*Task 1-5 — Context (fixture-constraints paragraph):*

> §8 fixture constraints apply. The migration framework generates the task IDs itself, so the fixture cannot choose them; the assertion must still be one an ascending-ID result could not satisfy.

**Proposed Text**:

*planning.md — Phase 1 Acceptance (one bullet):*

- [ ] A migration import whose source carries no creation times, its batch recording one creation second: listing it returns import order

*Task 1-5 (tick-a21794) — Problem:*

**Problem**: `tick migrate` is the in-process batch writer: one run creates every imported task back to back, so a source that carries no creation times of its own stamps the batch with one second, except for the occasional run that crosses a second boundary. Finer timestamps would not have separated them — 100 in-process stamps taken with tick's own ID generation in the loop produced one distinct millisecond. Unless each imported task carries a sequence, such a batch reads back in plan-dependent order. Imports that do carry historical creation times must keep that chronology rather than have import order override it.

*Task 1-5 — Acceptance Criteria (first criterion):*

- [ ] A beads source of five issues carrying no `created_at` and one priority, imported by a single `tick migrate` run whose imported tasks record one creation second: the stored tasks carry sequences in import order, and `tick list` returns them in import order — a result ascending task-ID order could not produce (§1.3, §8.1)

*Task 1-5 — Context (fixture-constraints paragraph):*

> §8 fixture constraints apply. The migration framework generates the task IDs itself, so the fixture cannot choose them; the assertion must still be one an ascending-ID result could not satisfy. Nor does the import guarantee the one-second tie the constraints require. Each imported task is stamped by its own `time.Now()` inside its own `Store.Mutate` (`internal/migrate/store_creator.go:60-63`). A five-issue import takes about 46ms, and 9 of 300 measured runs straddled a second boundary. The tie is a condition the test sets up, not a result it asserts of the import. A batch that straddles a boundary still lists in import order, by `created`, and proves nothing about the sequence.

**Resolution**: Fixed
**Notes**: Auto-approved. Applied to planning.md (Phase 1 Acceptance bullet), phase-1-tasks.md and tick-a21794 (description, verified against the detail file).

---

## Observations

- Task 2-3's assertions pass on their first run once Tasks 2-1 and 2-2 land, because `outputMutationResult` already renders through `queryShowData`. The task has no failing-test step, and the executor's "test passes immediately" check applies.
- Task 2-3's Context and the Phase 2 table row name `dep rm`, quoting §4.4. The command is `dep remove`. The line is an out-of-scope note, and nothing is built from it.
- A cache built at v3 by a Phase 1 build lacks the dependency ordinal, and Phase 2 leaves the version at 3, so that cache would not rebuild. This happens only if an intermediate build runs against a real project, which the Homebrew-only dogfooding rule excludes. The single 2→3 bump is §3.4's decision.
