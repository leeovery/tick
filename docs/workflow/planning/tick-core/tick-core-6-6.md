---
id: tick-core-6-6
phase: 6
status: pending
created: 2026-02-09
---

# Remove dead StubFormatter code

**Problem**: The `StubFormatter` type in `internal/cli/format.go` (lines 93-127) is annotated as a placeholder that "will be replaced by concrete Toon, Pretty, and JSON formatters" but all three concrete formatters now exist. StubFormatter is not referenced anywhere in production code paths (`newFormatter` only instantiates concrete formatters). It is dead code that adds maintenance noise.

**Solution**: Delete the `StubFormatter` struct and all its methods from `format.go`.

**Outcome**: No dead formatter code in the codebase. The concrete formatters (Toon, Pretty, JSON) are the only implementations.

**Do**:
1. In `internal/cli/format.go`, remove the `StubFormatter` struct definition and all its methods (FormatTaskList, FormatTaskDetail, FormatTransition, FormatDepChange, FormatStats, FormatMessage).
2. Check if any test files reference StubFormatter -- if so, update them to use a concrete formatter or a test-specific mock.
3. Run all tests.

**Acceptance Criteria**:
- `StubFormatter` type no longer exists in the codebase
- All tests pass
- No compilation errors

**Tests**:
- Full test suite passes after removal
- Grep for "StubFormatter" returns no hits in production code
