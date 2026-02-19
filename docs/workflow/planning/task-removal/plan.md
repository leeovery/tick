---
topic: task-removal
status: planning
format: tick
ext_id: tick-132530
specification: ../specification/task-removal/specification.md
spec_commit: 739e6f02a7f56e6c4d785b981f74bf75b08ef36a
created: 2026-02-18
updated: 2026-02-19
external_dependencies: []
task_list_gate_mode: gated
author_gate_mode: gated
finding_gate_mode: gated
review_cycle: 1
planning:
  phase: 3
  task: 5
---

# Plan: Task Removal

### Phase 1: Walking Skeleton — Single Task Removal
status: approved
ext_id: tick-cf0a05
approved_at: 2026-02-19

**Goal**: Remove a single task by ID using `--force`, filtering it from JSONL via `Store.Mutate()`, with output through all three formatters and help text registration.

**Why this order**: This is the thinnest end-to-end slice threading through every layer the feature needs: CLI argument parsing, `--force` flag, command dispatch in `App.Run`, the `RunRemove` handler, a `Store.Mutate` callback that filters a task from the slice, a new `FormatRemoval` method on the `Formatter` interface with implementations in all three formatters, and help text. It establishes every pattern subsequent phases extend.

**Acceptance**:
- [ ] `tick remove <id> --force` removes a task from tasks.jsonl and SQLite cache
- [ ] Removed task no longer appears in `tick list` or `tick show <id>`
- [ ] `FormatRemoval` method exists on all three formatters (toon, pretty, JSON) and outputs the removed task's ID and title
- [ ] Error returned when the provided task ID does not exist
- [ ] Error returned when no arguments provided, with message: `"task ID is required. Usage: tick remove <id> [<id>...]"`
- [ ] `tick help remove` displays usage, flags, cascade behavior note, and Git recovery note
- [ ] `--quiet` flag suppresses removal output
- [ ] All existing tests continue to pass

#### Tasks
| ID | Name | Edge Cases | Status | Ext ID |
|----|------|------------|--------|--------|
| task-removal-1-1 | Add FormatRemoval to Formatter interface and all implementations | none | authored | tick-7314b0 |
| task-removal-1-2 | Implement RunRemove handler with --force flag and wire into App.Run | case-insensitive ID matching | authored | tick-64566b |
| task-removal-1-3 | Handle remove error cases | no-args message matches spec exactly | authored | tick-0607a0 |
| task-removal-1-4 | Register remove command help text | none | authored | tick-1777bc |

### Phase 2: Interactive Confirmation Prompt
status: approved
ext_id: tick-fca658
approved_at: 2026-02-19

**Goal**: Add the interactive confirmation gate when `--force` is not provided, reading user input from stdin and writing prompts/abort messages to stderr.

**Why this order**: The confirmation prompt is the primary safety mechanism for this destructive command. It must be in place before cascade deletion is added, because cascade amplifies the blast radius and the prompt must surface it. This phase depends on the working removal pathway from Phase 1.

**Acceptance**:
- [ ] Without `--force`, `tick remove <id>` prompts on stderr showing task ID and title with `[y/N]` convention
- [ ] Entering `y` or `yes` (case-insensitive) proceeds with removal and outputs result to stdout
- [ ] Any other input including empty Enter aborts with `"Aborted."` on stderr and exit code 1
- [ ] Prompt text and abort message are written to stderr, not stdout
- [ ] Stdin is injectable on `App` (e.g., `Stdin io.Reader` field) for test isolation
- [ ] `--force` continues to skip the prompt entirely (Phase 1 behavior preserved)

#### Tasks
| ID | Name | Edge Cases | Status | Ext ID |
|----|------|------------|--------|--------|
| task-removal-2-1 | Add Stdin to App and thread through remove handler | none | authored | tick-8bc489 |
| task-removal-2-2 | Implement confirmation prompt with accept and abort paths | case-insensitive y/yes, empty Enter aborts, whitespace-padded input, --force bypass preserved | authored | tick-0c56d2 |

### Phase 3: Cascade Deletion, Dependency Cleanup, and Bulk Removal
status: approved
ext_id: tick-192a68
approved_at: 2026-02-19

**Goal**: Support cascade deletion of parent-child hierarchies, automatic dependency reference cleanup on surviving tasks, and bulk removal of multiple task IDs with deduplication — all in a single atomic `Store.Mutate` call.

**Why this order**: This is the most complex phase and depends on both the removal mechanism (Phase 1) and the confirmation prompt (Phase 2). Cascade deletion, dependency cleanup, and bulk removal are highly cohesive — they all operate within the same `Mutate` callback, share the same deduplication logic, and the confirmation prompt must surface the combined blast radius. Splitting them would create artificial phase boundaries with no independently valuable intermediate state.

**Acceptance**:
- [ ] Removing a parent task recursively removes all descendants (children, grandchildren, etc.) in a single `Mutate` call
- [ ] Removing a child task does not affect the parent
- [ ] Surviving tasks have all removed task IDs scrubbed from their `BlockedBy` arrays within the same `Mutate` call
- [ ] Formatter output reports which surviving tasks had dependency references cleaned (e.g., "Updated dependencies on tick-def, tick-ghi")
- [ ] Confirmation prompt (without `--force`) lists all tasks that will be removed, including cascaded descendants
- [ ] Multiple task IDs accepted as positional arguments for bulk removal
- [ ] Duplicate IDs in arguments are silently deduplicated
- [ ] A task appearing both as explicit argument and as cascaded descendant is only removed once
- [ ] If any provided task ID does not exist, the command fails before any removal occurs (all-or-nothing)
- [ ] Bulk removal with mixed cascade and non-cascade targets works atomically
- [ ] `--force` with cascade proceeds silently without confirmation
- [ ] All formatters (toon, pretty, JSON) render the combined removal + cascade + dependency cleanup output

#### Tasks
| ID | Name | Edge Cases | Status | Ext ID |
|----|------|------------|--------|--------|
| task-removal-3-1 | Bulk argument parsing with deduplication | duplicate IDs silently deduplicated, single ID still works, mixed flags and positional args | authored | tick-2a1fa5 |
| task-removal-3-2 | All-or-nothing ID validation for bulk removal | first ID valid but second invalid, all IDs invalid | authored | tick-37bab0 |
| task-removal-3-3 | Cascade descendant collection | deep hierarchy (3+ levels), task with no children, child removal does not cascade upward | authored | tick-5b74ec |
| task-removal-3-4 | Integrate cascade into removal flow with confirmation prompt | cascade with --force skips prompt, prompt lists all cascaded descendants, dependency cleanup covers all removed IDs | authored | tick-9e0c27 |
| task-removal-3-5 | Bulk and cascade interaction with cross-target deduplication | task appears as both explicit arg and cascaded descendant, two targets share a common descendant, bulk removal of unrelated leaf tasks | authored | tick-0424d3 |
