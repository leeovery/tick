---
topic: task-removal
status: in-progress
date: 2026-02-18
---

# Discussion: Task Removal

## Context

Tick currently supports status transitions (open → in_progress → done/cancelled) but has no mechanism to permanently remove a task from the data file. Cancelling a task keeps it in the JSONL log and SQLite cache — it's still visible in listings and still referenced by dependencies.

The need: a way to completely eliminate a task, removing it from the JSONL source of truth (and by extension the SQLite cache). This is a fundamentally different operation from cancellation — cancel marks a task as resolved-but-present; remove erases it entirely.

### Codebase Context

- **Terminology**: The codebase consistently uses "task" (not "issue")
- **Storage**: JSONL is the source of truth, rewritten atomically on every mutation via `Store.Mutate()`. SQLite cache is rebuilt from JSONL after each write. This means removal is mechanically straightforward — filter the task from the slice, rewrite.
- **Dependencies**: Tasks have `BlockedBy []string` (task IDs) and `Parent string`. Cancelling a task does NOT clean up BlockedBy arrays — cancelled tasks are treated as "resolved blockers" by query logic. Removing a task would leave orphaned IDs in other tasks' BlockedBy arrays.
- **Existing commands**: `cancel` is the closest analog — dispatched via `handleTransition()` → `RunTransition()`. No bulk operations exist for status transitions.
- **Doctor**: The `doctor` command validates JSONL integrity but currently doesn't check for orphaned dependency references.

### References

- Task struct: `internal/task/task.go`
- Dependency validation: `internal/task/dependency.go`
- Store.Mutate: `internal/storage/store.go`
- Transition handler: `internal/cli/transition.go`
- Command dispatch: `internal/cli/app.go`

## Questions

- [ ] What does "remove" mean at the storage level?
- [ ] Command naming: `remove`, `rm`, or both?
- [ ] Which tasks can be removed? Any status restrictions?
- [ ] How should dependency references be handled on removal?
- [ ] How should parent/child relationships be handled on removal?
- [ ] Should bulk removal be supported?

---

## What does "remove" mean at the storage level?

### Context

The JSONL file is rewritten atomically on every mutation via `Store.Mutate()`. A "remove" could mean either filtering the task out of the slice entirely (true deletion) or introducing a new status like `removed`/`deleted`. This is foundational — every other question depends on it.

### Options Considered

**Option A: True deletion — filter task from the slice**
- Task is completely gone from JSONL and cache
- No history preserved
- Clean and simple — consistent with what "remove" implies
- Mechanically trivial: return slice without the task from Mutate callback

**Option B: New status (e.g., `removed` or `deleted`)**
- Task remains in JSONL with a terminal status
- Preserves audit trail
- But: how is this meaningfully different from `cancelled`?
- Adds complexity: need to filter out of all queries, handle in ready logic, etc.

---

## Command naming: `remove`, `rm`, or both?

### Context

Need a command name. Existing commands: `init`, `create`, `list`, `show`, `update`, `start`, `done`, `cancel`, `reopen`, `ready`, `blocked`, `dep`, `stats`, `rebuild`, `doctor`, `migrate`, `help`.

---

## Which tasks can be removed? Any status restrictions?

### Context

Should removal be allowed from any status, or only certain ones? Cancellation is valid from `open` or `in_progress`. Should a `done` task be removable? What about already-cancelled tasks?

---

## How should dependency references be handled on removal?

### Context

Tasks reference other tasks via `BlockedBy []string`. If task A is in task B's BlockedBy and we remove task A, B's BlockedBy now contains an orphaned ID. Currently, cancelling a task does NOT clean up BlockedBy arrays — the query logic treats cancelled/done as "resolved." But a removed task won't exist at all.

---

## How should parent/child relationships be handled on removal?

### Context

Tasks have a `Parent string` field for hierarchy. Removing a parent could orphan children. Removing a child changes the parent's leaf-task status.

---

## Should bulk removal be supported?

### Context

No existing bulk status transitions exist. Each `cancel`, `done`, `start` etc. operates on a single task. Dependencies can be specified as comma-separated lists in create/update, but that's it.

---

## Summary

### Key Insights
*(To be filled as discussion progresses)*

### Current State
- Questions identified, discussion beginning

### Next Steps
- Work through each question with the user
