# Task tick-core-3-1: Dependency validation -- cycle detection & child-blocked-by-parent

## Task Summary

Implement pure validation functions that reject two categories of invalid `blocked_by` dependencies at write time: circular dependencies (detected via DFS/BFS with full cycle path in error) and child-blocked-by-parent (unworkable due to leaf-only ready rule). Both `ValidateDependency(tasks, taskID, newBlockedByID)` (single) and `ValidateDependencies(tasks, taskID, blockedByIDs)` (batch, fail-on-first) must be pure domain logic with no I/O.

**Acceptance Criteria:**
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

| Criterion | V5 | V6 |
|-----------|-----|-----|
| Self-reference rejected | PASS -- early check `if normTaskID == normNewDep`; tested in "it rejects direct self-reference" | PASS -- checked inside `detectCycle` via `if taskID == newBlockedByID`; tested in "it rejects direct self-reference" with exact message match |
| 2-node cycle detected with path | PASS -- BFS from `newBlockedByID` following `blockedByMap` edges; tested in "it rejects 2-node cycle with path" verifying path `tick-aaa -> tick-bbb -> tick-aaa` | PASS -- DFS from `newBlockedByID` following `blockedByMap` edges; tested in "it rejects 2-node cycle with path" with exact message match |
| 3+ node cycle detected with full path | PASS -- BFS reconstructs full path; tested in "it rejects 3+ node cycle with full path" verifying `tick-aaa -> tick-ccc -> tick-bbb -> tick-aaa` | PASS -- DFS reconstructs full path via backtracking; tested in "it rejects 3+ node cycle with full path" with exact message match |
| Child-blocked-by-parent rejected with descriptive error | PASS -- checks `taskParent == normNewDep`; error includes `"(would create unworkable task due to leaf-only ready rule)"`; tested with exact message match | PASS -- linear scan for task then `tasks[i].Parent == newBlockedByID`; tested with exact message match |
| Parent-blocked-by-child allowed | PASS -- tested in "it allows parent blocked by own child" | PASS -- tested in "it allows parent blocked by own child" |
| Sibling/cross-hierarchy deps allowed | PASS -- tested in "it allows sibling dependencies" and "it allows cross-hierarchy dependencies" | PASS -- tested in "it allows sibling dependencies" and "it allows cross-hierarchy dependencies" |
| Error messages match spec format | PARTIAL -- uses capitalized `"Cannot add dependency"` and includes extra rationale `"(would create unworkable task due to leaf-only ready rule)"` not in spec; spec says `"Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-parent"` | PARTIAL -- uses lowercase `"cannot add dependency"` deviating from spec casing; omits the rationale parenthetical; spec says `"Error: Cannot add dependency"` |
| Batch validation fails on first error | PASS -- sequential loop in `ValidateDependencies`; tested in "it validates multiple blocked_by IDs, fails on first error" | PASS -- sequential loop in `ValidateDependencies`; tested in "it validates multiple blocked_by IDs, fails on first error" |
| Pure functions -- no I/O | PASS -- no I/O, no context, no filesystem; operates on `[]Task` slice | PASS -- no I/O, no context, no filesystem; operates on `[]Task` slice |

## Implementation Comparison

### Approach

Both versions implement the same public API: `ValidateDependency(tasks []Task, taskID, newBlockedByID string) error` and `ValidateDependencies(tasks []Task, taskID string, blockedByIDs []string) error`, both in a new `internal/task/dependency.go` file. Both are pure functions with no I/O. The key structural differences are in graph traversal algorithm (BFS vs DFS), ID normalization, function decomposition, and error message content.

**Graph traversal algorithm:**

V5 uses **BFS** with an explicit queue of `node` structs that carry path history:
```go
// V5: dependency.go — BFS cycle detection
type node struct {
    id   string
    path []string // path of original IDs from newBlockedByID to this node
}

queue := []node{{id: normNewDep, path: []string{getOrigID(origID, newBlockedByID, normNewDep)}}}
visited := map[string]bool{normNewDep: true}

for len(queue) > 0 {
    current := queue[0]
    queue = queue[1:]

    for _, dep := range blockedByMap[current.id] {
        if dep == normTaskID {
            cyclePath := append([]string{taskID}, current.path...)
            cyclePath = append(cyclePath, dep)
            cyclePath[len(cyclePath)-1] = getOrigID(origID, taskID, normTaskID)
            return fmt.Errorf(
                "Cannot add dependency - creates cycle: %s",
                strings.Join(cyclePath, " \u2192 "),
            )
        }
        if !visited[dep] {
            visited[dep] = true
            newPath := make([]string, len(current.path))
            copy(newPath, current.path)
            newPath = append(newPath, getOrigID(origID, dep, dep))
            queue = append(queue, node{id: dep, path: newPath})
        }
    }
}
```

V6 uses **DFS** with recursive backtracking, mutating a shared `path` slice via pointer:
```go
// V6: dependency.go — DFS cycle detection
func detectCycle(tasks []Task, taskID, newBlockedByID string) error {
    if taskID == newBlockedByID {
        return fmt.Errorf(
            "cannot add dependency - creates cycle: %s \u2192 %s",
            taskID, taskID,
        )
    }

    blockedByMap := make(map[string][]string, len(tasks))
    for i := range tasks {
        if len(tasks[i].BlockedBy) > 0 {
            blockedByMap[tasks[i].ID] = tasks[i].BlockedBy
        }
    }

    visited := make(map[string]bool)
    path := []string{taskID, newBlockedByID}

    if found := dfs(newBlockedByID, taskID, blockedByMap, visited, &path); found {
        return fmt.Errorf(
            "cannot add dependency - creates cycle: %s",
            strings.Join(path, " \u2192 "),
        )
    }
    return nil
}

func dfs(current, target string, blockedByMap map[string][]string, visited map[string]bool, path *[]string) bool {
    if current == target {
        return true
    }
    if visited[current] {
        return false
    }
    visited[current] = true

    for _, dep := range blockedByMap[current] {
        *path = append(*path, dep)
        if dfs(dep, target, blockedByMap, visited, path) {
            return true
        }
        *path = (*path)[:len(*path)-1]
    }
    return false
}
```

**ID normalization:**

V5 normalizes all IDs to lowercase via `NormalizeID()` throughout the entire function -- 5 calls total. This ensures case-insensitive matching for cycle detection, child-blocked-by-parent checks, and BFS traversal:
```go
// V5: dependency.go — normalization throughout
normTaskID := NormalizeID(taskID)
normNewDep := NormalizeID(newBlockedByID)

if normTaskID == normNewDep { ... }

// builds map with normalized keys
nid := NormalizeID(t.ID)
origID[nid] = t.ID
if nid == normTaskID && t.Parent != "" {
    taskParent = NormalizeID(t.Parent)
}
for _, dep := range t.BlockedBy {
    blockedByMap[nid] = append(blockedByMap[nid], NormalizeID(dep))
}
```

V6 performs **zero** normalization -- all comparisons use raw string equality:
```go
// V6: dependency.go — no normalization
if taskID == newBlockedByID { ... }          // raw comparison
if tasks[i].ID == taskID { ... }             // raw comparison
blockedByMap[tasks[i].ID] = tasks[i].BlockedBy // raw keys
```

This is a correctness gap in V6. If IDs are ever stored or provided with different casing (e.g., `TICK-abc` vs `tick-abc`), V6 would fail to detect cycles or parent relationships. The project has `NormalizeID` available for exactly this purpose.

**Function decomposition:**

V5 puts all logic in a single `ValidateDependency` function (82 lines) plus one tiny helper `getOrigID` (7 lines). The function handles self-reference, map building, child-blocked-by-parent, and BFS cycle detection in sequence:
```go
// V5: single function flow
func ValidateDependency(tasks []Task, taskID, newBlockedByID string) error {
    // 1. Self-reference check
    // 2. Build lookup maps (single pass)
    // 3. Child-blocked-by-parent check
    // 4. BFS cycle detection
}
```

V6 decomposes into 4 functions -- `ValidateDependency` (delegator, 7 lines), `validateChildBlockedByParent` (11 lines), `detectCycle` (26 lines), and `dfs` (18 lines):
```go
// V6: decomposed into 4 functions
func ValidateDependency(tasks []Task, taskID, newBlockedByID string) error {
    if err := validateChildBlockedByParent(tasks, taskID, newBlockedByID); err != nil {
        return err
    }
    return detectCycle(tasks, taskID, newBlockedByID)
}
```

**Map building efficiency:**

V5 builds both maps (`origID` and `blockedByMap`) in a single pass over `tasks`, also extracting the task's parent in the same loop:
```go
// V5: single-pass map construction
origID := make(map[string]string, len(tasks))
blockedByMap := make(map[string][]string, len(tasks))
var taskParent string
for _, t := range tasks {
    nid := NormalizeID(t.ID)
    origID[nid] = t.ID
    if nid == normTaskID && t.Parent != "" {
        taskParent = NormalizeID(t.Parent)
    }
    for _, dep := range t.BlockedBy {
        blockedByMap[nid] = append(blockedByMap[nid], NormalizeID(dep))
    }
}
```

V6 iterates over `tasks` twice -- once in `validateChildBlockedByParent` (linear scan for parent check) and once in `detectCycle` (building `blockedByMap`):
```go
// V6: first pass in validateChildBlockedByParent
for i := range tasks {
    if tasks[i].ID == taskID && tasks[i].Parent == newBlockedByID && newBlockedByID != "" {
        return fmt.Errorf(...)
    }
}

// V6: second pass in detectCycle
blockedByMap := make(map[string][]string, len(tasks))
for i := range tasks {
    if len(tasks[i].BlockedBy) > 0 {
        blockedByMap[tasks[i].ID] = tasks[i].BlockedBy
    }
}
```

**Adjacency map construction detail:**

V6 directly references `tasks[i].BlockedBy` slices (sharing backing arrays with the input):
```go
// V6: shares slice backing array with input
blockedByMap[tasks[i].ID] = tasks[i].BlockedBy
```

V5 normalizes each dependency ID into a fresh slice:
```go
// V5: creates new slices with normalized IDs
blockedByMap[nid] = append(blockedByMap[nid], NormalizeID(dep))
```

V6's approach avoids allocation but couples the map to the input slice's lifecycle. For pure validation this is fine.

**Error message differences:**

V5 child-blocked-by-parent error includes an explanatory rationale:
```go
// V5 error:
"Cannot add dependency - tick-child cannot be blocked by its parent tick-parent (would create unworkable task due to leaf-only ready rule)"
```

V6 child-blocked-by-parent error is terse:
```go
// V6 error:
"cannot add dependency - tick-child cannot be blocked by its parent tick-parent"
```

V5 cycle errors use capitalized `"Cannot"`:
```go
// V5: "Cannot add dependency - creates cycle: tick-a -> tick-b -> tick-a"
```

V6 cycle errors use lowercase `"cannot"`:
```go
// V6: "cannot add dependency - creates cycle: tick-a -> tick-b -> tick-a"
```

The spec says: `"Error: Cannot add dependency - creates cycle: tick-a -> tick-b -> tick-a"`. Neither version includes the `"Error: "` prefix (standard Go practice). V5 matches the spec's `"Cannot"` casing; V6 follows Go convention with lowercase.

**Validation order:**

V5 checks in order: self-reference, build maps, child-blocked-by-parent, cycle detection (BFS).

V6 checks in order: child-blocked-by-parent (including linear scan), then `detectCycle` which checks self-reference first, then builds map, then DFS.

A subtle difference: if a task is both a child of AND would form a cycle with the blocked-by target, V5 would report self-reference first (if self-ref), then child-blocked-by-parent; V6 would report child-blocked-by-parent first. This is a minor behavioral difference since both error categories would be caught.

**origID map for display:**

V5 maintains an `origID` map to convert normalized IDs back to their original form for error messages:
```go
// V5: preserves original-case IDs for display
origID := make(map[string]string, len(tasks))
// ...
origID[nid] = t.ID
// ...
cyclePath[len(cyclePath)-1] = getOrigID(origID, taskID, normTaskID)
```

V6 has no such concern since it never normalizes IDs and uses them as-is in error messages.

### Code Quality

**V5:**
- 110 lines in `dependency.go` -- single cohesive function with one small helper
- Uses `NormalizeID` throughout for case-insensitive correctness -- consistent with the rest of the codebase
- BFS is slightly more complex to implement (explicit queue, path copying) but avoids recursion depth issues for very deep chains
- The `node` struct with path tracking requires `make`+`copy` for each new path, allocating more memory per BFS step
- `getOrigID` helper is well-named, unexported, and focused
- All exported functions documented with godoc: `ValidateDependency`, `ValidateDependencies`
- Single-pass map building is efficient
- The `append([]string{taskID}, current.path...)` pattern in cycle-found branch is slightly tricky -- creates a new slice with taskID prepended

**V6:**
- 99 lines in `dependency.go` -- 4 functions with clear separation of concerns
- Each function has a single responsibility: `ValidateDependency` (orchestrator), `validateChildBlockedByParent`, `detectCycle`, `dfs`
- DFS with pointer-to-slice path tracking (`*path`) is more memory-efficient -- mutates in place, backtracks by reslicing
- No ID normalization -- a correctness gap relative to V5
- `dfs` uses the standard recursive DFS pattern with `visited` map and path append/backtrack
- The `newBlockedByID != ""` guard in `validateChildBlockedByParent` is defensive but the spec doesn't mention empty-string blocked-by IDs
- Both exported functions documented; unexported functions also documented
- Iterates `tasks` twice (once per sub-function) instead of once
- The `for i := range tasks` idiom (index-based) avoids copying Task structs on each iteration -- correct for a slice of potentially large structs

**Both versions:**
- Use `strings.Join` with `" -> "` (arrow) for cycle path formatting
- Use `fmt.Errorf` for error construction
- No panics, no I/O, no context -- pure functions as specified
- Both `ValidateDependencies` implementations are identical in structure: sequential loop, fail on first error

### Test Quality

**V5 Tests** (180 lines in `dependency_test.go`):

All tests in `TestValidateDependency`:

1. `"it allows valid dependency between unrelated tasks"` -- two unrelated tasks, asserts no error via `t.Fatalf`
2. `"it rejects direct self-reference"` -- single task, `ValidateDependency(tasks, "tick-aaa", "tick-aaa")`, asserts error non-nil
3. `"it rejects 2-node cycle with path"` -- `tick-bbb` blocked by `tick-aaa`, adding `tick-aaa` blocked by `tick-bbb`; checks `"creates cycle"` and exact path `"tick-aaa -> tick-bbb -> tick-aaa"`
4. `"it rejects 3+ node cycle with full path"` -- 3-task chain, checks exact path `"tick-aaa -> tick-ccc -> tick-bbb -> tick-aaa"`
5. `"it rejects child blocked by own parent"` -- checks exact error message including `"(would create unworkable task due to leaf-only ready rule)"`
6. `"it allows parent blocked by own child"` -- asserts no error
7. `"it allows sibling dependencies"` -- 3-task family, asserts no error
8. `"it allows cross-hierarchy dependencies"` -- 2 parents + 2 children in different hierarchies, asserts no error
9. `"it returns cycle path format: tick-a -> tick-b -> tick-a"` -- exact error string match: `"Cannot add dependency - creates cycle: tick-a -> tick-b -> tick-a"`
10. `"it detects cycle through existing multi-hop chain"` -- 4-task chain, checks exact path `"tick-a -> tick-d -> tick-c -> tick-b -> tick-a"`

All tests in `TestValidateDependencies`:

11. `"it validates multiple blocked_by IDs, fails on first error"` -- batch of `["tick-ccc", "tick-bbb"]`, tick-ccc passes, tick-bbb creates cycle; asserts error mentions `"tick-bbb"`

**V6 Tests** (190 lines in `dependency_test.go`):

All tests in `TestValidateDependency`:

1. `"it allows valid dependency between unrelated tasks"` -- two unrelated tasks, asserts no error via `t.Errorf`
2. `"it rejects direct self-reference"` -- exact error match: `"cannot add dependency - creates cycle: tick-aaa -> tick-aaa"`
3. `"it rejects 2-node cycle with path"` -- exact error match: `"cannot add dependency - creates cycle: tick-bbb -> tick-aaa -> tick-bbb"`
4. `"it rejects 3+ node cycle with full path"` -- exact error match: `"cannot add dependency - creates cycle: tick-ccc -> tick-aaa -> tick-bbb -> tick-ccc"`
5. `"it rejects child blocked by own parent"` -- exact error match: `"cannot add dependency - tick-child cannot be blocked by its parent tick-parent"`
6. `"it allows parent blocked by own child"` -- asserts no error
7. `"it allows sibling dependencies"` -- asserts no error
8. `"it allows cross-hierarchy dependencies"` -- asserts no error
9. `"it returns cycle path format: tick-a -> tick-b -> tick-a"` -- exact error match: `"cannot add dependency - creates cycle: tick-b -> tick-a -> tick-b"`
10. `"it detects cycle through existing multi-hop chain"` -- exact error match: `"cannot add dependency - creates cycle: tick-d -> tick-a -> tick-b -> tick-c -> tick-d"`

All tests in `TestValidateDependencies`:

11. `"it validates multiple blocked_by IDs, fails on first error"` -- batch of `["tick-aaa", "tick-ccc"]` where `tick-aaa` creates cycle; exact error match

**Test approach differences:**

V5 test assertions use a mix of `strings.Contains` (for partial checks) and exact string match:
```go
// V5: partial + exact checks
if !strings.Contains(errMsg, "creates cycle") {
    t.Errorf("error should mention cycle: %q", errMsg)
}
wantPath := "tick-aaa \u2192 tick-bbb \u2192 tick-aaa"
if !strings.Contains(errMsg, wantPath) {
    t.Errorf("error should contain path %q, got: %q", wantPath, errMsg)
}
```

V6 test assertions use strict exact match everywhere:
```go
// V6: exact match only
expected := "cannot add dependency - creates cycle: tick-bbb \u2192 tick-aaa \u2192 tick-bbb"
if err.Error() != expected {
    t.Errorf("expected error %q, got %q", expected, err.Error())
}
```

V5 imports `"strings"` for `Contains`; V6 does not need it.

**Test scenario differences:**

The two versions set up their graph edges differently in some tests. For example, in the 2-node cycle test:

V5: `tick-bbb` blocked by `tick-aaa`; adding `tick-aaa` blocked by `tick-bbb` -> path: `tick-aaa -> tick-bbb -> tick-aaa`
V6: `tick-aaa` blocked by `tick-bbb`; adding `tick-bbb` blocked by `tick-aaa` -> path: `tick-bbb -> tick-aaa -> tick-bbb`

Both test the same logical scenario (2-node cycle) but from opposite perspectives. Both are correct.

**Batch test difference:**

V5 batch test puts the failing dep second (`["tick-ccc", "tick-bbb"]`) to prove tick-ccc is checked first (passes), then tick-bbb fails. V6 batch test puts the failing dep first (`["tick-aaa", "tick-ccc"]`) -- this still validates fail-on-first, but doesn't prove that valid deps before the failing one actually pass. V5's ordering is a stronger test because it demonstrates both pass-then-fail behavior.

**Error severity on test failure:**

V5 uses `t.Fatalf` for unexpected nil errors (stops test immediately) and `t.Errorf` for wrong messages (continues). V6 uses `t.Fatal` for nil checks and `t.Errorf` for message checks -- same pattern but V5 uses `t.Fatalf` with format string while V6 uses `t.Fatal` with plain string.

For success-path tests, V5 uses `t.Fatalf("unexpected error: %v", err)` which prints the error; V6 uses `t.Errorf("expected no error for valid dependency, got: %v", err)` which logs but does not stop the test. V5's `Fatalf` is more appropriate since subsequent assertions would be meaningless if the call errored.

| Aspect | V5 | V6 |
|--------|-----|-----|
| Test count | 11 | 11 |
| Spec test coverage | All 11 spec tests | All 11 spec tests |
| Assertion style | `Contains` + exact match | Exact match only |
| Batch test ordering | Failing dep second (stronger) | Failing dep first (weaker) |
| Success-path failure severity | `t.Fatalf` (stops test) | `t.Errorf` (continues) |
| Imports | `strings`, `testing` | `testing` only |

### Skill Compliance

| Skill Constraint | V5 | V6 |
|------------------|-----|-----|
| Use gofmt and golangci-lint on all code | PASS -- standard formatting | PASS -- standard formatting |
| Handle all errors explicitly | PASS -- all paths return or check errors | PASS -- all paths return or check errors |
| Write table-driven tests with subtests | PARTIAL -- uses subtests but not table-driven; each test is a distinct `t.Run` closure | PARTIAL -- uses subtests but not table-driven; each test is a distinct `t.Run` closure |
| Document all exported functions, types, and packages | PASS -- `ValidateDependency`, `ValidateDependencies` documented | PASS -- `ValidateDependency`, `ValidateDependencies` documented |
| Propagate errors with fmt.Errorf("%w", err) | N/A -- errors are terminal, not wrapped | N/A -- errors are terminal, not wrapped |
| MUST NOT ignore errors | PASS | PASS |
| MUST NOT use panic | PASS | PASS |
| MUST NOT hardcode configuration | PASS | PASS |

### Spec-vs-Convention Conflicts

1. **Error casing:** Spec says `"Cannot add dependency"`. V5 matches with `"Cannot"`. V6 uses `"cannot"` following Go convention (error strings should not be capitalized per Go Code Review Comments). Neither includes the spec's `"Error: "` prefix, which is standard Go practice since `err.Error()` already contextualizes.

2. **Child-blocked-by-parent error format:** Spec says `"Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-parent"`. V5 adds extra rationale: `"(would create unworkable task due to leaf-only ready rule)"`. V6 matches the spec format (minus "Error: " prefix and casing). V5 is more informative but deviates from spec; V6 is closer to spec format.

3. **ID normalization:** V5 normalizes all IDs via `NormalizeID` (lowercase) for case-insensitive comparison. V6 uses raw string equality. The spec does not explicitly require case-insensitive matching for dependency validation, but the codebase has `NormalizeID` and uses it elsewhere (e.g., in `Validate()` for self-reference and parent checks in `task.go`). V6's omission is inconsistent with the codebase convention.

4. **Table-driven tests:** The skill requires "table-driven tests with subtests." Neither version uses table-driven tests for this task -- both use individual `t.Run` subtests with distinct closures. The task spec itself lists 11 named tests rather than a pattern amenable to table-driven structure, so the individual-subtest approach is reasonable.

5. **Cycle path format:** Spec gives `"tick-a -> tick-b -> tick-a"`. Both versions use the Unicode arrow `\u2192` (right arrow) in `strings.Join`. This matches the spec's arrow notation.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed (task-relevant) | 2 (dependency.go, dependency_test.go) | 2 (dependency.go, dependency_test.go) |
| New files created | 2 | 2 |
| Implementation lines added | 110 | 99 |
| Test lines added | 180 | 190 |
| Total lines added | 290 | 289 |
| Functions (exported) | 2 (`ValidateDependency`, `ValidateDependencies`) | 2 (`ValidateDependency`, `ValidateDependencies`) |
| Functions (unexported) | 1 (`getOrigID`) | 3 (`validateChildBlockedByParent`, `detectCycle`, `dfs`) |
| Graph traversal algorithm | BFS (iterative queue) | DFS (recursive) |
| ID normalization calls | 5 | 0 |
| Task list iterations | 1 (single pass builds both maps) | 2 (one for parent check, one for map) |

## Verdict

**V5 is the stronger implementation.**

Both versions produce correct, working dependency validation with identical public APIs and both cover all 11 spec tests. The implementations are close in total line count (290 vs 289). The differences come down to correctness, consistency, and test rigor.

**V5 advantages:**
- **ID normalization throughout** -- uses `NormalizeID` consistently, matching the codebase convention established in `task.go` where `NormalizeID` is used for validation. V6 omits normalization entirely, creating a correctness gap if IDs with different casing are ever encountered.
- **Single-pass map construction** -- builds `origID`, `blockedByMap`, and extracts `taskParent` in one iteration over `tasks`. V6 iterates twice (once in `validateChildBlockedByParent`, once in `detectCycle`).
- **Original-ID preservation** -- maintains `origID` map to display original-case IDs in error messages even when comparison is case-insensitive. V6 has no such concern but only because it skips normalization.
- **Stronger batch test** -- places the failing dependency second in the list, proving that the first (valid) dependency passes before the second fails. V6 places the failing one first, which tests fail-on-first but not pass-before-fail.
- **Test failure severity** -- uses `t.Fatalf` for unexpected errors on success paths, stopping the test immediately rather than continuing with meaningless assertions (V6 uses `t.Errorf`).

**V6 advantages:**
- **Cleaner function decomposition** -- 4 focused functions vs V5's single 82-line function. Each V6 function has a clear single responsibility: orchestration, parent validation, cycle detection, DFS traversal.
- **DFS with backtracking is more memory-efficient** -- mutates a shared path slice in-place via pointer, avoiding V5's BFS pattern of `make`+`copy` for every queue entry's path.
- **Simpler adjacency map** -- directly references `tasks[i].BlockedBy` slices rather than normalizing and copying each element, though this saves correctness for brevity.
- **Child-blocked-by-parent error message** -- closer to spec format (no extra rationale parenthetical).
- **Error casing** -- lowercase `"cannot"` follows Go Code Review Comments convention (error strings should not be capitalized).
- **Empty-string guard** -- `newBlockedByID != ""` in `validateChildBlockedByParent` adds defensive protection against edge case of empty blocked-by ID.

**V6 weaknesses:**
- **Missing ID normalization** is the most significant issue. The codebase provides `NormalizeID` and uses it in `Validate()` for the same task struct fields. V6's `dependency.go` completely ignores it, creating inconsistency and a potential bug if task IDs are ever provided with variant casing.
- **Two iterations over tasks** instead of one is a minor efficiency concern but indicates less thoughtful data flow design.
- **Recursive DFS** could theoretically stack overflow on very deep dependency chains (thousands of hops), though this is unlikely in practice.

V5's correctness advantage (ID normalization) is the deciding factor. The BFS approach is slightly more verbose but avoids recursion risks and preserves original IDs for display. V6's superior decomposition is a legitimate design advantage, but the normalization omission is a gap that affects correctness guarantees in a way that matters for a CLI tool where users may type IDs in any case.
