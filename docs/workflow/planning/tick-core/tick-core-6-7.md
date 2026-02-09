---
id: tick-core-6-7
phase: 6
status: pending
created: 2026-02-09
---

# Replace interface{} Formatter parameters with type-safe signatures

**Problem**: Every Formatter method (FormatTaskList, FormatTaskDetail, FormatTransition, FormatDepChange, FormatStats) accepts `interface{}` as its data parameter and performs a runtime type assertion at the top of each implementation. With 3 formatters x 5 methods = 15 assertion sites, the surface area for silent type mismatches is significant. A caller passing the wrong type compiles successfully but panics or fails at runtime. Go's type system cannot catch regressions during refactoring.

**Solution**: Replace the Formatter interface methods with type-specific signatures. For example: `FormatTaskList(w io.Writer, rows []TaskRow) error`, `FormatTaskDetail(w io.Writer, data *showData) error`, `FormatTransition(w io.Writer, data *TransitionData) error`, `FormatDepChange(w io.Writer, data *DepChangeData) error`, `FormatStats(w io.Writer, data *StatsData) error`.

**Outcome**: All 15 runtime type assertions are eliminated. Passing the wrong type to a formatter method becomes a compile error.

**Do**:
1. In `internal/cli/format.go`, update the Formatter interface to use concrete types instead of `interface{}` for each method's data parameter.
2. Update `ToonFormatter`, `PrettyFormatter`, and `JSONFormatter` method signatures to match the new interface.
3. Remove the runtime type assertions at the top of each formatter method (the `data.(*SomeType)` calls).
4. Export `showData` and `relatedTask` types if they are used in the Formatter interface signatures (or keep them unexported if the interface and all implementations remain in the same package).
5. Update all call sites if any type adjustments are needed (e.g. pointer vs value).
6. Run all tests.

**Acceptance Criteria**:
- Formatter interface methods use concrete types, not interface{}
- No runtime type assertions in formatter implementations
- All existing tests pass
- Code compiles without errors

**Tests**:
- Full test suite passes after the refactor
- Verify that passing an incorrect type to a formatter method causes a compile error (manual check)
