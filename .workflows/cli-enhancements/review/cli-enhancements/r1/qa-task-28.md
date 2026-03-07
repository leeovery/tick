TASK: cli-enhancements-5-4 -- Extract shared validation helpers for type/tags/refs flags in CLI layer

ACCEPTANCE CRITERIA: Extract shared validation helpers for type/tags/refs flag parsing to reduce duplication across create and update commands. No edge cases specified.

STATUS: Complete

SPEC CONTEXT: The spec defines type validation (closed set: bug/feature/task/chore, case-insensitive, trimmed, stored lowercase), tags validation (kebab-case, dedup, max 30 chars/tag, max 10 tags), and refs validation (non-empty, no commas/whitespace, max 200 chars, max 10 refs, silent dedup). Both create and update commands perform these validations with slightly different empty-value error messages. The list command also validates type/tags for filtering but with different semantics (no empty check needed).

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/helpers.go:57-96
- Notes: Three shared helpers extracted:
  - `validateTypeFlag(value string) (string, error)` at line 59: normalizes via `task.NormalizeType`, checks non-empty via `task.ValidateTypeNotEmpty`, validates against allowed set via `task.ValidateType`
  - `validateTagsFlag(tags []string, emptyErr string) ([]string, error)` at line 73: deduplicates via `task.DeduplicateTags`, checks non-empty with caller-specified error message, validates via `task.ValidateTags`
  - `validateRefsFlag(refs []string, emptyErr string) ([]string, error)` at line 87: deduplicates via `task.DeduplicateRefs`, checks non-empty with caller-specified error message, validates via `task.ValidateRefs`
- Callers:
  - /Users/leeovery/Code/tick/internal/cli/create.go:131 (validateTypeFlag), :139 (validateTagsFlag), :147 (validateRefsFlag)
  - /Users/leeovery/Code/tick/internal/cli/update.go:164 (validateTypeFlag), :176 (validateTagsFlag), :188 (validateRefsFlag)
- Neither create.go nor update.go call the domain-level validation functions directly for type/tags/refs -- all go through the shared helpers. No duplication remains.
- The list command at /Users/leeovery/Code/tick/internal/cli/list.go:72,127 correctly does NOT use these helpers since list filtering has different semantics (no empty check, no dedup needed for type filter input).

TESTS:
- Status: Adequate
- Coverage: No direct unit tests for the three validation helpers in helpers_test.go. However, they are thoroughly exercised through integration tests:
  - create_test.go: tests invalid type values (line 860), empty type (line 877), tags/refs with create
  - update_test.go: tests invalid type (line 707), empty type (line 696), --type/--clear-type mutual exclusivity (line 669), --tags/--clear-tags mutual exclusivity (line 771), empty --tags (line 798), --refs/--clear-refs mutual exclusivity (line 917), empty --refs (line 944)
  - list_filter_test.go: tests invalid type filter (line 448)
- Notes: The integration tests provide high confidence since they exercise the full validation pipeline. Direct unit tests for these thin wrapper functions would add minimal value -- the domain-level validation functions (`ValidateType`, `ValidateTags`, `ValidateRefs`) have their own unit tests in internal/task/. The CLI helpers are essentially composition glue (normalize + empty-check + validate), and testing them through integration is appropriate for their simplicity.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, error wrapping patterns, proper Go naming conventions.
- SOLID principles: Good. Single responsibility -- each helper handles one flag type's validation pipeline. The helpers compose domain-level functions rather than duplicating logic.
- Complexity: Low. Each helper is 6-10 lines, linear flow, no branching beyond early returns on error.
- Modern idioms: Yes. Clean function signatures, appropriate use of multi-return (value, error), caller-parameterized error messages for the empty-value case.
- Readability: Good. Clear doc comments explain what each function does. The `emptyErr` parameter in tags/refs helpers makes the different error messages for create vs update contexts explicit and traceable.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `validateTypeFlag` does not take an `emptyErr` parameter like the tags/refs helpers do. This is acceptable because the type empty error message is the same in both create and update contexts (both delegate to `task.ValidateTypeNotEmpty` which has a fixed message). The asymmetry is justified by the different semantics.
