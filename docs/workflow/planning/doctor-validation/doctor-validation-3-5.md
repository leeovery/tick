---
id: doctor-validation-3-5
phase: 3
status: pending
created: 2026-01-31
---

# Child Blocked-By Parent Check

## Goal

Tasks 3-1 through 3-3 validate orphaned references and self-referential dependencies, and task 3-4 detects multi-task dependency cycles. None of these checks catch a subtler deadlock: a child task that lists its own parent in `blocked_by`. The tick-core specification explicitly forbids this because the leaf-only ready rule creates an unresolvable deadlock — a parent with open children never appears in `ready`, so the parent can never complete, so the child's `blocked_by` is never satisfied, so the child can never become ready either. This task implements the `ChildBlockedByParentCheck` that detects direct child-blocked-by-parent relationships. This is specification Error #9: "Child blocked_by parent — Deadlock condition — child can never become ready."

Only **direct** parent-child relationships are flagged. If a child is blocked by its grandparent (or any non-direct ancestor), this check does not flag it — the specification and tick-core rules define the constraint as "Child blocked_by parent," meaning the task's own `parent` field value appearing in its own `blocked_by` array. Indirect ancestor relationships are not checked.

## Implementation

- Create a `ChildBlockedByParentCheck` struct that implements the `Check` interface (from task 1-1). It needs access to the `.tick` directory path.

- Implement the `Run` method with the following logic:
  1. Call `ParseTaskRelationships(tickDir)` to get the task data. If the parser returns a file-not-found error, return a single failing `CheckResult` with Name `"Child blocked by parent"`, Severity `SeverityError`, Details `"tasks.jsonl not found"`, and Suggestion `"Run tick init or verify .tick directory"`. This is consistent with the pattern established in tasks 3-1 through 3-3.
  2. Iterate the task data. For each task where `Parent` is non-empty AND `BlockedBy` is non-empty:
     - Check if the task's `Parent` value appears in its `BlockedBy` slice.
     - If found, record a failing `CheckResult` with:
       - Name: `"Child blocked by parent"`
       - Severity: `SeverityError`
       - Details: `"tick-{child-id} is blocked by its parent tick-{parent-id}"` (e.g., `"tick-a1b2c3 is blocked by its parent tick-e5f6g7"`)
       - Suggestion: `"Manual fix required — child blocked by parent creates deadlock with leaf-only ready rule"`
  3. After checking all tasks, if no child-blocked-by-parent relationships were found, return a single passing `CheckResult` with Name `"Child blocked by parent"` and Passed `true`.
  4. If violations were found, return all the failing `CheckResult` entries (one per child task that has its parent in `blocked_by`). Do not include a passing result alongside failures.

- The detection logic is a simple per-task check: does this task's `Parent` appear in its `BlockedBy` slice? This requires no graph traversal and no lookup map — just a string comparison within each task's own data. The only data needed from other tasks is nothing; this is entirely local to each task record.

- **Only direct parent**: If task C has `parent: "tick-B"` and `blocked_by: ["tick-A"]`, and task B has `parent: "tick-A"`, the check does NOT flag task C even though tick-A is its grandparent. The check only compares each task's `Parent` field against its own `BlockedBy` entries. Grandparent/ancestor traversal is not performed.

- **One error per child**: If a task lists its parent in `blocked_by` (possibly alongside other dependencies), that produces exactly one failing result for that task. Even if the parent ID appears multiple times in `blocked_by` due to data corruption, report one error per child task — the condition is "child is blocked by its parent," not "child has N blocked_by entries matching parent."

- **Multiple children blocked by same parent**: If tasks `tick-aaa` and `tick-bbb` both have `parent: "tick-ppp"` and both have `"tick-ppp"` in their `blocked_by`, two separate failing results are produced — one per child. Each result identifies the specific child and parent IDs.

- The check does **not** normalize IDs before comparison. IDs in `tasks.jsonl` should already be lowercase per write-time normalization. Compare as-is, consistent with tasks 3-1 through 3-3.

- Tasks with no `parent` (empty string from the parser) are root tasks and cannot be blocked by a parent they don't have. Skip them.

- Tasks with no `blocked_by` (empty slice from the parser) have no dependencies at all and trivially cannot be blocked by their parent. Skip them.

- A task that has a `parent` but whose `blocked_by` contains only other valid task IDs (not the parent) is perfectly valid. The check does not flag it.

## Tests

- `"it returns passing result when no child tasks are blocked by their parent"`
- `"it returns passing result for empty file (zero bytes)"`
- `"it returns passing result when tasks have parents and blocked_by but no overlap"`
- `"it returns passing result when task has parent but empty blocked_by (no dependencies)"`
- `"it returns passing result when task has blocked_by but no parent (root task)"`
- `"it returns failing result when a child has its direct parent in blocked_by"`
- `"it returns failing result when child is blocked by parent among other valid deps"`
- `"it returns one failing result per child when multiple children are blocked by the same parent"`
- `"it does not flag child blocked by grandparent (only direct parent checked)"`
- `"it includes child ID and parent ID in error details"`
- `"it follows wording 'tick-{child} is blocked by its parent tick-{parent}' in details"`
- `"it reports one error per child even if parent appears multiple times in blocked_by"`
- `"it skips unparseable lines — does not report them as child-blocked-by-parent"`
- `"it returns failing result when tasks.jsonl does not exist"`
- `"it suggests fix mentioning deadlock with leaf-only ready rule"`
- `"it uses CheckResult Name 'Child blocked by parent' for all results"`
- `"it uses SeverityError for all failure cases"`
- `"it does not normalize IDs before comparison (compares as-is)"`
- `"it does not modify tasks.jsonl (read-only verification)"`

## Edge Cases

- **Missing `tasks.jsonl`**: The file does not exist in the `.tick/` directory. The parser returns an error, and the check translates it into a single failing `CheckResult` with a suggestion to initialize. Consistent with the Phase 2 and Phase 3 pattern.

- **Empty file (zero bytes)**: The file exists but has no content. The parser returns an empty slice. No tasks means no parent-child relationships to validate. The check returns a single passing result.

- **Direct child blocked by parent**: A task like `{"id": "tick-child1", "parent": "tick-epic1", "blocked_by": ["tick-epic1"]}` has its own parent in `blocked_by`. This creates a deadlock: the parent cannot be ready while it has open children (leaf-only rule), and the child cannot be ready while the parent is not done (dependency rule). The check flags this with a single failing result identifying both the child and parent IDs.

- **Child blocked by grandparent (not flagged)**: Task C has `parent: "tick-B"` and `blocked_by: ["tick-A"]`. Task B has `parent: "tick-A"`. Although tick-A is C's grandparent, the check only examines C's own `Parent` field (`tick-B`) against C's `BlockedBy` entries. Since `tick-B` is not in C's `blocked_by`, no error is reported for C. The specification constraint is specifically "Child blocked_by parent" — the immediate parent relationship — not "child blocked_by ancestor."

- **Multiple children blocked by same parent**: Tasks `tick-sub1` and `tick-sub2` both have `parent: "tick-epic"` and both include `"tick-epic"` in their `blocked_by`. Two separate failing results are produced — one per child. Each result identifies the specific child ID and the shared parent ID. The parent being the same does not collapse the errors.

- **Child blocked by parent among other valid deps**: A task like `{"id": "tick-child1", "parent": "tick-epic1", "blocked_by": ["tick-other1", "tick-epic1", "tick-other2"]}` has its parent among other valid dependencies. The check still detects it — the parent ID's presence in `blocked_by` is all that matters, regardless of other entries. One failing result is produced for this task.

- **Task with parent but no blocked_by (valid)**: A task like `{"id": "tick-child1", "parent": "tick-epic1"}` has a parent but no dependencies. The parser sets `BlockedBy` to an empty slice. The check skips it — there are no `blocked_by` entries to compare against the parent. This is a normal, valid task structure.

- **Duplicate parent in blocked_by**: A task like `{"id": "tick-child1", "parent": "tick-epic1", "blocked_by": ["tick-epic1", "tick-epic1"]}` lists its parent twice. Only one error is reported for this task — the condition is per-child, not per-entry.

- **Unparseable lines skipped**: Lines that are not valid JSON are skipped by the shared parser. They do not contribute to the check results. The relationship checks each task's own `Parent` and `BlockedBy` fields independently.

## Acceptance Criteria

- [ ] `ChildBlockedByParentCheck` implements the `Check` interface
- [ ] Check reuses `ParseTaskRelationships` from task 3-1
- [ ] Passing check returns `CheckResult` with Name `"Child blocked by parent"` and Passed `true`
- [ ] Each child blocked by its parent produces its own failing `CheckResult` with child ID and parent ID in details
- [ ] Details follow wording: `"tick-{child} is blocked by its parent tick-{parent}"`
- [ ] Only direct parent-child relationships flagged — grandparent/ancestor not checked
- [ ] Multiple children blocked by same parent each produce separate failing results
- [ ] Child blocked by parent detected even among other valid `blocked_by` entries
- [ ] Duplicate parent entries in `blocked_by` produce one error per child (not per entry)
- [ ] Tasks with no parent (root tasks) are not flagged
- [ ] Tasks with parent but empty `blocked_by` are not flagged
- [ ] Unparseable lines skipped by parser — not examined by this check
- [ ] Missing `tasks.jsonl` returns error-severity failure with init suggestion
- [ ] Suggestion mentions deadlock with leaf-only ready rule
- [ ] All failures use `SeverityError`
- [ ] Check is read-only — never modifies `tasks.jsonl`
- [ ] Tests written and passing for all edge cases

## Context

The specification defines this as Error #9: "Child blocked_by parent — Deadlock condition — child can never become ready." The fix suggestion table maps it to "Manual fix required" (under "All other errors").

The tick-core specification explicitly forbids this relationship in its Dependency Validation Rules table: "Child blocked_by parent — No — Creates deadlock with leaf-only rule." The write-time error message is: `"Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-epic (would create unworkable task due to leaf-only ready rule)"`. Doctor catches cases where this invariant has been violated despite write-time validation — perhaps through manual file edits, data corruption, or bugs in write-time validation.

The deadlock mechanism is: the leaf-only ready rule says a parent with open children never appears in `ready` (condition 3 of the ready query). So the parent cannot be worked on or completed while it has open children. But if a child is blocked by the parent, the child cannot become `ready` until the parent is `done`. The parent cannot become `done` while the child is open. Neither can make progress — a permanent deadlock.

The check only examines **direct** parent-child relationships (the task's own `Parent` field vs. its own `BlockedBy` entries). It does not traverse the hierarchy to find grandparent or ancestor relationships. This is consistent with the specification's wording ("Child blocked_by parent") and the tick-core validation rule which checks `blocked_by` against the task's immediate `parent` field.

The `blocked_by` field is: type `array`, required `No`, default `[]`. The `parent` field is: type `string`, required `No`, default `null`. The parser (from task 3-1) normalizes null/absent `parent` to empty string and null/absent `blocked_by` to empty slice.

This is a Go project. The check implements the `Check` interface defined in task 1-1, reuses the `ParseTaskRelationships` parser from task 3-1, and will be registered with the `DiagnosticRunner` in task 3-7.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
