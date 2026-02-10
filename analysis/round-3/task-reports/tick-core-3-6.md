# Task 3-6: Parent scoping -- --parent flag with recursive descendant CTE (V5 Only)

## Task Plan Summary

This task adds a `--parent <id>` flag to `tick list` (and by extension `tick ready` and `tick blocked`) that restricts query results to descendants of the specified parent task. The implementation uses a recursive CTE in SQLite to collect all descendant IDs, then applies existing query filters (ready, blocked, status, priority) within that narrowed set. The parent task itself is excluded from results. Key behaviors include: non-existent parent ID returns an error, parent with no descendants returns empty result (exit 0), deep nesting (3+ levels) is supported, and all existing filters compose cleanly as AND conditions.

## Note

This task exists only in V5. V4 does not implement the --parent flag. This is a standalone quality assessment, not a comparison.

## V5 Implementation

### Architecture & Design

The implementation follows an elegant **pre-filter / post-filter architecture** that is the core design insight of this task. The `--parent` flag narrows the candidate set (pre-filter), and all existing filters (ready, blocked, status, priority) apply unchanged within that set (post-filters). This avoids special-case logic entirely.

**Key architectural decisions:**

1. **Delegation pattern for `runReady` and `runBlocked`**: Both commands were refactored from ~50 lines of duplicated query logic into 3-line delegators:

   ```go
   // ready.go lines 31-33
   func runReady(ctx *Context) error {
       ctx.Args = append([]string{"--ready"}, ctx.Args...)
       return runList(ctx)
   }
   ```

   ```go
   // blocked.go lines 34-36
   func runBlocked(ctx *Context) error {
       ctx.Args = append([]string{"--blocked"}, ctx.Args...)
       return runList(ctx)
   }
   ```

   This is a significant refactoring win. Instead of needing to add `--parent` support to three separate command handlers, the implementation consolidates all list/ready/blocked logic into `runList`. The `ready` and `blocked` commands simply prepend their respective flags and delegate. This means `--parent` automatically works with `tick ready --parent X` and `tick blocked --parent X` with zero additional code.

2. **Recursive CTE for descendant collection** (`list.go` lines 179-203): The CTE is implemented exactly as specified in the plan:

   ```go
   query := `
   WITH RECURSIVE descendants(id) AS (
     SELECT id FROM tasks WHERE parent = ?
     UNION ALL
     SELECT t.id FROM tasks t
     JOIN descendants d ON t.parent = d.id
   )
   SELECT id FROM descendants`
   ```

   The CTE starts from direct children of the parent (`WHERE parent = ?`) and recursively joins, collecting all descendants at any depth. The parent itself is never included since the seed query selects children *of* the parent, not the parent itself.

3. **Two-phase query execution within `store.Query`** (`list.go` lines 238-257): The parent validation, descendant collection, and main query all execute within a single `store.Query` callback. This ensures they share the same database connection and benefit from the same lock/freshness guarantee. This is a sound design choice -- the parent existence check and descendant query are read operations that should be consistent with the main query.

4. **`appendDescendantFilter` as a composable query builder** (`list.go` lines 161-174): The function takes a query string and params, and appends an `AND id IN (...)` clause if descendantIDs is non-nil. Using `nil` as the sentinel for "no parent filter" (vs an empty slice for "parent has no descendants") is a clean semantic distinction.

5. **`buildWrappedFilterQuery` consolidation** (`list.go` lines 119-137): The implementation further refactored `buildReadyFilterQuery` and `buildBlockedFilterQuery` into thin wrappers around a shared `buildWrappedFilterQuery`. This eliminates duplication between the two wrapped query builders while keeping the API stable.

### Code Quality

**Strengths:**

- **All errors are explicitly handled with `fmt.Errorf("%w", ...)` wrapping** throughout the new code. For example:
  - `list.go` line 191: `return nil, fmt.Errorf("querying descendants: %w", err)`
  - `list.go` line 199: `return nil, fmt.Errorf("scanning descendant ID: %w", err)`
  - `list.go` line 210: `return nil, fmt.Errorf("checking parent task: %w", err)`

- **Every exported and unexported function has a doc comment** explaining its purpose. Examples:
  - `list.go` line 161: `// appendDescendantFilter adds an AND id IN (...) clause...`
  - `list.go` line 177: `// queryDescendantIDs executes a recursive CTE to collect all descendant task IDs...`
  - `list.go` line 206: `// parentTaskExists checks whether a task with the given ID exists...`

- **Proper use of `defer rows.Close()`** at `list.go` line 193 to avoid resource leaks.

- **Case-insensitive ID normalization** at `list.go` line 72: `f.parent = task.NormalizeID(args[i])` -- delegates to the shared `NormalizeID` function which calls `strings.ToLower()`, consistent with all other ID handling in the codebase.

- **Parameterized SQL** throughout -- no string interpolation of user input into SQL. The `appendDescendantFilter` generates `?` placeholders and appends values to the params slice.

- **Clean error message**: `list.go` line 247: `return fmt.Errorf("Task '%s' not found", filters.parent)` -- matches the exact error format specified in the plan.

**Minor observations:**

- **`appendDescendantFilter` checks for `nil` not `len == 0`** (`list.go` line 165): `if descendantIDs == nil`. This is deliberate -- `nil` means "no --parent filter" while an empty non-nil slice would mean "parent exists but has no descendants". However, the empty-slice case is handled earlier in `runList` (line 254: `if len(descendantIDs) == 0 { return nil }`), so `appendDescendantFilter` never receives an empty non-nil slice. While correct, this implicit contract could be more explicitly documented. If the early return were ever removed, `appendDescendantFilter` would produce `AND id IN ()` which is valid but always-false SQL -- not a crash, but potentially confusing.

- **`[]interface{}` instead of `[]any`**: The code uses `[]interface{}` (e.g., `list.go` line 98). In Go 1.18+ `any` is an alias for `interface{}`, and modern Go style prefers `any`. This is a minor stylistic point and consistent with the rest of the codebase.

- **`ctx.Args` mutation in `runReady`/`runBlocked`**: `ctx.Args = append([]string{"--ready"}, ctx.Args...)` modifies the `Args` slice on the shared `Context`. This is safe because the context is used once per invocation and not shared concurrently, but it is a slight mutation-of-input pattern. No bug here, but worth noting.

### Test Coverage

The test file `parent_scope_test.go` contains **15 test cases** organized under a single `TestParentScope` parent test, using Go subtests (`t.Run`). This matches the 15 tests specified in the plan exactly.

**Test-by-test coverage assessment:**

| Plan Test | Implemented | Lines |
|-----------|-------------|-------|
| "it returns all descendants of parent (direct children)" | Yes | 13-40 |
| "it returns all descendants recursively (3+ levels deep)" | Yes | 42-70 |
| "it excludes parent task itself from results" | Yes | 72-97 |
| "it returns empty result when parent has no descendants" | Yes | 99-115 |
| "it errors with 'Task not found' for non-existent parent ID" | Yes | 117-133 |
| "it returns only ready tasks within parent scope with tick ready --parent" | Yes | 135-164 |
| "it returns only blocked tasks within parent scope with tick blocked --parent" | Yes | 166-196 |
| "it combines --parent with --status filter" | Yes | 198-231 |
| "it combines --parent with --priority filter" | Yes | 233-263 |
| "it combines --parent with --ready and --priority" | Yes | 265-298 |
| "it combines --parent with --blocked and --status" | Yes | 300-327 |
| "it handles case-insensitive parent ID" | Yes | 329-347 |
| "it excludes tasks outside the parent subtree" | Yes | 349-377 |
| "it outputs IDs only with --quiet within scoped set" | Yes | 379-415 |
| "it returns 'No tasks found.' when descendants exist but none match filters" | Yes | 417-437 |

**100% coverage of plan-specified tests.**

**Test quality observations:**

- Tests are **integration-level** -- they exercise the full stack from CLI entry point (`Run(...)`) through flag parsing, store creation, SQL execution, and output formatting. This validates the entire chain.

- Each test creates an isolated temp directory via `initTickProjectWithTasks`, ensuring no cross-test contamination.

- Tests use both **positive and negative assertions**: they check that expected tasks appear AND that unexpected tasks do not appear.

- The "ready within parent scope" test (line 135) correctly sets up a task blocked by an *external* blocker (outside the parent subtree) to verify the ready filter works within the scoped set.

- The "blocked within parent scope" test (line 166) correctly sets up an outside-blocked task to verify it is excluded from parent-scoped blocked results.

- The 3-level deep recursion test (line 42) tests grandparent -> parent -> child -> grandchild, verifying all three descendants appear.

- The case-insensitive test (line 329) passes `"TICK-AAAAAA"` and expects results, confirming normalization works.

**Potential gaps:**

- No unit test for `appendDescendantFilter` in isolation, though it is thoroughly exercised through integration tests.
- No unit test for `queryDescendantIDs` in isolation, though again exercised through integration tests.
- No test for `--parent` with an empty string value (edge case: `tick list --parent ""`). The `NormalizeID` would return `""`, and `parentTaskExists` would check for a task with ID `""` which would not exist, returning "Task '' not found". This is acceptable behavior.
- No benchmark tests for the recursive CTE with large hierarchies. Given this is a CLI tool, this is acceptable.

### Spec Compliance

Checking each acceptance criterion from the plan:

| Criterion | Status | Evidence |
|-----------|--------|----------|
| `tick list --parent <id>` returns only descendants | Pass | Test "direct children" and "excludes tasks outside subtree" |
| `tick ready --parent <id>` returns only ready descendants | Pass | Test "ready tasks within parent scope" |
| `tick blocked --parent <id>` returns only blocked descendants | Pass | Test "blocked tasks within parent scope" |
| Parent task excluded from results | Pass | Test "excludes parent task itself"; CTE starts from `WHERE parent = ?` |
| Non-existent parent returns error | Pass | Test "errors with Task not found"; error message matches spec format |
| Parent with no descendants returns empty | Pass | Test "empty result when parent has no descendants" |
| Deep nesting (3+ levels) | Pass | Test "recursively 3+ levels deep" with 4-level hierarchy |
| Composes with --status | Pass | Test "combines --parent with --status" |
| Composes with --priority | Pass | Test "combines --parent with --priority" |
| Composes with --ready | Pass | Test "combines --parent with --ready and --priority" |
| Composes with --blocked | Pass | Test "combines --parent with --blocked and --status" |
| Case-insensitive parent ID | Pass | Test "handles case-insensitive parent ID" |
| --quiet outputs IDs only within scoped set | Pass | Test "outputs IDs only with --quiet" |

**All 13 acceptance criteria are satisfied.**

Additionally, the implementation follows the plan's specified approach:
- Recursive CTE matches the exact SQL from the plan (Do step 1)
- `--parent` flag registered on list command (Do step 2)
- Parent ID validated with existence check (Do step 3)
- Integration as pre-filter before other filters (Do step 4)
- Composition verified with all filter combinations (Do step 5)

### golang-pro Skill Compliance

Assessing against the skill's constraints:

**MUST DO compliance:**

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Handle all errors explicitly | Pass | All error returns checked and wrapped with `%w` |
| Write table-driven tests with subtests | Partial | Uses subtests (`t.Run`) but not table-driven format. Each test is a standalone `t.Run` block. This is acceptable for integration tests where each scenario has unique setup, but deviates from the skill's preference for table-driven tests. |
| Document all exported functions/types | Pass | All functions have doc comments |
| Propagate errors with `fmt.Errorf("%w", err)` | Pass | All error wrapping uses `%w` |
| Use `context.Context` to all blocking operations | N/A | No blocking operations introduced (all SQL queries are fast local operations) |

**MUST NOT DO compliance:**

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Ignore errors | Pass | No ignored errors |
| Use panic for normal error handling | Pass | No panics |
| Create goroutines without lifecycle management | Pass | No goroutines introduced |
| Hardcode configuration | Pass | No hardcoded config |

**Additional skill requirements:**

- **Interface definitions (contracts first)**: The existing `Formatter` interface is reused; no new interfaces were needed.
- **Proper package structure**: All new code is in `internal/cli`, consistent with the project structure.
- **Test file with table-driven tests**: Tests use subtests but not table-driven patterns. For integration tests with varying setup (different task hierarchies, different command args), standalone subtests are arguably more readable than a table. This is a minor deviation.

## Quality Assessment

### Strengths

1. **Excellent refactoring of `runReady` and `runBlocked`**: Eliminating ~100 lines of duplicated query/output logic across three command handlers was an outstanding design decision. The delegation pattern (`ctx.Args = append([]string{"--ready"}, ctx.Args...)`) is simple, correct, and makes `--parent` available to all three commands automatically. This goes beyond the task plan's requirements and improves the overall codebase quality.

2. **Clean pre-filter / post-filter separation**: The `--parent` flag collects descendant IDs, which are passed as a separate parameter through the query builder chain. At no point does `--parent` logic need to know about ready/blocked/status/priority semantics. This is textbook separation of concerns.

3. **Defensive handling of edge cases**: The empty-descendants early return (`list.go` lines 253-256) avoids generating an `AND id IN ()` SQL clause. The `nil` vs empty slice distinction in `appendDescendantFilter` is semantically clear.

4. **DRY query construction**: The introduction of `buildWrappedFilterQuery` consolidates ready/blocked wrapping logic into a single function, further reducing duplication beyond what the task strictly required.

5. **Comprehensive test coverage**: 15 tests covering all specified scenarios, including multi-level composition tests (e.g., `--parent + --ready + --priority`). Each test has both inclusion and exclusion assertions.

6. **The recursive CTE is correct and efficient**: It uses the existing `idx_tasks_parent` index, starts from children (excluding the parent itself naturally), and uses `UNION ALL` (not `UNION`) which is correct since task IDs are unique and there can be no duplicates in a tree structure.

### Weaknesses

1. **Potential scalability concern with `IN (...)` clause**: The `appendDescendantFilter` generates `AND id IN (?,?,?,...,?)` with one placeholder per descendant. For very large subtrees (thousands of tasks), this could hit SQLite's `SQLITE_MAX_VARIABLE_NUMBER` limit (default 999 in older SQLite, 32766 in newer). A more scalable approach would be to use a temporary table or embed the CTE directly in the main query. For a CLI task tracker, this is unlikely to be a practical problem, but it is a theoretical limitation.

2. **`appendDescendantFilter` nil-check contract is implicit**: The function checks `descendantIDs == nil` but the only caller ensures empty slices are handled by early return. If future code changes the caller's behavior, a `len(descendantIDs) == 0` case would produce valid but empty-result SQL. Adding a comment or handling both nil and empty-slice explicitly would be more defensive.

3. **No `--parent` flag documentation in `printUsage`**: The `printUsage` function in `cli.go` (lines 184-210) lists commands and global flags but does not document subcommand-specific flags like `--parent`. This is consistent with how `--ready`, `--blocked`, `--status`, and `--priority` are also not documented in the help output, so this is a systemic gap rather than a task-specific weakness.

4. **Error message format inconsistency**: The error `"Task '%s' not found"` (`list.go` line 247) uses a capital "T" in "Task", while the plan specifies `"Error: Task '<id>' not found"`. The "Error: " prefix is actually added by the `Run` function's error handler (`cli.go` line 89), so the full output matches the plan. However, other error messages in the codebase (like `"tick directory does not exist"` in `engine/store.go`) use lowercase. The uppercase "Task" is intentional to match the plan and looks correct in context.

5. **Test uses `--pretty` flag inconsistently**: Some tests pass `--pretty` (e.g., line 80 "excludes parent task itself", line 105 "no descendants", line 426 "no match filters") while others omit it (e.g., line 24 "direct children"). This does not affect correctness since the tests check for `strings.Contains` on task IDs regardless of output format, but the inconsistency suggests the tests were written or modified at different times. The `--pretty` flag is needed when checking for "No tasks found." to ensure the output format is predictable.

### Overall Quality Rating

**Excellent**

This is a high-quality implementation that goes beyond the plan requirements. The refactoring of `runReady` and `runBlocked` into delegators was a significant architectural improvement that reduced code duplication by ~100 lines while simultaneously enabling the `--parent` flag to work with all three commands automatically. The recursive CTE is implemented correctly and efficiently. Error handling, documentation, and test coverage are thorough. All 13 acceptance criteria are met, all 15 specified tests are implemented, and the code follows idiomatic Go patterns. The weaknesses identified are minor and theoretical rather than practical.
