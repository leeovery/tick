---
id: doctor-validation-5-2
phase: 5
status: pending
created: 2026-02-13
---

# Extract assertReadOnly test helper

**Problem**: Every check test file (10 files) contains a near-identical "does not modify tasks.jsonl (read-only verification)" test: write content, read file bytes before check, run check, read file bytes after, compare. The pattern is 12-15 lines per file, totaling ~130 lines of duplicated scaffolding. Only the check type instantiated and the JSONL content vary.

**Solution**: Extract a shared test helper `assertReadOnly(t *testing.T, tickDir string, content []byte, runCheck func())` in a common test helper file. Each test file calls this with its specific content and a closure that runs the check.

**Outcome**: ~130 lines of duplicated test scaffolding replaced by ~15 lines of helper plus ~10 one-line calls. Read-only verification logic maintained in one place.

**Do**:
1. Create or identify an existing shared test helper file in `internal/doctor/` (e.g., `helpers_test.go` if it exists, or the file where setupTickDir and writeJSONL already live).
2. Add `assertReadOnly(t *testing.T, tickDir string, content []byte, runCheck func())` that: writes content via writeJSONL, reads file bytes before, calls runCheck(), reads file bytes after, compares with t.Helper() for correct line reporting.
3. In each of the 10 test files, replace the inline read-only verification test body with a call to assertReadOnly, passing the check-specific content and a closure like `func() { check := &OrphanedParentCheck{}; check.Run(ctx, tickDir) }`.
4. Run `go test ./internal/doctor/...` to verify all read-only tests still pass.

**Acceptance Criteria**:
- assertReadOnly helper exists in a shared test file in internal/doctor/.
- All 10 read-only verification tests use the helper instead of inline logic.
- All tests pass. No test behavior changes.
- Helper uses t.Helper() for proper error attribution.

**Tests**:
- All 10 existing "does not modify tasks.jsonl" tests pass using the new helper.
- No other tests are affected.
