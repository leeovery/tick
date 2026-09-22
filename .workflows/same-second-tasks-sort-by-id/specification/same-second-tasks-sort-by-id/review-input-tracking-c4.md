# Review Tracking: Same-Second Tasks Sort By Id - Input Review

## Findings

### 1. Several tasks created in one write each need their own number

**Source**: `.workflows/same-second-tasks-sort-by-id/investigation/same-second-tasks-sort-by-id.md` — Fix Direction → Chosen Approach ("Each task gains a monotonic creation sequence — the next number above the highest currently in the file, assigned at creation"); Testing Recommendations → Ordering ("An in-process batch, not just separate invocations — the migration framework is the natural fixture, and the assertion is on *ordering* with the explicit note that its timestamps are identical"); Analysis → H4 (`migrate/store_creator.go` "appends per task and stamps `time.Now().UTC().Truncate(time.Second)` when the provider gives no creation time — so a whole import lands in one second")
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: Section 2.2 (Assignment)

**Problem**:
A project imported through the migration framework is the case with the most at stake here: every task lands in one wall-clock second, so the creation sequence is the only thing that can order them. If the tasks an import creates before the file is written back all take the same number, the import reads back in random task-ID order — the exact shuffle this work exists to remove — and `tick doctor` then warns that every imported task shares its number with every other, on a file the user has done nothing wrong to produce. The assignment rule is written for one creation against the task set as it was read, and says nothing about a write that creates several tasks at once, so an implementer can satisfy it and still ship this.

**Proposal**:
Say in the assignment rule that each new task takes the next number above every task already in the set, counting the ones added earlier in the same write. The investigation already decides it — the sequence is monotonic and assigned per task at creation, and it requires an in-process batch (the migration framework, timestamps identical) to be asserted for ordering, which can only pass if each task in one write takes its own number. Carrying that across closes the hole without changing the mechanism.

**Current**:
A new task takes the next number above the highest sequence currently in the file. Sequences are positive: numbering begins at 1, so an absent or zero value on a record means it carries no sequence. The highest is taken from the task set as read, not from the stored bytes: backfill (§2.3) runs first, so records that reached the file without a sequence already carry one and the next number sits above those too.

**Proposed Text**:
A new task takes the next number above the highest sequence currently in the file. Sequences are positive: numbering begins at 1, so an absent or zero value on a record means it carries no sequence. The highest is taken from the task set as read, not from the stored bytes: backfill (§2.3) runs first, so records that reached the file without a sequence already carry one and the next number sits above those too.

The rule is per task, not per write. Where one write creates several tasks — an import through `internal/migrate`, whose whole batch shares a single creation second — each takes the next number above every task already in the set, including the ones added earlier in that same write. A batch is therefore numbered in the order its tasks are appended, which is the order it was authored.

**Resolution**: Pending
**Notes**:

---

## Observations

- Neither source nor specification states the cache column's type for `seq`; the neighbouring `created` column is `TEXT` and compares lexically (investigation H6), which would order 10 before 2.
- The investigation's reason for reserving zero — under Go's `omitempty` a zero sequence vanishes from the record, so that task would be re-backfilled differently on every read until a write froze it wrong — is not carried; the 1-based rule the specification does state prevents the failure regardless.
- The name the duplicate-sequence check prints in `tick doctor` output is unstated; §6 turns on a user finding the diagnostic from the README's "duplicate creation sequences" entry.
