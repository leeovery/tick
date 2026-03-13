TASK: cli-enhancements-1-4 -- Integrate ResolveID into dep add/rm

ACCEPTANCE CRITERIA:
- dep add and dep rm resolve both positional task ID arguments through store.ResolveID
- Edge case: both arguments resolving to same task produces an error (self-reference/cycle)

STATUS: Complete

SPEC CONTEXT: The specification requires partial ID matching to be centralized via ResolveID in the storage layer, applied everywhere an ID is accepted including dep add/rm. Both `tick-a3f` and `a3f` input forms must be accepted, with minimum 3 hex chars, ambiguous prefix errors, and not-found errors.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/cli/dep.go:71-78 (RunDepAdd calls store.ResolveID on both taskID and blockedByID)
  - /Users/leeovery/Code/tick/internal/cli/dep.go:148-155 (RunDepRm calls store.ResolveID on both taskID and blockedByID)
  - /Users/leeovery/Code/tick/internal/cli/dep.go:38-55 (parseDepArgs normalizes IDs via task.NormalizeID before resolution)
- Notes: Both RunDepAdd and RunDepRm follow the same pattern: parse args, open store, resolve both IDs via ResolveID, then proceed with the mutation. The ResolveID call happens before any Mutate/Query logic, which is correct -- errors from resolution (not found, ambiguous) short-circuit before the mutation callback. The self-reference case (both args resolving to the same task) is caught by task.ValidateDependency inside the Mutate callback, which checks for cycles including self-references.

TESTS:
- Status: Adequate
- Coverage:
  - TestDepAddPartialID (dep_test.go:601-770):
    - Both args as partial IDs (line 604)
    - Mixed full and partial IDs (line 633)
    - Both partial IDs resolving to same task -- the explicit edge case (line 662)
    - Ambiguous prefix in first arg (line 678)
    - Ambiguous prefix in second arg (line 702)
    - Not-found prefix in first arg (line 726)
    - Full IDs still work (line 742)
  - TestDepRmPartialID (dep_test.go:772-855):
    - Both args as partial IDs (line 775)
    - Not-found prefix in dep rm (line 805)
    - Full IDs still work in dep rm (line 826)
  - Pre-existing tests in TestDepAdd/TestDepRm cover full-ID scenarios, error cases (task not found, duplicate dependency, self-reference, cycle, child-blocked-by-parent), normalization, quiet mode, timestamp updates, persistence
- Notes: The "both arguments resolving to same task" edge case is only tested for dep add, not dep rm. For dep rm, if both IDs resolve to the same task, the mutation would look for that task ID in its own BlockedBy list and return "not a dependency" error (unless the task actually has itself in BlockedBy, which ValidateDependency prevents). This is acceptable behavior and is implicitly covered by the existing "dependency not found in blocked_by" test. No ambiguous-prefix test for dep rm's second arg, but this is adequately covered by the ResolveID unit tests in resolve_id_test.go and the dep add tests that prove the pattern works for both positions.

CODE QUALITY:
- Project conventions: Followed. Uses the established handler signature pattern, stdlib testing with t.Run subtests, t.Helper on test helpers, error wrapping with %w, DI via App struct.
- SOLID principles: Good. parseDepArgs extracts arg parsing into a reusable function. ResolveID is centralized in the storage layer. RunDepAdd and RunDepRm each have single responsibility.
- Complexity: Low. Linear flow: parse args -> open store -> resolve IDs -> mutate -> output. No nested conditionals or complex branching beyond the mutation logic.
- Modern idioms: Yes. Uses range loops, deferred Close, error propagation patterns consistent with idiomatic Go.
- Readability: Good. Clear function names, well-structured code, comments explain intent.
- Issues: Minor redundancy -- parseDepArgs calls task.NormalizeID which lowercases, then ResolveID also lowercases internally. This is harmless (idempotent) and consistent with how other commands work.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The double-normalization in parseDepArgs (NormalizeID) followed by ResolveID (which also lowercases) is harmless but slightly redundant. This is consistent across all commands, so not worth changing in isolation.
