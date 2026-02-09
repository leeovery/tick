---
id: tick-core-6-4
phase: 6
status: pending
created: 2026-02-09
---

# Extract shared formatter methods (FormatTransition, FormatDepChange, FormatMessage)

**Problem**: `ToonFormatter.FormatTransition` and `PrettyFormatter.FormatTransition` produce byte-identical output (same type assertion, same fmt.Fprintf with arrow). `FormatDepChange` is also identical across both formatters. `FormatMessage` is identical across all three formatters (Toon, Pretty, Stub). These are ~50 lines of duplicated logic that will be copied into any future formatter and risk drift if the format changes.

**Solution**: Extract shared implementations into package-level helper functions (e.g. `formatTransitionText`, `formatDepChangeText`) that both ToonFormatter and PrettyFormatter call. For FormatMessage, either embed a common base struct or use a standalone helper.

**Outcome**: Each shared formatting operation is implemented once. Toon and Pretty formatters delegate to the shared implementation for operations where their output is identical.

**Do**:
1. In `internal/cli/format.go` (or a new helpers file in the same package), define `formatTransitionText(w io.Writer, data interface{}) error` containing the shared transition formatting logic currently in both formatters.
2. Define `formatDepChangeText(w io.Writer, data interface{}) error` with the shared dep change formatting logic.
3. Update `ToonFormatter.FormatTransition`, `PrettyFormatter.FormatTransition`, `ToonFormatter.FormatDepChange`, and `PrettyFormatter.FormatDepChange` to delegate to these helpers.
4. For FormatMessage, define a package-level `formatMessageText(w io.Writer, msg string)` and have all formatters call it (or embed a base struct).
5. Run all formatter tests.

**Acceptance Criteria**:
- FormatTransition logic exists in one shared function, called by both Toon and Pretty formatters
- FormatDepChange logic exists in one shared function, called by both Toon and Pretty formatters
- FormatMessage logic exists in one shared function or base struct
- All existing formatter and command tests pass unchanged

**Tests**:
- All existing tests for transition output, dep change output, and message output pass
- No new tests needed -- this is a pure refactor with existing coverage
