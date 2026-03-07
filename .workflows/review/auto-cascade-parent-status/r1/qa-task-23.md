TASK: Extract parent validation and reopen helper

ACCEPTANCE CRITERIA: Parent validation (ValidateAddChild) and done-parent reopen logic should be extracted into a helper function to reduce duplication between RunCreate and RunUpdate.

STATUS: Complete

SPEC CONTEXT: Rule 6 (new child added to done parent triggers reopen) and Rule 7 (block adding child to cancelled parent) require ValidateAddChild + conditional reopen logic at every call site that sets a parent. The spec notes ValidateAddChild is pure validation; the caller handles the reopen. This pattern repeats in both RunCreate and RunUpdate.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/helpers.go:110-133
- Notes: `validateAndReopenParent` consolidates ValidateAddChild (Rule 7) and done-parent reopen via ApplyWithCascades (Rule 6) into a single helper. Both RunCreate (create.go:228) and RunUpdate (update.go:304) call the helper. No direct ValidateAddChild calls remain in create.go or update.go. The helper returns a 4-tuple (TransitionResult, []CascadeChange, bool, error) which cleanly communicates whether a reopen occurred and its results.

TESTS:
- Status: Adequate
- Coverage: TestValidateAndReopenParent (helpers_test.go:410-511) covers 6 subtests: open parent (no-op), in_progress parent (no-op), cancelled parent (Rule 7 error), done parent (Rule 6 reopen), case-insensitive ID matching, parent not found (no-op). All key behaviors tested.
- Notes: Good coverage of the distinct code paths. Tests verify both the boolean return and the transition result values. Would fail if the feature broke.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing, t.Run subtests, error wrapping, consistent with other helpers in the same file.
- SOLID principles: Good -- single responsibility (validate + reopen for parent), clean separation from callers.
- Complexity: Low -- linear scan with early returns, straightforward control flow.
- Modern idioms: Yes -- idiomatic Go, range loop with index for in-place mutation.
- Readability: Good -- clear doc comment explaining Rules 6 and 7, descriptive variable names.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The function silently returns no-op when the parent ID is not found in the tasks slice (line 132). This is safe because parent ID resolution happens upstream, but a comment explaining this assumption could help future readers.
