TASK: Extract cascade output helper function (acps-4-4)

ACCEPTANCE CRITERIA: Cascade output formatting logic (building CascadeResult and calling FormatCascadeTransition) should be extracted into a helper to reduce duplication across RunTransition, RunCreate, and RunUpdate.

STATUS: Complete

SPEC CONTEXT: The spec defines FormatCascadeTransition as a Formatter interface method that renders cascade output. Three CLI commands (transition, create, update) need to conditionally output either a simple transition or a cascade transition. Before this task, each had inline logic choosing between FormatTransition and FormatCascadeTransition.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/helpers.go:98-108 (outputTransitionOrCascade function)
- Notes: The helper cleanly encapsulates the branching logic: nil or empty cascade uses FormatTransition, otherwise FormatCascadeTransition. All three callers (transition.go:56, create.go:285, update.go:420+425) now use the helper with no direct FormatTransition/FormatCascadeTransition calls remaining. The buildCascadeResult function remains in transition.go:66-138, which is reasonable since it is a data-construction concern rather than output formatting.

TESTS:
- Status: Adequate
- Coverage: helpers_test.go:293-408 (TestOutputTransitionOrCascade) covers 6 subtests: nil cascade result, empty cascade list, cascade with entries, parity with inline FormatTransition pattern, parity with inline FormatCascadeTransition pattern, JSON formatter. Tests across multiple formatter types (Toon, Pretty, JSON).
- Notes: Good balance. Tests verify behavioral equivalence with the old inline pattern, ensuring the refactor is safe. No over-testing observed.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, error wrapping conventions.
- SOLID principles: Good. Single responsibility -- helper does one thing (choose and output the right format). Open/closed -- works with any Formatter implementation.
- Complexity: Low. Simple nil/length check with two branches.
- Modern idioms: Yes. Pointer-based optional parameter (*CascadeResult) is idiomatic Go.
- Readability: Good. Clear doc comment explains the contract (callers must build CascadeResult inside Mutate closure).
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
