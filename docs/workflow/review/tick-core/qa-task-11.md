TASK: Dependency validation -- cycle detection & child-blocked-by-parent (tick-core-3-1)

ACCEPTANCE CRITERIA:
- Self-reference rejected
- 2-node cycle detected with path
- 3+ node cycle detected with full path
- Child-blocked-by-parent rejected with descriptive error
- Parent-blocked-by-child allowed
- Sibling/cross-hierarchy deps allowed
- Error messages match spec format
- Batch validation fails on first error
- Pure functions -- no I/O

STATUS: Complete

SPEC CONTEXT: The spec (hierarchy-dependency-model section) defines that `blocked_by` must not create cycles and that child-blocked-by-parent creates a deadlock with the leaf-only ready rule. Validation must happen at write time before persisting. Error messages must include cycle path using arrow notation and an explanatory second line for child-blocked-by-parent.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/task/dependency.go (all 127 lines)
- Key functions:
  - `ValidateDependency()` (line 12) -- main entry point, checks child-blocked-by-parent then cycle
  - `ValidateDependencies()` (line 25) -- batch validation, sequential fail-on-first
  - `validateChildBlockedByParent()` (line 36) -- checks if task's parent equals newBlockedByID
  - `detectCycle()` (line 51) -- self-reference check + DFS graph traversal
  - `dfs()` (line 107) -- recursive DFS with path tracking for error reporting
- Notes: Clean separation of concerns. Self-reference handled as special case before DFS. Cycle path reconstructed using original IDs (not normalized) for readable error messages. All ID comparisons use NormalizeID for case-insensitivity. Integration confirmed in `/Users/leeovery/Code/tick/internal/cli/dep.go:104`, `/Users/leeovery/Code/tick/internal/cli/create.go:155,160`, `/Users/leeovery/Code/tick/internal/cli/update.go:189`.

TESTS:
- Status: Adequate
- Coverage: All 11 tests from the task spec are present plus 3 additional mixed-case ID tests
  - "it allows valid dependency between unrelated tasks" (line 9)
  - "it rejects direct self-reference" (line 21)
  - "it rejects 2-node cycle with path" (line 37)
  - "it rejects 3+ node cycle with full path" (line 56)
  - "it rejects child blocked by own parent" (line 76)
  - "it allows parent blocked by own child" (line 93)
  - "it allows sibling dependencies" (line 105)
  - "it allows cross-hierarchy dependencies" (line 118)
  - "it returns cycle path format: tick-a -> tick-b -> tick-a" (line 132)
  - "it validates multiple blocked_by IDs, fails on first error" (line 217)
  - "it detects cycle through existing multi-hop chain" (line 149)
  - Mixed-case cycle detection (line 172)
  - Mixed-case child-blocked-by-parent (line 188)
  - Mixed-case valid dependency (line 203)
- Location: /Users/leeovery/Code/tick/internal/task/dependency_test.go (237 lines)
- Notes: Tests verify exact error strings including Unicode arrow character. Each test is focused and non-redundant. No unnecessary mocking -- pure function tests with in-memory Task slices. Tests would fail if the feature broke. No over-testing detected.

CODE QUALITY:
- Project conventions: Followed -- table-driven subtests not used for the main test function, but each subtest is a standalone t.Run with descriptive names which is acceptable given each case has unique setup. Mixed-case tests properly grouped under a separate TestValidateDependencyMixedCase function.
- SOLID principles: Good -- single responsibility (validation only, no I/O), functions are small and focused, dependency on []Task slice (not interface) is appropriate for pure domain logic.
- Complexity: Low -- detectCycle is the most complex function with DFS traversal, but path is clear and well-commented. Cyclomatic complexity is reasonable.
- Modern idioms: Yes -- uses range over slices, string building with strings.Join, Unicode escape in format strings, proper error wrapping.
- Readability: Good -- function names are self-documenting, comments explain the algorithm, parameter names are clear (taskID, newBlockedByID).
- Issues: None significant.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Error message capitalization: The implementation uses lowercase "cannot add dependency" while the spec uses "Cannot add dependency". This is addressed by a separate task (tick-core-6-7, Phase 6) and is the correct Go convention since the CLI dispatcher prepends "Error: " (see /Users/leeovery/Code/tick/internal/cli/app.go:94). The combined output reads "Error: cannot add dependency..." which is acceptable Go style.
- The DFS builds adjacency maps on every call to ValidateDependency. For ValidateDependencies (batch), this means rebuilding maps per-blockedByID. For the expected scale (hundreds of tasks), this is negligible, but could be optimized if batch validation becomes a hot path.
- The child-blocked-by-parent check only validates direct parent, not grandparent. This matches the task spec and edge case documentation exactly.
