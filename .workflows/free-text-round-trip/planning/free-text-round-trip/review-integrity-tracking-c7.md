# Review Tracking: Free Text Round Trip - Integrity

## Findings

### 1. Phase 1's ordering rationale claims the detail document is produced by "five of the eight listed commands", and no set of eight commands exists anywhere in the plan or the specification

**Severity**: Minor
**Plan Reference**: Phase 1 (Task detail document becomes conformant TOON), "Why this order" — `planning.md:10`
**Category**: Phase Structure — the phase's stated rationale contains an unverifiable count
**Move**: settled
**Change Type**: update-phase

**Problem**:
"Why this order" is the paragraph a reader consults to understand why Phase 1 comes first and what it is worth to the rest of the plan — the same role this field plays in every other phase, and the one cycle 6 already corrected once for Phase 3. Phase 1's version opens: "It is the document five of the eight listed commands produce." Five is right — the task-detail document is produced by `show`, `create`, `update`, `note add` and `note remove`, exactly as the phase's own Goal states and task `free-text-round-trip-1-1`'s Problem confirms ("Five commands emit this document"). Eight is not grounded anywhere. The word "eight" appears nowhere in the specification (`grep -in eight` over `specification.md` returns nothing), and nowhere else in `planning.md` either. The only inventory of "listed commands" the plan or spec names is §3.1's must-parse table, which task `free-text-round-trip-6-3`'s own Context quotes in full: task detail (`show`, `create`, `update`, `note add`, `note remove` — 5), task list (`list`, `ready`, `blocked` — 3), stats (`stats` — 1), dependency graph (`dep tree` — 1), status change (`start`, `done`, `cancel`, `reopen` — 4). That is fourteen commands, not eight, and it is the same fourteen that task `free-text-round-trip-6-3`'s `TestConformanceInventoryCoversEveryCommand` guard partitions into must-parse / prose / out-of-scope. A reader who tries to verify "eight" against the one table the plan itself treats as authoritative for "listed commands" gets fourteen, and has no way to reconcile the two.

**Proposal**:
Correct the count to the one the plan's own §3.1 inventory supports — fourteen — and name the source explicitly so the claim is checkable rather than asserted. Nothing else in the sentence changes, and no task or acceptance criterion is affected: this is a rationale-paragraph fact, not a decision.

**Current**:
```
**Why this order**: It is the document five of the eight listed commands produce, and its header is the line a reader fails on before it sees anything else. Every later phase consumes it: the `changed` table becomes a section inside it, field selection projects from it, the round-trip fixture reads through it. It also deletes the mechanism behind every malformed section — hand-assembled string building and header surgery — establishing the pattern the remaining phases follow.
```

**Proposed Text**:
```
**Why this order**: It is the document five of the fourteen commands in §3.1's must-parse inventory produce, and its header is the line a reader fails on before it sees anything else. Every later phase consumes it: the `changed` table becomes a section inside it, field selection projects from it, the round-trip fixture reads through it. It also deletes the mechanism behind every malformed section — hand-assembled string building and header surgery — establishing the pattern the remaining phases follow.
```

**Resolution**: Pending
**Notes**:

---

## Verification Notes (no finding)

Recorded so the checks that were run and came back clean are visible, and are not re-run from scratch next cycle.

- **All 38 tasks read in full from tick** (`tick show` on every `task_map` entry) and checked against every review-integrity criterion: Task Template Compliance, Vertical Slicing, Phase Structure, Dependencies and Ordering, Task Self-Containment, Scope and Granularity, Acceptance Criteria Quality. No template-field omissions, no vague or subjective acceptance criteria, no missing edge-case coverage found.
- **Artifact sync spot-checked**: task `free-text-round-trip-1-1` compared word-for-word between `phase-1-tasks.md` and its tick record (`tick-36cfbf`) — identical. Consistent with cycle 5's full field-for-field sync check across all 38 tasks, which this cycle did not need to repeat.
- **Phase containers in tick carry no rationale text** (re-confirmed by inspecting all six phase containers — `tick-063549`, `tick-41bed7`, `tick-0540b5`, `tick-ec0b32`, `tick-a92b82`, `tick-9a4efb` — directly): the "Why this order" narrative lives only in `planning.md`, so the fix above touches a single location.
- **A full sweep for the cycle-6 defect class** (a phase's "Why this order" asserting independence/shared-code status from another phase) — re-read all six phases' rationale paragraphs; cycle 6's fix to Phase 3 holds verbatim in the current `planning.md`, and no new instance of that pattern was introduced.
- **A sweep for numeric/count claims in `planning.md`** (`grep -E` for number-words one through fourteen) found one unverifiable count — the finding above — and no others; every other numeric claim checked (Phase 2's "two independent cascade blocks", Phase 3's "three of its six tasks", Phase 4's "two of its formatter tasks", Phase 6's "update's four branches", the "seven" count-zero headers in Phase 6's acceptance criteria against task `free-text-round-trip-6-6`'s own list) is internally consistent and traceable to the tasks or context that state it.
- **Dependencies and ordering re-checked**: the tick `blocked_by` edges present (README chore tasks in Phases 2-5 each blocked on `free-text-round-trip-1-6`'s tick record, and `free-text-round-trip-5-5` blocked on `free-text-round-trip-4-6`) are genuine cross-phase or convergence dependencies; no missing edge found given natural creation-date ordering within and across phases (phase task timestamps increase monotonically phase by phase, confirmed via `tick show`).
- **Comments-are-not-task-content and scope/granularity** were not re-run in full detail beyond the read-through above — cycle 6's verification notes cover them exhaustively and nothing in this cycle's read contradicts them.
