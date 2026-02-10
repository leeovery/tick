---
id: tick-core-7-4
phase: 7
status: pending
created: 2026-02-10
---

# Remove dead VerboseLog function

**Problem**: The standalone function `VerboseLog(w io.Writer, verbose bool, msg string)` in format.go (lines 186-192) is defined and tested (format_test.go lines 381-400) but never called in any production code path. All production verbose logging uses the `VerboseLogger` struct (verbose.go) instead. This is dead code left over from before VerboseLogger was introduced.

**Solution**: Remove VerboseLog from format.go and its associated tests from format_test.go.

**Outcome**: No dead code in the verbose logging surface. All verbose logging flows through VerboseLogger.

**Do**:
1. Remove the `VerboseLog` function from `internal/cli/format.go` (lines 186-192)
2. Remove the associated test(s) from `internal/cli/format_test.go` (lines 381-400)
3. Remove any unused imports that result from the removal
4. Verify no production code references VerboseLog (it should only be referenced from its own test)
5. Run all tests to confirm nothing breaks

**Acceptance Criteria**:
- VerboseLog function no longer exists in format.go
- No test references to VerboseLog remain
- All existing tests pass
- No production code is affected

**Tests**:
- Verify via grep that no production code calls VerboseLog (validation step, not a new test)
- All existing tests pass after removal
