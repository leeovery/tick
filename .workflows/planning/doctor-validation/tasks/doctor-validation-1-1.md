---
id: doctor-validation-1-1
phase: 1
status: completed
created: 2026-01-30
---

# Check Interface & Diagnostic Runner

## Goal

Doctor needs a framework to register, execute, and collect results from multiple diagnostic checks. Without this foundation, individual checks (cache staleness, JSONL syntax, orphan detection, etc.) have no way to run or report results. This task defines the check interface that all 10 checks will implement and the diagnostic runner that iterates through registered checks, collects results, and returns a unified result set. The runner must always execute all checks — it never short-circuits on the first failure.

## Implementation

- Define a `CheckResult` struct with fields: `Name` (string — the check's display label, e.g. "Cache", "JSONL syntax"), `Passed` (bool), `Severity` (enum: `error` or `warning`), `Details` (string — human-readable description of what's wrong, empty when passed), `Suggestion` (string — actionable fix text, empty when passed or when no suggestion applies)
- Define a `Check` interface with a single method: `Run(ctx) -> []CheckResult`. A single check can return multiple results (e.g., 5 orphaned references produce 5 `CheckResult` entries). A passing check returns exactly one result with `Passed: true`. A failing check returns one or more results with `Passed: false`.
- Define a `Severity` type (string enum) with two constants: `SeverityError` and `SeverityWarning`. Errors affect exit code; warnings do not.
- Define a `DiagnosticRunner` struct that holds an ordered slice of `Check` implementations
- Implement `Register(check Check)` to append a check to the runner's slice
- Implement `RunAll(ctx) -> DiagnosticReport` that iterates every registered check, calls `Run()`, and collects all `CheckResult` entries into a `DiagnosticReport`
- Define `DiagnosticReport` struct with fields: `Results` (all `CheckResult` entries in registration order) and computed accessors: `HasErrors() bool` (any result with `Passed: false` and `Severity: error`), `ErrorCount() int`, `WarningCount() int`
- The runner must call every check regardless of prior failures — no short-circuit logic
- The runner with zero registered checks returns an empty `DiagnosticReport` (zero results, `HasErrors()` returns false)
- The `ctx` parameter should carry the path to the `.tick` directory so checks can locate `tasks.jsonl` and `cache.db`

## Tests

- `"it returns empty report when zero checks are registered"`
- `"it runs a single passing check and returns one result with Passed true"`
- `"it runs a single failing check and returns one result with Passed false"`
- `"it runs all checks when all pass — report has no errors"`
- `"it runs all checks when all fail — report collects all failures"`
- `"it runs all checks with mixed pass/fail — report contains both"`
- `"it does not short-circuit — failing check does not prevent subsequent checks from running"`
- `"it preserves registration order in results"`
- `"it collects multiple results from a single check (e.g., check returns 3 failures)"`
- `"HasErrors returns true when any error-severity result has Passed false"`
- `"HasErrors returns false when only warnings exist (no errors)"`
- `"ErrorCount counts only error-severity failures"`
- `"WarningCount counts only warning-severity failures"`
- `"HasErrors returns false for empty report"`

## Edge Cases

- **Zero checks registered**: Runner returns empty `DiagnosticReport`. `HasErrors()` is false, `ErrorCount()` is 0, `WarningCount()` is 0. This is a valid state — not an error.
- **All checks pass**: Every `CheckResult` has `Passed: true`. `HasErrors()` is false. This is the happy path for a healthy data store.
- **All checks fail**: Every `CheckResult` has `Passed: false`. All failures collected. Runner does not stop early.
- **Mixed pass/fail**: Some checks pass, some fail. Report contains all results. `HasErrors()` reflects whether any error-severity failures exist.
- **Single check returns multiple results**: A check may find multiple issues (e.g., 5 orphaned references). All results are included in the report individually. This is distinct from multiple checks each returning one result.
- **Warnings only (no errors)**: `HasErrors()` returns false. This distinction matters for exit code logic (built in task 1-2) — warnings alone do not cause exit code 1.

## Acceptance Criteria

- [ ] `Check` interface defined with `Run` method returning `[]CheckResult`
- [ ] `CheckResult` struct has `Name`, `Passed`, `Severity`, `Details`, `Suggestion` fields
- [ ] `Severity` type with `SeverityError` and `SeverityWarning` constants
- [ ] `DiagnosticRunner` registers checks and runs all of them via `RunAll`
- [ ] `DiagnosticReport` collects all results with `HasErrors()`, `ErrorCount()`, `WarningCount()` accessors
- [ ] Zero registered checks produces empty report (not an error)
- [ ] Runner never short-circuits — all checks run regardless of prior failures
- [ ] Single check returning multiple results produces multiple entries in report
- [ ] Tests written and passing for all edge cases (zero, all-pass, all-fail, mixed, multi-result)

## Context

The specification requires doctor to "run all checks" — it completes all validations before reporting, never stops early. This is design principle #4. The runner is the enforcement point for this guarantee.

Doctor performs two categories of checks: errors and warnings. The specification defines 9 error checks and 1 warning check across three phases. The `Severity` distinction is needed from the start because exit code logic (task 1-2) depends on it — exit 0 when no errors (warnings allowed), exit 1 when any error found.

The `Check` interface must support returning multiple results from a single check because the specification states: "Doctor lists each error individually. If there are 5 orphaned references, all 5 are shown with their specific details."

The `Suggestion` field on `CheckResult` supports the fix suggestion mechanism — some checks have specific suggestions (e.g., cache staleness suggests "Run `tick rebuild` to refresh cache") while others use "Manual fix required". The suggestion is part of the check result, not the formatter, because each check knows its own remedy.

This is a Go project. The tick-core dependency provides the data structures (Task, ID format, cache hash mechanism) that checks will validate against, but the runner itself has no dependency on tick-core internals — it only defines the interface and execution framework.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
