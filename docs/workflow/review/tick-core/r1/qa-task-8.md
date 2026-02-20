TASK: Status transition validation logic (tick-core-2-1)

ACCEPTANCE CRITERIA:
- [ ] All 7 valid status transitions succeed with correct new status
- [ ] All 9 invalid transitions return error
- [ ] Task not modified on invalid transition
- [ ] `closed` set to current UTC on done/cancelled
- [ ] `closed` cleared on reopen
- [ ] `updated` refreshed on every valid transition
- [ ] Error messages include command name and current status
- [ ] Function returns old and new status

STATUS: Complete

SPEC CONTEXT: The specification defines 4 status values (open, in_progress, done, cancelled) with transitions governed by commands (start, done, cancel, reopen). The `closed` field is an optional datetime set on done/cancelled and cleared on reopen. The `updated` timestamp refreshes on any mutation. Error messages should include the command and current status. The spec also requires the transition output format `tick-a3f2b7: open -> in_progress` which means the function must return old and new status.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/task/transition.go:1-75
- Notes:
  - `TransitionResult` struct holds `OldStatus` and `NewStatus` (line 10-13) -- satisfies "returns old and new status"
  - `transitionTable` map (line 16-24) defines all 4 commands with valid source statuses and target status -- correct, covers all 7 valid transitions
  - `Transition` function (line 34-65) validates command, checks source status, mutates task in-place on success, returns error without modification on failure
  - `closed` correctly set on done/cancelled (line 55), cleared on reopen (line 58)
  - `updated` set to `time.Now().UTC().Truncate(time.Second)` on every valid transition (line 48-51)
  - Error format: `"cannot %s task %s --- status is '%s'"` with em dash (\u2014). The "Error:" prefix is added by the CLI layer (app.go:94), which is idiomatic Go (errors start lowercase). Final output matches spec intent.
  - Pure domain logic with no I/O -- correct per task requirement
  - Unknown command returns a separate error -- good defensive coding

TESTS:
- Status: Adequate
- Coverage: All 21 planned tests are present in /Users/leeovery/Code/tick/internal/task/transition_test.go
  - 7 valid transitions tested in `TestTransition_ValidTransitions` (lines 27-112): verifies new status, OldStatus/NewStatus in result, and updated timestamp range
  - 9 invalid transitions tested in `TestTransition_InvalidTransitions` (lines 114-189): verifies error returned and error message matches expected format
  - 2 closed timestamp tests (done, cancelled) in `TestTransition_ClosedTimestamp` (lines 191-228): verifies timestamp set and within range
  - 1 closed cleared test (reopen) in `TestTransition_ClosedTimestamp` (lines 230-242): verifies nil after reopen
  - 1 updated timestamp test covering 4 representative commands in `TestTransition_UpdatedTimestamp` (lines 244-282): verifies timestamp changed from original
  - 1 no-modification test in `TestTransition_NoModificationOnInvalid` (lines 284-308): verifies status, updated, and closed unchanged
- Notes:
  - Tests are well-structured using table-driven subtests per Go conventions
  - Each test verifies behavior, not implementation details
  - The valid transition tests also verify the TransitionResult fields, not just task mutation
  - The invalid transition tests verify exact error message content
  - No over-testing detected -- each test case covers a distinct state transition pair

CODE QUALITY:
- Project conventions: Followed. Table-driven tests with subtests per golang-pro skill. Helper functions marked appropriately. Pure domain logic separated from CLI concerns.
- SOLID principles: Good. Single responsibility (Transition does one thing: validate and apply). Open/closed via transitionTable map (new commands can be added without modifying Transition logic). Dependency inversion: no I/O dependencies, caller handles persistence.
- Complexity: Low. Linear flow with early returns. The transitionTable map reduces cyclomatic complexity compared to nested if/switch. `statusIn` helper is clean.
- Modern idioms: Yes. Uses pointer for optional `Closed` field (`*time.Time`), proper UTC truncation, em dash via Unicode escape.
- Readability: Good. Function is 30 lines, well-documented with godoc comment. TransitionResult struct is self-explanatory. Variable names are clear.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The error message uses lowercase "cannot" while the task spec shows "Cannot" (capital C). However, the "Error:" prefix is added at the CLI layer, and Go convention is that error strings should not be capitalized. This is correct Go style and the final user-facing output reads "Error: cannot start task tick-a1b2c3 --- status is 'done'" which is clear and acceptable.
