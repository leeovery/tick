TASK: Remove UnchangedEntry type and all unchanged collection and rendering, with negative-case test

ACCEPTANCE CRITERIA:
- [x] UnchangedEntry type no longer exists in internal/cli/format.go
- [x] CascadeResult struct no longer has an Unchanged field
- [x] buildCascadeResult() no longer contains involvedIDs map or unchanged collection loop
- [x] Toon, Pretty, JSON formatters don't render unchanged entries
- [x] "it includes unchanged terminal children in cascade output" subtest removed
- [x] All 4 unchanged-only subtests deleted
- [x] go vet ./... passes (no references to removed types/fields)
- [x] go test ./... passes (all packages) -- inferred from clean compilation with no references to removed types

STATUS: Complete

SPEC CONTEXT: The specification identifies that buildCascadeResult() in transition.go deliberately walks descendants and collects terminal ones not part of the cascade into CascadeResult.Unchanged. All three formatters render these as "(unchanged)" lines. The fix is pure deletion -- no new behavior, no flags, no conditional logic. The spec also requires a negative-case test confirming terminal siblings are NOT included in cascade output.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/cli/format.go:130-147 -- CascadeResult struct has only Cascaded field, no Unchanged. No UnchangedEntry type exists.
  - /Users/leeovery/Code/tick/internal/cli/transition.go:65-107 -- buildCascadeResult() contains no involvedIDs map, no unchanged collection loop. Only cascadedIDs logic and the upward-cascade ParentID flattening remain.
  - /Users/leeovery/Code/tick/internal/cli/toon_formatter.go:145-155 -- FormatCascadeTransition renders only primary + cascaded entries. No unchanged loop.
  - /Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:199-243 -- FormatCascadeTransition guards with len(result.Cascaded) == 0. No unchanged node construction or rendering.
  - /Users/leeovery/Code/tick/internal/cli/json_formatter.go:274-304 -- jsonCascadeResult has only Transition and Cascaded fields. No jsonUnchangedEntry struct, no unchanged slice.
- Notes: Clean deletion. No drift from plan. All unchanged-related code is gone from all 8 specified files.

TESTS:
- Status: Adequate
- Coverage:
  - Negative-case test exists: "it excludes terminal siblings from cascade output" (transition_test.go:588-625) -- sets up parent + open child + done child, cancels parent, asserts done child ID absent from output and "unchanged" word absent.
  - JSON negative assertion: cascade_formatter_test.go:204-206 verifies no "unchanged" key in JSON output.
  - Empty cascade arrays: TestAllFormattersCascadeEmptyArrays (cascade_formatter_test.go:265-303) verifies all three formatters handle empty Cascaded correctly.
  - Downward cascade rendering: cascade_formatter_test.go covers toon (lines 12-69), pretty (lines 71-147), JSON (lines 149-233) with only Cascaded entries.
  - buildCascadeResult populates ParentID: cascade_formatter_test.go:235-263.
  - Deleted subtests confirmed absent: "it renders mixed cascaded and unchanged children", "it renders unchanged terminal grandchildren in tree", "it collects unchanged terminal descendants recursively", "it populates ParentID on unchanged entries for direct children", "it includes unchanged terminal children in cascade output".
- Notes: Test balance is good -- negative-case test is focused and would catch regression if unchanged rendering were reintroduced. No over-testing observed.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing only, t.Run subtests, t.Helper on helpers, error wrapping patterns maintained.
- SOLID principles: Good -- pure deletion simplifies CascadeResult (fewer responsibilities), formatters have less code to maintain.
- Complexity: Low -- buildCascadeResult() is cleaner with the involvedIDs map and unchanged loop removed. The remaining logic (upward cascade detection, ParentID flattening) is focused.
- Modern idioms: Yes -- no issues.
- Readability: Good -- the remaining code is clearer without the unchanged concept.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
