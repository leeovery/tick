---
id: doctor-validation-1-2
phase: 1
status: pending
created: 2026-01-30
---

# Output Formatter & Exit Code Logic

## Goal

The diagnostic runner (task 1-1) collects `CheckResult` entries into a `DiagnosticReport`, but there is no way to present those results to the user or determine the process exit code. This task builds the output formatter that converts a `DiagnosticReport` into the human-readable text format defined by the specification, and the exit code function that maps report state to 0 (clean) or 1 (errors found). Without this task, checks can run but produce no visible output and the CLI has no way to signal success or failure.

## Implementation

- Define a `Formatter` struct (or function) that accepts a `DiagnosticReport` and an `io.Writer`, and writes the formatted output. This keeps formatting testable without capturing stdout — tests pass a `bytes.Buffer` as the writer.
- Iterate over `DiagnosticReport.Results` in order. For each `CheckResult`:
  - If `Passed` is true, write a line: `✓ {Name}: OK`
  - If `Passed` is false, write a line: `✗ {Name}: {Details}`
  - If `Passed` is false and `Suggestion` is non-empty, write an indented follow-up line: `  → {Suggestion}`
  - If `Passed` is false and `Suggestion` is empty, no suggestion line is written
- After all results, write a blank line followed by a summary line:
  - Zero issues: `No issues found.`
  - One issue: `1 issue found.`
  - More than one issue: `{N} issues found.` where N is the count of all results with `Passed: false` (both errors and warnings count in the display total)
- Warnings use the same `✗` marker as errors in output (the specification shows only `✓` and `✗`; severity distinction affects exit code, not display symbol)
- Define an `ExitCode(report DiagnosticReport) int` function:
  - Returns `0` if `report.HasErrors()` is false (no error-severity failures — warnings alone do not trigger exit 1)
  - Returns `1` if `report.HasErrors()` is true (at least one error-severity failure)
- The formatter and exit code logic are pure functions of `DiagnosticReport` — no side effects beyond writing to the provided writer

## Tests

- `"it formats zero results as empty output with 'No issues found.' summary"`
- `"it formats a single passing check as '✓ {Name}: OK'"`
- `"it formats multiple passing checks each on their own line"`
- `"it formats a single failing check as '✗ {Name}: {Details}' with suggestion on next line"`
- `"it formats a failing check without suggestion — no suggestion line emitted"`
- `"it formats multiple failures from different checks each individually"`
- `"it formats multiple failures from one check (e.g., 3 orphaned references) as 3 separate ✗ lines"`
- `"it shows summary '1 issue found.' for exactly one failure"`
- `"it shows summary '{N} issues found.' for multiple failures"`
- `"it shows summary 'No issues found.' when all checks pass"`
- `"it includes both errors and warnings in the summary issue count"`
- `"it preserves result order from DiagnosticReport in output"`
- `"exit code returns 0 when report has no errors (empty report)"`
- `"exit code returns 0 when report has only passing checks"`
- `"exit code returns 0 when report has only warnings (no errors)"`
- `"exit code returns 1 when report has at least one error-severity failure"`
- `"exit code returns 1 when report has mixed errors and warnings"`

## Edge Cases

- **Zero issues (empty report)**: Formatter produces only the summary line `No issues found.` with no check lines above it. Exit code is 0. This occurs when zero checks are registered or all checks pass with no results — the formatter handles both identically since it operates on the results slice.
- **Single issue**: Exactly one `CheckResult` with `Passed: false`. Summary reads `1 issue found.` (singular). Validates the singular/plural grammar.
- **Multiple issues from one check**: A single check (e.g., orphaned references) may return 3 failures. The formatter writes 3 separate `✗` lines, one per `CheckResult`. The summary counts all 3. This is distinct from 3 different checks each finding 1 issue.
- **Suggestion text present vs absent**: Some checks include a suggestion (e.g., cache staleness: "Run `tick rebuild` to refresh cache"), others may not. When `Suggestion` is empty, the `→` line is omitted entirely — no blank line, no arrow with empty text. When present, the suggestion appears indented on the line immediately after the `✗` line.
- **Warnings only**: All failures are `SeverityWarning`. They appear in output as `✗` lines (same display as errors). They count in the summary total. But exit code is 0 because `HasErrors()` is false — only error-severity failures affect exit code.
- **Mixed pass and fail**: Some checks pass, some fail. Passing checks show `✓`, failures show `✗`. Output interleaves them in registration order. Summary counts only failures.

## Acceptance Criteria

- [ ] Formatter writes `✓ {Name}: OK` for each passing result
- [ ] Formatter writes `✗ {Name}: {Details}` for each failing result
- [ ] Suggestion line `  → {Suggestion}` appears only when suggestion is non-empty
- [ ] Summary line uses correct grammar: "No issues found." / "1 issue found." / "{N} issues found."
- [ ] Summary counts all failures (errors + warnings) for display
- [ ] Exit code returns 0 when no error-severity failures exist
- [ ] Exit code returns 1 when any error-severity failure exists
- [ ] Warnings do not affect exit code (exit 0 when only warnings)
- [ ] Output order matches result order in DiagnosticReport
- [ ] Formatter writes to provided `io.Writer` (testable without stdout capture)
- [ ] Tests written and passing for all edge cases (zero issues, single issue, multi-issue from one check, suggestion present/absent, warnings-only)

## Context

The specification defines the exact output format:

```
✓ Cache: OK
✓ JSONL syntax: OK
✓ ID uniqueness: OK
✗ Orphaned reference: tick-a1b2c3 references non-existent parent tick-missing
  → Manual fix required

1 issue found.
```

Key formatting rules from the spec:
- `✓` for passing checks, `✗` for failures with details and suggested action
- Summary count at end
- Doctor outputs human-readable text only — no TOON/JSON variants
- Exit code 0 means all checks passed (no errors, warnings allowed); exit code 1 means one or more errors found

The exit code distinction between errors and warnings is critical: the specification states exit code 0 when "no errors, warnings allowed." This means a report with only warning-severity failures still exits 0. The `HasErrors()` method from `DiagnosticReport` (task 1-1) provides exactly this check.

The `Suggestion` field on `CheckResult` (task 1-1) carries check-specific fix text. The formatter's job is purely presentational — it renders whatever suggestion text the check provided, or omits the suggestion line if empty. The formatter does not decide what suggestions to show.

This is a Go project. The formatter and exit code function depend only on the types defined in task 1-1 (`DiagnosticReport`, `CheckResult`, `Severity`).

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
