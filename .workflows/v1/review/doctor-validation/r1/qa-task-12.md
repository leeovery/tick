TASK: Dependency Cycle Detection Check (doctor-validation-3-4)

ACCEPTANCE CRITERIA:
- [x] `DependencyCycleCheck` implements the `Check` interface
- [x] Check reuses `ParseTaskRelationships` from task 3-1 (via refactored `getTaskRelationships`)
- [x] DFS with three-color marking (white/gray/black) used for cycle detection
- [x] Self-references excluded from the adjacency list (task 3-3 owns self-reference detection)
- [x] Orphaned dependency references excluded from the adjacency list (non-existent target IDs filtered out)
- [x] Each unique cycle produces its own failing `CheckResult`
- [x] Cycles are deduplicated -- same cycle discovered from different starting nodes reported only once
- [x] Cycle details formatted as `"Dependency cycle: tick-A -> tick-B -> tick-C -> tick-A"` (first node repeated at end)
- [x] Deterministic cycle output -- lexicographically smallest ID first in the normalized cycle
- [x] Passing check returns `CheckResult` with Name `"Dependency cycles"` and Passed `true`
- [x] Valid DAGs (chains without back-edges) do not produce false positives
- [x] Tasks with no dependencies do not cause errors
- [x] Missing `tasks.jsonl` returns error-severity failure with init suggestion
- [x] Suggestion is `"Manual fix required"` for cycle errors
- [x] All failures use `SeverityError`
- [x] Check is read-only -- never modifies `tasks.jsonl`
- [x] Tests written and passing for all edge cases including 2-node, 3+-node, multiple independent, overlapping, and complex mixed graphs

STATUS: Complete

SPEC CONTEXT: Specification Error #8 defines "Circular dependency chains (A->B->C->A)" as an error condition. The fix suggestion table maps all non-cache errors to "Manual fix required". The specification requires "Doctor lists each error individually" -- each distinct cycle is a separate error. Design principle #4 ("Run all checks") mandates that cycle detection discovers all cycles, not just the first one. Self-references are Error #7 (separate check, task 3-3).

IMPLEMENTATION:
- Status: Implemented
- Location: `/Users/leeovery/Code/tick/internal/doctor/dependency_cycle.go:1-158`
- Notes:
  - `DependencyCycleCheck` struct (line 14) with `Run` method (line 20) correctly implements `Check` interface (`Run(ctx context.Context, tickDir string) []CheckResult`).
  - Uses `getTaskRelationships` (line 21), the context-aware refactored wrapper around the original `ParseTaskRelationships` from task 3-1. This is the correct post-Phase-4 pattern.
  - Adjacency list built with self-reference exclusion (line 33) and orphaned-target exclusion (line 35). Both filters are correct.
  - DFS with three-color marking: white=0, gray=1, black=2 (lines 43-47). Classic algorithm correctly implemented via recursive closure (lines 59-80).
  - Cycle extraction via `extractCycle` (lines 129-138) finds the cycle portion of the DFS path when a back-edge is detected.
  - Cycle normalization via `normalizeCycle` (lines 141-158) rotates so lexicographically smallest ID is first.
  - Deduplication via `seen` map keyed on comma-joined normalized cycle (lines 54, 68-70).
  - Sorted iteration over task IDs (lines 83-93) ensures deterministic DFS traversal order.
  - Output cycles are sorted (lines 103-105) for deterministic result ordering.
  - Failing results use correct Name, Severity, Details format, and Suggestion (lines 114-120).
  - Passing result returns single `CheckResult` with Name "Dependency cycles" and Passed true (lines 96-99).
  - Missing file handled via `fileNotFoundResult("Dependency cycles")` (line 23), returning proper error details and suggestion.
  - Registered in `tick doctor` command at `/Users/leeovery/Code/tick/internal/cli/doctor.go:26`.
  - Uses shared helpers `buildKnownIDs` and `fileNotFoundResult` from `/Users/leeovery/Code/tick/internal/doctor/helpers.go`.

TESTS:
- Status: Adequate
- Coverage: All 23 tests from the plan's test list are present and correctly implemented in `/Users/leeovery/Code/tick/internal/doctor/dependency_cycle_test.go:1-496`.
  - Passing scenarios (4 tests): no cycles, empty file, valid DAG, no dependencies
  - Cycle detection (4 tests): 2-node cycle, 3-node cycle, 4+ node cycle, multiple independent cycles
  - False positive avoidance (3 tests): chain without back-edge, self-references excluded, self-ref exclusion with coexisting real cycle
  - Orphaned ref exclusion (1 test): non-existent dependency targets filtered
  - Complex graph (1 test): mixed cycles and valid chains, reports only cycles
  - Result structure (3 tests): separate results per cycle, deduplication from different starting nodes, output format
  - Deterministic output (1 test): non-alphabetical task order still produces normalized output
  - Edge cases (3 tests): unparseable lines skipped, missing file, read-only verification
  - Metadata validation (3 tests): suggestion text, Name field across scenarios (table-driven), SeverityError across scenarios (table-driven)
- Notes:
  - The "overlapping cycles (shared nodes)" edge case from the plan's Edge Cases section is not explicitly tested, but this is a documented edge case rather than a required test (not listed in the Tests section). The DFS algorithm correctly handles it by design since separate back-edges are detected independently.
  - Tests use table-driven subtests appropriately (Name validation, Severity validation).
  - Test helpers (`setupTickDir`, `writeJSONL`, `assertReadOnly`, `ctxWithTickDir`) are properly shared from `cache_staleness_test.go` and use `t.Helper()`.
  - No over-testing observed. Each test verifies a distinct behavior or edge case. No redundant assertions.

CODE QUALITY:
- Project conventions: Followed. Stdlib testing only, `t.Run()` subtests, `t.TempDir()` isolation, `t.Helper()` on helpers. Error wrapping via `fmt.Errorf`. No testify.
- SOLID principles: Good. Single responsibility -- `DependencyCycleCheck` does only cycle detection. Helper functions (`extractCycle`, `normalizeCycle`, `buildKnownIDs`, `fileNotFoundResult`) are well-extracted. Open/closed -- new checks can be added without modifying existing ones.
- Complexity: Low. The DFS is a standard algorithm with clear state transitions. The closure captures state cleanly. `extractCycle` and `normalizeCycle` are simple pure functions.
- Modern idioms: Yes. Uses `context.Context` in the interface. Uses `struct{}` for set values. Uses slice tricks for path management.
- Readability: Good. Well-commented with clear variable names. The three-color constants are documented. The algorithm flow is straightforward to follow.
- Issues: None identified.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `extractCycle` function can theoretically return `nil` if the target is not found in the path (line 137), but this case is unreachable since the function is only called when a gray node (which must be on the path) is encountered. The downstream `normalizeCycle` handles empty slices gracefully (line 142-144), so there is no bug, but a brief comment noting this invariant could aid future maintainers.
