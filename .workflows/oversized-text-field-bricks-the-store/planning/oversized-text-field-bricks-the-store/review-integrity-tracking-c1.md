# Review Tracking: Oversized Text Field Bricks The Store - Integrity

## Findings

### 1. The read-error criterion for a line with no `id` promises an error the loader never raises

**Severity**: Important
**Plan Reference**: Phase 1 / Task oversized-text-field-bricks-the-store-1-6 (tick-2d2b01), Acceptance Criteria
**Category**: Acceptance Criteria Quality
**Move**: settled
**Change Type**: update-task

**Problem**:
Task 1-6 has a criterion saying a line with a missing `id` key produces an error naming the line alone. That is false for a line that is otherwise complete. Such a line loads today and every command opens the store. (Measured with the installed binary: a full task record without `id` makes `tick list` exit 0, listing the task with an empty ID.) If the executor builds the fixture from a complete record, no error appears. The only way to make the criterion pass as written is to make the loader refuse any line without an `id`. §3 never decided that. It governs how a failing line is named, not which lines fail. After that change, a store holding such a line from a hand edit or another tool would stop opening for every command. That is the kind of failure this fix exists to end, and it would show the first time `tick list` runs on that store.

**Proposal**:
Limit the criterion to a line that fails to load, since that is all §3 governs: "Every error from reading `tasks.jsonl` into the store names the line that stopped it … A line that is not valid JSON, or has no string `id`, is named by line alone." For the missing-key case, the fixture must fail for some other reason. For example, `{"title":"T"}` fails on its empty `created` timestamp. Which lines the loader accepts does not change.

**Current**:
- [ ] A line with no string `id` (the key is missing, the `id` is not a string, or the value is not a JSON object, such as `null` or `[]`): the error names the line by number alone, followed by the reason (§3)

**Proposed Text**:
- [ ] A line that fails to load and carries no string `id`: the `id` key is missing (such as `{"title":"T"}`, which fails on its empty `created` timestamp), the `id` is not a string, or the value is not a JSON object (such as `null` or `[]`). The error names the line by number alone, followed by the reason (§3)

**Resolution**: Pending
**Notes**:

---

### 2. Phase 3 claims independence from Phase 1 that its own tasks and graph contradict

**Severity**: Important
**Plan Reference**: Phase 3: Description cap, **Why this order** (planning.md)
**Category**: Phase Structure
**Move**: settled
**Change Type**: update-task

**Problem**:
Phase 3's "Why this order" says nothing in Phase 3 depends on Phases 1 and 2. The rest of the plan says otherwise:
- Task 3-1 is blocked by Task 1-1.
- Tasks 3-1 and 3-2 both have a multibyte boundary scenario: a description of exactly 50,000 multibyte characters. That takes the task's line past 65,536 bytes, and such a line reads back only through Phase 1's reader.

If someone takes the sentence at its word and moves Phase 3 ahead of Phase 1, or lands it on its own, Task 3-1 stays blocked. Task 3-2's acceptance fails: `tick update --description` with 50,000 multibyte characters commits the write, then exits 1 on its read-back with `bufio.Scanner: token too long`, and the store is left unreadable.

**Proposal**:
State the dependency the graph already records. The cap check itself needs nothing from the earlier phases, but its multibyte boundary scenarios need Phase 1's reader (§2.1, §6.1). Remove the sentence that claims full independence.

**Current**:
**Why this order**: The cap is hygiene, not the store's safety device (§6.1), and nothing in it depends on Phases 1 and 2. It lands after the defect fixes so that the store's safety never depends on it.

**Proposed Text**:
**Why this order**: The cap is hygiene, not the store's safety device (§6.1). It lands after the defect fixes so that the store's safety never depends on it. The cap's own check needs nothing from Phases 1 and 2, but its multibyte boundary scenarios do. A description of 50,000 multibyte characters takes the task's line past 65,536 bytes, and that line reads back only through Phase 1's shared line reader (§2.1, §6.1). Task 3-1 depends on Task 1-1 for that reason.

**Resolution**: Pending
**Notes**:

---

## Observations

- Task 1-2 AC3 says `tick update <oversized-id>` succeeds but names no flag. A call with no flags is refused ("at least one flag is required"), so the executor has to pick a flag. The task's Context already steers them away from setting an over-cap description.
- Tasks 1-2 (the ~1 MiB task in AC4) and 1-4 (its near-ceiling records) lack the note that Tasks 1-3 and 1-5 carry: their large records must not come from setting a description over Phase 3's 50,000-character cap. The raw-fixture helper `setupTickProjectWithTasks` makes writing the records directly the natural path anyway.
- Task 2-4 AC3's fixture list leaves out malformed JSON, which Phase 2's acceptance includes. AC1's general "a line that fails to load" still covers it.
- Task 3-2's multibyte scenario at 50,000 characters also reads back a line past 65,536 bytes, but unlike 3-1 it has no edge to Task 1-1. Creation order runs Phase 1 first, so nothing runs out of order.
- Task 1-5's Do, "drops the ceiling of its `bufio.NewScanner`", can be read as keeping a `bufio.Scanner`, which always has some maximum token size. AC3 (no limit of any size) is the binding criterion.
