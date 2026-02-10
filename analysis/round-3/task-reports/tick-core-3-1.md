# Task tick-core-3-1: Dependency validation -- cycle detection & child-blocked-by-parent

## Task Summary

This task implements pure validation functions that reject two categories of invalid dependencies at write time: circular dependencies (deadlocks) and child-blocked-by-parent (unworkable due to leaf-only ready rule). The primary function `ValidateDependency(tasks []Task, taskID, newBlockedByID string) error` checks both cycle and child-blocked-by-parent conditions. Cycle detection uses DFS/BFS from `newBlockedByID` following `blocked_by` edges to determine if `taskID` is reachable, returning the full cycle path on error. Child-blocked-by-parent checks if the task's direct parent equals `newBlockedByID`. A batch function `ValidateDependencies` validates multiple blocked_by IDs sequentially, failing on the first error. All functions are pure domain logic with no I/O.

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

| Criterion | V4 | V5 |
|-----------|-----|-----|
| Self-reference rejected | PASS -- `checkCycle` returns error when `taskID == newBlockedByID` with message `"Error: Cannot add dependency - creates cycle: tick-aaa -> tick-aaa"`; tested in `"it rejects direct self-reference"` | PASS -- inline check `if normTaskID == normNewDep` returns error `"Cannot add dependency - creates cycle: tick-aaa -> tick-aaa"`; tested in `"it rejects direct self-reference"` |
| 2-node cycle detected with path | PASS -- BFS in `checkCycle` finds cycle; tested in `"it rejects 2-node cycle with path"` with exact match `"Error: Cannot add dependency - creates cycle: tick-aaa -> tick-bbb -> tick-aaa"` | PASS -- BFS finds cycle; tested in `"it rejects 2-node cycle with path"` using `strings.Contains` for path `"tick-aaa -> tick-bbb -> tick-aaa"` |
| 3+ node cycle detected with full path | PASS -- BFS with `reconstructPath` traces full chain; tested in `"it rejects 3+ node cycle with full path"` with exact match on 3-node path | PASS -- BFS with inline path tracking traces full chain; tested in `"it rejects 3+ node cycle with full path"` using `strings.Contains` on 3-node path |
| Child-blocked-by-parent rejected with descriptive error | PASS -- `checkChildBlockedByParent` scans tasks for parent match; error `"Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-parent"`; tested with exact match | PASS -- inline parent check after map build; error `"Cannot add dependency - tick-child cannot be blocked by its parent tick-parent (would create unworkable task due to leaf-only ready rule)"`; tested with exact match |
| Parent-blocked-by-child allowed | PASS -- tested in `"it allows parent blocked by own child"` | PASS -- tested in `"it allows parent blocked by own child"` |
| Sibling/cross-hierarchy deps allowed | PASS -- tested in `"it allows sibling dependencies"` and `"it allows cross-hierarchy dependencies"` | PASS -- tested in `"it allows sibling dependencies"` and `"it allows cross-hierarchy dependencies"` |
| Error messages match spec format | PARTIAL -- includes `"Error: "` prefix matching spec exactly; uses unicode arrow matching spec | FAIL -- omits `"Error: "` prefix that spec requires (spec says `"Error: Cannot add dependency - creates cycle: ..."`); also appends `"(would create unworkable task due to leaf-only ready rule)"` to child-blocked-by-parent message which spec does not include |
| Batch validation fails on first error | PASS -- `ValidateDependencies` iterates and returns first error; tested in `"it validates multiple blocked_by IDs, fails on first error"` with child-blocked-by-parent as the failing case | PASS -- `ValidateDependencies` iterates and returns first error; tested in `"it validates multiple blocked_by IDs, fails on first error"` with cycle as the failing case |
| Pure functions -- no I/O | PASS -- all functions operate only on `[]Task` slice, no file/network operations | PASS -- all functions operate only on `[]Task` slice, no file/network operations |

## Implementation Comparison

### Approach

**V4: Separated helper functions with BFS + path reconstruction**

V4 decomposes the validation into four well-separated helper functions:

1. `ValidateDependency` -- orchestrates checks, calling child-blocked-by-parent first, then cycle detection
2. `checkChildBlockedByParent` -- linear scan of tasks for parent match
3. `checkCycle` -- builds blocked-by map, runs BFS, uses `reconstructPath` for cycle path
4. `buildBlockedByMap` -- creates `map[string][]string` lookup
5. `reconstructPath` -- traces parent map backwards from end to start, then reverses
6. `formatCyclePath` -- joins `taskID + path + taskID` with unicode arrow

The orchestration in `ValidateDependency`:

```go
func ValidateDependency(tasks []Task, taskID, newBlockedByID string) error {
    if err := checkChildBlockedByParent(tasks, taskID, newBlockedByID); err != nil {
        return err
    }
    if err := checkCycle(tasks, taskID, newBlockedByID); err != nil {
        return err
    }
    return nil
}
```

The BFS cycle detection uses a `parent` map for path reconstruction:

```go
parent := make(map[string]string)
visited := make(map[string]bool)
queue := []string{newBlockedByID}
visited[newBlockedByID] = true

for len(queue) > 0 {
    current := queue[0]
    queue = queue[1:]
    for _, blocker := range blockedByMap[current] {
        if blocker == taskID {
            path := reconstructPath(parent, newBlockedByID, current)
            return fmt.Errorf(
                "Error: Cannot add dependency - creates cycle: %s",
                formatCyclePath(taskID, path),
            )
        }
        if !visited[blocker] {
            visited[blocker] = true
            parent[blocker] = current
            queue = append(queue, blocker)
        }
    }
}
```

Path reconstruction traces backwards through the parent map:

```go
func reconstructPath(parent map[string]string, start, end string) []string {
    path := []string{end}
    current := end
    for current != start {
        current = parent[current]
        path = append(path, current)
    }
    for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
        path[i], path[j] = path[j], path[i]
    }
    return path
}
```

V4 does NOT use `NormalizeID` anywhere in the dependency validation code. All comparisons are case-sensitive using raw task IDs.

**V5: Monolithic function with BFS + inline path tracking and ID normalization**

V5 puts all logic into a single `ValidateDependency` function with inline checks rather than extracted helpers:

1. `ValidateDependency` -- contains self-reference check, map building, child-blocked-by-parent check, and BFS cycle detection all in one function
2. `ValidateDependencies` -- batch wrapper (same as V4)
3. `getOrigID` -- small helper for ID display fallback

Key architectural difference: V5 normalizes all IDs via `NormalizeID` for case-insensitive comparison:

```go
normTaskID := NormalizeID(taskID)
normNewDep := NormalizeID(newBlockedByID)

if normTaskID == normNewDep {
    return fmt.Errorf("Cannot add dependency - creates cycle: %s → %s", taskID, taskID)
}
```

V5 builds both an `origID` map (normalized -> original) and a `blockedByMap` in a single pass:

```go
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

V5's BFS carries the path inline in each queue node rather than using a separate parent map:

```go
type node struct {
    id   string
    path []string
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
                strings.Join(cyclePath, " → "),
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

**Key structural differences:**

1. **ID normalization:** V5 normalizes all IDs via `NormalizeID(strings.ToLower)` for case-insensitive matching and maintains an `origID` map to display original-case IDs in error messages. V4 uses raw IDs throughout, relying on IDs being normalized upstream. V5's approach is more defensive and self-contained, but adds complexity (the `getOrigID` helper, the `origID` map, normalizing every blocked_by entry).

2. **Function decomposition:** V4 extracts 6 functions total (`ValidateDependency`, `ValidateDependencies`, `checkChildBlockedByParent`, `checkCycle`, `buildBlockedByMap`, `reconstructPath`, `formatCyclePath`). V5 has 3 functions (`ValidateDependency`, `ValidateDependencies`, `getOrigID`). V4's decomposition follows the Go principle of small, focused functions with single responsibilities. V5's monolithic approach is denser but keeps all logic visible in one place.

3. **Path tracking:** V4 uses a `parent` map during BFS and reconstructs the path after cycle detection by tracing backwards. V5 carries the path forward in each queue node as a `[]string` slice. V5's approach copies the path slice on each enqueue (`make` + `copy` + `append`), which allocates more memory for deep graphs. V4's approach only allocates the parent map entries and reconstructs once at the end, which is more memory-efficient for large graphs.

4. **Child-blocked-by-parent timing:** V4 checks child-blocked-by-parent BEFORE building the blocked-by map (via a separate linear scan). V5 checks it AFTER building the map (since it extracts the parent during the same pass). V4's approach means invalid parent deps are rejected without the cost of building the full map. V5's approach is more efficient in the happy path since it only traverses tasks once.

5. **Error message format:** V4 prefixes errors with `"Error: "` matching the spec exactly. V5 omits the `"Error: "` prefix. V5 also appends explanatory context `"(would create unworkable task due to leaf-only ready rule)"` to the child-blocked-by-parent error, which the spec does not include.

### Code Quality

**Function decomposition and naming:**

V4's helper functions have clear, descriptive names:
- `checkChildBlockedByParent` -- exactly describes what it does
- `checkCycle` -- concise and accurate
- `buildBlockedByMap` -- clear purpose
- `reconstructPath` -- describes the algorithm step
- `formatCyclePath` -- describes the formatting step

Each function has a doc comment:

```go
// checkChildBlockedByParent rejects a dependency where a task would be
// blocked by its own direct parent. This creates an unworkable state
// due to the leaf-only ready rule.
func checkChildBlockedByParent(tasks []Task, taskID, newBlockedByID string) error {
```

```go
// buildBlockedByMap creates a lookup from task ID to its blocked_by list.
func buildBlockedByMap(tasks []Task) map[string][]string {
```

V5's `ValidateDependency` is 66 lines of logic in a single function (excluding the struct type definition), which is approaching the threshold where extraction would improve readability. The `getOrigID` helper is documented:

```go
// getOrigID returns the original (non-normalized) ID for display purposes.
// It prefers the ID stored in the origID map; if not found, falls back to
// the provided fallback.
func getOrigID(origID map[string]string, fallback, normKey string) string {
```

**Error handling:**

V4: All errors are returned explicitly. No ignored errors. Error messages include full context:

```go
return fmt.Errorf(
    "Error: Cannot add dependency - %s cannot be blocked by its parent %s",
    taskID, newBlockedByID,
)
```

V5: Same explicit error handling pattern. Error messages include more context but deviate from spec:

```go
return fmt.Errorf(
    "Cannot add dependency - %s cannot be blocked by its parent %s (would create unworkable task due to leaf-only ready rule)",
    taskID, newBlockedByID,
)
```

**Memory allocation patterns:**

V4's `buildBlockedByMap` pre-allocates the map:

```go
m := make(map[string][]string, len(tasks))
for _, t := range tasks {
    if len(t.BlockedBy) > 0 {
        m[t.ID] = t.BlockedBy
    }
}
```

This directly references the existing `t.BlockedBy` slices rather than copying them, which is efficient. V5 builds new slices via `append`:

```go
for _, dep := range t.BlockedBy {
    blockedByMap[nid] = append(blockedByMap[nid], NormalizeID(dep))
}
```

This creates new slice allocations for every dependency because it normalizes each entry. V5 also allocates the `origID` map. For small task lists this is negligible, but V4's approach is more memory-efficient.

V5's path-per-node BFS also allocates a new slice for every enqueued node:

```go
newPath := make([]string, len(current.path))
copy(newPath, current.path)
newPath = append(newPath, getOrigID(origID, dep, dep))
queue = append(queue, node{id: dep, path: newPath})
```

V4's parent-map approach allocates only a single `map[string]string` entry per visited node, then reconstructs the path once at the end. This is O(n) space vs V5's O(n * path_length) in the worst case.

**Test helper:**

V4 defines a `depTask` helper to reduce boilerplate in test task construction:

```go
func depTask(id string, parent string, blockedBy ...string) Task {
    return Task{
        ID:        id,
        Title:     "Test task " + id,
        Status:    StatusOpen,
        Priority:  2,
        BlockedBy: blockedBy,
        Parent:    parent,
    }
}
```

V5 constructs tasks inline using struct literals:

```go
{ID: "tick-aaa", BlockedBy: nil},
{ID: "tick-bbb", BlockedBy: []string{"tick-aaa"}},
```

V4's helper is more DRY -- it sets `Title`, `Status`, and `Priority` automatically, ensuring valid tasks. V5's inline construction omits required fields (`Title`, `Status`, `Priority`), which works because Go zero-values them, but the resulting Task structs have empty titles and zero priority, which are technically invalid per the model constraints (though the dependency validation does not check these). V4's approach is more robust from a test hygiene perspective.

### Test Quality

#### V4 Test Functions

All tests are under `TestValidateDependency` (1 top-level function) and `TestValidateDependencies` (1 top-level function), using subtests:

**`TestValidateDependency`** (9 subtests):

1. `"it allows valid dependency between unrelated tasks"` -- two unrelated tasks `tick-aaa` and `tick-bbb`, validates no error returned. Uses `t.Errorf`.
2. `"it rejects direct self-reference"` -- single task `tick-aaa`, validates error returned, checks `strings.Contains(err.Error(), "creates cycle")` and `strings.Contains(err.Error(), "tick-aaa")`. Uses `t.Fatal` for nil check.
3. `"it rejects 2-node cycle with path"` -- `tick-bbb` blocked by `tick-aaa`, adding `tick-aaa` blocked by `tick-bbb`. Exact match: `"Error: Cannot add dependency - creates cycle: tick-aaa -> tick-bbb -> tick-aaa"`. Uses `t.Fatal` + `t.Errorf`.
4. `"it rejects 3+ node cycle with full path"` -- `tick-bbb` blocked by `tick-aaa`, `tick-ccc` blocked by `tick-bbb`, adding `tick-aaa` blocked by `tick-ccc`. Exact match: `"Error: Cannot add dependency - creates cycle: tick-aaa -> tick-ccc -> tick-bbb -> tick-aaa"`.
5. `"it rejects child blocked by own parent"` -- `tick-child` with parent `tick-parent`, adding `tick-child` blocked by `tick-parent`. Exact match: `"Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-parent"`.
6. `"it allows parent blocked by own child"` -- `tick-parent` blocked by `tick-child` (where child has parent `tick-parent`). Validates no error. Uses `t.Errorf`.
7. `"it allows sibling dependencies"` -- `tick-sib2` blocked by `tick-sib1` (both children of `tick-parent`). Validates no error.
8. `"it allows cross-hierarchy dependencies"` -- `tick-c1` (child of `tick-p1`) blocked by `tick-c2` (child of `tick-p2`). Validates no error.
9. `"it returns cycle path format: tick-a -> tick-b -> tick-a"` -- uses shorter IDs `tick-a` and `tick-b`. Exact match: `"Error: Cannot add dependency - creates cycle: tick-a -> tick-b -> tick-a"`.

**`TestValidateDependencies`** (2 subtests):

10. `"it validates multiple blocked_by IDs, fails on first error"` -- `tick-child` with parent `tick-parent`, batch `["tick-other", "tick-parent"]`. Exact match on child-blocked-by-parent error for `tick-parent`.
11. `"it succeeds when all dependencies are valid"` -- `tick-aaa` with batch `["tick-bbb", "tick-ccc"]`. Validates no error.

**V4 edge cases covered:** 9 + 2 = 11 test scenarios matching all 11 specified tests in the plan.

**V4 additional test: `"it detects cycle through existing multi-hop chain"`** -- 4-node chain `tick-a -> tick-b -> tick-c -> tick-d`, adding `tick-a` blocked by `tick-d`. Exact match: `"Error: Cannot add dependency - creates cycle: tick-a -> tick-d -> tick-c -> tick-b -> tick-a"`. This is test #10 under `TestValidateDependency`, the 10th subtest.

Total V4: 2 top-level functions, 12 subtests.

#### V5 Test Functions

All tests are under `TestValidateDependency` (1 top-level function) and `TestValidateDependencies` (1 top-level function):

**`TestValidateDependency`** (9 subtests):

1. `"it allows valid dependency between unrelated tasks"` -- inline struct construction `{ID: "tick-aaa", BlockedBy: nil}`. Uses `t.Fatalf`.
2. `"it rejects direct self-reference"` -- single task, checks `err == nil` only. Uses `t.Fatal`. Does NOT verify error message content.
3. `"it rejects 2-node cycle with path"` -- uses `strings.Contains` for `"creates cycle"` and `"tick-aaa -> tick-bbb -> tick-aaa"`. Does NOT do exact match.
4. `"it rejects 3+ node cycle with full path"` -- uses `strings.Contains` for `"tick-aaa -> tick-ccc -> tick-bbb -> tick-aaa"`. Does NOT do exact match.
5. `"it rejects child blocked by own parent"` -- exact match: `"Cannot add dependency - tick-child cannot be blocked by its parent tick-parent (would create unworkable task due to leaf-only ready rule)"`.
6. `"it allows parent blocked by own child"` -- validates no error. Uses `t.Fatalf`.
7. `"it allows sibling dependencies"` -- validates no error.
8. `"it allows cross-hierarchy dependencies"` -- validates no error.
9. `"it returns cycle path format: tick-a -> tick-b -> tick-a"` -- exact match: `"Cannot add dependency - creates cycle: tick-a -> tick-b -> tick-a"`.

**`TestValidateDependencies`** (1 subtest):

10. `"it validates multiple blocked_by IDs, fails on first error"` -- `tick-aaa` with batch `["tick-ccc", "tick-bbb"]`, where `tick-bbb` is blocked by `tick-aaa` creating a cycle. Uses `strings.Contains` to check `"tick-bbb"` appears in error.

**V5 edge cases covered:** 9 + 1 = 10 test scenarios.

**V5 additional test: `"it detects cycle through existing multi-hop chain"`** -- 4-node chain, adding `tick-a` blocked by `tick-d`. Uses `strings.Contains` for path. This is the 10th subtest under `TestValidateDependency`.

Total V5: 2 top-level functions, 11 subtests.

#### Test Coverage Diff

| Edge Case | V4 | V5 |
|-----------|-----|-----|
| Valid dependency between unrelated tasks | Yes | Yes |
| Direct self-reference rejection | Yes (checks "creates cycle" + task ID in message) | Yes (checks error non-nil only -- no message verification) |
| 2-node cycle with path | Yes (exact match) | Yes (`strings.Contains`) |
| 3+ node cycle with full path | Yes (exact match) | Yes (`strings.Contains`) |
| Child blocked by parent | Yes (exact match) | Yes (exact match, different message) |
| Parent blocked by child allowed | Yes | Yes |
| Sibling dependencies allowed | Yes | Yes |
| Cross-hierarchy dependencies allowed | Yes | Yes |
| Cycle path format verification | Yes (exact match with `Error:` prefix) | Yes (exact match without `Error:` prefix) |
| Multi-hop chain cycle (4 nodes) | Yes (exact match) | Yes (`strings.Contains`) |
| Batch fails on first error | Yes (child-blocked-by-parent) | Yes (cycle error) |
| Batch succeeds when all valid | Yes | No -- MISSING |
| Self-reference error message content | Yes (checks "creates cycle" and task ID) | No (only checks error is non-nil) |

**Notable coverage differences:**

- **V4 has an extra test:** `"it succeeds when all dependencies are valid"` in `TestValidateDependencies`. V5 only tests the failure case for batch validation, not the success case. This is a test gap in V5.
- **V4 assertions are stronger:** For most cycle detection tests, V4 uses exact string match (`err.Error() != want`) while V5 uses substring match (`strings.Contains`). V4's exact matches catch any regression in error message formatting. V5's substring matches are more lenient, which makes them less precise at catching regressions.
- **V4 verifies self-reference error content:** V4's self-reference test checks both `"creates cycle"` and the task ID appear in the error message. V5 only checks that an error was returned, not its content. This means V5 would pass even if the self-reference returned a completely wrong error message.
- **V4 test helpers produce valid tasks:** V4's `depTask` helper sets Title, Status, and Priority. V5's inline structs leave these zero-valued.

### Skill Compliance

| Constraint | V4 | V5 |
|------------|-----|-----|
| Handle all errors explicitly (no naked returns) | PASS -- all error returns are explicit; no `_` assignments | PASS -- all error returns are explicit; no `_` assignments |
| Write table-driven tests with subtests | PARTIAL -- uses subtests via `t.Run` throughout, but no table-driven tests; each case is an individual subtest | PARTIAL -- uses subtests via `t.Run` throughout, but no table-driven tests; each case is an individual subtest |
| Document all exported functions, types, and packages | PASS -- `ValidateDependency` and `ValidateDependencies` are both documented with doc comments explaining parameters, behavior, and purity constraint | PASS -- `ValidateDependency` and `ValidateDependencies` are both documented with doc comments explaining parameters, behavior, and purity constraint |
| Propagate errors with `fmt.Errorf("%w", err)` | N/A -- these functions create errors, they don't wrap existing ones | N/A -- these functions create errors, they don't wrap existing ones |
| No hardcoded configuration | PASS -- no magic values | PASS -- no magic values |
| No panic for normal error handling | PASS -- no panics | PASS -- no panics |
| Avoid `_` assignment without justification | PASS -- no ignored errors | PASS -- no ignored errors |
| Use gofmt and golangci-lint | PASS -- code is properly formatted | PASS -- code is properly formatted |

### Spec-vs-Convention Conflicts

**1. Error message `"Error:"` prefix**

- **Spec says:** `"Error: Cannot add dependency - creates cycle: tick-a -> tick-b -> tick-a"` and `"Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-parent"` -- both start with `"Error: "`.
- **Go convention:** Error strings should not be capitalized or end with punctuation. The `"Error: "` prefix is unusual for Go error values, but this is a user-facing CLI error message.
- **V4 chose:** Includes `"Error: "` prefix, matching spec exactly.
- **V5 chose:** Omits `"Error: "` prefix, producing `"Cannot add dependency - creates cycle: ..."`.
- **Assessment:** The spec explicitly defines the error format with the `"Error: "` prefix. For pure domain validation functions that return `error` values, the Go convention would be to omit the prefix and let the CLI layer add it. V5's choice is more Go-idiomatic for a library function, but the spec is explicit. Since these errors are defined in the task plan with exact format strings, V4's literal compliance is the safer interpretation. This is a judgment call where both approaches have merit, but V4 aligns with the spec.

**2. Extra context in child-blocked-by-parent error**

- **Spec says:** `"Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-parent"`
- **V5 adds:** `"(would create unworkable task due to leaf-only ready rule)"` -- additional context not in spec.
- **V4 chose:** Matches spec exactly.
- **Assessment:** V5's addition is informative but deviates from the spec format. The spec defines exact error messages. Adding unsolicited context could break downstream parsing or test expectations. V4's approach of matching the spec exactly is preferable.

No other spec-vs-convention conflicts identified.

## Diff Stats

| Metric | V4 | V5 |
|--------|-----|-----|
| Files changed | 4 (dependency.go, dependency_test.go, 2 docs) | 4 (dependency.go, dependency_test.go, 2 docs) |
| Lines added (total) | 351 | 293 |
| Lines added (internal/) | 348 | 290 |
| Impl LOC (dependency.go) | 135 | 110 |
| Test LOC (dependency_test.go) | 213 | 180 |
| Top-level test functions | 2 | 2 |
| Total test subtests | 12 | 11 |

## Verdict

**V4 is the better implementation.**

The margin is moderate, driven by three factors:

1. **Spec compliance (V4 advantage):** V4's error messages exactly match the spec-defined formats including the `"Error: "` prefix. V5 omits this prefix and adds unsolicited explanatory text to the child-blocked-by-parent error. The task plan defines explicit error format strings, and V4 matches them. This is a clear compliance gap in V5.

2. **Test rigor (V4 advantage):** V4 has 12 subtests vs V5's 11 -- V4 includes a batch success test that V5 omits. More importantly, V4 uses exact string matching for most error assertions (`err.Error() != want`), while V5 uses substring matching (`strings.Contains`) for cycle path errors. V4's self-reference test verifies the error message content ("creates cycle" and task ID), while V5 only checks that an error was returned. These differences mean V4's test suite catches more categories of regression. V4's `depTask` helper also produces more realistic test tasks with all required fields populated.

3. **Code decomposition (V4 advantage):** V4 extracts 6 focused helper functions with clear single responsibilities. V5 puts nearly all logic into a single 66-line `ValidateDependency` function. V4's approach is more aligned with Go's preference for small, readable functions, and makes the algorithm easier to follow and maintain.

4. **ID normalization (V5 advantage):** V5's case-insensitive ID comparison via `NormalizeID` is more defensive and consistent with the `task.go` model which defines `NormalizeID` for this purpose. V4 relies on IDs being normalized before reaching the validation layer. V5's approach is more self-contained, but adds complexity (origID map, extra allocations) that may be unnecessary if normalization is guaranteed upstream.

5. **Memory efficiency (V4 advantage):** V4's parent-map BFS with post-hoc path reconstruction is more memory-efficient than V5's path-per-node approach which allocates and copies a path slice for every enqueued node. V4's `buildBlockedByMap` references existing slices rather than creating new normalized copies. For small task lists the difference is negligible, but V4's approach scales better.

Overall, V4 wins on spec compliance, test quality, and code organization. V5's ID normalization is a genuine improvement in robustness, but it does not outweigh V4's advantages in the other areas. The spec compliance gap (missing `"Error: "` prefix and added unsolicited text) is V5's most significant shortcoming.
