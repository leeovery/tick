TASK: Extract Rule 3 completion evaluation into shared function (acps-4-3)

ACCEPTANCE CRITERIA:
- Rule 3 completion evaluation logic extracted into a shared function
- No duplication between Cascades() and CLI reparenting code
- Both call sites use the shared function

STATUS: Complete

SPEC CONTEXT: Rule 3 (upward completion cascade) states that when all children of a parent reach a terminal state, the parent auto-transitions to done (if any child done) or cancelled (if all cancelled). This logic is needed in two places: the Cascades() cascade engine and the CLI reparenting code in RunUpdate (which must re-evaluate the original parent after a child is moved away).

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/task/cascades.go:173-224 (EvaluateParentCompletion function)
- Callers:
  - /Users/leeovery/Code/tick/internal/task/cascades.go:118 (cascadeUpwardCompletion)
  - /Users/leeovery/Code/tick/internal/cli/update.go:137 (evaluateRule3)
- Notes: The function is exported as `EvaluateParentCompletion` with a clean signature returning `(action string, shouldComplete bool)`. Both call sites use it -- no duplication of the terminal-check logic exists in CLI code. The function handles all edge cases: parent not found, parent already terminal, no children, mixed done/cancelled children.

TESTS:
- Status: Adequate
- Coverage: TestEvaluateParentCompletion at /Users/leeovery/Code/tick/internal/task/cascades_test.go:851-995 covers 9 subtests:
  - All children terminal with at least one done -> done
  - All children cancelled -> cancel
  - Some children non-terminal -> false
  - Parent not found -> false
  - Parent already terminal -> false
  - Parent has no children -> false
  - Single child done -> done
  - Single child cancelled -> cancel
  - Case-insensitive ID matching
- Integration coverage via update_test.go Rule 3 tests (lines 1031-1234) verifying the CLI call path
- Notes: Good coverage of all logical branches. Tests are focused without redundancy.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, idiomatic Go patterns.
- SOLID principles: Good. Single responsibility -- function does one thing (evaluate completion). Clean separation from mutation logic.
- Complexity: Low. Linear scan with early returns. Clear control flow.
- Modern idioms: Yes. Named return values used appropriately for documentation.
- Readability: Good. Well-documented with godoc comment explaining the algorithm and when it triggers.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
