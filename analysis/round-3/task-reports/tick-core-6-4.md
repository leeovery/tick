# Task 6-4: Extract shared formatter methods (FormatTransition, FormatDepChange, FormatMessage) (V5 Only -- Phase 6 Refinement)

## Task Plan Summary

The task addresses code duplication across `ToonFormatter`, `PrettyFormatter`, and `StubFormatter`. Three formatting operations -- `FormatTransition`, `FormatDepChange`, and `FormatMessage` -- produced byte-identical output across multiple formatters (~50 lines of duplicated logic). The plan calls for extracting shared implementations into package-level helper functions (`formatTransitionText`, `formatDepChangeText`, `formatMessageText`) in the `cli` package, then having each formatter delegate to the shared helpers. This is a pure refactor: no new tests are needed because existing coverage validates output correctness.

## Note

This is a Phase 6 analysis refinement task that only exists in V5. It addresses code duplication found during post-implementation analysis. This is a standalone quality assessment, not a comparison.

## V5 Implementation

### Architecture & Design

The implementation follows the plan precisely. Three package-level helper functions were added to `/private/tmp/tick-analysis-worktrees/v5/internal/cli/format.go` (lines 128-161 at commit `bba1728`):

1. **`formatTransitionText(w io.Writer, data interface{}) error`** (lines 128-137): Extracts the type assertion (`data.(*TransitionData)`) and the `fmt.Fprintf` call with the arrow-formatted transition line.

2. **`formatDepChangeText(w io.Writer, data interface{}) error`** (lines 139-155): Extracts the type assertion (`data.(*DepChangeData)`) and the switch on `d.Action` for "added"/"removed" cases with a default error.

3. **`formatMessageText(w io.Writer, msg string)`** (lines 158-161): Wraps `fmt.Fprintln(w, msg)` into a named function.

The delegation pattern is clean. Both `ToonFormatter` and `PrettyFormatter` now have one-line methods for these operations:

```go
// toon_formatter.go line 165-166
func (f *ToonFormatter) FormatTransition(w io.Writer, data interface{}) error {
    return formatTransitionText(w, data)
}
```

```go
// pretty_formatter.go line 129-130
func (f *PrettyFormatter) FormatTransition(w io.Writer, data interface{}) error {
    return formatTransitionText(w, data)
}
```

The same pattern applies to `FormatDepChange` and `FormatMessage` on both formatters, plus `StubFormatter.FormatMessage` which was also updated to delegate (format.go line 124-125).

**Design decision: Placement in `format.go`**. The helpers were placed in the existing `format.go` file alongside the `Formatter` interface and `StubFormatter`, rather than creating a new file. This is a reasonable choice for three small functions totaling ~34 lines, keeping shared formatter infrastructure in one location.

**Design decision: `interface{}` parameter retention**. At the time of this commit, the `Formatter` interface used `interface{}` parameters for all methods except `FormatMessage`. The shared helpers therefore also accept `interface{}` and perform type assertions internally. This matches the existing interface contract. Notably, subsequent task T6-7 later addressed this by replacing the `interface{}` parameters with concrete types, which then allowed the helpers to also use typed parameters. The task correctly chose not to change the interface signature as that was outside its scope.

### Code Quality

**Positive aspects:**

- Every exported and unexported function has a doc comment. For example:
  ```go
  // formatTransitionText writes a plain-text status transition line.
  // Data must be *TransitionData. Shared by ToonFormatter and PrettyFormatter.
  ```
  The comments accurately describe the expected data type and which formatters use the helper.

- The refactoring is mechanical and behavior-preserving. The extracted code is character-for-character identical to what was previously duplicated. The diff confirms this: the removed blocks in `toon_formatter.go` (lines 163-181 pre-commit) and `pretty_formatter.go` (lines 127-145 pre-commit) match exactly the new helper bodies.

- The error messages in type assertions are preserved verbatim (e.g., `"FormatTransition: expected *TransitionData, got %T"`), maintaining the same diagnostic behavior.

- Naming follows Go convention: unexported package-level functions with descriptive `formatXxxText` names that clearly convey "shared text formatting helpers."

- The `formatMessageText` function, while arguably trivial (a single `fmt.Fprintln` call), is justified because it prevents drift: all three formatters (Toon, Pretty, Stub) now call the same function, ensuring consistent newline-terminated output.

**Minor observations:**

- The `formatDepChangeText` function retains a `default` case returning an error for unknown actions (`fmt.Errorf("FormatDepChange: unknown action %q", d.Action)`). This is defensive and correct. The error message prefix "FormatDepChange" in the helper technically refers to the caller's name rather than the helper's name, but this is acceptable since it matches what the user-facing error message should say.

- The comment on `formatMessageText` (line 159) says "Shared by ToonFormatter, PrettyFormatter, and StubFormatter" but the comment on `formatTransitionText` (line 129) says "Shared by ToonFormatter and PrettyFormatter." This asymmetry is accurate -- the StubFormatter's `FormatTransition` returns nil (it is a stub), only `FormatMessage` was non-trivially implemented on StubFormatter.

- No `import` changes were needed. The `fmt` and `io` packages were already imported in `format.go`.

### Test Coverage

The task plan explicitly states: "No new tests needed -- this is a pure refactor with existing coverage." This is correct. The commit modifies zero test files.

Existing test coverage is thorough:

- **`toon_formatter_test.go`** (lines 318-393): `TestToonFormatterFormatTransitionAndDep` tests transition output (arrow format), dep add, dep remove, and message output against exact expected strings.

- **`pretty_formatter_test.go`** (lines 331-406): `TestPrettyFormatterTransitionAndDep` has matching tests for the same operations on `PrettyFormatter`.

- **`formatter_integration_test.go`** (lines 120-183): Integration tests run full CLI invocations with `--toon` and `--json` flags for transitions and dependency changes, verifying end-to-end behavior through the real `Run()` function.

These existing tests validate that the refactored helpers produce identical output. Since the tests pass without modification, the refactor is proven behavior-preserving.

One gap worth noting: there are no direct unit tests for the package-level helper functions themselves (`formatTransitionText`, `formatDepChangeText`, `formatMessageText`). This is acceptable because they are exercised indirectly through every formatter test. Adding direct tests would be redundant for a pure extraction refactor.

### Spec Compliance

Checking each acceptance criterion:

1. **"FormatTransition logic exists in one shared function, called by both Toon and Pretty formatters"** -- Met. `formatTransitionText` is defined once at format.go:128-137, called by `ToonFormatter.FormatTransition` (toon_formatter.go:165-166) and `PrettyFormatter.FormatTransition` (pretty_formatter.go:129-130).

2. **"FormatDepChange logic exists in one shared function, called by both Toon and Pretty formatters"** -- Met. `formatDepChangeText` is defined once at format.go:139-155, called by `ToonFormatter.FormatDepChange` (toon_formatter.go:170-172) and `PrettyFormatter.FormatDepChange` (pretty_formatter.go:134-136).

3. **"FormatMessage logic exists in one shared function or base struct"** -- Met. `formatMessageText` is defined once at format.go:158-161, called by `ToonFormatter.FormatMessage` (toon_formatter.go:209-210), `PrettyFormatter.FormatMessage` (pretty_formatter.go:203-204), and `StubFormatter.FormatMessage` (format.go:124-125). The plan offered either a shared function or a base struct; a shared function was chosen, which is simpler and more idiomatic for Go.

4. **"All existing formatter and command tests pass unchanged"** -- Met. Zero test files were modified. The tests exercise exact string matching of output, so any behavioral change would cause failures.

All five "Do" items from the plan are addressed:
- Do #1: `formatTransitionText` in `format.go` -- done.
- Do #2: `formatDepChangeText` in `format.go` -- done.
- Do #3: `ToonFormatter.FormatTransition`, `PrettyFormatter.FormatTransition`, `ToonFormatter.FormatDepChange`, `PrettyFormatter.FormatDepChange` all delegate -- done.
- Do #4: `formatMessageText` with all three formatters calling it -- done.
- Do #5: All formatter tests pass -- done (no test changes).

### golang-pro Skill Compliance

Evaluating against the SKILL.md constraints:

**MUST DO checklist:**

- **"Use gofmt and golangci-lint on all code"**: The code is properly formatted. Indentation, spacing, and line breaks are consistent with `gofmt` output.

- **"Handle all errors explicitly (no naked returns)"**: `formatTransitionText` returns the `err` from `fmt.Fprintf`. `formatDepChangeText` returns errors for both action cases and the default. The `formatMessageText` function mirrors the existing `FormatMessage` convention of not returning an error (matching the `Formatter` interface signature). This is consistent, not a violation.

- **"Document all exported functions, types, and packages"**: All three helpers are unexported (lowercase), so exported-documentation requirements don't strictly apply. Nevertheless, all three have doc comments -- exceeding the requirement.

- **"Propagate errors with fmt.Errorf("%w", err)"**: The type assertion errors use `fmt.Errorf` without `%w` wrapping, but these are not wrapping existing errors -- they are creating new sentinel-style errors for type mismatches. This is correct; `%w` is for wrapping received errors, not for constructing new ones.

**MUST NOT DO checklist:**

- **"Ignore errors"**: No errors are ignored. `formatMessageText` calls `fmt.Fprintln` without checking the error, matching the existing pattern established by the Formatter interface where `FormatMessage` has no error return. This is a known design choice in the interface, not a violation introduced by this task.

- **"Use panic for normal error handling"**: No panics.

- **"Use reflection without performance justification"**: No reflection. Type assertions (`data.(*TransitionData)`) are not reflection; they are Go's standard type assertion mechanism.

## Quality Assessment

### Strengths

1. **Surgical precision**: The refactoring is perfectly mechanical. The extracted code is byte-identical to the duplicated code it replaces. This eliminates risk of behavioral regression.

2. **Correct scope discipline**: The task resisted the temptation to also change the `interface{}` parameters to concrete types, leaving that for the subsequent T6-7 task. This demonstrates good task isolation.

3. **Complete coverage**: All three operations mentioned in the plan (`FormatTransition`, `FormatDepChange`, `FormatMessage`) were extracted. The `StubFormatter.FormatMessage` was also updated to use the shared helper, even though the plan primarily mentioned Toon and Pretty -- this was a reasonable inclusion.

4. **Documentation quality**: Every helper function has a clear, accurate doc comment identifying its purpose and which formatters share it.

5. **Net code reduction**: The diff shows +45 lines / -44 lines across source files, but the actual duplication eliminated is ~50 lines. Each formatter method body went from 5-15 lines to a single return statement. This makes future maintenance significantly easier -- a format change needs to happen in one place.

6. **Zero test modification**: Confirming the refactoring is behavior-preserving by not needing any test changes is the gold standard for extract-method refactoring.

### Weaknesses

1. **`interface{}` type assertions in helpers**: At this commit, the shared helpers still use `interface{}` parameters with runtime type assertions. While this matches the existing Formatter interface, the helpers could have been written with concrete types even within this commit, since they are package-internal and only called from methods that already have `interface{}`. However, this was correctly deferred to T6-7, so this is more of an observation than a flaw.

2. **`formatMessageText` is trivially thin**: The function wraps a single `fmt.Fprintln` call. While justified for consistency and drift prevention, it adds a layer of indirection for a one-liner. This is a very minor concern -- the consistency benefit outweighs the indirection cost.

3. **No direct unit tests for helpers**: While existing tests cover the helpers indirectly, direct tests for `formatTransitionText`, `formatDepChangeText`, and `formatMessageText` would provide documentation-as-tests and make it explicit what the helper's contract is. This is standard practice in the golang-pro skill's testing guidance. However, the plan explicitly states no new tests are needed, so this is compliant with the spec.

4. **JSONFormatter excluded from `FormatMessage` sharing**: The `JSONFormatter.FormatMessage` method (json_formatter.go line 205-206) writes `writeJSON(w, jsonMessage{Message: msg})` which produces different output (JSON vs plain text). This is correctly excluded from sharing because the output format differs. No weakness here -- just noting completeness of analysis.

### Overall Quality Rating

**Excellent** -- This is a textbook extract-method refactoring executed with precision. Every acceptance criterion is met. The code changes are minimal, surgical, and behavior-preserving as proven by zero test modifications. The implementation follows Go conventions, maintains proper documentation, and demonstrates good task scoping discipline by not overreaching into interface signature changes. The only minor observations (thin wrapper for `formatMessageText`, retained `interface{}` parameters) are either justified design choices or correctly deferred to subsequent tasks.
