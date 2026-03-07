TASK: Defensive copy of task data for cascade display output (acps-4-6)

ACCEPTANCE CRITERIA:
- No reference to the Mutate callback's tasks slice is held after Mutate returns
- Display output is built from copied data
- All existing CLI output tests pass unchanged

STATUS: Complete

SPEC CONTEXT: The spec requires cascade display output showing primary transitions, cascaded changes, and unchanged terminal children across Toon, Pretty, and JSON formats. The concern is that `Store.Mutate` owns the tasks slice lifecycle -- if display code holds a reference to pointers within that slice after Mutate returns, future Store implementation changes could cause stale/corrupt reads.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/transition.go:34-49` -- buildCascadeResult called inside Mutate closure
  - `/Users/leeovery/Code/tick/internal/cli/create.go:244` -- buildCascadeResult called inside Mutate closure
  - `/Users/leeovery/Code/tick/internal/cli/update.go:321,383` -- both R6 and R3 cascade results built inside Mutate closure
  - `/Users/leeovery/Code/tick/internal/cli/transition.go:66-138` -- buildCascadeResult extracts string values into value-type CascadeResult
  - `/Users/leeovery/Code/tick/internal/cli/format.go:130-156` -- CascadeResult, CascadeEntry, UnchangedEntry are pure value types (strings only, no pointers)
  - `/Users/leeovery/Code/tick/internal/cli/helpers.go:101` -- comment documents the pattern: "Callers must build the CascadeResult inside the Mutate closure where the tasks slice is still valid."
- Notes: The defensive copy strategy is clean. All three CLI command files (transition.go, create.go, update.go) consistently call `buildCascadeResult` inside the Mutate closure. The resulting `CascadeResult` struct and its nested types contain only string fields -- no task pointers leak out of the closure. After Mutate returns, only value-type data (strings) is used for display output.

TESTS:
- Status: Adequate
- Coverage:
  - `helpers_test.go:329-395` tests outputTransitionOrCascade with cascade results, verifying correct output from built CascadeResult
  - `cascade_formatter_test.go:337-400` tests buildCascadeResult with various cascade scenarios (downward, unchanged children)
  - `transition_test.go:581-625` tests buildCascadeResult for upward cascade ParentID flattening
  - Integration tests in transition_test.go, create_test.go, update_test.go exercise the full flow through Mutate
- Notes: No dedicated test verifies that post-Mutate access to the tasks slice would fail (which would be testing implementation details). The existing tests adequately verify that the output is correct, which is the observable behavior. This is appropriate -- the task is a structural safety improvement, not a behavioral change.

CODE QUALITY:
- Project conventions: Followed -- uses the established pattern of capturing results via closure variables
- SOLID principles: Good -- buildCascadeResult has a single responsibility (extract display data from mutable state), and the CascadeResult type cleanly separates display concerns from domain mutation
- Complexity: Low -- the defensive copy is implicit in the value-type extraction, no complex copying logic needed
- Modern idioms: Yes -- Go value semantics handle the defensive copy naturally through struct assignment
- Readability: Good -- the comment on helpers.go:101 documents the pattern requirement for future maintainers
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The approach is elegant: rather than explicit deep-copy of task structs, the code extracts only the needed display fields into value-type structs inside the closure. This is both more efficient and more maintainable than a traditional defensive copy.
