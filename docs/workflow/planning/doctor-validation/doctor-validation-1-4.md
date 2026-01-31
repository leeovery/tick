---
id: doctor-validation-1-4
phase: 1
status: pending
created: 2026-01-30
---

# tick doctor Command Wiring

## Goal

Tasks 1-1 through 1-3 built the diagnostic runner, output formatter, and cache staleness check as standalone, testable units. But there is no way for a user or agent to invoke `tick doctor` from the CLI. This task wires the `doctor` subcommand into the existing CLI framework (established in tick-core-1-5), connecting the command to the diagnostic runner, registering the cache staleness check, formatting output to stdout, and returning the correct exit code. It also verifies the critical invariant that doctor never modifies any data — it is a strictly read-only operation. Without this task, the entire doctor framework exists but is unreachable.

## Implementation

- Add a `doctor` subcommand handler in the CLI dispatch (alongside existing commands like `init`, `create`, `list`, etc.). The handler is invoked when the user runs `tick doctor`.
- In the handler:
  1. Discover the `.tick/` directory using the existing directory discovery helper (walks up from cwd). If `.tick/` is not found, write an error to stderr: `Error: Not a tick project (no .tick directory found)` and return exit code 1. This is the same error as other commands — doctor requires an initialized project.
  2. Create a `DiagnosticRunner` instance (from task 1-1).
  3. Register the `CacheStalenessCheck` (from task 1-3), passing the `.tick/` directory path.
  4. Call `RunAll()` to execute all registered checks and get a `DiagnosticReport`.
  5. Create the formatter (from task 1-2) and write the formatted output to `os.Stdout`.
  6. Compute the exit code using `ExitCode(report)` (from task 1-2).
  7. Exit with the computed code.
- Doctor takes no flags or arguments. It does not support `--quiet`, `--verbose`, `--toon`, `--pretty`, or `--json` — the specification states doctor outputs human-readable text only, with no structured output variants. Ignore format flags for this command.
- Doctor does not acquire a write lock. It is a read-only operation. If it reads files, it opens them in read-only mode. It must not write to `tasks.jsonl`, `cache.db`, or any other file in `.tick/`. The cache staleness check (task 1-3) already opens `cache.db` in read-only mode; this task verifies the end-to-end guarantee.
- Doctor does not trigger a cache rebuild. Even if the cache is stale, doctor reports the staleness and suggests `tick rebuild` — it does not auto-rebuild. This is distinct from normal read commands which auto-rebuild on stale cache.

## Tests

- `"it exits 0 and prints all-pass output when data store is healthy"`
- `"it exits 1 and prints failure output when cache is stale"`
- `"it prints formatted output to stdout with ✓/✗ markers and summary line"`
- `"it errors with exit code 1 when .tick directory is not found"`
- `"it prints 'Not a tick project' error to stderr when .tick directory is missing"`
- `"it does not modify tasks.jsonl (file unchanged after doctor runs)"`
- `"it does not modify cache.db (file unchanged after doctor runs)"`
- `"it does not create any new files in .tick directory"`
- `"it does not trigger a cache rebuild when cache is stale"`
- `"it registers and runs the cache staleness check"`
- `"it shows 'No issues found.' summary when all checks pass"`
- `"it shows '1 issue found.' summary when cache staleness fails"`

## Edge Cases

- **`.tick` directory not found**: Doctor needs an initialized tick project just like any other command. When no `.tick/` directory is found by walking up from cwd, doctor must exit with code 1 and print the standard error to stderr. It must not print diagnostic output (no `✓` or `✗` lines) — the error is a precondition failure, not a diagnostic result.
- **Doctor never modifies data (read-only verification)**: This is the most critical invariant of doctor. The end-to-end test must verify that after running `tick doctor`, the byte contents of `tasks.jsonl` and `cache.db` are identical to before the run, and no new files are created in `.tick/`. This covers: no accidental writes to JSONL, no cache rebuild triggered, no lock file left behind, no temp files created. The specification states doctor "never modifies data" and is "diagnostic only." This edge case is verified via file checksums or stat comparison before and after.
- **Stale cache does not trigger rebuild**: Normal tick read operations (list, show, ready) auto-rebuild when the cache is stale. Doctor must explicitly not do this. When doctor detects a stale cache, it reports the staleness and exits — it does not attempt to fix it. This requires that the doctor command handler bypasses the normal storage engine read path (which includes auto-rebuild) and instead uses the runner/check architecture directly.
- **Empty project (no tasks)**: When `.tick/` exists with an empty `tasks.jsonl` and a matching `cache.db`, doctor should report all checks passing. This is a valid healthy state.

## Acceptance Criteria

- [ ] `tick doctor` subcommand registered in CLI dispatch
- [ ] Doctor discovers `.tick/` directory via existing helper
- [ ] Missing `.tick/` directory produces error to stderr with exit code 1
- [ ] DiagnosticRunner created with CacheStalenessCheck registered
- [ ] Formatted output written to stdout (✓/✗ markers, details, suggestions, summary)
- [ ] Exit code 0 when all checks pass, exit code 1 when any error found
- [ ] Doctor outputs human-readable text only (no TOON/JSON variants)
- [ ] Doctor does not modify `tasks.jsonl` — verified by before/after comparison
- [ ] Doctor does not modify `cache.db` — verified by before/after comparison
- [ ] Doctor does not create new files in `.tick/`
- [ ] Doctor does not trigger cache rebuild when cache is stale
- [ ] Tests written and passing for all edge cases

## Context

The specification defines `tick doctor` in the CLI command reference table: "Run diagnostics and validation." It also establishes that doctor is "diagnostic only — it reports problems and suggests remedies but never modifies data." Design principle #1: "Report, don't fix — Doctor diagnoses and suggests; user/agent decides what to run." Design principle #2: "Human-focused — Debugging tool for humans; agents don't need to parse diagnostic output."

The output format section states: "Doctor outputs human-readable text only. No TOON/JSON variants." This means doctor should ignore the normal TTY detection and format flag machinery — it always outputs the same format regardless of context. The rationale: "Doctor is a debugging/maintenance tool run by humans investigating issues. Agents use normal operations (`ready`, `start`, `done`) — they don't parse diagnostics."

Exit codes: 0 = all checks passed (no errors, warnings allowed), 1 = one or more errors found. This maps directly to `ExitCode(report)` from task 1-2.

The `.tick/` directory discovery pattern is established in tick-core-1-5: walk up from cwd looking for `.tick/`, error if not found. Doctor reuses this same helper and error message.

This is a Go project. The CLI dispatch pattern from tick-core-1-5 uses `os.Args` subcommand routing. Doctor plugs into this same routing table. The diagnostic runner, formatter, and cache check are all from earlier tasks in this phase.

Specification reference: `docs/workflow/specification/doctor-validation.md` (for ambiguity resolution)
