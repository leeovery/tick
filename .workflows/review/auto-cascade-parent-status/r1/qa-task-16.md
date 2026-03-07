TASK: Wire ApplyWithCascades into RunTransition

ACCEPTANCE CRITERIA:
- RunTransition uses StateMachine.ApplyWithCascades() and persists all changes atomically in a single Store.Mutate call
- Non-cascade single-task transitions still use existing FormatTransition with no visual regression

STATUS: Complete

SPEC CONTEXT: The specification requires that all cascade changes are persisted atomically within a single Store.Mutate call (temp file + fsync + rename). RunTransition should call ApplyWithCascades(), receive the primary result plus cascade changes, and write everything in one atomic operation. For single-task transitions (no cascades), the existing FormatTransition output must be preserved with no visual regression. Edge cases: no cascades, quiet mode suppresses output, task not found.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/transition.go:14-60
- Notes: RunTransition correctly instantiates a StateMachine, calls sm.ApplyWithCascades() inside Store.Mutate, and returns the modified tasks slice atomically. The cascade result is built inside the Mutate closure (where the tasks slice is valid), and output dispatch uses the outputTransitionOrCascade helper in helpers.go:102-108. The helper falls back to FormatTransition when cr is nil or has no cascaded entries, preserving non-cascade visual output. Quiet mode (fc.Quiet) correctly suppresses all output at line 55. Task-not-found returns an error at line 49. All acceptance criteria are met with no drift from the plan.

TESTS:
- Status: Adequate
- Coverage:
  - Single-task transition with FormatTransition output verified (line 389-407: "it transitions single task with no cascades using FormatTransition")
  - Downward cascade output verified (line 409-442: "it renders cascade output when done triggers downward cascade")
  - Upward cascade output verified (line 444-470: "it renders cascade output when start triggers upward cascade")
  - Atomic persistence of cascade changes verified (line 472-512: "it persists all cascade changes atomically")
  - Unchanged terminal children in cascade output verified (line 514-546)
  - Quiet mode suppression with cascades verified (line 622-645: "it suppresses cascade output in quiet mode")
  - Quiet mode suppression without cascades verified (line 180-197)
  - Task not found error verified (line 214-224)
  - Missing ID argument error verified (line 199-212)
  - Invalid transition error verified (line 226-241)
  - 3-level upward cascade rendering verified (line 548-579)
  - buildCascadeResult flat ParentIDs for upward cascade verified (line 581-620)
  - outputTransitionOrCascade helper tested independently in helpers_test.go:293-408 covering nil cascade, empty cascade, cascade with entries, all three formatters, and output equivalence with inline pattern
- Notes: All three edge cases from the task definition are covered. Tests verify both persisted state and output format. Test balance is good -- each test verifies a distinct behavior without redundancy.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, t.Helper on helpers, error wrapping with fmt.Errorf, handler signature matches the RunCommand pattern.
- SOLID principles: Good. RunTransition has single responsibility (orchestrate transition). Output dispatch is extracted to a helper. buildCascadeResult is separated from the main flow.
- Complexity: Low. The main function is a straightforward linear flow: validate args, open store, resolve ID, mutate, output. The buildCascadeResult function has moderate complexity with the upward cascade detection logic but is clear and well-commented.
- Modern idioms: Yes. Uses range over slice indices for in-place mutation, deferred Close, closure-based Mutate pattern.
- Readability: Good. Function and variable names are clear. Comments explain the upward cascade detection logic in buildCascadeResult. The outputTransitionOrCascade helper has a clear doc comment explaining when to use each format path.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The buildCascadeResult function (line 66-138) is somewhat long at ~70 lines. It handles upward detection, cascade entry building, and unchanged collection. Could potentially be split, but the current structure is readable and each section is well-commented, so this is a style preference not a concern.
