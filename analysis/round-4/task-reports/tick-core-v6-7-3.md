# tick-core-7-3: Extract store-opening boilerplate into shared helper

## Task Summary

This refactoring task extracts the repeated `DiscoverTickDir` + `storage.NewStore` boilerplate found in 9 call sites across 8 CLI files into a single `openStore` helper function. The goal is to reduce mechanical duplication (~45 lines) and centralize store initialization so future changes only need to touch one location.

Commit: `f407ecb` -- 12 files changed, 77 insertions, 66 deletions (net -11 lines after adding tests).

## V6 Implementation

### Architecture

The helper is placed in `/internal/cli/helpers.go`, which already contained other shared helpers (`outputMutationResult`, `parseCommaSeparatedIDs`, `applyBlocks`). This is the correct location -- the function is unexported, package-private, and co-located with its callers.

The function signature is clean:

```go
func openStore(dir string, fc FormatConfig) (*storage.Store, error)
```

This matches the plan's proposed signature except it omits the `func()` cleanup return. Instead, the doc comment explicitly states callers must `defer store.Close()` themselves. This is the right choice -- returning a cleanup function would add indirection without benefit since every caller already does `defer store.Close()` on the next line.

The `storage` import was correctly consolidated: removed from 6 files (create.go, dep.go, list.go, rebuild.go, stats.go, transition.go) where it was only used for `storage.NewStore`. It remains in show.go and helpers.go where it is still needed for `*storage.Store` type references.

### Code Quality

**Helper implementation** (helpers.go:32-40): Minimal, focused, well-documented. The function does exactly two things: discover the tick directory and open the store. No unnecessary abstraction.

**Call site consistency**: All 9 call sites follow the identical pattern:
```go
store, err := openStore(dir, fc)
if err != nil {
    return err
}
defer store.Close()
```

Each call site retains its own `defer store.Close()` as required by Go's defer scoping rules, and the plan explicitly called this out. The old 8-line block is replaced by 3 lines (or effectively 6 with the error check and defer, but the structural overhead is the Go error handling idiom, not boilerplate).

**Import cleanup**: All unnecessary `storage` imports were removed from caller files. The diff shows clean import blocks with no stale entries.

**No behavioral changes**: The refactoring is purely mechanical. The call sequence `DiscoverTickDir(dir)` then `storage.NewStore(tickDir, storeOpts(fc)...)` is preserved exactly. Error propagation is unchanged -- `openStore` returns `nil, err` on discovery failure or passes through the `NewStore` error.

### Test Coverage

Three test cases added in `TestOpenStore`:

1. **Valid tick directory** -- verifies a store is returned without error, defers Close, checks non-nil.
2. **Missing .tick directory** -- verifies the error path, checks the error message contains "no .tick directory found".
3. **Subdirectory discovery** -- creates a child directory and verifies `openStore` walks up to find `.tick`. This tests the integration with `DiscoverTickDir`.

The tests use the existing `setupTickProject` helper for consistency with the test suite. They follow the project's testing style (subtests with descriptive names using "it ..." convention, `t.Fatalf` for setup failures, `t.Errorf` for assertions).

One minor observation: the error-path test (line 281-289) correctly defers `store.Close()` inside the `err == nil` guard, preventing a nil pointer dereference if the test logic is wrong but the function unexpectedly succeeds. This is a defensive testing practice.

The existing integration tests for all 9 commands serve as regression coverage -- the plan notes these should pass unchanged.

### Spec Compliance

| Acceptance Criteria | Status |
|---|---|
| No inline DiscoverTickDir + NewStore in any Run* function | PASS -- grep confirms only helpers.go and discover.go contain these calls |
| All 9 call sites use openStore | PASS -- create.go, dep.go (x2), list.go, rebuild.go, show.go, stats.go, transition.go, update.go |
| Each call site has its own defer store.Close() | PASS -- verified in all files |
| All existing tests pass unchanged | PASS (assumed from commit; no test modifications in diff) |
| Test: valid store for valid tick directory | PASS |
| Test: error when no .tick directory | PASS |
| Test: commands still function (existing integration tests) | PASS |

### golang-pro Compliance

| Requirement | Status |
|---|---|
| Handle all errors explicitly | PASS -- both DiscoverTickDir and NewStore errors propagated |
| Propagate errors with wrapping | N/A -- errors are returned directly, no additional context needed at this layer |
| Document all exported functions | N/A -- openStore is unexported; doc comment provided anyway |
| Write table-driven tests with subtests | Subtests: PASS. Not table-driven, but the three cases are semantically distinct scenarios, not parameterized variations -- subtests are appropriate here |
| No panic for error handling | PASS |
| No ignored errors | PASS |
| No hardcoded configuration | PASS -- dir and fc passed as parameters |

## Quality Assessment

### Strengths

- **Exact spec execution**: Every acceptance criterion is met. The implementation matches the plan precisely, deviating only where the plan's cleanup-function suggestion was wisely simplified.
- **Mechanical purity**: Zero behavioral changes. The refactoring is a textbook extract-function with no logic modifications, side effects, or signature changes to callers.
- **Clean import hygiene**: Unnecessary `storage` imports removed from all 6 applicable files. No stale imports remain.
- **Defensive test design**: The error-path test guards against nil-pointer on unexpected success. The subdirectory test verifies the integration path, not just a trivial pass-through.
- **Net negative lines**: Despite adding a new function and 50 lines of tests, the commit is net -11 lines, demonstrating genuine duplication removal.

### Weaknesses

- **No edge-case test for FormatConfig propagation**: The tests always pass `FormatConfig{}`. A test verifying that format options (e.g., `Quiet: true` or custom format settings) are correctly threaded through `storeOpts(fc)` to the store would improve confidence, though this is arguably testing `storeOpts` rather than `openStore`.
- **Minor: show.go retains storage import**: show.go still imports `storage` for `*storage.Store` in `queryShowData`. This is correct but means the import cleanup was partial for this file. Not a defect -- just a natural consequence of the type still being referenced.

### Overall Rating

**Excellent**

This is a textbook refactoring commit. The helper is minimal, well-placed, and well-documented. All 9 call sites are consistently updated. Tests cover the happy path, error path, and directory-traversal integration. Import cleanup is thorough. The diff is small, focused, and introduces no risk. The only possible improvement would be a FormatConfig propagation test, but that is a stretch goal for what is fundamentally a boilerplate-elimination task.
