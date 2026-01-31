---
id: doctor-validation-3-2
phase: 3
status: pending
created: 2026-01-31
---

# Orphaned Dependency Reference Check

## Goal

Task 3-1 built the shared `ParseTaskRelationships` parser and the orphaned parent check. The parser extracts `BlockedBy` (a slice of dependency IDs) from each task, but no check yet validates that those dependency IDs actually reference existing tasks. If a task's `blocked_by` array contains an ID that does not exist in `tasks.jsonl` — due to task deletion, manual editing, or data corruption — the task is blocked on a phantom dependency that can never be resolved. This task implements the `OrphanedDependencyCheck` that detects and individually reports each reference to a non-existent task in any task's `blocked_by` array. This is specification Error #6: "Task depends on non-existent task."

## Implementation

- Create an `OrphanedDependencyCheck` struct that implements the `Check` interface (from task 1-1). It needs access to the `.tick` directory path.

- Implement the `Run` method with the following logic:
  1. Call `ParseTaskRelationships(tickDir)` to get the task data. If the parser returns a file-not-found error, return a single failing `CheckResult` with Name `"Orphaned dependencies"`, Severity `SeverityError`, Details `"tasks.jsonl not found"`, and Suggestion `"Run tick init or verify .tick directory"`. This is consistent with the Phase 2 and task 3-1 pattern for missing files.
  2. Build a set of all known task IDs from the returned data (iterate the slice, add each `ID` to a `map[string]bool`).
  3. Iterate the task data. For each task, iterate its `BlockedBy` slice. For each dependency ID in `BlockedBy`:
     - Check if the dependency ID exists in the known-ID set.
     - If not found, record a failing `CheckResult` with Name `"Orphaned dependencies"`, Severity `SeverityError`, Details following the pattern: `"tick-{task-id} depends on non-existent task tick-{dep-id}"` (e.g., `"tick-a1b2c3 depends on non-existent task tick-missing"`), and Suggestion `"Manual fix required"`.
  4. After checking all tasks and all their dependencies, if no orphaned dependencies were found, return a single passing `CheckResult` with Name `"Orphaned dependencies"` and Passed `true`.
  5. If orphaned dependencies were found, return all the failing `CheckResult` entries (one per orphaned dependency reference). Do not include a passing result alongside failures.

- Each orphaned dependency reference is reported individually. If task `tick-aaa111` has `blocked_by: ["tick-missing1", "tick-missing2"]`, that produces two separate failing results. If task `tick-bbb222` also references `tick-missing1`, that produces a third failing result. The error is per-reference, not per-missing-ID or per-task.

- The check name `"Orphaned dependencies"` is the check category label, consistent with the naming pattern from prior checks (`"Orphaned parents"`, `"JSONL syntax"`, `"ID format"`, `"ID uniqueness"`). The Details field carries the specific task and dependency IDs.

- The check does **not** normalize IDs to lowercase before comparison. IDs in `tasks.jsonl` should already be lowercase (write-time normalization). The orphaned dependency check compares IDs as-is, consistent with the orphaned parent check (task 3-1).

- Tasks with `blocked_by` set to `null`, absent, or an empty array are valid — they simply have no dependencies. The parser (from task 3-1) represents these as an empty slice for `BlockedBy`. The check skips tasks with empty `BlockedBy` — there are no references to validate.

- A `blocked_by` entry that references a valid existing task is not flagged, even if that reference might also be caught by other checks (e.g., self-reference caught by task 3-3, or cycle caught by task 3-4). This check is solely concerned with whether the referenced ID exists.

## Tests

- `"it returns passing result when all blocked_by references are valid"`
- `"it returns passing result for empty file (zero bytes)"`
- `"it returns passing result when no tasks have blocked_by entries (all empty arrays)"`
- `"it returns failing result when a task references a non-existent dependency"`
- `"it returns one failing result per orphaned dependency when a single task has multiple orphaned deps"`
- `"it returns one failing result per orphaned dependency across multiple tasks"`
- `"it includes task ID and missing dependency ID in error details"`
- `"it follows pattern: 'tick-{task} depends on non-existent task tick-{dep}'"`
- `"it treats empty blocked_by array as valid (no references to check)"`
- `"it treats null blocked_by as valid (parser returns empty slice)"`
- `"it treats absent blocked_by field as valid (parser returns empty slice)"`
- `"it reports each invalid ref individually when blocked_by contains mix of valid and invalid refs"`
- `"it does not flag valid references in a mixed blocked_by array"`
- `"it skips unparseable lines — does not report them as orphaned dependencies"`
- `"it returns failing result when tasks.jsonl does not exist"`
- `"it suggests 'Manual fix required' for orphaned dependency errors"`
- `"it uses CheckResult Name 'Orphaned dependencies' for all results"`
- `"it uses SeverityError for all failure cases"`
- `"it does not normalize IDs before comparison (compares as-is)"`
- `"it does not modify tasks.jsonl (read-only verification)"`

## Edge Cases

- **Missing `tasks.jsonl`**: The file does not exist in the `.tick/` directory. The parser returns an error, and the check translates it into a single failing `CheckResult` with a suggestion to initialize. This is consistent with the Phase 2 pattern and task 3-1.

- **Empty file (zero bytes)**: The file exists but has no content. The parser returns an empty slice. No tasks means no `blocked_by` references to validate. The check returns a single passing result.

- **Single orphaned dependency**: One task has one `blocked_by` entry referencing an ID that does not exist. Produces exactly one failing result with the task ID and the missing dependency ID in the details.

- **Multiple orphaned deps on same task**: A task has `blocked_by: ["tick-missing1", "tick-missing2"]` where both IDs do not exist. Produces two separate failing results, one for each orphaned reference. Both results identify the same task ID but different missing dependency IDs.

- **Multiple orphaned deps across tasks**: Different tasks each reference non-existent dependencies. Each orphaned reference produces its own failing result. For example, `tick-aaa` depending on `tick-missing` and `tick-bbb` depending on `tick-gone` produces two results.

- **All deps valid**: Every task's `blocked_by` entries reference IDs that exist in the file. The check returns a single passing result. This is the happy path confirming no false positives.

- **Empty `blocked_by` array**: A task has `"blocked_by": []`. The parser returns an empty slice for `BlockedBy`. The check has no references to validate for this task and simply skips it. This is a valid state — the task has no dependencies.

- **Unparseable lines skipped**: Lines that are not valid JSON are skipped by the parser (per task 3-1). They do not contribute to the known-ID set, and their `blocked_by` entries (if any) are not examined. If a task's `blocked_by` references an ID that was on an unparseable line, that dependency IS missing from the known-ID set, and the reference IS reported as orphaned. This is correct — from the perspective of parseable data, the dependency target does not exist.

- **`blocked_by` contains mix of valid and invalid refs**: A task has `blocked_by: ["tick-exists", "tick-missing"]` where `tick-exists` is a valid task but `tick-missing` is not. Only `tick-missing` produces a failing result. `tick-exists` is silently accepted. Each entry in the array is checked independently.

## Acceptance Criteria

- [ ] `OrphanedDependencyCheck` implements the `Check` interface
- [ ] Check reuses `ParseTaskRelationships` from task 3-1 (no duplicate file parsing)
- [ ] Passing check returns `CheckResult` with Name `"Orphaned dependencies"` and Passed `true`
- [ ] Each orphaned dependency reference produces its own failing `CheckResult` with task ID and missing dep ID in details
- [ ] Details follow pattern: `"tick-{task} depends on non-existent task tick-{dep}"`
- [ ] Mixed valid/invalid refs in same `blocked_by` array: only invalid refs produce failures
- [ ] Null, absent, or empty `blocked_by` treated as valid (no references to check)
- [ ] Unparseable lines skipped by parser — not flagged as orphaned dependencies
- [ ] Missing `tasks.jsonl` returns error-severity failure with init suggestion
- [ ] Suggestion is `"Manual fix required"` for orphaned dependency errors
- [ ] All failures use `SeverityError`
- [ ] Check is read-only — never modifies `tasks.jsonl`
- [ ] Tests written and passing for all edge cases

## Context

The specification defines orphaned dependency references as Error #6: "Task depends on non-existent task." Unlike orphaned parent references (Error #5), the tick-core specification does not provide an explicit doctor message format for orphaned dependencies. The details pattern `"tick-{task} depends on non-existent task tick-{dep}"` is derived from the specification's description ("Task depends on non-existent task") and follows the same structure as the orphaned parent message.

The tick-core specification says `blocked_by` is: type `array`, required `No`, default `[]`. The JSONL format says "Optional fields omitted when empty/null (not serialized as null)." So a task with no dependencies will either have `"blocked_by": []`, `"blocked_by": null`, or no `blocked_by` key at all. All three representations mean "no dependencies" and the parser (from task 3-1) normalizes all of them to an empty slice.

The tick-core specification states that `blocked_by` "Must reference existing task IDs" and that validation happens at write time. Doctor catches cases where this invariant has been violated — perhaps through manual file edits, concurrent modifications, or task deletion without dependency cleanup.

The specification requires doctor to report each error individually: "Doctor lists each error individually. If there are 5 orphaned references, all 5 are shown with their specific details." This applies to dependency references just as it does to parent references — each invalid `blocked_by` entry is a separate error.

This check is solely concerned with existence. A `blocked_by` entry that references a valid ID is accepted by this check even if it would be caught by other checks (self-reference by task 3-3, cycle by task 3-4, child-blocked-by-parent by task 3-5). Each check validates one specific constraint independently.

This is a Go project. The check implements the `Check` interface defined in task 1-1, reuses the `ParseTaskRelationships` parser from task 3-1, and will be registered with the `DiagnosticRunner` in task 3-7.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
