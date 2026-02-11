# Task Report: tick-core-7-1 — Extract shared ready-query SQL conditions

**Commit:** `7f1a3c7` (V6 only)

## Task Summary

The task eliminates SQL duplication where "ready" and "blocked" WHERE conditions were independently authored in three locations across `list.go` and `stats.go`. The solution extracts these SQL fragments into a new `query_helpers.go` file, providing composable helper functions that both consumers reference. This ensures the definition of "ready" exists in exactly one place.

---

## V6 Implementation

### Architecture

The refactor introduces `/internal/cli/query_helpers.go` with a clear hierarchy of helpers:

- `ReadyNoUnclosedBlockers()` -- single NOT EXISTS subquery for blocker check
- `ReadyNoOpenChildren()` -- single NOT EXISTS subquery for children check
- `ReadyConditions()` -- composes status + both NOT EXISTS into a `[]string`
- `BlockedConditions()` -- open status + De Morgan inverse (EXISTS OR EXISTS)
- `ReadyWhereClause()` -- joins conditions with AND for inline SQL embedding

The layered design is sound. `ReadyConditions()` composes the two granular helpers, and `ReadyWhereClause()` is a convenience for `stats.go` which needs a single string rather than a slice. The file placement inside `internal/cli` is appropriate since both consumers live in that package.

**Concern: `BlockedConditions()` does not reuse `ReadyNoUnclosedBlockers()` / `ReadyNoOpenChildren()`.** The blocked SQL (EXISTS instead of NOT EXISTS) is hardcoded as a new string literal rather than being mechanically derived from the ready sub-expressions. If the ready conditions change (e.g., adding a new status exclusion), `BlockedConditions()` must be manually updated in tandem. This partially undermines the single-source-of-truth goal stated in the plan: "The blocked conditions can be derived as the negation." In practice, the blocked condition's EXISTS clauses mirror the ready NOT EXISTS clauses but are not programmatically derived from them. The duplication is now reduced from 3 sites to 2 co-located sites (same file), which is a significant improvement but not full elimination.

### Code Quality

**Strengths:**

- Every exported function has a doc comment explaining purpose and assumptions (e.g., "Assumes the outer query aliases the tasks table as `t`").
- The API surface is minimal and well-layered -- callers pick the appropriate abstraction level.
- The `list.go` diff is clean: 30 lines of inline SQL replaced by two single-line calls.
- `stats.go` compacts a 14-line query definition into a single line using `ReadyWhereClause()`.

**Concerns:**

- `ReadyWhereClause()` returns a string with embedded `\n\t\t\t  AND ` separators. This hardcodes an indentation level. If a future caller uses a different indentation context, the generated SQL will be awkwardly formatted. This is minor since SQL formatting doesn't affect correctness, but it couples presentation to the helper.
- The `stats.go` readyQuery construction on line 79 uses raw escape sequences (`"\n\t\t\tSELECT COUNT(*) FROM tasks t\n\t\t\tWHERE " + ReadyWhereClause()`). This is less readable than the previous raw string literal. A backtick-based template or a dedicated `ReadyCountQuery()` helper would be cleaner.
- All helpers return fresh string allocations on every call. For this use case the overhead is negligible, but `const` or package-level `var` would be more conventional for static SQL fragments. The function approach does provide flexibility for future parameterization.

### Test Coverage

The test file `query_helpers_test.go` contains 5 subtests inside a single top-level `TestReadyConditions`:

| Test | What it verifies |
|------|-----------------|
| no-unclosed-blockers non-empty | `ReadyNoUnclosedBlockers()` returns non-empty string |
| no-open-children non-empty | `ReadyNoOpenChildren()` returns non-empty string |
| ReadyConditions structure | Returns exactly 3 elements; first is status open; others match sub-helpers |
| BlockedConditions structure | Returns exactly 2 elements; first is status open; second is non-empty |
| ReadyWhereClause non-empty | Returns non-empty string |

**Weaknesses in test coverage:**

- Tests only verify non-emptiness and slice length -- they do not assert that the SQL contains expected keywords (e.g., `NOT EXISTS`, `dependencies`, `tasks child`). A change that returns a syntactically different query would pass all tests.
- `BlockedConditions` test only checks `conditions[1] == ""` -- it does not verify the blocked condition contains `EXISTS` or the OR disjunction. This is the weakest assertion in the suite.
- No integration/behavioral tests are included in this commit to verify the refactored queries produce identical results. The plan states "All existing list and stats tests pass unchanged" and "Test that list --ready still returns correct results after refactor." The commit relies on pre-existing tests for behavioral verification, which is acceptable if those tests exist and were run. However, no evidence of test execution is included in the commit.
- The plan's acceptance criterion "Test that the shared ready conditions produce correct SQL fragments" implies asserting on content, not just non-emptiness.
- No table-driven test structure is used, which the golang-pro skill mandates ("Write table-driven tests with subtests").

### Spec Compliance

| Acceptance Criterion | Status |
|---------------------|--------|
| Ready NOT EXISTS subqueries in exactly one location | **Partial** -- ready conditions are in one place; blocked conditions duplicate the inner SQL |
| `buildListQuery` uses shared helper for ready and blocked | **Met** -- uses `ReadyConditions()` and `BlockedConditions()` |
| `RunStats` uses shared helper for ready count | **Met** -- uses `ReadyWhereClause()` |
| All existing tests pass unchanged | **Assumed met** -- no test modifications in diff |
| `tick ready`, `tick blocked`, `tick stats` produce identical output | **Assumed met** -- SQL is character-identical to originals |

### golang-pro Compliance

| Requirement | Status |
|------------|--------|
| Document all exported functions | **Met** -- all 5 exports have doc comments |
| Handle all errors explicitly | **N/A** -- no error paths in these helpers |
| Write table-driven tests with subtests | **Not met** -- uses individual subtests, not table-driven |
| Propagate errors with `fmt.Errorf("%w", err)` | **N/A** |
| Use gofmt | **Appears met** -- formatting is consistent |

---

## Quality Assessment

### Strengths

1. **Effective duplication reduction.** Three independent SQL locations reduced to shared helpers. The risk of drift between `list.go` and `stats.go` is eliminated.
2. **Clean API design.** The layered helper functions (granular sub-conditions, composed conditions, WHERE clause) give callers the right abstraction level without over-engineering.
3. **Minimal, surgical diff.** The refactor touches only the relevant lines in `list.go` and `stats.go`. No unrelated changes, no scope creep.
4. **Good documentation.** Every exported function has a clear doc comment including the table alias assumption.

### Weaknesses

1. **`BlockedConditions()` does not derive from ready helpers.** The EXISTS subqueries in blocked are independent string literals that mirror but do not reuse the NOT EXISTS counterparts. This leaves a residual duplication risk within the same file.
2. **Tests are superficial.** Assertions check for non-emptiness and slice length but not SQL content. A broken refactor that returns wrong SQL would pass all tests. The plan's test criteria ("Test that the shared ready conditions produce correct SQL fragments") are not fully satisfied.
3. **No table-driven tests** as required by the golang-pro skill.
4. **`ReadyWhereClause()` hardcodes formatting.** The `\n\t\t\t  AND ` joiner embeds indentation assumptions, and the `stats.go` call site uses raw escape sequences that are harder to read than the original multi-line string.

### Overall Rating

**Good.** The refactor achieves its primary goal of centralizing ready/blocked SQL semantics and the code is clean, well-documented, and minimal. The main gaps are: (1) `BlockedConditions()` not being mechanically derived from the ready sub-expressions, which the plan implied but the implementation skipped; and (2) test assertions that verify structure but not content, falling short of the plan's testing criteria and the golang-pro table-driven test mandate. These are moderate issues that don't affect correctness today but reduce the long-term protection the refactor was meant to provide.
