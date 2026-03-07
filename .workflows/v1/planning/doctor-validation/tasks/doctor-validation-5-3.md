---
id: doctor-validation-5-3
phase: 5
status: pending
created: 2026-02-13
---

# Use DiagnosticReport methods for issue count in FormatReport

**Problem**: FormatReport in format.go maintains a local issueCount variable that counts all non-passing results in its own loop. DiagnosticReport already provides ErrorCount() and WarningCount() methods. The format function's count is logically ErrorCount() + WarningCount() but computed independently. If a new severity level is added or the definition of "issue" changes, FormatReport and the report methods could produce inconsistent numbers.

**Solution**: Replace the local issueCount tracking in FormatReport with `report.ErrorCount() + report.WarningCount()` computed after the display loop.

**Outcome**: Issue count derived from canonical methods. Single source of truth for what constitutes an "issue". FormatReport no longer maintains independent counting logic.

**Do**:
1. In `internal/doctor/format.go`, remove the `issueCount := 0` declaration and the `issueCount++` increment inside the loop.
2. After the display loop (before the switch statement), add `issueCount := report.ErrorCount() + report.WarningCount()`.
3. The rest of the switch statement remains unchanged.
4. Run `go test ./internal/doctor/...` to verify all format tests pass.

**Acceptance Criteria**:
- FormatReport does not maintain its own counter during iteration.
- issueCount is derived from report.ErrorCount() + report.WarningCount().
- All existing format tests pass without modification.

**Tests**:
- All existing FormatReport tests pass (behavior is identical since ErrorCount()+WarningCount() equals the current counting logic).
- ExitCode tests unaffected.
