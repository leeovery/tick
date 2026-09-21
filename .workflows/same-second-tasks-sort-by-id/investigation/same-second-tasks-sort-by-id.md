# Investigation: Same Second Tasks Sort By Id

## Symptoms

### Problem Description

**Expected behavior:**
Tasks listed back in creation order come back in the order they were authored. A batch written one after another — by an agent, an importer, or a shell loop — reads back in the sequence it was written.

**Actual behavior:**
Tasks created within the same wall-clock second come back in task-ID order, which is effectively random — IDs are three random bytes. The creation-date sort ties on every row in the batch and falls through to the secondary key.

### Manifestation

- `tick ready` returned a five-task phase shuffled out of authoring order
- The plan adapter's "N of M in phase" positions were visibly wrong
- No error, no warning — the ordering is silently wrong; a batch with real sequencing between tasks would execute out of order with no signal

### Reproduction Steps

1. Create several tasks in quick succession (scripted, so they share a wall-clock second)
2. List them back — `tick list`, `tick ready`, or `tick blocked`
3. Observe the order does not match the order of creation

**Reproducibility:** Always, when the batch shares a second. A human typing `tick add` repeatedly will not hit it; anything scripted hits it every time.

### Environment

- **Affected environments:** Local — all usage
- **User conditions:** Any batch created fast enough to share a wall-clock second. The faster the writer, the more reliably authoring order is lost.

### Impact

- **Severity:** Medium — silent wrong ordering, no data loss
- **Scope:** Every consumer that treats tick as an ordered list rather than a bag of tasks: the workflow system's plan adapter and its implementation loop, `tick list` on a freshly imported project, the migration framework's output, a dependency chain authored in one pass
- **Business impact:** Trust in tick as a plan store. The observed case had independent tasks so nothing broke; a sequenced batch would run out of order undetected.

### References

- Seed: `seeds/2026-09-21-same-second-tasks-sort-by-id.md` (inbox:bug)
- Discovery session: `.workflows/same-second-tasks-sort-by-id/discovery/sessions/session-001.md`
- Observed while dogfooding tick as the plan store for the `free-text-round-trip` workflow (Phase 10)

### Scope Note (from discovery)

Getting creation order right for new tasks going forward is confirmed sufficient. What the fix should do about tasks already stored at second granularity — existing `tasks.jsonl` files that already carry ties — was deliberately left to this investigation.

---

## Analysis

### Hypotheses

**Checkpoint depth:** {to be agreed}

(ledger to be populated at Step 5)

### Code Trace

(to be populated)

**Ground named by the seed:**
- `internal/cli/query_helpers.go` and the list/ready/blocked query paths carrying the `ORDER BY` clauses
- the task record's `created` field in `internal/task/`
- the SQLite cache schema in `internal/storage/cache.go`

### Root Cause

(to be populated)

### Contributing Factors

(to be populated)

### Why It Wasn't Caught

(to be populated)

### Blast Radius

(to be populated)

---

## Fix Direction

(to be populated after findings sign-off)

---

## Notes

(none yet)
