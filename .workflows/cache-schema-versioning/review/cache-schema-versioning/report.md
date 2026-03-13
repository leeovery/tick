# Implementation Review: Cache Schema Versioning

**Plan**: cache-schema-versioning
**QA Verdict**: Approve

## Summary

Clean, focused bugfix that adds schema versioning to the SQLite cache. All three tasks are fully implemented with no drift from the plan or specification. The implementation is minimal and well-placed: a version constant, metadata storage during rebuild, an early check in `ensureFresh()`, and a `recreateCache()` helper that cleanly encapsulates the close-delete-reopen sequence. Tests cover every acceptance criterion including the exact "no such column: t.type" bug scenario that motivated the fix.

## QA Verification

### Specification Compliance

Implementation aligns precisely with the specification:
- Schema version constant at `cache.go:14`
- Version stored in metadata during `Rebuild()` at `cache.go:226-231`
- Version check before `IsFresh()` in `ensureFresh()` at `store.go:395-410`
- Delete+rebuild on mismatch or missing version via `recreateCache()` at `store.go:135-147`
- No `ALTER TABLE` migrations — full rebuild as specified

No deviations from spec design decisions.

### Plan Completion

- [x] Phase 1 acceptance criteria met
- [x] All 3 tasks completed
- [x] No scope creep

All 7 phase-level acceptance criteria verified:
1. `const schemaVersion = 1` exists
2. `Rebuild()` stores `schema_version` in metadata
3. `ensureFresh()` checks version before `IsFresh()`, triggers delete+rebuild on mismatch
4. Test: wrong schema version triggers rebuild
5. Test: missing schema version triggers rebuild
6. Test: correct version preserved
7. Test: queries succeed after version-triggered rebuild

### Code Quality

No issues found. The implementation follows project conventions (error wrapping, DI via struct fields, functional options). `recreateCache()` is a well-factored helper with a `reason` parameter for observability. Cyclomatic complexity is low throughout.

### Test Quality

Tests adequately verify requirements. 11 new tests across `cache_test.go` and `store_test.go`:
- 5 unit tests for cache-level version storage and retrieval
- 4 integration tests for store-level version check behavior
- 2 end-to-end tests for the full upgrade scenario

Each test covers a distinct scenario with focused assertions. No redundant or over-tested cases. The end-to-end test at `store_test.go:1513` directly replicates the original bug by dropping+recreating the tasks table without the `type` column.

### Required Changes

None.

## Recommendations

1. **`containsSubstring` helper** (`store_test.go:1744`): This is functionally equivalent to `strings.Contains` — the length checks add no value. Consider simplifying.
2. **`sql.ErrNoRows` comparison** (`cache.go:265`): Uses `==` rather than `errors.Is()`. Safe with `database/sql` but `errors.Is()` is more idiomatic Go. This is a pre-existing pattern (also at `cache.go:246`), not introduced by this change.
3. **`recreateCache` nil guard** (`store.go:136`): `s.cache.Close()` is called without a nil check. Currently safe because all callers guarantee non-nil, but a guard would be more defensive.
