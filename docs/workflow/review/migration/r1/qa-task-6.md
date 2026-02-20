TASK: Engine Continue-on-Error (migration-2-1)

ACCEPTANCE CRITERIA:
- Engine.Run continues processing remaining tasks when CreateTask returns an error
- Each insertion failure produces a Result{Success: false, Err: <original error>} in the results slice
- Engine.Run returns nil error when provider succeeds, even if all tasks fail insertion
- Engine.Run still returns an error immediately when provider.Tasks() fails (unchanged)
- Results slice contains one entry per task from the provider, in order, regardless of success/failure
- Phase 1 fail-fast test is replaced with continue-on-error tests
- All tests written and passing

STATUS: Complete

SPEC CONTEXT: The specification explicitly requires "Continue on error, report failures at end." When a task fails to import: (1) log the failure with reason, (2) continue processing remaining tasks, (3) report summary at end. No rollback -- successfully imported tasks remain even if others fail. The engine's error return should only signal provider-level failures (cannot read source data), not per-task failures.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/migrate/engine.go:59-89
- Notes: The `Engine.Run` method handles both validation and insertion failures identically: record a `Result{Title: task.Title, Success: false, Err: err}` and continue to the next task. The loop always completes and returns `(results, nil)`. The only early return with a non-nil error is when `provider.Tasks()` fails (line 62). The insertion failure path at lines 80-83 appends a failure Result with the original CreateTask error and uses `continue`, which is exactly the continue-on-error behavior specified. The implementation is clean, minimal, and directly matches the acceptance criteria.

TESTS:
- Status: Adequate
- Coverage: All 9 tests specified in the plan are present in the test file, plus several pre-existing tests from Phase 1 that cover related behavior:
  - "it continues processing after CreateTask fails and records failure Result" (line 324) -- verifies engine continues past insertion failure, checks all 3 tasks sent to creator
  - "it returns nil error when all tasks fail insertion" (line 368) -- edge case: all fail, error is nil
  - "it returns nil error with mixed validation and insertion failures" (line 399) -- mixed failure types, 4 tasks
  - "failure Result from insertion contains the original CreateTask error" (line 447) -- error identity preserved
  - "results slice contains entries for all tasks in provider order regardless of success or failure" (line 475) -- ordering preserved
  - "successful tasks are persisted even when later tasks fail insertion" (line 605) -- valid tasks after failures still reach creator
  - "all tasks fail insertion returns results with all failures and nil error" (line 645) -- edge case with distinct errors per task
  - "mixed failures: validation then insertion then success produces three Results in order" (line 680) -- ordering across failure types, verifies creator call count
  - "it returns error immediately when provider.Tasks() fails" (line 283) -- unchanged provider-level error behavior
  - Phase 1 fail-fast test for CreateTask failure is confirmed removed (no matches found)
- Notes: Tests are well-structured with clear names matching the plan. Each test verifies distinct behavior. The mockTaskCreator with per-call error map is an effective test double. Edge cases (all fail, mixed failures) are covered as specified.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing only, t.Run subtests, error wrapping, DI via struct fields, functional mockTaskCreator with per-call control
- SOLID principles: Good -- Engine has single responsibility (orchestrate iteration), TaskCreator interface isolates persistence, Provider interface isolates data source. Open/closed: new failure types don't require engine changes.
- Complexity: Low -- Engine.Run is a simple linear loop with two conditional branches (validate, create). No nested complexity. Cyclomatic complexity is approximately 4.
- Modern idioms: Yes -- clean for-range loop, named returns avoided (explicit returns), pre-allocated slice with `make([]Result, 0, len(tasks))`
- Readability: Good -- clear variable names, the flow is obvious (validate -> create -> append result), doc comment on Run accurately describes the contract
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The mockTaskCreator in engine_test.go tracks both `calls` (all tasks) and `callIdx` but the `ids` return values are not consumed by any test (the engine discards the ID returned by CreateTask). This is harmless but slightly over-specified in the mock. Not worth changing since it matches the TaskCreator interface signature.
