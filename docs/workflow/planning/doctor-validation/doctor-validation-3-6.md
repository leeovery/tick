---
id: doctor-validation-3-6
phase: 3
status: pending
created: 2026-01-31
---

# Parent Done With Open Children Warning

## Goal

Tasks 3-1 through 3-5 cover all nine error-severity relationship checks, but none detect a suspicious-but-allowed state: a parent task marked `done` while one or more of its children are still open (status `open` or `in_progress`). The specification defines this as Warning #1: "Parent marked done while children still open — allowed but suspicious." This is the **only warning-severity check** in the entire doctor suite. Unlike all error checks, this finding uses `SeverityWarning` and does not affect exit code — if this check produces warnings and no other check produces errors, `tick doctor` exits 0.

The tick-core specification allows this state (it is not a hard constraint violation), but it is worth surfacing because it often indicates a parent was prematurely completed or children were created after the parent was closed. Doctor reports it so humans can investigate.

## Implementation

- Create a `ParentDoneWithOpenChildrenCheck` struct that implements the `Check` interface (from task 1-1). It needs access to the `.tick` directory path.

- Implement the `Run` method with the following logic:
  1. Call `ParseTaskRelationships(tickDir)` to get the task data. If the parser returns a file-not-found error, return a single failing `CheckResult` with Name `"Parent done with open children"`, Severity `SeverityError` (not warning — a missing file is a genuine error preventing the check from running), Details `"tasks.jsonl not found"`, and Suggestion `"Run tick init or verify .tick directory"`. This is consistent with the pattern established in tasks 3-1 through 3-5.
  2. Build two data structures from the returned slice:
     - A **status map**: `map[string]string` mapping each task's `ID` to its `Status`.
     - A **children map**: `map[string][]string` mapping each parent ID to a list of child IDs. Built by iterating all tasks: if a task has a non-empty `Parent`, append the task's `ID` to `children[task.Parent]`.
  3. Iterate the children map. For each parent ID that has children:
     - Look up the parent's status in the status map. If the parent ID is not found in the status map (the parent task doesn't exist as a parseable record), skip it — the orphaned parent check (task 3-1) handles that case. This check only examines parents that exist in the file.
     - If the parent's status is `"done"`, iterate its children. For each child:
       - Look up the child's status in the status map. If the child's status is `"open"` or `"in_progress"` (i.e., NOT `"done"` and NOT `"cancelled"`), record a failing `CheckResult` with:
         - Name: `"Parent done with open children"`
         - Passed: `false`
         - Severity: `SeverityWarning`
         - Details: `"tick-{parent-id} is done but has open child tick-{child-id}"` (e.g., `"tick-e1f2a3 is done but has open child tick-a1b2c3"`)
         - Suggestion: `"Review whether parent was completed prematurely"`
  4. After checking all parent-child relationships, if no warnings were found, return a single passing `CheckResult` with Name `"Parent done with open children"` and Passed `true`.
  5. If warnings were found, return all the failing `CheckResult` entries (one per parent-child pair where parent is done and child is open). Do not include a passing result alongside failures.

- **"Open" means NOT done and NOT cancelled.** The tick-core statuses are: `open`, `in_progress`, `done`, `cancelled`. A child is considered "open" (in the colloquial sense used by this check) if its status is `open` or `in_progress`. Children with status `done` or `cancelled` are not flagged.

- **Only `done` parents are flagged.** A parent with status `open`, `in_progress`, or `cancelled` is not flagged by this check, even if it has open children. Only `done` parents are suspicious — the concern is that a parent was marked complete while work remains.

- **Each parent-child pair produces its own warning.** If parent P is `done` and has children C1 (`open`), C2 (`in_progress`), and C3 (`done`), two warnings are produced: one for P+C1 and one for P+C2. C3 is not flagged because its status is `done`.

- **Missing file is SeverityError, not SeverityWarning.** The file-not-found condition prevents the check from running at all — it is a genuine error, not a warning. Only the actual "parent done with open children" finding uses `SeverityWarning`.

- The check does **not** normalize IDs before comparison. IDs in `tasks.jsonl` should already be lowercase per write-time normalization. Compare as-is, consistent with all other Phase 3 checks.

- The check is **read-only** — it never modifies `tasks.jsonl`.

- The formatter (task 1-2) is responsible for rendering the appropriate output marker (`⚠` or similar) based on `SeverityWarning`. This check just sets the severity correctly; it does not control display.

## Tests

- `"it returns passing result when no parent is done with open children"`
- `"it returns passing result for empty file (zero bytes)"`
- `"it returns passing result when parent is done and all children are done"`
- `"it returns passing result when parent is done and all children are cancelled"`
- `"it returns passing result when parent is done with mix of done and cancelled children (no open)"`
- `"it returns passing result when parent is open with open children (not flagged)"`
- `"it returns passing result when parent is in_progress with open children (not flagged)"`
- `"it returns passing result when parent is cancelled with open children (not flagged)"`
- `"it returns warning result when parent is done with one open child (status open)"`
- `"it returns warning result when parent is done with one in_progress child"`
- `"it returns one warning per open child when parent is done with multiple open children"`
- `"it returns warnings only for open children — done and cancelled children of same parent not flagged"`
- `"it returns warnings across multiple parents — each done parent with open children produces own warnings"`
- `"it uses SeverityWarning for parent-done-with-open-children findings"`
- `"it uses SeverityError (not SeverityWarning) when tasks.jsonl does not exist"`
- `"it includes parent ID and child ID in warning details"`
- `"it follows wording 'tick-{parent} is done but has open child tick-{child}' in details"`
- `"it suggests 'Review whether parent was completed prematurely' for warnings"`
- `"it uses CheckResult Name 'Parent done with open children' for all results"`
- `"it skips parent IDs that do not exist as parseable tasks (orphaned parent — handled by task 3-1)"`
- `"it skips unparseable lines — does not report them in parent-child analysis"`
- `"it returns failing result when tasks.jsonl does not exist"`
- `"it does not normalize IDs before comparison (compares as-is)"`
- `"it does not modify tasks.jsonl (read-only verification)"`
- `"warnings-only report produces exit code 0 when combined with DiagnosticRunner (integration)"`

## Edge Cases

- **Missing `tasks.jsonl`**: The file does not exist in the `.tick/` directory. The parser returns an error, and the check translates it into a single failing `CheckResult` with `SeverityError` (not warning — a missing file prevents the check from running). Suggestion: `"Run tick init or verify .tick directory"`. Consistent with all other Phase 3 checks.

- **Empty file (zero bytes)**: The file exists but has no content. The parser returns an empty slice. No tasks means no parent-child relationships to analyze. The check returns a single passing result.

- **Parent done with one open child**: A single parent task with status `done` has one child with status `open`. One warning is produced identifying both the parent and child IDs. This is the simplest positive case.

- **Parent done with multiple open children**: Parent P is `done` and has children C1 (`open`), C2 (`in_progress`), C3 (`done`). Two warnings produced — one for P+C1 (status `open`) and one for P+C2 (status `in_progress`). C3 is not flagged because it is `done`. Each warning is a separate `CheckResult` entry with its own details identifying the specific child.

- **Parent done with all children done**: Parent P is `done` and all its children are also `done`. No warnings produced — this is the normal completion state. The check returns a passing result (assuming no other parents trigger warnings).

- **Parent done with cancelled children only**: Parent P is `done` and all its children are `cancelled`. No warnings produced — cancelled children are not "open." Cancelling children before completing a parent is a valid workflow.

- **Parent open with open children (not flagged)**: Parent P has status `open` and has children with status `open`. This is perfectly normal — the parent is not done, so there is nothing suspicious. The check does not flag this. Only `done` parents with open children trigger a warning.

- **Parent done with in_progress child**: Parent P is `done` and child C has status `in_progress`. This is flagged — `in_progress` is an "open" status (not `done`, not `cancelled`). It is arguably more suspicious than a merely `open` child because someone is actively working on a child of a completed parent.

- **Warnings-only produces exit code 0**: If this check produces warnings and all other checks pass (no errors from any check), the `DiagnosticRunner`'s report has `HasErrors() == false` and the exit code function returns 0. Warnings appear in the output (with `⚠` or appropriate marker from the formatter) and count in the summary total, but do not cause a non-zero exit. This is a critical behavioral distinction from all error checks and must be validated at the integration level (task 3-7 registration test).

- **Parent ID exists in children map but not in status map**: If a task references a parent that does not exist as a parseable record in the file, the children map will have an entry for that parent ID, but the status map will not. The check skips this parent — the orphaned parent check (task 3-1) is responsible for flagging non-existent parent references. This check only analyzes parents that are themselves present in the data.

## Acceptance Criteria

- [ ] `ParentDoneWithOpenChildrenCheck` implements the `Check` interface
- [ ] Check reuses `ParseTaskRelationships` from task 3-1
- [ ] Passing check returns `CheckResult` with Name `"Parent done with open children"` and Passed `true`
- [ ] Each done parent + open child pair produces its own failing `CheckResult` with `SeverityWarning`
- [ ] Details follow wording: `"tick-{parent} is done but has open child tick-{child}"`
- [ ] Suggestion is `"Review whether parent was completed prematurely"` for warning results
- [ ] "Open" child means status `open` or `in_progress` (not `done`, not `cancelled`)
- [ ] Only parents with status `done` are flagged (not `open`, `in_progress`, or `cancelled`)
- [ ] Parent done with all children `done` produces no warnings
- [ ] Parent done with all children `cancelled` produces no warnings
- [ ] Parent done with mix of done/cancelled/open children produces warnings only for open children
- [ ] Parent not done with open children produces no warnings
- [ ] Missing `tasks.jsonl` returns `SeverityError` failure with init suggestion (not SeverityWarning)
- [ ] Parent IDs not found in status map (non-existent parent tasks) are skipped
- [ ] Unparseable lines skipped by parser — not included in analysis
- [ ] IDs compared as-is (no normalization)
- [ ] Check is read-only — never modifies `tasks.jsonl`
- [ ] Tests written and passing for all edge cases

## Context

The specification defines this as Warning #1: "Parent marked done while children still open — allowed but suspicious." It is the only warning in the entire doctor validation suite. All other checks (Error #1 through Error #9) use `SeverityError`.

The specification's exit code table states: exit code 0 means "All checks passed (no errors, warnings allowed)." This explicitly permits warnings without affecting the exit code. The `DiagnosticReport.HasErrors()` method (task 1-1) counts only error-severity failures, so warnings are excluded from exit code determination.

The specification's fix suggestion table covers errors only ("All other errors" maps to "Manual fix required"). Since this is a warning (suspicious but allowed), the suggestion should be informational rather than prescriptive. The suggestion `"Review whether parent was completed prematurely"` conveys that this warrants investigation without implying something is broken.

The tick-core specification defines four task statuses: `open`, `in_progress`, `done`, `cancelled`. For this check, "open" children are those with status `open` or `in_progress` — any status that represents incomplete work. Children with status `done` or `cancelled` are considered resolved and do not trigger warnings.

Parent-child relationships are determined by the `parent` field: a task is a child of another task if its `parent` field contains that task's ID. A task is a parent if at least one other task references it as `parent`. The `ParseTaskRelationships` function (task 3-1) provides the `Parent` and `Status` fields needed for this analysis.

The Phase 3 acceptance criteria states: "Warning: parent done with open children reported with ⚠ or appropriate marker." The formatter (task 1-2) is responsible for rendering the visual marker based on severity. This check's responsibility is to set `SeverityWarning` correctly on the `CheckResult`; the display representation is not this check's concern.

This is a Go project. The check implements the `Check` interface defined in task 1-1, reuses `ParseTaskRelationships` from task 3-1, and will be registered with the `DiagnosticRunner` in task 3-7.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
