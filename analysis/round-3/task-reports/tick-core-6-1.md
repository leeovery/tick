# Task 6-1: Add dependency validation to create --blocked-by and --blocks paths (V5 Only -- Phase 6 Refinement)

## Task Plan Summary

The task addresses a validation gap where `create --blocked-by`, `create --blocks`, and `update --blocks` paths bypassed the full dependency validation (cycle detection and child-blocked-by-parent checks) that `dep add` correctly enforced via `task.ValidateDependency`/`task.ValidateDependencies`. The fix adds these validation calls to the three affected code paths so that all dependency-creating operations are subject to identical constraints.

## Note

This is a Phase 6 analysis refinement task that only exists in V5. It addresses a validation gap found during post-implementation analysis. This is a standalone quality assessment, not a comparison.

## V5 Implementation

### Architecture & Design

The implementation follows a thoughtful architectural approach with one notable design decision: the **stub task pattern** in `create.go`.

Since the new task does not yet exist in the task list when validation runs, the implementation constructs a temporary stub task with the new task's ID, parent, and blocked_by fields, then appends it to a copy of the task list:

```go
// create.go, lines 92-105
stubTask := task.NewTask(id, title)
stubTask.Parent = parent
if len(blockedBy) > 0 {
    stubTask.BlockedBy = blockedBy
}
tasksWithNew := make([]task.Task, len(tasks)+1)
copy(tasksWithNew, tasks)
tasksWithNew[len(tasks)] = stubTask
```

This is a critical design choice. Without it, `task.ValidateDependency` would fail to detect the child-blocked-by-parent rule because the new task's parent field would not be visible in the task list. The BFS-based cycle detection also needs the new task's edges to reason about cycles involving both `--blocked-by` and `--blocks` simultaneously.

For `update.go`, no stub is needed because the task already exists in the list. The implementation is correspondingly simpler -- just a loop calling `ValidateDependency` for each block target:

```go
// update.go, lines 101-105
for _, blockID := range blocks {
    if err := task.ValidateDependency(tasks, blockID, id); err != nil {
        return nil, err
    }
}
```

The argument order `(tasks, blockID, id)` is correct: `--blocks <blockID>` means "add `id` to `blockID`'s `blocked_by`", so the validation is "would adding `id` as a dependency of `blockID` create a cycle?" -- i.e., `ValidateDependency(tasks, blockID, id)` checks if BFS from `id` following blocked_by edges reaches `blockID`.

**Placement within the mutation flow**: Validation is positioned after `validateIDsExist` (ensuring all referenced IDs are present) and before the actual task construction/mutation. This ordering is correct -- existence is a precondition for meaningful cycle detection.

### Code Quality

**Positive aspects:**

1. **Clear comments**: The new code block in `create.go` opens with a descriptive comment explaining both what and why:
   ```go
   // Validate dependencies (cycle detection, child-blocked-by-parent).
   // Build a temporary task list that includes the new task so validators
   // can see its parent and blocked_by edges.
   ```

2. **Consistent error handling**: All errors are returned immediately as `(nil, err)`, matching the existing pattern throughout both files. No errors are silently ignored.

3. **Reuse of existing validators**: Rather than duplicating validation logic, the implementation calls the same `task.ValidateDependency` and `task.ValidateDependencies` functions used by `dep add`. This is the correct approach -- single source of truth for dependency rules.

4. **Idiomatic Go**: Error propagation via `fmt.Errorf` wrapping in the underlying validators, explicit nil checks, and clean control flow.

**Minor observations:**

1. **Allocation in the happy path**: `tasksWithNew` allocates a new slice of `len(tasks)+1` and copies all tasks every time `create` runs, even when neither `--blocked-by` nor `--blocks` is provided. While the copy is O(n) and tasks are value types (not large), wrapping the allocation inside an `if len(blockedBy) > 0 || len(blocks) > 0` guard would avoid unnecessary work. However, this is a negligible concern for a CLI tool where task counts are small.

2. **The stub task has incomplete fields**: `stubTask` only sets `ID`, `Title`, `Parent`, and `BlockedBy`. Fields like `Status`, `Priority`, `Description`, and timestamps are defaults from `NewTask`. This is correct because `ValidateDependency` only inspects `ID`, `Parent`, and `BlockedBy` during its BFS. The incomplete fields have no impact on correctness.

3. **No duplicate dependency check in create/update --blocks paths**: The `dep add` command (dep.go line ~82-87) checks for duplicate dependencies before calling `ValidateDependency`. The `create --blocks` and `update --blocks` paths do not check for duplicates. This is acceptable because `--blocks` is additive (append to blocked_by) and the task plan did not specify duplicate checking as a requirement. However, it means a user could run `update --blocks tick-aaa` twice and get duplicate entries in the target's `blocked_by` array. This is a pre-existing behavior, not introduced by this task.

### Test Coverage

The implementation adds **8 new test cases** across two test files:

**create_test.go (4 new tests):**

| Test | Scenario | Validates |
|------|----------|-----------|
| `it rejects --blocked-by that would create child-blocked-by-parent` | Create child with `--parent A --blocked-by A` | Child-blocked-by-parent rule via `ValidateDependencies` |
| `it rejects --blocked-by that would create a cycle via --blocks` | A blocked_by B; create C `--blocked-by A --blocks B` forming B->C->A->B cycle | Cycle detection across combined `--blocked-by` + `--blocks` |
| `it rejects --blocks that would create a direct cycle` | Create B `--blocked-by A --blocks A` forming A<->B direct cycle | Direct cycle detection via `--blocks` |
| `it accepts valid --blocked-by and --blocks dependencies` | Independent A, B; create C `--blocked-by A --blocks B` | Happy path: valid combined dependencies |

**update_test.go (3 new tests):**

| Test | Scenario | Validates |
|------|----------|-----------|
| `it rejects --blocks that would create a cycle` | C blocked_by A; update C `--blocks A` forming A<->C direct cycle | Direct cycle via update --blocks |
| `it rejects --blocks that would create an indirect cycle` | A blocked_by B, B blocked_by C; update A `--blocks C` forming C->A->B->C cycle | Multi-hop indirect cycle via update --blocks |
| `it accepts valid --blocks dependency` | Independent A, B; update A `--blocks B` | Happy path: valid update --blocks |

**Test quality assessment:**

1. **Comprehensive error path verification**: Each rejection test checks (a) exit code is 1, (b) stderr contains the expected keyword ("cycle" or "parent"), and (c) no mutation occurred (task count or blocked_by state unchanged). This three-point verification is thorough.

2. **Detailed comments in test code**: The `indirect cycle` test in update_test.go includes an extensive comment block (lines ~477-489) showing the developer's reasoning about BFS traversal direction. While verbose, this demonstrates the developer understood the algorithm well enough to catch a subtle directionality issue and chose the correct test scenario.

3. **Coverage against acceptance criteria**: All 5 acceptance criteria from the plan are met:
   - `create --blocked-by` rejects cycles: Covered by "cycle via --blocks" and "direct cycle" tests
   - `create --blocked-by <parent>` rejects child-blocked-by-parent: Covered by first new create test
   - `create --blocks` rejects cycles: Covered by "direct cycle" test
   - `update --blocks` rejects cycles: Covered by both update cycle tests
   - Valid dependencies still succeed: Covered by happy path tests in both files

4. **Missing test from plan**: The plan specified "Test create with --blocked-by forming a direct cycle (A blocks B, create C --blocked-by A,C where C blocks A)". The implementation tests a slightly different variant (--blocked-by A --blocks A creating A<->B, rather than a 3-task cycle purely through --blocked-by). However, the pure `--blocked-by` cycle scenario is already covered by the existing `task.ValidateDependencies` unit tests in `dependency_test.go`, so this is not a gap -- it's just tested at a different level.

5. **No test for --blocked-by only cycle (without --blocks)**: A test like "create C --blocked-by A where A is already blocked_by something that eventually reaches C" doesn't apply because C doesn't exist yet, so no cycle can form through --blocked-by alone on a new task. The developer correctly recognized this.

### Spec Compliance

The implementation fulfills all requirements from the task plan:

1. **Do item 1**: "In create.go, call `task.ValidateDependencies(tasks, id, blockedBy)` after validateIDsExist" -- Done at line 107-109 of create.go, with the additional stub task enhancement.

2. **Do item 2**: "In create.go, for each blockID in blocks, call `task.ValidateDependency(tasks, blockID, id)`" -- Done at lines 111-115 of create.go.

3. **Do item 3**: "In update.go, for each blockID in blocks, call `task.ValidateDependency(tasks, blockID, id)`" -- Done at lines 101-105 of update.go.

4. **Do item 4**: "Add tests covering (a)-(e)" -- All five test categories are covered across the 7 new test cases.

5. **All acceptance criteria met**: Error messages are consistent with `dep add` because the same `ValidateDependency` function produces them.

### golang-pro Skill Compliance

**MUST DO compliance:**

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Use gofmt/golangci-lint | Assumed (code follows standard formatting) | Consistent indentation, brace style |
| context.Context for blocking ops | N/A | No new blocking operations introduced |
| Handle all errors explicitly | Pass | All `ValidateDependency`/`ValidateDependencies` errors checked and returned |
| Table-driven tests with subtests | Partial | Tests use subtests (t.Run) but are not table-driven; each is a distinct scenario |
| Document exported functions | N/A | No new exported functions introduced |
| Propagate errors with fmt.Errorf("%w") | N/A | Errors from validators are returned unwrapped (pass-through), which is acceptable |
| Race detector on tests | Not verified | Cannot run tests in this environment |

**MUST NOT DO compliance:**

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Ignore errors | Pass | No `_` assignments for error returns |
| Panic for error handling | Pass | No panics |
| Goroutines without lifecycle | N/A | No goroutines introduced |
| Reflection without justification | Pass | No reflection |
| Hardcode configuration | Pass | No hardcoded config |

**Partial compliance on table-driven tests**: The skill file mandates "Write table-driven tests with subtests". The new tests use subtests (`t.Run`) but are individually crafted scenarios rather than table-driven. For these specific tests, table-driven format would be awkward because each scenario has different setup (different existing tasks, different CLI args, different error expectations). The current approach is pragmatic and readable.

## Quality Assessment

### Strengths

1. **Correct identification of the stub task requirement**: The most technically challenging aspect of this task was recognizing that `create` needs a stub task in the validation list. Without it, `ValidateDependency` would not see the new task's parent or blocked_by edges, leading to false negatives. The implementation handles this correctly.

2. **Argument order correctness for --blocks**: The `--blocks` flag semantics are reversed from `--blocked-by` (it modifies the *target* task, not the current task). The implementation correctly passes `(tasks, blockID, id)` rather than `(tasks, id, blockID)`, matching the semantic meaning.

3. **Thorough test verification**: Each negative test verifies three things: exit code, error message content, and absence of mutation. This guards against partial-mutation bugs where validation fails but state has already changed.

4. **Minimal, focused changes**: The diff only touches 4 source files (create.go, create_test.go, update.go, update_test.go) plus 2 tracking files. No unnecessary refactoring or scope creep.

5. **Consistency with existing patterns**: The new validation code follows the same style as the surrounding code in both create.go and update.go. The test helper functions (`initTickProjectWithTasks`, `readTasksFromFile`) are reused rather than duplicated.

### Weaknesses

1. **Unnecessary allocation when no deps**: The `tasksWithNew` slice is always allocated in `create.go` even when `len(blockedBy) == 0 && len(blocks) == 0`. A simple guard would avoid this:
   ```go
   if len(blockedBy) > 0 || len(blocks) > 0 {
       // build stub and validate
   }
   ```
   Impact: Negligible for a CLI tool. Correctness is unaffected.

2. **Verbose test comments**: The indirect cycle test in `update_test.go` (lines ~477-489) contains a lengthy reasoning trace that reads like developer notes rather than test documentation. While it shows understanding, it could be condensed to 2-3 lines explaining the setup and expected outcome.

3. **No test for --blocked-by only with child-blocked-by-parent in create + update**: While the child-blocked-by-parent test exists for `create`, there is no equivalent for `update --blocked-by` (which doesn't exist as a flag). This is not actually a gap since `update` doesn't have `--blocked-by`, but it's worth noting that the update path for blocked_by changes goes through `dep add/rm`, not `update`.

4. **Stub task does not include --blocks edges**: The `stubTask` in create.go sets `Parent` and `BlockedBy` but does not model the `--blocks` effect (where target tasks would gain the new task in their blocked_by). This means `ValidateDependencies` for `--blocked-by` runs against a task list that doesn't yet reflect `--blocks` mutations. In practice this is not a bug because: (a) `--blocked-by` validation checks edges FROM the new task, and (b) `--blocks` validation runs after and checks edges TO the target tasks from the new task. The sequential validation order handles both directions. However, a pathological case where `--blocked-by` and `--blocks` together create a cycle not detectable by either individual check could theoretically slip through. Examining the BFS algorithm: `ValidateDependency(tasksWithNew, blockID, id)` does BFS from `id` following blocked_by edges. Since `stubTask.BlockedBy = blockedBy` is set, the BFS from `id` CAN follow the `--blocked-by` edges. So the `--blocks` validation DOES see the `--blocked-by` edges. This means combined cycles ARE correctly detected. The test "it rejects --blocked-by that would create a cycle via --blocks" confirms this.

### Overall Quality Rating

**Excellent** -- The implementation is correct, well-tested, minimal in scope, and demonstrates a strong understanding of the dependency graph algorithms involved. The stub task pattern is a thoughtful solution to the "task doesn't exist yet" problem during create-time validation. All acceptance criteria are met. The weaknesses are cosmetic (unnecessary allocation, verbose comments) rather than functional. The code integrates cleanly with the existing validation infrastructure by reusing `ValidateDependency`/`ValidateDependencies` rather than duplicating logic.
