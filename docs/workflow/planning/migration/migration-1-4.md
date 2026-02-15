---
id: migration-1-4
phase: 1
status: completed
created: 2026-01-31
---

# Migration Output - Per-Task & Summary

## Goal

The migration engine (migration-1-3) returns `[]Result` to its caller, but nothing formats those results for human consumption. Without an output layer, users running `tick migrate --from beads` see nothing — no per-task feedback, no summary. This task builds the presenter that turns `[]Result` into the spec's output format: a header line, one checkmark line per task, and a summary line showing the count of imported tasks.

Phase 1 only renders the success path (all checkmarks, imported count). Phase 2 will add failure markers (`✗`) and mixed success/failure summary.

## Implementation

- Create a `Presenter` (or `Formatter`) in `internal/migrate/` that accepts an `io.Writer` and the provider name, and exposes methods to render migration output.
- Implement three output responsibilities:

  **1. Header line** — printed before any tasks are processed:
  ```
  Importing from beads...
  ```
  Method: `WriteHeader(w io.Writer, providerName string)` or equivalent.

- **2. Per-task line** — printed for each result as it is processed:
  ```
    ✓ Task: Implement login flow
  ```
  Method: `WriteResult(w io.Writer, r Result)`. In Phase 1, all results passed to this function are successful, so always render the `✓` prefix. Handle `Success: false` gracefully (only print `✓` for successful results) for forward-compatibility with Phase 2.

- **3. Summary line** — printed after all tasks, preceded by a blank line:
  ```

  Done: 3 imported, 0 failed
  ```
  Method: `WriteSummary(w io.Writer, results []Result)`. Count successful results for the "imported" number. Include ", 0 failed" portion to establish the format for Phase 2.

- Alternatively, implement a single `Present(w io.Writer, providerName string, results []Result)` function that calls all three steps in sequence. Choose whichever API makes the caller (migration-1-5 CLI command) cleanest.

- **Title truncation for long titles**: The spec does not prescribe truncation, and terminal wrapping is acceptable. Do NOT truncate titles. Print them in full.

- **Zero tasks**: When `results` is empty, still print the header and summary:
  ```
  Importing from beads...

  Done: 0 imported, 0 failed
  ```

- Use `fmt.Fprintf` to the provided `io.Writer` for all output. This makes the presenter testable (write to a `bytes.Buffer` in tests) and flexible (the CLI will pass `os.Stdout`).

- The presenter is a pure formatting concern — it does NOT call the engine or provider. It receives already-computed results.

## Tests

- `"WriteHeader prints 'Importing from <provider>...' with provider name"`
- `"WriteResult prints checkmark and task title for successful result"`
- `"WriteSummary prints 'Done: N imported, 0 failed' with correct count"`
- `"Present renders full output: header, per-task lines, blank line, summary"`
- `"Present with zero results prints header and summary with zero counts"`
- `"Present with single result prints one task line and count of 1"`
- `"Present with multiple results prints each task on its own line"`
- `"per-task lines are indented with two spaces"`
- `"long titles are printed in full without truncation"`
- `"summary line is separated from task lines by a blank line"`
- `"provider name in header matches what provider.Name() returns"`

## Edge Cases

**Zero tasks imported**: The provider returned an empty `[]MigratedTask` (e.g., empty beads file). The engine produces an empty `[]Result`. The presenter prints the header and summary with zero counts. No per-task lines. No error message — this is normal.

**Long titles**: A task title that is very long (e.g., 200+ characters). The presenter prints it in full. The test verifies the full title appears in the output without truncation or corruption. Terminal line wrapping is the user's concern, not ours.

## Acceptance Criteria

- [ ] Presenter exists in `internal/migrate/` and writes to an `io.Writer`
- [ ] Header line prints `Importing from <provider>...` using the provider name
- [ ] Each successful result prints `  ✓ Task: <title>` (two-space indent, checkmark, task prefix)
- [ ] Summary line prints `Done: N imported, 0 failed` with the correct count of successful results
- [ ] A blank line separates the last per-task line from the summary line
- [ ] Zero results produce header + blank line + summary with `0 imported, 0 failed`
- [ ] Long titles are printed in full without truncation
- [ ] All output goes to the provided `io.Writer` (not hardcoded to stdout)
- [ ] All tests written and passing

## Context

The specification defines the output format:

```
Importing from beads...
  ✓ Task: Implement login flow
  ✓ Task: Fix database connection
  ✓ Task: Add unit tests
  ✗ Task: Broken entry (skipped: missing title)

Done: 3 imported, 1 failed
```

Phase 1 only implements success output (the `✓` lines and the imported count). The `✗` failure lines and mixed summary are Phase 2 scope. However, including `0 failed` in the Phase 1 summary line establishes the format so Phase 2 only needs to change the count, not the format.

The `Result` struct (from migration-1-1) has `Title string`, `Success bool`, and `Err error`. The presenter should only print `✓` for `Success: true` results, keeping it correct for Phase 2 without extra work.

The engine (migration-1-3) returns `[]Result` to the caller. The CLI command (migration-1-5) will call the engine, then pass results to this presenter.

Specification reference: `docs/workflow/specification/migration.md`
