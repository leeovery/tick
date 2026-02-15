TASK: tick start, done, cancel, reopen commands (tick-core-2-2)

ACCEPTANCE CRITERIA:
- All four commands transition correctly and output transition line
- Invalid transitions return error to stderr with exit code 1
- Missing/not-found task ID returns error with exit code 1
- --quiet suppresses success output
- Input IDs normalized to lowercase
- Timestamps managed correctly (closed set/cleared, updated refreshed)
- Mutation persisted through storage engine

STATUS: Complete

SPEC CONTEXT:
The spec defines four status transition commands (start, done, cancel, reopen) with a fixed transition table. Output format is `tick-a3f2b7: open -> in_progress` with Unicode arrow. --quiet suppresses stdout on success. All errors to stderr with exit code 1. IDs case-insensitive, normalized to lowercase. `closed` timestamp set on done/cancel, cleared on reopen. `updated` refreshed on every mutation.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/cli/transition.go (CLI command handler, 50 lines)
  - /Users/leeovery/Code/tick/internal/task/transition.go (pure domain transition logic, 75 lines)
  - /Users/leeovery/Code/tick/internal/cli/app.go:76 (command registration: "start", "done", "cancel", "reopen")
  - /Users/leeovery/Code/tick/internal/cli/format.go:136 (FormatTransition with Unicode arrow)
- Notes:
  - All four commands share a single `RunTransition` handler parameterized by command name (DRY)
  - Transition table in task/transition.go maps command -> valid from-statuses + target status
  - ID normalization via `task.NormalizeID(args[0])` at line 18 of transition.go
  - Missing ID check at line 14, task-not-found at line 39, invalid transition delegated to domain logic
  - --quiet check at line 45 suppresses output; errors still propagated via return
  - Errors bubble up through app.go:94 which prefixes "Error: " and writes to stderr
  - Mutation persisted via `store.Mutate` callback pattern (atomic write)
  - Closed timestamp set/cleared in domain logic (transition.go:53-58)
  - Updated timestamp refreshed in domain logic (transition.go:51)
  - Output format uses Unicode right arrow (U+2192) matching spec: `tick-a3f2b7: open -> in_progress`

TESTS:
- Status: Adequate
- Coverage:
  - /Users/leeovery/Code/tick/internal/cli/transition_test.go: 18 test cases covering all planned scenarios
  - /Users/leeovery/Code/tick/internal/task/transition_test.go: Domain-level tests for all valid transitions, all invalid transitions, closed timestamp, updated timestamp, no-modification-on-invalid
  - /Users/leeovery/Code/tick/internal/cli/base_formatter_test.go: FormatTransition output format verification with Unicode arrow
  - Tests match the planned test list exactly (all 18 tests from the task plan are present)
  - CLI tests use full App.Run() flow through setupTickProjectWithTasks helpers, verifying end-to-end behavior
  - Persistence verified by reading back from JSONL on disk (readPersistedTasks)
  - Edge cases covered: uppercase ID input, missing ID, not-found ID, invalid transition, --quiet flag, stderr-only errors, exit code 1, closed timestamp set/cleared, updated timestamp refreshed
- Notes:
  - Good separation: domain logic tested independently in task/transition_test.go, CLI integration tested in cli/transition_test.go
  - Table-driven tests used appropriately (valid transitions, invalid transitions, closed timestamp sub-tests)
  - No over-testing detected; each test verifies a distinct behavior

CODE QUALITY:
- Project conventions: Followed -- table-driven tests, proper error handling, exported function documentation
- SOLID principles: Good -- single responsibility (RunTransition handles CLI, Transition handles domain), open/closed (transition table is data-driven), dependency inversion (Formatter interface)
- Complexity: Low -- RunTransition is ~35 lines, Transition is ~30 lines, linear code paths
- Modern idioms: Yes -- idiomatic Go error handling, pointer receiver for mutation, Mutate callback pattern
- Readability: Good -- self-documenting, clear variable names, comments where needed
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Minor casing difference in error message: code produces "task ID is required" (lowercase t) while the task doc shows "Task ID is required" (uppercase T). The test is flexible enough to pass either way, and the spec does not prescribe exact casing for this message. Not a blocking issue.
