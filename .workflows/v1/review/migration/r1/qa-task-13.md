TASK: Consolidate inconsistent empty-title fallback strings (migration-3-3)

ACCEPTANCE CRITERIA:
- A single `FallbackTitle` constant exists in `migrate.go`.
- All three usage sites reference the constant.
- No hardcoded fallback title strings remain in the migrate package.
- Tests pass with the consolidated value.

STATUS: Complete

SPEC CONTEXT: The migration output format shows failed tasks with a title display (e.g., "Task: Broken entry (skipped: missing title)"). When a task has no title, a fallback string is needed. The spec does not prescribe a specific fallback; this task consolidates three independently written fallback strings into one constant.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/migrate/migrate.go:20-21` -- `FallbackTitle` constant defined as `"(untitled)"`
  - `/Users/leeovery/Code/tick/internal/migrate/engine.go:74` -- uses `FallbackTitle` in validation failure path
  - `/Users/leeovery/Code/tick/internal/migrate/presenter.go:28` -- uses `FallbackTitle` in `WriteResult` for failed results with empty title
  - `/Users/leeovery/Code/tick/internal/migrate/presenter.go:65` -- uses `FallbackTitle` in `WriteFailures` for failures with empty title
- Notes: All three production usage sites correctly reference the exported constant. No hardcoded `"(unknown)"` strings remain anywhere in the package. The constant value `"(untitled)"` was chosen as the canonical value per the task rationale, since it more accurately describes the situation.

TESTS:
- Status: Adequate
- Coverage:
  - `engine_test.go:594-595` -- tests that empty-title tasks get `"(untitled)"` as the result title
  - `presenter_test.go:104-114` -- tests `WriteResult` with empty title, expects `"(untitled)"` in output
  - `presenter_test.go:261-273` -- tests `WriteFailures` with empty title, expects `"(untitled)"` in output
  - `presenter_test.go:482-494` -- tests `Present` integration with an untitled failed task
- Notes: Test files use literal `"(untitled)"` strings in expected output rather than referencing the `FallbackTitle` constant. This is acceptable test practice -- hardcoded expected values in tests serve as independent specifications and prevent tests from silently passing if someone changes the constant to an incorrect value. The tests previously used `"(unknown)"` and have been updated to the new value.

CODE QUALITY:
- Project conventions: Followed. Exported constant with doc comment. Error wrapping with `fmt.Errorf`. Standard Go testing patterns.
- SOLID principles: Good. Single definition point (DRY), referenced from all consumers.
- Complexity: Low. Straightforward constant extraction.
- Modern idioms: Yes. Exported package-level constant is idiomatic Go for shared sentinel values.
- Readability: Good. The constant name `FallbackTitle` is self-explanatory, and the doc comment clarifies its purpose.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Test files contain hardcoded `"(untitled)"` strings (5 occurrences across `engine_test.go` and `presenter_test.go`). The acceptance criteria states "No hardcoded fallback title strings remain in the migrate package" which could be read to include test files. However, using literal expected values in tests is good practice for independent verification, so this is not a concern. The important thing is that all production code paths use the constant.
