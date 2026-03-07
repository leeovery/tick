TASK: Wire ValidateAddChild and done-parent reopen cascade into RunCreate

ACCEPTANCE CRITERIA:
- Creating a task with a done parent triggers reopen cascade (Rule 6) with cascade output
- All existing tests pass with no regressions
- Edge cases: parent cancelled (error), parent open (no cascade), parent done (reopen cascade)

STATUS: Complete

SPEC CONTEXT: Rule 6 states adding a non-terminal child to a done parent triggers parent reopen to open. Rule 7 blocks adding a child to a cancelled parent. ValidateAddChild handles only validation (cancelled check); the done-parent reopen is the caller's responsibility. Changes must be persisted atomically in a single Store.Mutate call.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/create.go:180-288 (RunCreate with parent validation and reopen cascade), /Users/leeovery/Code/tick/internal/cli/helpers.go:113-133 (validateAndReopenParent helper), /Users/leeovery/Code/tick/internal/task/state_machine.go:71-76 (ValidateAddChild)
- Notes: Implementation correctly follows the spec. ValidateAddChild is pure validation (cancelled check only). RunCreate calls validateAndReopenParent inside the Mutate closure, which calls ValidateAddChild then ApplyWithCascades for done parents. Cascade result is built inside Mutate while tasks slice is valid (defensive copy concern addressed). Cascade output is displayed after the created task detail, respecting quiet mode. All changes (parent reopen + new child) persisted atomically in a single Mutate call.

TESTS:
- Status: Adequate
- Coverage:
  - Cancelled parent blocks create with correct error message (create_test.go:461)
  - Done parent triggers reopen, parent status becomes open, Closed cleared (create_test.go:1070)
  - Done grandparent cascade: both parent and grandparent reopened (create_test.go:1120)
  - Open parent: no cascade, status unchanged, no cascade output (create_test.go:1158)
  - In-progress parent: no cascade, status unchanged (create_test.go:1186)
  - Atomic persistence of parent reopen + new child (create_test.go:1214)
  - Cascade output appears after task detail in stdout (create_test.go:1255)
  - Unit tests for validateAndReopenParent helper: open no-op, in_progress no-op, cancelled error, done reopen, normalized ID, nonexistent ID (helpers_test.go:410-509)
- Notes: All three edge cases from the plan are covered. Tests verify both persistence and output. The helper function has thorough unit tests. No over-testing detected -- each test covers a distinct scenario.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, "it does X" naming, Store.Mutate for writes, error wrapping patterns.
- SOLID principles: Good. validateAndReopenParent extracted as a reusable helper (also used in update.go). ValidateAddChild is pure validation per spec. Separation of validation, mutation, and display.
- Complexity: Low. The parent handling in RunCreate is a clean if-block. validateAndReopenParent is a linear scan with early returns.
- Modern idioms: Yes. Standard Go patterns throughout.
- Readability: Good. Comments reference rule numbers (Rule 6, Rule 7). Variable names are clear. The flow is: validate -> reopen if needed -> build cascade result -> create task -> output.
- Issues: None identified.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Line 70 of state_machine.go has a minor typo: `/ Note:` should be `// Note:` (missing leading slash in comment). This is cosmetic and pre-existing from an earlier task.
