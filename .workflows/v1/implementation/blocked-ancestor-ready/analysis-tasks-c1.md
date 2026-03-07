---
topic: blocked-ancestor-ready
cycle: 1
total_proposed: 1
---
# Analysis Tasks: blocked-ancestor-ready (Cycle 1)

## Task 1: Compose BlockedConditions() from ReadyNo*() helpers instead of duplicating SQL
status: approved
severity: high
sources: duplication, architecture

**Problem**: `BlockedConditions()` in `internal/cli/query_helpers.go` (lines 64-94) contains three independently-written EXISTS subqueries whose SQL bodies are character-for-character identical to the NOT EXISTS subqueries returned by `ReadyNoUnclosedBlockers()`, `ReadyNoOpenChildren()`, and `ReadyNoBlockedAncestor()`. Only the EXISTS vs NOT EXISTS wrapper differs. This violates the project's "Compose, Don't Duplicate" principle: if any ReadyNo*() helper is modified, BlockedConditions() must be manually updated in lockstep or it silently drifts.

**Solution**: Refactor `BlockedConditions()` to derive its OR clause from the existing ReadyNo*() helpers by mechanically negating them -- stripping the leading "NOT " from each helper's output to convert `NOT EXISTS (...)` into `EXISTS (...)`. This eliminates ~25 lines of duplicated SQL and guarantees the two stay in sync.

**Outcome**: `BlockedConditions()` contains zero hand-written SQL subquery bodies. All SQL logic lives exclusively in the ReadyNo*() helpers, and `BlockedConditions()` composes from them. Any future change to a ready helper automatically propagates to blocked logic.

**Do**:
1. In `internal/cli/query_helpers.go`, add a small unexported helper function (e.g., `negateNotExists(s string) string`) that takes a `NOT EXISTS (...)` string and returns the `EXISTS (...)` form by stripping the leading `"NOT "` prefix via `strings.TrimPrefix` or `strings.CutPrefix`.
2. Rewrite `BlockedConditions()` to build its OR clause by calling `negateNotExists()` on each of `ReadyNoUnclosedBlockers()`, `ReadyNoOpenChildren()`, and `ReadyNoBlockedAncestor()`, joining the results with `"\n\t\t\t\tOR "` and wrapping in parentheses.
3. Delete the three hand-written EXISTS subquery blocks currently in `BlockedConditions()`.
4. Run `go test ./internal/cli/...` to confirm all existing ready, blocked, list, and stats tests still pass.
5. Verify with `go vet ./...` and `gofmt -d ./internal/cli/query_helpers.go` that the code is clean.

**Acceptance Criteria**:
- `BlockedConditions()` contains no SQL string literals other than `t.status = 'open'`
- `BlockedConditions()` calls all three `ReadyNo*()` helpers (directly or via negation)
- All existing tests in `internal/cli/` pass without modification
- `go vet ./...` reports no issues

**Tests**:
- Add a unit test in `internal/cli/query_helpers_test.go` that verifies `BlockedConditions()` output contains the same subquery bodies as the negated `ReadyNo*()` helpers (e.g., confirm each ReadyNo*() body appears in the BlockedConditions output with EXISTS instead of NOT EXISTS). This acts as a drift-detection test.
