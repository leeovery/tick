---
id: doctor-validation-3-3
phase: 3
status: pending
created: 2026-01-31
---

# Self-Referential Dependency Check

## Goal

After Task 3-1 established the shared `ParseTaskRelationships` parser and Task 3-2 catches orphaned dependency references (tasks depending on non-existent tasks), there is still no check for a task that depends on itself. A self-referential dependency is a degenerate cycle: the task can never become ready because it is waiting on its own completion. This is specification Error #7: "Task depends on itself." Each self-referential dependency must be reported individually per the specification's rule that "Doctor lists each error individually."

## Implementation

- Create a `SelfReferentialDepCheck` struct that implements the `Check` interface (from task 1-1). It needs access to the `.tick` directory path.

- Implement the `Run` method with the following logic:
  1. Call `ParseTaskRelationships(tickDir)` to get the task data. If the parser returns a file-not-found error, return a single failing `CheckResult` with Name `"Self-referential dependencies"`, Severity `SeverityError`, Details `"tasks.jsonl not found"`, and Suggestion `"Run tick init or verify .tick directory"`. This is consistent with the pattern established in tasks 3-1 and 3-2.
  2. Iterate the task data. For each task, iterate its `BlockedBy` slice. If any entry in `BlockedBy` equals the task's own `ID`, record a failing `CheckResult` with:
     - Name: `"Self-referential dependencies"`
     - Severity: `SeverityError`
     - Details: `"tick-{id} depends on itself"` (e.g., `"tick-a1b2c3 depends on itself"`)
     - Suggestion: `"Manual fix required"`
  3. A task may list itself multiple times in `blocked_by`. If `blocked_by: ["tick-abc123", "tick-abc123"]` for task `tick-abc123`, report one error per self-referential task, not per duplicate entry. The specification says "Task depends on itself" (singular condition per task).
  4. After checking all tasks, if no self-referential dependencies were found, return a single passing `CheckResult` with Name `"Self-referential dependencies"` and Passed `true`.
  5. If self-referential dependencies were found, return all the failing `CheckResult` entries (one per self-referential task). Do not include a passing result alongside failures.

- The check does **not** normalize IDs before comparison. IDs in `tasks.jsonl` should already be lowercase per write-time normalization. Compare as-is, consistent with tasks 3-1 and 3-2.

- This check only detects self-references (`A depends on A`). Multi-task cycles (`A→B→C→A`) are handled by the cycle detection check (task 3-4). Task 3-4's edge cases note "self-reference not double-reported (handled by task 3-3)", confirming that 3-3 owns self-referential detection exclusively.

- A task with an empty `BlockedBy` slice (no dependencies) is trivially not self-referential. Skip it.

- A task that lists itself among other valid dependencies (e.g., `blocked_by: ["tick-other1", "tick-abc123", "tick-other2"]` for task `tick-abc123`) is still self-referential. The presence of valid deps alongside the self-reference does not excuse it.

## Tests

- `"it returns passing result when no tasks have self-referential dependencies"`
- `"it returns passing result for empty file (zero bytes)"`
- `"it returns passing result when tasks have dependencies but none are self-referential"`
- `"it returns failing result when a task lists itself in blocked_by"`
- `"it detects self-reference among other valid dependencies in blocked_by"`
- `"it returns one failing result per self-referential task when multiple tasks each reference themselves"`
- `"it includes the self-referential task ID in error details"`
- `"it follows wording 'tick-{id} depends on itself' in details"`
- `"it returns failing result when a task's blocked_by contains only itself"`
- `"it skips unparseable lines — does not report them as self-referential"`
- `"it returns failing result when tasks.jsonl does not exist"`
- `"it suggests 'Manual fix required' for self-referential dependency errors"`
- `"it uses CheckResult Name 'Self-referential dependencies' for all results"`
- `"it uses SeverityError for all failure cases"`
- `"it skips tasks with empty blocked_by (no dependencies)"`
- `"it does not normalize IDs before comparison (compares as-is)"`
- `"it does not modify tasks.jsonl (read-only verification)"`

## Edge Cases

- **Missing `tasks.jsonl`**: The file does not exist in the `.tick/` directory. The parser returns an error, and the check translates it into a single failing `CheckResult` with a suggestion to initialize. Consistent with the Phase 2 and Phase 3 pattern.

- **Empty file (zero bytes)**: The file exists but has no content. The parser returns an empty slice. No tasks means no dependencies to check. The check returns a single passing result.

- **Task blocked by itself among other valid deps**: A task like `{"id": "tick-abc123", "blocked_by": ["tick-def456", "tick-abc123", "tick-ghi789"]}` has a self-reference buried among valid dependencies. The check must still detect and report it. The other dependencies are irrelevant to the self-reference detection — the check iterates all entries in `blocked_by`.

- **Multiple tasks each self-referential**: If `tick-aaa111` depends on itself and `tick-bbb222` depends on itself, two separate failing results are produced — one per self-referential task. Each result includes the specific task ID in its details.

- **Task with only self-reference**: A task like `{"id": "tick-abc123", "blocked_by": ["tick-abc123"]}` whose only dependency is itself. Reported as a single failing result. This is the most direct form of the error.

- **Unparseable lines skipped**: Lines that are not valid JSON are skipped by the shared parser. They do not contribute to the check results. If a task's `blocked_by` references an ID on an unparseable line, the orphaned dependency check (task 3-2) catches that — this check only cares about self-references where the task's own ID matches an entry in its own `blocked_by`.

- **Tasks with no blocked_by field or empty array**: Perfectly valid. The parser returns an empty `BlockedBy` slice for these. The check skips them — no entries to compare against the task's own ID.

- **Duplicate self-references in blocked_by**: If a task lists itself twice in `blocked_by` (e.g., `["tick-abc123", "tick-abc123"]` for task `tick-abc123`), report one error for the task, not two. The condition is "task depends on itself" — a per-task finding, not per-entry.

## Acceptance Criteria

- [ ] `SelfReferentialDepCheck` implements the `Check` interface
- [ ] Check reuses `ParseTaskRelationships` from task 3-1
- [ ] Passing check returns `CheckResult` with Name `"Self-referential dependencies"` and Passed `true`
- [ ] Each self-referential task produces its own failing `CheckResult` with the task ID in details
- [ ] Details follow wording: `"tick-{id} depends on itself"`
- [ ] Self-reference detected even when mixed with other valid dependencies in `blocked_by`
- [ ] Multiple self-referential tasks each produce separate failing results
- [ ] Tasks with empty or absent `blocked_by` are not flagged
- [ ] Duplicate self-references in the same task's `blocked_by` produce one error (per-task, not per-entry)
- [ ] Unparseable lines skipped by parser — not flagged as self-referential
- [ ] Missing `tasks.jsonl` returns error-severity failure with init suggestion
- [ ] Suggestion is `"Manual fix required"` for self-referential dependency errors
- [ ] All failures use `SeverityError`
- [ ] Check is read-only — never modifies `tasks.jsonl`
- [ ] Tests written and passing for all edge cases

## Context

The specification defines self-referential dependencies as Error #7: "Task depends on itself." The fix suggestion table maps this to "Manual fix required" (under "All other errors").

The tick-core specification describes the `blocked_by` field as: type `array`, required `No`, default `[]`. It contains task IDs that must be completed before the task becomes ready. The tick-core dependency model says: "A task's `blocked_by` array lists task IDs that must reach `done` (or `cancelled`) before the task becomes `ready`." A self-reference creates a deadlock — the task waits for itself to complete, which can never happen.

The specification requires doctor to report each error individually: "Doctor lists each error individually. If there are 5 orphaned references, all 5 are shown with their specific details." Applied to this check: if 3 tasks are self-referential, 3 errors are reported.

Task 3-4 (dependency cycle detection) explicitly excludes self-references from its scope per its edge case note: "self-reference not double-reported (handled by task 3-3)." This ensures clean separation — 3-3 owns all self-referential detection, 3-4 handles multi-task cycles only.

This check reuses the `ParseTaskRelationships` function built in task 3-1. It follows the same patterns established across Phase 2 and Phase 3: missing file returns error with init suggestion, empty file returns passing result, unparseable lines skipped, one failing `CheckResult` per individual finding, read-only file access.

This is a Go project. The check implements the `Check` interface defined in task 1-1 and will be registered with the `DiagnosticRunner` in task 3-7.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
