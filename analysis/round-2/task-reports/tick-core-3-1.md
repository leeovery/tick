# Task tick-core-3-1: Dependency Validation

## Task Summary

Implement dependency validation that rejects two categories of invalid dependencies at write time:

1. **Circular dependencies** (deadlocks) -- DFS/BFS from `newBlockedByID` following `blocked_by` edges to detect if `taskID` is reachable. Error must include full cycle path.
2. **Child-blocked-by-parent** -- If task's parent equals `newBlockedByID`, reject. Unworkable due to leaf-only ready rule.

Required API:
- `ValidateDependency(tasks []Task, taskID, newBlockedByID string) error`
- `ValidateDependencies(tasks, taskID, blockedByIDs)` for batch -- sequential, fail on first error

Allowed: parent blocked by child, sibling deps, cross-hierarchy deps. Pure functions, no I/O.

Error formats from spec:
- `Error: Cannot add dependency - creates cycle: tick-a -> tick-b -> tick-a`
- `Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-parent`

### Acceptance Criteria

1. Self-reference rejected
2. 2-node cycle detected with path
3. 3+ node cycle detected with full path
4. Child-blocked-by-parent rejected with descriptive error
5. Parent-blocked-by-child allowed
6. Sibling/cross-hierarchy deps allowed
7. Error messages match spec format
8. Batch validation fails on first error
9. Pure functions -- no I/O

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| Self-reference rejected | PASS -- tested in "it rejects direct self-reference" | PASS -- tested in "it rejects direct self-reference" |
| 2-node cycle detected with path | PASS -- tested with exact message match | PASS -- tested with exact message match |
| 3+ node cycle detected with full path | PASS -- tested with 3-node and 4-node chains | PASS -- tested with 3-node and 4-node chains |
| Child-blocked-by-parent rejected | PASS -- tested in "it rejects child blocked by own parent" | PASS -- tested in "it rejects child blocked by own parent" |
| Parent-blocked-by-child allowed | PASS -- tested explicitly | PASS -- tested explicitly |
| Sibling/cross-hierarchy deps allowed | PASS -- both tested explicitly | PASS -- both tested explicitly |
| Error messages match spec format | **FAIL** -- uses `"Cannot add dependency"`, missing `"Error: "` prefix required by spec | PASS -- uses `"Error: Cannot add dependency"` matching spec exactly |
| Batch validation fails on first error | PASS -- tested in TestValidateDependencies | PASS -- tested in TestValidateDependencies |
| Pure functions -- no I/O | PASS -- no I/O anywhere | PASS -- no I/O anywhere |

## Implementation Comparison

### Approach

**File organization:**
- V2 adds all validation logic directly into `internal/task/task.go` (existing file), appending ~105 lines. Tests go into `task_test.go`.
- V4 creates a dedicated `internal/task/dependency.go` (135 lines) and `internal/task/dependency_test.go` (213 lines). Cleaner separation of concerns.

**Case normalization:**
- V2 uses `NormalizeID()` throughout, normalizing every ID comparison to lowercase. This is defensive: it ensures case-insensitive matching of task IDs, parent IDs, and blocked_by IDs:
  ```go
  normalizedTaskID := NormalizeID(taskID)
  normalizedBlockedByID := NormalizeID(newBlockedByID)
  // ...
  taskMap[NormalizeID(t.ID)] = t
  // ...
  if NormalizeID(task.Parent) == normalizedBlockedByID {
  ```
- V4 does **no** case normalization whatsoever. All comparisons are direct string equality (`t.ID == taskID`, `blocker == taskID`). This assumes IDs are already normalized before reaching validation.

**Graph traversal algorithm:**
- V2 uses **DFS** (stack-based) via an explicit stack with `frame` structs that carry the path:
  ```go
  type frame struct {
      id   string
      path []string
  }
  stack := []frame{{id: startID, path: []string{startID}}}
  // Pop from end (LIFO)
  current := stack[len(stack)-1]
  stack = stack[:len(stack)-1]
  ```
- V4 uses **BFS** (queue-based) with a separate `parent` map for path reconstruction:
  ```go
  parent := make(map[string]string)
  queue := []string{newBlockedByID}
  // Dequeue from front (FIFO)
  current := queue[0]
  queue = queue[1:]
  ```

**Path reconstruction:**
- V2 carries the full path in each stack frame, allocating a new slice at every edge:
  ```go
  newPath := make([]string, len(current.path)+1)
  copy(newPath, current.path)
  newPath[len(current.path)] = normalizedDep
  ```
  This is O(path_length) memory per frame and does extra copying, but yields the path directly when a cycle is found.

- V4 stores only a `parent` map and reconstructs the path at the end via backtracking:
  ```go
  func reconstructPath(parent map[string]string, start, end string) []string {
      path := []string{end}
      current := end
      for current != start {
          current = parent[current]
          path = append(path, current)
      }
      // Reverse
      for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
          path[i], path[j] = path[j], path[i]
      }
      return path
  }
  ```
  More memory-efficient for wide graphs since each node stores only one parent pointer.

**Function decomposition:**
- V2 has two functions: `ValidateDependency` (main) and `detectCycle` (private helper). The cycle detection is separated but child-blocked-by-parent check is inline.
- V4 decomposes into five functions: `ValidateDependency`, `ValidateDependencies`, `checkChildBlockedByParent`, `checkCycle`, `buildBlockedByMap`, `reconstructPath`, and `formatCyclePath`. This is significantly more modular.

**Child-blocked-by-parent check:**
- V2 builds a map and does a single lookup:
  ```go
  if task, ok := taskMap[normalizedTaskID]; ok {
      if NormalizeID(task.Parent) == normalizedBlockedByID {
  ```
- V4 iterates the full task list:
  ```go
  func checkChildBlockedByParent(tasks []Task, taskID, newBlockedByID string) error {
      for _, t := range tasks {
          if t.ID == taskID && t.Parent == newBlockedByID {
  ```
  V2's map approach is O(1) lookup after O(n) build (map is reused for cycle detection). V4 does a linear scan specifically for this check, then `buildBlockedByMap` does another linear scan for cycle detection -- two passes total.

**Check ordering:**
- V2: self-reference first, then build map, then child-blocked-by-parent, then cycle detection.
- V4: child-blocked-by-parent first, then cycle detection (which includes self-reference). This means V4's self-reference check is inside `checkCycle`, not a separate early-return.

**Error message format:**
- V2: `"Cannot add dependency - creates cycle: ..."` and `"Cannot add dependency - %s cannot be blocked by its parent %s"`
- V4: `"Error: Cannot add dependency - creates cycle: ..."` and `"Error: Cannot add dependency - %s cannot be blocked by its parent %s"`
- The spec explicitly requires the `"Error: "` prefix. **V2 does not match the spec; V4 does.**

### Code Quality

**Go idioms:**
- Both versions follow standard Go patterns. Neither uses custom error types (both use `fmt.Errorf`).
- V4's decomposition into small single-purpose functions (`checkChildBlockedByParent`, `checkCycle`, `buildBlockedByMap`, `reconstructPath`, `formatCyclePath`) is more idiomatic Go -- small functions with clear names.
- V2's `detectCycle` function is larger and does more work internally.

**Naming:**
- V2: `detectCycle(taskMap, targetID, startID)` -- parameter naming could be clearer. `targetID` is the task we're checking *for*, `startID` is where we begin traversal.
- V4: `checkCycle(tasks, taskID, newBlockedByID)` matches the public API naming, making the relationship clearer.

**DRY:**
- V2 builds the `taskMap` once and reuses it for both child-blocked-by-parent and cycle detection. More efficient.
- V4 has `checkChildBlockedByParent` do its own linear scan, then `checkCycle` calls `buildBlockedByMap` for another scan. Two separate passes where V2 does one.

**Type safety:**
- Both versions are equivalent in type safety. Neither uses custom error types. Both operate on `[]Task` and return `error`.

**Case normalization (significant quality difference):**
- V2 normalizes every ID comparison via `NormalizeID()` (6 call sites in the implementation). This is defensive and correct if IDs could arrive in mixed case.
- V4 performs zero normalization. If callers pass IDs in different cases, the validation would miss cycles or parent relationships. Whether this is a bug depends on the system's invariants -- if IDs are guaranteed lowercase at this layer, V4 is fine; if not, V4 has a latent bug.

**Error message formatting -- V2 path display (lines 229-237):**
```go
parts := make([]string, 0, len(path)+1)
parts = append(parts, taskID)
for _, id := range path {
    if t, ok := taskMap[id]; ok {
        parts = append(parts, t.ID)
    } else {
        parts = append(parts, id)
    }
}
parts = append(parts, taskID)
```
V2 looks up original-cased IDs from the taskMap to display in error messages. This preserves the user's original casing in output even though comparisons are case-insensitive.

**Error message formatting -- V4 (line 131):**
```go
func formatCyclePath(taskID string, path []string) string {
    parts := make([]string, 0, len(path)+2)
    parts = append(parts, taskID)
    parts = append(parts, path...)
    parts = append(parts, taskID)
    return strings.Join(parts, " \u2192 ")
}
```
Clean, dedicated formatting function. Simpler since no case translation needed.

### Test Quality

**V2 Test Functions (in `task_test.go`):**

`TestValidateDependency` -- 9 subtests:
1. `"it allows valid dependency between unrelated tasks"` -- two tasks, no blockers, expects nil error
2. `"it rejects direct self-reference"` -- same ID for task and blocker, checks "creates cycle" substring
3. `"it rejects 2-node cycle with path"` -- tick-aaa blocked by tick-bbb already; adding tick-bbb blocked by tick-aaa; exact message match
4. `"it rejects 3+ node cycle with full path"` -- 3-task chain; exact message match with full path
5. `"it rejects child blocked by own parent"` -- child with Parent field set; exact message match
6. `"it allows parent blocked by own child"` -- reverse direction; expects nil error
7. `"it allows sibling dependencies"` -- two children of same parent; expects nil error
8. `"it allows cross-hierarchy dependencies"` -- children of different parents; expects nil error
9. `"it returns cycle path format: tick-a -> tick-b -> tick-a"` -- 2-node cycle with short IDs; exact match
10. `"it detects cycle through existing multi-hop chain"` -- 4-task chain; exact message with full path

`TestValidateDependencies` -- 1 subtest:
1. `"it validates multiple blocked_by IDs, fails on first error"` -- batch with valid then invalid; checks "creates cycle" substring

Total: 11 subtests across 2 test functions.

**V4 Test Functions (in `dependency_test.go`):**

`TestValidateDependency` -- 10 subtests:
1. `"it allows valid dependency between unrelated tasks"` -- same as V2
2. `"it rejects direct self-reference"` -- checks "creates cycle" and task ID substrings
3. `"it rejects 2-node cycle with path"` -- exact message match with `"Error: "` prefix
4. `"it rejects 3+ node cycle with full path"` -- exact message match with `"Error: "` prefix
5. `"it rejects child blocked by own parent"` -- exact message match with `"Error: "` prefix
6. `"it allows parent blocked by own child"` -- expects nil error
7. `"it allows sibling dependencies"` -- expects nil error
8. `"it allows cross-hierarchy dependencies"` -- expects nil error
9. `"it returns cycle path format: tick-a -> tick-b -> tick-a"` -- exact match with `"Error: "` prefix
10. `"it detects cycle through existing multi-hop chain"` -- 4-task chain; exact message with `"Error: "` prefix

`TestValidateDependencies` -- 2 subtests:
1. `"it validates multiple blocked_by IDs, fails on first error"` -- batch with valid then invalid (child-blocked-by-parent); exact message match
2. `"it succeeds when all dependencies are valid"` -- batch with all valid deps; expects nil error

Total: 12 subtests across 2 test functions.

**Test infrastructure:**
- V2 constructs `Task` structs inline with minimal fields: `Task{ID: "tick-aaa", BlockedBy: nil}`.
- V4 uses a `depTask` helper function for cleaner test setup:
  ```go
  func depTask(id string, parent string, blockedBy ...string) Task {
      return Task{
          ID: id, Title: "Test task " + id, Status: StatusOpen,
          Priority: 2, BlockedBy: blockedBy, Parent: parent,
      }
  }
  ```
  V4's helper creates more complete Task objects (with Title, Status, Priority set). V2's minimal structs are sufficient since validation only reads ID, Parent, and BlockedBy.

**Test gap analysis:**
- V4 has an extra test: `"it succeeds when all dependencies are valid"` in TestValidateDependencies. V2 only tests the failure case.
- V2's batch test uses a cycle error; V4's batch test uses a child-blocked-by-parent error. V4's approach tests a different code path in the batch function.
- V4's batch failure test asserts the **exact** error message; V2's only checks for `"creates cycle"` substring. V4 is more precise.
- Neither version tests: empty `blockedByIDs` slice, non-existent task IDs, tasks with multiple blocked_by entries branching.

**Assertion style:**
- Both use standard library `testing` with `t.Fatal` / `t.Errorf`.
- Both use exact string matching for error messages where appropriate and substring matching for less critical checks.
- V4 is slightly more consistent: the batch failure test asserts the exact message while V2 only checks a substring.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 2 (task.go, task_test.go) | 2 (dependency.go, dependency_test.go) |
| Lines added | 286 | 348 |
| Impl LOC | 105 | 135 |
| Test LOC | 181 | 213 |
| Test functions | 2 (11 subtests) | 2 (12 subtests) |

## Verdict

**V4 is the better implementation**, for three reasons:

1. **Spec compliance on error format.** V2 uses `"Cannot add dependency"` while the spec explicitly requires `"Error: Cannot add dependency"`. V4 matches the spec. This is a clear functional defect in V2 -- its tests assert the wrong format and pass, but the output would be wrong in production. This is the most significant difference.

2. **Better code organization.** V4 creates a dedicated `dependency.go` file with well-decomposed helper functions (`checkChildBlockedByParent`, `checkCycle`, `buildBlockedByMap`, `reconstructPath`, `formatCyclePath`). V2 stuffs everything into the already-large `task.go`. V4's approach is more maintainable and follows the single-responsibility principle.

3. **More thorough test coverage.** V4 has 12 subtests vs V2's 11, with the extra test covering the happy path for batch validation. V4's batch failure test also asserts exact error messages rather than substrings, and uses a different error type (child-blocked-by-parent) than the single-validation tests, exercising more code paths.

V2 has one advantage: **case normalization**. V2 applies `NormalizeID()` at every comparison point, which is more defensive. V4 does none, assuming IDs arrive pre-normalized. Whether this matters depends on the system's guarantees, but V2's approach is safer. V2 also reuses its `taskMap` for both the parent check and cycle detection (single pass), while V4 does two linear scans. These are minor efficiency and robustness points that don't outweigh V2's spec non-compliance.
