TASK: JSONL Syntax Check (doctor-validation-2-1)

ACCEPTANCE CRITERIA:
- [x] `JsonlSyntaxCheck` implements the `Check` interface
- [x] Passing check returns `CheckResult` with Name `"JSONL syntax"` and Passed `true`
- [x] Each malformed line produces its own failing `CheckResult` with line number in details
- [x] Blank and whitespace-only lines are silently skipped (not errors, not counted as valid)
- [x] Line numbers are 1-based and count blank lines (blank line on line 3 means next content is line 4)
- [x] Empty file (zero bytes) returns passing result
- [x] Missing `tasks.jsonl` returns error-severity failure with init suggestion
- [x] Only JSON syntax is validated -- field names, types, and values are not inspected
- [x] Long malformed lines are truncated in details output
- [x] Suggestion is `"Manual fix required"` for syntax errors
- [x] Check is read-only -- never modifies `tasks.jsonl`
- [x] All failures use `SeverityError`
- [x] Tests written and passing for all edge cases

STATUS: Complete

SPEC CONTEXT: Specification Error #2 defines "Malformed JSON lines that can't be parsed." The spec states "Schema validation (field types, required fields, valid enum values) happens at write time, not in doctor. Doctor catches corruption and edge cases that slipped through." Each error must be reported individually per the spec: "Doctor lists each error individually. If there are 5 orphaned references, all 5 are shown with their specific details." The fix suggestion table maps "All other errors" to "Manual fix required."

IMPLEMENTATION:
- Status: Implemented
- Location: `/Users/leeovery/Code/tick/internal/doctor/jsonl_syntax.go:1-57`
- Supporting code: `/Users/leeovery/Code/tick/internal/doctor/jsonl_reader.go` (shared `ScanJSONLines` / `getJSONLines`), `/Users/leeovery/Code/tick/internal/doctor/helpers.go:14-22` (`fileNotFoundResult` helper)
- Notes: The implementation correctly delegates to the shared `getJSONLines` function which reads `tasks.jsonl` line by line via `ScanJSONLines`, skips blank/whitespace-only lines while preserving 1-based line numbers, and parses into `map[string]interface{}`. The syntax check then uses `json.Valid` as an additional validation for lines where `Parsed` is nil -- this is important because `ScanJSONLines` only parses into `map[string]interface{}`, meaning valid JSON arrays (e.g., `[]`) or primitives would have nil `Parsed` but are syntactically valid. The `json.Valid` fallback correctly handles this case. Truncation at 80 chars with `"..."` suffix is implemented. The check is registered in `/Users/leeovery/Code/tick/internal/cli/doctor.go:20`. No drift from the plan.

TESTS:
- Status: Adequate
- Coverage: All 18 test cases specified in the plan task are present and accounted for:
  1. All lines valid JSON -- passing result
  2. Empty file (zero bytes) -- passing result
  3. File contains only blank lines -- passing result
  4. File contains only whitespace-only lines -- passing result
  5. Single malformed line with line number in details -- failing result
  6. All lines malformed -- one failing result per line
  7. Mixed valid and malformed lines -- only failing results returned
  8. Blank lines skipped without counting as valid or invalid
  9. Trailing newline producing empty last line -- handled correctly
  10. Missing tasks.jsonl -- failing result
  11. Suggestion is "Manual fix required" for syntax errors
  12. Suggestion is "Run tick init or verify .tick directory" for missing file
  13. Name is "JSONL syntax" for all result types (table-driven across passing/failing/missing)
  14. SeverityError for all failure cases (table-driven across syntax error and missing file)
  15. Correct 1-based line numbers with blank lines still counting
  16. Long malformed line content truncated in details
  17. Does not validate JSON field names or values -- `{}`, `[]`, and `{"bogus_field":999}` all pass
  18. Read-only verification via `assertReadOnly` helper
- Notes: Tests are well-structured with subtests. Table-driven tests are used where appropriate (Name check, Severity check). Each test verifies specific behavior rather than implementation details. The `assertReadOnly` helper is a shared test utility that compares file content before/after execution. No over-testing detected -- each test covers a distinct aspect of the acceptance criteria.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib `testing` only (no testify), `t.Run()` subtests, `t.TempDir()` for isolation, `t.Helper()` on helpers. Error wrapping with `fmt.Errorf` in the reader layer. Implements the `Check` interface correctly with `Run(ctx context.Context, tickDir string) []CheckResult` signature.
- SOLID principles: Good. `JsonlSyntaxCheck` has a single responsibility (syntax validation). It depends on the `Check` interface (DIP). The shared `getJSONLines` abstraction allows context-based caching without changing the check's code (OCP). The `fileNotFoundResult` helper eliminates duplication across checks (DRY).
- Complexity: Low. The `Run` method is a straightforward linear scan with clear branching: file-not-found returns early, then iterates lines, accumulates failures, and returns either failures or a single pass result. Cyclomatic complexity is approximately 4.
- Modern idioms: Yes. Uses `json.Valid` for syntax-only checking rather than parsing into a throwaway structure. Uses `context.Context` for pre-scanned data propagation. Uses value receiver on the struct (appropriate since no mutation needed).
- Readability: Good. The code is concise (57 lines including comments), well-commented with a doc comment on both the struct and the `Run` method, and the logic flow is immediately clear. Variable names are descriptive (`preview`, `failures`, `lines`).
- Issues: None identified.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The implementation wisely uses `json.Valid` as a secondary check for lines where `Parsed` is nil (since `ScanJSONLines` only unmarshals into `map[string]interface{}`). This correctly handles edge cases like valid JSON arrays or primitives that fail map unmarshaling. This is well-tested in the "does not validate JSON field names or values" test case which includes `[]`.
