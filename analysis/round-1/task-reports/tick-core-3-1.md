# Task tick-core-3-1: Dependency validation -- cycle detection & child-blocked-by-parent

## Task Summary

Reject two categories of invalid dependencies at write time:

1. **Circular dependencies** (deadlocks) -- DFS/BFS from `newBlockedByID` following `blocked_by` edges; return error with full cycle path.
2. **Child-blocked-by-parent** -- if a task's parent equals `newBlockedByID`, reject (unworkable due to leaf-only ready rule).

Required functions:
- `ValidateDependency(tasks []Task, taskID, newBlockedByID string) error`
- `ValidateDependencies(tasks, taskID, blockedByIDs)` -- batch, sequential, fail-fast

Error formats specified:
- Cycle: `Error: Cannot add dependency - creates cycle: tick-a -> tick-b -> tick-a`
- Parent: `Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-parent`

Allowed: parent blocked by child, sibling deps, cross-hierarchy deps. Pure domain logic, no I/O.

### Acceptance Criteria (from plan)

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

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| 1. Self-reference rejected | PASS -- explicit `taskID == newBlockedByID` check returns error | PASS -- explicit check with `NormalizeID`, error says "creates cycle" | PASS -- handled by `detectCycle` self-reference check returning `[]string{taskID, taskID}` |
| 2. 2-node cycle with path | PASS -- BFS detects, path includes both nodes + arrows | PASS -- DFS detects, exact path format `tick-bbb -> tick-aaa -> tick-bbb` | PASS -- BFS detects, exact path format |
| 3. 3+ node cycle with full path | PASS -- BFS reconstructs full path through chain | PASS -- DFS reconstructs full path, test asserts exact string | PASS -- BFS reconstructs full path, test asserts exact string |
| 4. Child-blocked-by-parent rejected | PASS -- checks `t.Parent == newBlockedByID` | PASS -- checks with `NormalizeID` normalization | PASS -- checks with `NormalizeID` normalization, adds extra explanation in error |
| 5. Parent-blocked-by-child allowed | PASS -- tested, no error | PASS -- tested, no error | PASS -- tested, no error |
| 6. Sibling/cross-hierarchy allowed | PASS -- both tested | PASS -- both tested | PASS -- both tested |
| 7. Error messages match spec format | PARTIAL -- self-ref says "task cannot be blocked by itself" (no "Cannot add dependency" prefix); cycle uses em-dash instead of hyphen; child-blocked-by-parent uses em-dash | PASS -- exact match: "Cannot add dependency - creates cycle: ..." and "Cannot add dependency - %s cannot be blocked by its parent %s" | PARTIAL -- lowercase "cannot" instead of "Cannot"; adds extra line "(would create unworkable task due to leaf-only ready rule)" |
| 8. Batch validation fails on first error | PASS -- `ValidateDependencies` loops, returns first error | PASS -- identical pattern | PASS -- identical pattern |
| 9. Pure functions -- no I/O | PASS -- no imports beyond fmt/strings | PASS -- no I/O in added code | PASS -- no I/O in added code |

## Implementation Comparison

### Approach

#### File Organization

**V1** and **V3** create a dedicated `internal/task/dependency.go` file (with matching `dependency_test.go`). **V2** appends the validation functions into the existing `internal/task/task.go` (and tests into `task_test.go`).

V1/V3's approach is cleaner -- separating dependency validation into its own file follows the single-responsibility principle. V2's approach conflates task model/creation logic with dependency graph validation in the same file.

#### ID Normalization

**V1** performs NO ID normalization -- it compares raw string IDs directly:
```go
// V1 dependency.go line 13
if taskID == newBlockedByID {
    return fmt.Errorf("task cannot be blocked by itself")
}
```

**V2** and **V3** both normalize IDs via `NormalizeID()` (lowercase normalization) before all comparisons:
```go
// V2 task.go (added lines)
normalizedTaskID := NormalizeID(taskID)
normalizedBlockedByID := NormalizeID(newBlockedByID)
```
```go
// V3 dependency.go line 13-14
normalizedTaskID := NormalizeID(taskID)
normalizedBlockedByID := NormalizeID(newBlockedByID)
```

This is a meaningful difference. V1 would fail to detect cycles if IDs were passed in different cases (e.g., `TICK-AAA` vs `tick-aaa`). V2 and V3 are more robust.

#### Cycle Detection Algorithm

All three use BFS/DFS graph traversal with path tracking:

**V1** -- BFS with queue (slice used as FIFO):
```go
// V1 dependency.go lines 54-83
type entry struct {
    id   string
    path []string
}
queue := []entry{{id: startID, path: []string{startID}}}
// ... dequeue from front: queue[0] / queue = queue[1:]
```

**V2** -- DFS with stack (slice used as LIFO):
```go
// V2 task.go (detectCycle function)
type frame struct {
    id   string
    path []string
}
stack := []frame{{id: startID, path: []string{startID}}}
// ... pop from end: stack[len(stack)-1] / stack = stack[:len(stack)-1]
```

**V3** -- BFS with queue (same pattern as V1):
```go
// V3 dependency.go lines 107-145
type node struct {
    id   string
    path []string
}
queue := []node{{id: newBlockedByID, path: []string{newBlockedByID}}}
// ... dequeue from front: queue[0] / queue = queue[1:]
```

For cycle detection in DAGs, BFS vs DFS are functionally equivalent. However, BFS finds the shortest path first, which means V1 and V3 will report the shortest cycle when multiple cycles exist, while V2's DFS may report a longer path depending on traversal order.

#### Self-Reference Handling

**V1** handles self-reference as a separate check BEFORE cycle detection with a distinct error message:
```go
// V1 dependency.go line 13-14
if taskID == newBlockedByID {
    return fmt.Errorf("task cannot be blocked by itself")
}
```

**V2** handles self-reference as a separate check with a "creates cycle" message that matches the spec format:
```go
// V2 task.go
if normalizedTaskID == normalizedBlockedByID {
    return fmt.Errorf("Cannot add dependency - creates cycle: %s -> %s", taskID, taskID)
}
```

**V3** handles self-reference INSIDE `detectCycle` rather than in the main function:
```go
// V3 dependency.go line 67-69
if taskID == newBlockedByID {
    return []string{taskID, taskID}
}
```
This means self-reference goes through the same "creates cycle" error path, which is semantically correct (self-reference IS a cycle) and produces a consistent error format.

#### Child-blocked-by-parent Check Order

**V1** and **V2** check child-blocked-by-parent BEFORE cycle detection.
**V3** checks child-blocked-by-parent FIRST (before cycle detection), same order, but adds an explanatory suffix to the error message:
```go
// V3 dependency.go line 24-27
return fmt.Errorf("cannot add dependency - %s cannot be blocked by its parent %s\n       (would create unworkable task due to leaf-only ready rule)",
    normalizedTaskID, normalizedBlockedByID)
```

#### Cycle Path Format

**V1** builds path during BFS and appends targetID at the end:
```go
// V1: path starts with startID (newBlockedByID), found nodes appended, targetID added at end
// Result for 2-node: [tick-aaa001, tick-bbb002, tick-ccc003] joined with " -> "
return append(current.path, targetID)
```
The path format is: `newBlockedByID -> ... -> taskID`, which means the path starts from the blocker and ends at the task being blocked. This does NOT match the spec format which starts from `taskID`.

**V2** builds path during DFS, then constructs the full cycle in `ValidateDependency`:
```go
// V2: detectCycle returns path from startID to near-targetID
// ValidateDependency then wraps: taskID + path + taskID
parts := append(parts, taskID)
for _, id := range path { parts = append(parts, ...) }
parts = append(parts, taskID)
```
Result: `taskID -> newBlockedByID -> ... -> taskID`. This matches the spec format: the cycle starts and ends with the task that would be blocked.

**V3** builds path during BFS starting from `newBlockedByID`, then appends `newBlockedByID` again at the end:
```go
// V3: cyclePath starts with newBlockedByID, ends with taskID, then newBlockedByID appended
cyclePath := append(newPath, newBlockedByID)
```
Result: `newBlockedByID -> ... -> taskID -> newBlockedByID`. Same as V2's format: the cycle starts and ends with the blocker ID. Both V2 and V3 tests assert exact path strings matching this format.

#### Error Message Format

The spec requires: `"Error: Cannot add dependency - creates cycle: tick-a -> tick-b -> tick-a"`

| Aspect | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Self-ref error | `"task cannot be blocked by itself"` | `"Cannot add dependency - creates cycle: tick-aaa -> tick-aaa"` | `"cannot add dependency - creates cycle: tick-aaaaaa -> tick-aaaaaa"` |
| Cycle error prefix | `"Cannot add dependency \u2014 creates cycle: "` (em-dash) | `"Cannot add dependency - creates cycle: "` (hyphen) | `"cannot add dependency - creates cycle: "` (lowercase c, hyphen) |
| Parent error prefix | `"Cannot add dependency \u2014 "` (em-dash) | `"Cannot add dependency - "` (hyphen) | `"cannot add dependency - "` (lowercase c, hyphen) |
| Parent error suffix | none | none | `"\n       (would create unworkable task...)"` |

V2 matches the spec format most closely. V1 uses em-dashes (`\u2014`) instead of hyphens. V3 uses lowercase "cannot" instead of uppercase "Cannot".

#### detectCycle Return Signature

**V1**: `func detectCycle(...) []string` -- returns nil or path slice.
**V2**: `func detectCycle(...) ([]string, bool)` -- returns path + boolean, Go-idiomatic "comma ok" pattern.
**V3**: `func detectCycle(...) []string` -- returns nil or path slice, same as V1.

V2's approach is more idiomatic Go for a "found or not" function.

#### Task Map Value Type

**V1** stores pointers in the map:
```go
taskMap := make(map[string]*Task, len(tasks))
for i := range tasks {
    taskMap[tasks[i].ID] = &tasks[i]
}
```

**V2** stores values:
```go
taskMap := make(map[string]Task, len(tasks))
for _, t := range tasks {
    taskMap[NormalizeID(t.ID)] = t
}
```

**V3** stores values:
```go
taskByID := make(map[string]Task)
for _, t := range tasks {
    taskByID[NormalizeID(t.ID)] = t
}
```

V1's pointer approach avoids copying Task structs and is slightly more efficient. V2 and V3 copy tasks into the map. V1 also pre-sizes the map (`len(tasks)`), V2 pre-sizes, V3 does not -- a minor efficiency gap.

#### Code Verbosity

V3's `detectCycle` function is 179 lines total for the file, with extensive inline comments explaining the BFS logic step by step (approximately 30 lines of comments within `detectCycle` alone). V1 is 92 lines. V2 adds 105 lines to an existing file. The core logic is equivalent across all three; V3 is significantly more verbose due to comments.

### Code Quality

#### Naming

- **V1**: `entry` struct with `id`/`path` fields, `taskMap`, `detectCycle`, `formatPath`
- **V2**: `frame` struct with `id`/`path` fields, `taskMap`, `detectCycle`
- **V3**: `node` struct with `id`/`path` fields, `taskByID`, `detectCycle`

V3's `taskByID` is the most descriptive map name. V2's `frame` is less intuitive than V1's `entry` or V3's `node` for a graph traversal context.

#### Separation of Concerns

**V1**: `formatPath` is extracted as a separate helper:
```go
func formatPath(path []string) string {
    return strings.Join(path, " -> ")
}
```

**V2** and **V3** inline the `strings.Join` call directly. V1's extraction is marginally cleaner but for a one-liner it is arguably over-engineering.

#### Error Handling Consistency

**V1** has inconsistent error prefixes: self-reference says `"task cannot be blocked by itself"` while cycle/parent errors say `"Cannot add dependency..."`. V2 and V3 use consistent prefixes for all error cases.

#### ID Normalization in detectCycle

**V1** does NOT normalize IDs in `detectCycle` -- it compares raw BlockedBy values against raw targetID:
```go
// V1 detectCycle
for _, blockerID := range t.BlockedBy {
    if blockerID == targetID {
```

**V2** normalizes inside `detectCycle`:
```go
// V2 detectCycle
for _, dep := range task.BlockedBy {
    normalizedDep := NormalizeID(dep)
    if normalizedDep == targetID {
```

**V3** also normalizes inside `detectCycle`:
```go
// V3 detectCycle
for _, blockedBy := range task.BlockedBy {
    normalizedBlockedBy := NormalizeID(blockedBy)
    if normalizedBlockedBy == taskID {
```

V2 and V3 are defensively correct. V1 could miss cycles if `BlockedBy` entries have different casing than the task IDs.

#### V3 Extra Existence Check

V3 adds an early return if `newBlockedByID` doesn't exist in the task map:
```go
// V3 dependency.go line 102-104
if _, exists := taskByID[newBlockedByID]; !exists {
    return nil
}
```
This is a minor optimization that avoids BFS when the blocker task doesn't exist. V1 and V2 would simply traverse an empty BFS (since the starting node has no blocked_by edges), achieving the same result less explicitly.

### Test Quality

#### V1 Test Functions (dependency_test.go, 163 lines)

**TestValidateDependency** (9 subtests):
1. `"allows valid dependency between unrelated tasks"` -- two unrelated tasks, expects nil error
2. `"rejects direct self-reference"` -- same ID for task and blocker, expects error
3. `"rejects 2-node cycle with path"` -- A blocked by B, add B blocked by A; asserts "cycle" and arrow in message
4. `"rejects 3+ node cycle with full path"` -- A->B->C chain, add C->A; asserts "cycle" and all three IDs in message
5. `"rejects child blocked by own parent"` -- child with Parent field, blocked by parent; asserts "parent" in message
6. `"allows parent blocked by own child"` -- reverse of above, expects nil
7. `"allows sibling dependencies"` -- two children of same parent, expects nil
8. `"allows cross-hierarchy dependencies"` -- children of different parents, expects nil
9. `"detects cycle through existing multi-hop chain"` -- 4-node chain A->B->C->D, add D->A; asserts "cycle"

**TestValidateDependencies** (2 subtests):
1. `"validates multiple blocked_by IDs, fails on first error"` -- second ID is self-ref, expects error
2. `"passes when all dependencies valid"` -- three unrelated tasks, expects nil

**Total: 11 test functions**

V1 test style: uses `strings.Contains` for most assertions rather than exact string matching. Does NOT assert exact error message format for most cases. Self-reference test only checks `err != nil`, not the message content.

#### V2 Test Functions (task_test.go, 181 lines added)

**TestValidateDependency** (9 subtests):
1. `"it allows valid dependency between unrelated tasks"` -- two unrelated tasks, expects nil
2. `"it rejects direct self-reference"` -- asserts `strings.Contains(err.Error(), "creates cycle")`
3. `"it rejects 2-node cycle with path"` -- exact string match: `"Cannot add dependency - creates cycle: tick-bbb -> tick-aaa -> tick-bbb"`
4. `"it rejects 3+ node cycle with full path"` -- exact string match: `"Cannot add dependency - creates cycle: tick-ccc -> tick-aaa -> tick-bbb -> tick-ccc"`
5. `"it rejects child blocked by own parent"` -- exact string match: `"Cannot add dependency - tick-child cannot be blocked by its parent tick-parent"`
6. `"it allows parent blocked by own child"` -- expects nil
7. `"it allows sibling dependencies"` -- expects nil
8. `"it allows cross-hierarchy dependencies"` -- expects nil
9. `"it returns cycle path format: tick-a -> tick-b -> tick-a"` -- exact string match for 2-node cycle format
10. `"it detects cycle through existing multi-hop chain"` -- 4-node chain, exact string match: `"Cannot add dependency - creates cycle: tick-d -> tick-a -> tick-b -> tick-c -> tick-d"`

**TestValidateDependencies** (1 subtest):
1. `"it validates multiple blocked_by IDs, fails on first error"` -- tick-bbb blocked by [tick-ccc, tick-aaa] where tick-aaa creates cycle; asserts "creates cycle"

**Total: 11 test functions**

V2 test style: uses exact string matching for cycle and parent errors (5 exact assertions). Prefixes test names with `"it ..."` matching spec test names. Most thorough error message validation.

#### V3 Test Functions (dependency_test.go, 248 lines)

**TestValidateDependency** (9 subtests):
1. `"it allows valid dependency between unrelated tasks"` -- expects nil
2. `"it rejects direct self-reference"` -- asserts `strings.Contains(err.Error(), "creates cycle")`
3. `"it rejects 2-node cycle with path"` -- exact string match: `"cannot add dependency - creates cycle: tick-aaaaaa -> tick-bbbbbb -> tick-aaaaaa"`
4. `"it rejects 3+ node cycle with full path"` -- exact string match for 3-node cycle
5. `"it rejects child blocked by own parent"` -- exact string match including the extra explanation line
6. `"it allows parent blocked by own child"` -- expects nil
7. `"it allows sibling dependencies"` -- expects nil
8. `"it allows cross-hierarchy dependencies"` -- expects nil
9. `"it returns cycle path format: tick-a -> tick-b -> tick-a"` -- 4-node cycle, exact string match
10. `"it detects cycle through existing multi-hop chain"` -- 3-node chain with different IDs (tick-x, tick-y, tick-z), asserts "creates cycle"

**TestValidateDependencies** (3 subtests):
1. `"it validates multiple blocked_by IDs, fails on first error"` -- asserts "creates cycle"
2. `"it succeeds when all dependencies are valid"` -- expects nil
3. `"it returns empty error for empty blocked_by list"` -- tests both nil and empty slice

**Total: 12 test functions**

V3 test style: uses helper functions `makeTask` and `makeTaskWithParent` to reduce boilerplate. Tasks are created with full field population (Title, Status, Priority, Created, Updated), making them more realistic. Uses exact string matching for most error assertions.

#### Test Helper Usage

**V1** and **V2**: inline Task struct literals directly in each test.
**V3**: uses `makeTask` and `makeTaskWithParent` helpers:
```go
makeTask := func(id string, blockedBy ...string) Task {
    return Task{
        ID: id, Title: "Task " + id, Status: StatusOpen,
        Priority: 2, BlockedBy: blockedBy,
        Created: "2026-01-19T10:00:00Z", Updated: "2026-01-19T10:00:00Z",
    }
}
```
This reduces repetition but also means the helpers are redefined in both `TestValidateDependency` and `TestValidateDependencies` (duplicated).

#### Test Gaps

| Test | V1 | V2 | V3 |
|------|-----|-----|-----|
| Self-reference error message content | NO (only checks err != nil) | YES (checks "creates cycle") | YES (checks "creates cycle") |
| Exact cycle path format (2-node) | NO (checks Contains) | YES (exact match) | YES (exact match) |
| Exact cycle path format (3-node) | NO (checks Contains) | YES (exact match) | YES (exact match) |
| Exact parent error message | NO (checks Contains "parent") | YES (exact match) | YES (exact match with extra text) |
| Exact multi-hop cycle path | NO (checks Contains "cycle") | YES (exact match) | NO (checks Contains "creates cycle") |
| All-valid batch | YES | NO | YES |
| Empty batch input | NO | NO | YES (nil and empty slice) |
| Cycle path format test (spec test name) | NO (separate test absent) | YES (dedicated subtest) | YES (dedicated subtest, uses 4-node) |

**Tests unique to V1**: "passes when all dependencies valid" batch test (V3 also has this).
**Tests unique to V2**: None -- all V2 tests are present in at least one other version.
**Tests unique to V3**: Empty blocked_by list (nil and empty slice) for `ValidateDependencies`.

**Tests in all 3**: allows valid dependency, rejects self-reference, rejects 2-node cycle, rejects 3+ node cycle, rejects child-blocked-by-parent, allows parent-blocked-by-child, allows siblings, allows cross-hierarchy, detects multi-hop cycle, batch fails on first error.

V2 has the strongest exact-match assertions (all error messages checked character-by-character). V3 has the most test functions (12) and tests the empty-input edge case that neither V1 nor V2 covers. V1 has the weakest assertions, relying mostly on `strings.Contains` rather than exact matches.

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 2 | 4 (2 impl-relevant) | 5 (2 impl-relevant) |
| Lines added | 255 | 289 (286 impl-relevant) | 444 (427 impl-relevant) |
| Impl LOC | 92 | 105 | 179 |
| Test LOC | 163 | 181 | 248 |
| Test functions | 11 | 11 | 12 |

Note: V2 and V3 also modify docs/workflow files (planning status + implementation log), which are non-functional changes. V3 additionally creates a `tick-core-context.md` doc file.

## Verdict

**V2 is the best implementation** of this task, with V3 as a close second.

**Why V2 wins:**

1. **Spec compliance**: V2's error messages exactly match the specified format (`"Cannot add dependency - creates cycle: ..."` and `"Cannot add dependency - %s cannot be blocked by its parent %s"`). V1 uses em-dashes and inconsistent self-reference messages. V3 uses lowercase "cannot" and adds an unsolicited explanation line to the parent error.

2. **ID normalization**: V2 normalizes all IDs via `NormalizeID()` at every comparison point, matching V3 and surpassing V1 which performs no normalization at all. This is critical for correctness given the codebase's established pattern of case-insensitive IDs.

3. **Test rigor**: V2 uses exact string matching for 5 error messages (2-node cycle, 3-node cycle, 4-node multi-hop cycle, child-blocked-by-parent, cycle path format). This catches any regression in error formatting. V1 only uses `strings.Contains`, which would pass even if the format changed substantially.

4. **Go idioms**: V2's `detectCycle` returns `([]string, bool)`, the standard Go "comma ok" pattern. V1 and V3 use nil-vs-non-nil slice, which works but is less idiomatic.

5. **Code conciseness**: V2 is 105 implementation lines vs V3's 179 (93 of V3's extra lines are verbose inline comments explaining BFS logic that is straightforward to any Go developer). V1 is most concise at 92 lines but lacks normalization.

**V2's one weakness**: It places the code in `task.go` rather than a dedicated `dependency.go` file. V1 and V3's file separation is architecturally cleaner. However, this is a minor organizational issue that doesn't affect correctness or maintainability significantly at this codebase's current size.

**V3's strengths over V2**: more test functions (12 vs 11), tests the empty-input edge case, uses test helpers to reduce boilerplate, and uses a separate file. V3's weakness is the spec-divergent error format (lowercase, extra explanation text) and excessive verbosity in comments.

**V1 is weakest**: no ID normalization (correctness risk), inconsistent error messages (em-dashes, non-standard self-reference message), and weakest test assertions (Contains-only, no exact format checks).
