---
id: migration-2-2
phase: 2
status: completed
created: 2026-01-31
---

# Presenter Failure Output

## Goal

Phase 1's presenter (migration-1-4) only renders success lines (`✓`) and always shows `0 failed` in the summary. After migration-2-1, the engine returns `[]Result` containing both successes and failures. The presenter needs to render failure lines (`✗`) with the skip reason inline, compute the correct failed count in the summary, and append a "Failures:" detail section listing each failed task with its reason. Without this, users see no indication of which tasks failed or why.

## Implementation

- Modify the existing presenter in `internal/migrate/` (the `WriteResult`, `WriteSummary` functions, or equivalent) to handle `Result{Success: false}` entries.

- **WriteResult changes** — when `r.Success` is `false`, render the failure line format:
  ```
    ✗ Task: Broken entry (skipped: missing title)
  ```
  The reason text comes from `r.Err.Error()`. The format is: two-space indent, `✗`, space, `Task: `, title, ` (skipped: `, error reason, `)`.

  If `r.Title` is empty (failure with empty title), use a fallback. Render `Task: (unknown)` so the line is still parseable and the failure is visible. The `(skipped: reason)` suffix still applies.

- **WriteSummary changes** — count failures (`Success: false`) separately from successes:
  ```
  Done: 2 imported, 1 failed
  ```
  Phase 1 already includes `0 failed` in the format, so this is changing the count from hardcoded zero to an actual count of failed results.

- **WriteFailures** (new function) — after the summary line, if there are any failures, print a blank line followed by a `Failures:` section:
  ```

  Failures:
  - Task "foo": Missing required field
  - Task "bar": Invalid date format
  ```
  Each line: `- Task "`, title, `": `, error reason. For empty titles, use `(unknown)` as the title in quotes.

  Method signature: `WriteFailures(w io.Writer, results []Result)` or equivalent. This function is a no-op (writes nothing) when there are zero failures. It should filter results to only `Success: false` entries.

- **Integration with Present()** — if there is a top-level `Present` function, update its flow to call `WriteFailures` after `WriteSummary` when failures exist. The full output sequence becomes: header, per-task lines, blank line, summary, and conditionally: blank line, failures section.

- **Special characters in error reasons**: Error reasons may contain quotes, angle brackets, newlines, or other special characters. Print them verbatim using `fmt.Fprintf` — no escaping or sanitization.

## Tests

- `"WriteResult prints cross mark and skip reason for failed result"`
- `"WriteResult prints checkmark for successful result"` (regression — still works)
- `"WriteResult prints '(unknown)' as title when failed result has empty title"`
- `"WriteSummary counts failures correctly in summary line"`
- `"WriteSummary prints 'Done: 2 imported, 1 failed' for mixed results"`
- `"WriteSummary prints 'Done: 0 imported, 3 failed' when all tasks fail"`
- `"WriteSummary prints 'Done: 3 imported, 0 failed' when no failures"` (regression)
- `"WriteFailures prints failure detail section with each failure listed"`
- `"WriteFailures prints nothing when there are zero failures"`
- `"WriteFailures uses '(unknown)' for failures with empty title"`
- `"WriteFailures preserves special characters in failure reason"`
- `"Present renders full output with failures: header, results, summary, failure detail"`
- `"Present omits failure detail section when all results are successful"`
- `"Present with all failures shows zero imported and failure detail section"`

## Edge Cases

**Failure with empty title**: A `Result{Title: "", Success: false, Err: errors.New("missing title")}` arrives — the task had no title. The per-task line renders as `  ✗ Task: (unknown) (skipped: missing title)` and the failure detail renders as `- Task "(unknown)": missing title`. This ensures every failure is visible even when the title is missing.

**Failure reason with special characters**: The `Err.Error()` string contains quotes, colons, unicode, or other characters. The presenter prints the reason verbatim without escaping. Tests verify the exact error string appears in output.

**Zero failures (detail section omitted)**: When all results have `Success: true`, the `WriteFailures` function writes nothing. The output stops after the summary line — no blank line, no `Failures:` header, no trailing content. This is the Phase 1 behavior preserved.

## Acceptance Criteria

- [ ] Failed results render as `  ✗ Task: <title> (skipped: <reason>)` with two-space indent
- [ ] Failed results with empty title use `(unknown)` as the title placeholder
- [ ] Summary line shows correct count of both imported and failed tasks
- [ ] Failures detail section prints after summary when failures exist, with format `- Task "<title>": <reason>`
- [ ] Failures detail section is completely absent (no blank line, no header) when there are zero failures
- [ ] Special characters in error reasons are printed verbatim
- [ ] Successful result rendering is unchanged (regression)
- [ ] All output goes to the provided `io.Writer`
- [ ] All tests written and passing

## Context

The specification defines the complete output format including failures:

```
Importing from beads...
  ✓ Task: Implement login flow
  ✓ Task: Fix database connection
  ✗ Task: Broken entry (skipped: missing title)

Done: 2 imported, 1 failed

Failures:
- Task "foo": Missing required field
- Task "bar": Invalid date format
```

The `Result` struct (from migration-1-1) has `Title string`, `Success bool`, and `Err error`. Phase 1's presenter (migration-1-4) established `WriteHeader`, `WriteResult`, `WriteSummary` writing to `io.Writer`. This task extends those functions and adds `WriteFailures` for the detail section.

After migration-2-1, the engine returns all outcomes in `[]Result` — failures have `Success: false` and `Err` set to a non-nil error with the reason. The presenter uses `Err.Error()` to extract the human-readable reason string.

Specification reference: `docs/workflow/specification/migration.md`
