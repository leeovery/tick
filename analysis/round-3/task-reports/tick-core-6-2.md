# Task 6-2: Extract shared ready/blocked SQL WHERE clauses (V5 Only -- Phase 6 Refinement)

## Task Plan Summary

This task addresses code duplication found during post-implementation analysis. `StatsReadyCountQuery` in `stats.go` duplicated the WHERE clause from `ReadyQuery` in `ready.go` (three conditions: status=open, NOT EXISTS unclosed blockers, NOT EXISTS open children). Similarly, `StatsBlockedCountQuery` duplicated the WHERE clause from `BlockedQuery` in `blocked.go`. Additionally, `buildReadyFilterQuery` and `buildBlockedFilterQuery` in `list.go` were near-identical functions (~15 lines each) differing only in which inner query constant they wrapped. The solution: extract shared WHERE clause constants (`readyWhereClause`, `blockedWhereClause`) and a shared `buildWrappedFilterQuery` function.

## Note

This is a Phase 6 analysis refinement task that only exists in V5. It addresses code duplication found during post-implementation analysis. This is a standalone quality assessment, not a comparison.

## V5 Implementation

### Architecture & Design

The implementation follows the plan precisely with a clean, idiomatic approach to eliminating SQL query duplication through Go's compile-time string constant concatenation.

**WHERE clause extraction** -- Two new unexported constants were introduced:

1. `readyWhereClause` in `/private/tmp/tick-analysis-worktrees/v5/internal/cli/ready.go` (lines 6-17):
```go
const readyWhereClause = `t.status = 'open'
  AND NOT EXISTS (
    SELECT 1 FROM dependencies d
    JOIN tasks blocker ON blocker.id = d.blocked_by
    WHERE d.task_id = t.id
      AND blocker.status NOT IN ('done', 'cancelled')
  )
  AND NOT EXISTS (
    SELECT 1 FROM tasks child
    WHERE child.parent = t.id
      AND child.status IN ('open', 'in_progress')
  )`
```

2. `blockedWhereClause` in `/private/tmp/tick-analysis-worktrees/v5/internal/cli/blocked.go` (lines 6-19):
```go
const blockedWhereClause = `t.status = 'open'
  AND (
    EXISTS (
      SELECT 1 FROM dependencies d
      JOIN tasks blocker ON blocker.id = d.blocked_by
      WHERE d.task_id = t.id
        AND blocker.status NOT IN ('done', 'cancelled')
    )
    OR EXISTS (
      SELECT 1 FROM tasks child
      WHERE child.parent = t.id
        AND child.status IN ('open', 'in_progress')
    )
  )`
```

Both constants are **unexported** (lowercase), which is the correct choice -- they are implementation details of the `cli` package, not part of its public API. They remain in their respective domain files (`ready.go`, `blocked.go`), preserving semantic locality.

**Constant composition** -- The exported query constants are now built via Go's compile-time string concatenation:

In `ready.go` (lines 22-27):
```go
const ReadyQuery = `
SELECT t.id, t.status, t.priority, t.title
FROM tasks t
WHERE ` + readyWhereClause + `
ORDER BY t.priority ASC, t.created ASC
`
```

In `stats.go` (lines 28-31):
```go
const StatsReadyCountQuery = `
SELECT COUNT(*) FROM tasks t
WHERE ` + readyWhereClause + `
`
```

The same pattern is applied for `BlockedQuery` and `StatsBlockedCountQuery`. This ensures the WHERE logic is defined exactly once and composed into both listing and counting queries at compile time. There is zero runtime overhead -- the Go compiler resolves these concatenations into single string literals.

**Shared filter query builder** -- In `/private/tmp/tick-analysis-worktrees/v5/internal/cli/list.go`, the previously duplicated `buildReadyFilterQuery` and `buildBlockedFilterQuery` are refactored to delegate to a shared function (lines 119-137):

```go
func buildWrappedFilterQuery(innerQuery, alias string, f listFilters, descendantIDs []string) (string, []interface{}) {
	q := `SELECT id, status, priority, title FROM (` + innerQuery + `) AS ` + alias + ` WHERE 1=1`
	var params []interface{}

	q, params = appendDescendantFilter(q, params, descendantIDs)

	if f.status != "" {
		q += ` AND status = ?`
		params = append(params, f.status)
	}
	if f.hasPri {
		q += ` AND priority = ?`
		params = append(params, f.priority)
	}

	return q, params
}
```

The two original functions are now thin wrappers (lines 110-117):

```go
func buildReadyFilterQuery(f listFilters, descendantIDs []string) (string, []interface{}) {
	return buildWrappedFilterQuery(ReadyQuery, "ready", f, descendantIDs)
}

func buildBlockedFilterQuery(f listFilters, descendantIDs []string) (string, []interface{}) {
	return buildWrappedFilterQuery(BlockedQuery, "blocked", f, descendantIDs)
}
```

The `alias` parameter is a nice touch -- it preserves the existing `AS ready` / `AS blocked` subquery aliases, which aids readability in query debugging and logging. Both wrapper functions remain to serve as semantic entry points so callers don't need to know the alias convention.

### Code Quality

**Net reduction**: The diff shows 45 insertions vs 70 deletions, a net reduction of 25 lines. The stats.go file alone lost 24 lines of duplicated SQL. This is an excellent signal for a pure-refactor deduplication task.

**Documentation quality**: Comments are updated throughout:
- The `readyWhereClause` comment (ready.go lines 3-5) explicitly states it is "Shared by ReadyQuery and StatsReadyCountQuery" -- this is important for maintainability, as it tells future developers that modifications will propagate to multiple queries.
- The `blockedWhereClause` comment (blocked.go lines 3-5) follows the same pattern.
- The `StatsReadyCountQuery` comment (stats.go lines 26-27) says "It reuses readyWhereClause so that the definition stays in one place."
- The `buildWrappedFilterQuery` comment (list.go lines 119-120) describes its purpose clearly.
- Redundant comment text that described the WHERE conditions in detail was properly removed from `ReadyQuery`, `BlockedQuery`, `StatsReadyCountQuery`, and `StatsBlockedCountQuery` since those details now live on the shared constants. The remaining comments on the composed queries are shorter but still meaningful.

**Naming conventions**: All names follow Go conventions. `readyWhereClause` and `blockedWhereClause` are descriptive, unexported, and use camelCase. `buildWrappedFilterQuery` clearly conveys its purpose. The `innerQuery` and `alias` parameter names are clear.

**SQL formatting**: The WHERE clause constants begin the SQL on the same line as the backtick-quoted string opening, which means the composed queries read naturally with `WHERE ` + readyWhereClause yielding valid SQL with proper indentation. The formatting is consistent and readable.

**No functional changes**: The refactoring is purely structural. The final SQL strings produced by constant concatenation are byte-for-byte identical to the originals. This is verifiable by comparing the pre/post versions of each query constant.

### Test Coverage

The plan explicitly states: "No new tests needed -- this is a pure refactor with existing coverage." The implementation correctly follows this guidance.

Existing test coverage is comprehensive across all affected code paths:

1. **Ready query tests** (`ready_test.go`): 9 test cases covering open tasks, blocker filtering, parent/child relationships, deep nesting, ordering, empty results, aligned output, quiet mode -- all exercise `ReadyQuery` which now uses `readyWhereClause`.

2. **Blocked query tests** (`blocked_test.go`): 6 test cases for `TestBlockedQuery` plus 3 for `TestBlockedCommand` plus 3 for `TestCancelUnblocksDependents` -- all exercise `BlockedQuery` which now uses `blockedWhereClause`.

3. **Stats tests** (`stats_test.go`): 7 test cases including "it counts ready and blocked tasks correctly" (line 60) which directly exercises `StatsReadyCountQuery` and `StatsBlockedCountQuery` through JSON output verification, confirming the shared WHERE clauses produce correct counts.

4. **List tests** (`list_test.go`): 16 test cases including "it filters to ready tasks with --ready" (line 171) and "it filters to blocked tasks with --blocked" (line 213) which exercise the `buildReadyFilterQuery` -> `buildWrappedFilterQuery` and `buildBlockedFilterQuery` -> `buildWrappedFilterQuery` code paths. "it combines --ready with --priority" (line 317) exercises the combined filter logic inside `buildWrappedFilterQuery`.

These tests collectively validate that the refactored query composition produces identical results to the original inlined queries.

### Spec Compliance

All four acceptance criteria from the task plan are met:

1. **"ReadyQuery and StatsReadyCountQuery share the same WHERE clause constant"** -- Both use `readyWhereClause` via compile-time concatenation (`ready.go` line 25, `stats.go` line 30).

2. **"BlockedQuery and StatsBlockedCountQuery share the same WHERE clause constant"** -- Both use `blockedWhereClause` via compile-time concatenation (`blocked.go` line 28, `stats.go` line 37).

3. **"buildReadyFilterQuery and buildBlockedFilterQuery are collapsed into calls to a shared function"** -- Both delegate to `buildWrappedFilterQuery` (`list.go` lines 111, 116).

4. **"All existing list, ready, blocked, and stats tests pass unchanged"** -- No test files were modified in the diff. All test assertions remain identical.

The four "Do" items from the plan are also fully satisfied:
- Item 1: `readyWhereClause` and `blockedWhereClause` defined as string constants.
- Item 2: `ReadyQuery` and `StatsReadyCountQuery` composed from `readyWhereClause`; same for blocked.
- Item 3: `buildWrappedFilterQuery` extracted in `list.go` with the specified signature `(innerQuery string, f listFilters, descendantIDs []string)` -- the actual signature adds `alias string` as a second parameter, which is a reasonable enhancement over the plan.
- Item 4: Existing tests verify no behavioral changes.

### golang-pro Skill Compliance

**MUST DO items**:
- **gofmt/golangci-lint**: The code follows standard Go formatting. No lint issues are apparent.
- **context.Context**: Not applicable -- this refactoring does not introduce blocking operations.
- **Error handling**: Not applicable -- no new error paths introduced; constants are resolved at compile time.
- **Table-driven tests**: Not applicable -- no new tests required for this pure refactoring.
- **Documentation**: All exported functions and constants have GoDoc comments. The new unexported constants also have comments explaining their shared purpose.
- **Error propagation**: Not applicable.

**MUST NOT DO items**:
- No errors are ignored.
- No panics introduced.
- No goroutines created.
- No reflection used.
- No hardcoded configuration.

The implementation is clean, minimal, and focused -- consistent with Go proverbs of simplicity and clarity.

## Quality Assessment

### Strengths

1. **Precise deduplication with zero runtime cost**: Using Go's compile-time string constant concatenation eliminates the WHERE clause duplication with no runtime overhead whatsoever. The final query strings are identical to the originals at the binary level. This is the ideal technique for this problem.

2. **Semantic locality preserved**: The `readyWhereClause` constant lives in `ready.go` alongside `ReadyQuery`, and `blockedWhereClause` lives in `blocked.go` alongside `BlockedQuery`. A developer modifying the ready/blocked logic will naturally find and update the single source of truth in the file they are already editing.

3. **Excellent change ratio**: 45 insertions, 70 deletions (net -25 lines) for a pure refactor demonstrates genuine deduplication without introducing unnecessary abstraction layers.

4. **Proper visibility**: Both WHERE clause constants are unexported, correctly treating them as internal implementation details. The public API (`ReadyQuery`, `BlockedQuery`, `StatsReadyCountQuery`, `StatsBlockedCountQuery`) is unchanged.

5. **Thoughtful alias parameter**: The `buildWrappedFilterQuery` function accepts an `alias` parameter, which preserves the existing `AS ready` / `AS blocked` subquery naming in the composed SQL. This is a small detail that aids debugging and maintains backward compatibility in SQL query plans.

6. **Wrapper functions retained**: Rather than having callers invoke `buildWrappedFilterQuery` directly with raw query strings and alias names, the two thin wrappers `buildReadyFilterQuery` and `buildBlockedFilterQuery` are retained. This preserves a clean, domain-specific API for `buildListQuery` to call, and encapsulates the query-to-alias mapping.

7. **Comments updated accurately**: Shared-purpose documentation on the constants prevents future developers from modifying one usage site without realizing the constant is shared.

### Weaknesses

1. **Minor: Plan signature deviation**: The plan specified `buildWrappedFilterQuery(innerQuery string, f listFilters, descendantIDs []string)` but the implementation adds `alias string` as a second parameter. This is a reasonable enhancement that improves SQL readability, but it is a minor deviation from the spec. In practice, this is a net positive.

2. **Minor: No compile-time verification test**: While the existing tests comprehensively cover the query behavior, there is no explicit test that asserts the string content of `ReadyQuery` contains the same WHERE conditions as `StatsReadyCountQuery`. Such a test could catch accidental divergence if someone were to edit the composed query constants directly (e.g., adding an extra condition to `ReadyQuery` but not through `readyWhereClause`). However, this risk is low given the constant composition pattern, and the existing behavioral tests would likely catch any such discrepancy.

3. **Negligible: SQL string boundary readability**: The `WHERE ` + readyWhereClause concatenation leaves a slightly unusual visual break in the SQL string at the WHERE keyword boundary. The constant starts with `t.status = 'open'` rather than including the `WHERE` keyword itself. This is the correct design choice (the constant represents only the conditions, not the clause keyword), but reading the composed query requires mental concatenation across lines 24-25 of `ready.go`. This is an inherent trade-off of the pattern and not a real issue.

### Overall Quality Rating

**Excellent** -- This is a textbook refactoring task executed with precision. The implementation eliminates all identified duplication (WHERE clauses in 4 query constants, filter builder functions in list.go) using Go's most appropriate mechanism (compile-time string concatenation for constants, function extraction for builders). The change is minimal, focused, and produces identical runtime behavior with zero overhead. All acceptance criteria are met. All existing tests pass unchanged. Documentation is updated accurately. The code follows idiomatic Go patterns throughout. The only deviation from the plan (adding the `alias` parameter) is a net improvement.
