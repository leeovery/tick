TASK: Add explanatory second line to child-blocked-by-parent error

ACCEPTANCE CRITERIA:
- Error message matches spec lines 407-408 exactly (two lines, correct capitalization)
- Existing dependency validation tests pass with updated assertions
- The explanatory rationale line is present in the error output

STATUS: Issues Found

SPEC CONTEXT: Spec lines 405-411 define error message format for child-blocked-by-parent: "Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-epic" followed by "(would create unworkable task due to leaf-only ready rule)". The `Error:` prefix is added by the CLI layer (`internal/cli/app.go:94`), so the error string itself should start with "Cannot" (capital C).

IMPLEMENTATION:
- Status: Partial
- Location: /Users/leeovery/Code/tick/internal/task/dependency.go:39-42
- Notes: The two-line error message with the explanatory rationale about the leaf-only ready rule IS present. However, the task explicitly requires fixing capitalization from lowercase "cannot" to uppercase "Cannot" to match the spec. The implementation still uses lowercase "cannot add dependency" at line 40. The task description specifically identifies this as a problem: "uses lowercase 'cannot' vs the spec's 'Cannot'" and instructs "fix capitalization to match spec." This was not done.

  Additionally, the spec shows the second line indented with spaces to align under the message text, but the implementation uses a bare `\n` without indentation. The task does not explicitly call this out, so it is non-blocking.

TESTS:
- Status: Adequate (modulo the capitalization issue)
- Coverage: Test at dependency_test.go:76-91 verifies both lines are present and checks exact string match. Test at dependency_test.go:188-201 verifies mixed-case ID handling. The tests assert lowercase "cannot" consistent with the current (incorrect per task) implementation.
- Notes: Tests do verify the two-line format, the rationale text, and exact message match. They would fail if the feature broke. The test assertion at line 87 uses lowercase "cannot" matching the implementation but not the spec. If the capitalization fix were applied, these tests would need updating (as the task instructs in step 4). Tests are focused and not over-tested.

CODE QUALITY:
- Project conventions: Go convention says error strings should not be capitalized. The lowercase implementation follows Go idiom. However, the spec and task explicitly require capital C. This is a spec-vs-convention tension that was explicitly resolved by the task in favor of the spec.
- SOLID principles: Good. validateChildBlockedByParent is a focused single-responsibility function.
- Complexity: Low. Simple loop with clear logic.
- Modern idioms: Yes. Standard fmt.Errorf pattern.
- Readability: Good. Clear function name, good doc comment.
- Issues: None beyond the capitalization discrepancy.

BLOCKING ISSUES:
- Capitalization not fixed: The error message at dependency.go:40 starts with lowercase "cannot" but the task explicitly requires "Cannot" (capital C) to match spec lines 407-408. The acceptance criterion "correct capitalization" is not met. The test at dependency_test.go:87 also asserts lowercase, so it would need updating. This is the primary purpose of the task and was not completed.

NON-BLOCKING NOTES:
- The spec shows the second line indented with spaces for visual alignment under the first line text (after "Error: " prefix). Current implementation uses bare "\n" without indentation. Consider adding padding to match spec format exactly, though the task does not explicitly require this.
- Go convention prefers lowercase error strings. If the project decides to follow Go convention over spec format, this should be documented as a deliberate deviation from the spec.
