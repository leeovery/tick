TASK: Surface beads provider parse/validation errors as failed results instead of silently dropping

ACCEPTANCE CRITERIA:
- Empty-title entries from the JSONL file appear as failed results in migration output with a validation error message.
- Entries that fail validation (e.g., out-of-range priority) appear as failed results rather than being silently dropped.
- Malformed JSON lines produce visible failures in the output.
- Valid entries continue to import successfully.

STATUS: Complete

SPEC CONTEXT: The specification (Error Handling section) states: "Continue on error, report failures at end. When a task fails to import: 1. Log the failure with reason, 2. Continue processing remaining tasks, 3. Report summary at end." The spec output format shows failures inline (cross mark with "(skipped: reason)") and a "Failures:" detail section listing each failed task with its reason. This task ensures the beads provider does not silently drop entries before they reach the engine, which is the component responsible for validation and failure reporting.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/migrate/beads/beads.go:86-105` (Tasks method)
  - `/Users/leeovery/Code/tick/internal/migrate/beads/beads.go:116-138` (mapToMigratedTask)
- Notes:
  - Malformed JSON lines (line 93-101): Creates a sentinel `MigratedTask{Title: "(malformed entry)", Status: task.Status("(invalid)")}`. The "(invalid)" status forces engine validation to reject it, making the failure visible. This approach is effective and aligns with the task's "Do" instructions.
  - Empty-title entries: The provider no longer skips them. `mapToMigratedTask` preserves the empty title as-is (line 116 comment confirms this). The engine's `Validate()` at `engine.go:71` catches it and produces a failed Result with `FallbackTitle = "(untitled)"`.
  - Validation removed from provider: No `Validate()` call exists in the provider. The engine is the single validation point at `engine.go:71`. This eliminates the double-validation issue described in the task.
  - Blank lines (whitespace-only) in the JSONL file are still skipped (line 88-90), which is correct -- these are not data entries, they are file formatting artifacts.
  - All acceptance criteria are met: empty titles, invalid priorities, and malformed JSON all flow through to the engine and appear as failed Results in the output.

TESTS:
- Status: Adequate
- Coverage:
  - **Beads provider unit tests** (`/Users/leeovery/Code/tick/internal/migrate/beads/beads_test.go`):
    - Line 280-303: "Tasks returns malformed JSON lines as sentinel MigratedTask entries" -- verifies malformed JSON produces `(malformed entry)` title, 3 entries returned (not 2).
    - Line 305-324: "Tasks returns entries with empty title for engine validation" -- verifies empty-title entries are returned, not skipped.
    - Line 326-345: "Tasks returns entries with whitespace-only title for engine validation" -- verifies whitespace titles pass through.
    - Line 347-373: "Tasks returns entries with invalid priority for engine validation" -- verifies priority=99 passes through with value intact.
    - Line 375-406: "Tasks returns all entries from mixed valid invalid and malformed JSONL" -- comprehensive mix test: valid, empty-title, malformed JSON, invalid priority, valid. All 5 returned.
  - **mapToMigratedTask unit test** (`beads_test.go:549-565`): Verifies empty title is preserved in mapping.
  - **Engine tests** (`/Users/leeovery/Code/tick/internal/migrate/engine_test.go`):
    - Line 169-206: Validates that tasks with invalid status fail validation in the engine.
    - Line 577-603: Validates that empty-title tasks get `FallbackTitle` and fail with error.
  - **CLI integration test** (`/Users/leeovery/Code/tick/internal/cli/migrate_test.go:315-348`): "migrate shows failures for invalid entries from beads provider" -- end-to-end test with a fixture containing a valid task, empty-title entry, malformed JSON, and invalid priority. Asserts: "Done: 1 imported, 3 failed", presence of "Failures:" section, "title is required" error, "priority must be" error, and "(malformed entry)" sentinel in output.
- Notes: Test coverage is thorough and well-balanced. Unit tests verify provider behavior (entries returned, not dropped). Integration test verifies the full pipeline from JSONL file to CLI output. Tests would break if the feature regressed (e.g., if someone re-added the skip logic, the count assertions would fail). No over-testing observed -- each test targets a distinct scenario.

CODE QUALITY:
- Project conventions: Followed. Uses `t.Helper()`, `t.TempDir()`, `t.Run()` subtests, stdlib testing only (no testify). Error wrapping with `fmt.Errorf`. Handler follows project patterns.
- SOLID principles: Good. Single responsibility -- the provider reads and maps, the engine validates and reports. Removing `Validate()` from the provider eliminated a responsibility violation. The sentinel approach for malformed JSON keeps the provider's contract simple (returns all entries).
- Complexity: Low. The `Tasks()` method is a straightforward scanner loop with two branches (malformed JSON vs valid JSON). No nested conditionals or complex control flow.
- Modern idioms: Yes. Uses `*int` for optional priority (nullable pattern). Uses `bufio.Scanner` idiomatically. Sentinel values with descriptive titles are a pragmatic approach.
- Readability: Good. Clear comments on lines 61-65 explain the strategy ("Malformed JSON lines are returned as sentinel entries... Empty titles and validation failures are left for the engine to handle"). The sentinel value `"(malformed entry)"` with `"(invalid)"` status is self-documenting.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The sentinel `Status: task.Status("(invalid)")` on malformed entries is a pragmatic choice that works because the engine's validation rejects unknown statuses. An alternative would be to introduce a dedicated error field on MigratedTask, but the current approach is simpler and effective for the single-provider use case.
- The malformed entry sentinel does not include the line number or the original JSON error message. The task's "Do" section suggested including "a title describing the line number/error." The current `"(malformed entry)"` title is less specific but still makes the failure visible. This is minor -- a user encountering malformed JSON in their beads file would need to inspect the file manually regardless.
