TASK: Pretty format cascade tree rendering (auto-cascade-parent-status-4-1)

ACCEPTANCE CRITERIA:
- Pretty format downward cascade output shows grandchildren indented under their parent child entries with nested box-drawing characters
- Pretty format upward cascade output renders correctly for multi-level ancestor chains
- Unchanged terminal descendants at all hierarchy levels appear in the output
- Toon format remains flat with (auto) and (unchanged) markers
- JSON format remains a flat array structure

STATUS: Complete

SPEC CONTEXT: The spec (lines 123-132) shows Pretty format rendering cascades as a nested tree with box-drawing characters where grandchildren are indented under their parent children. Unchanged terminal children appear at the bottom with "(unchanged)". Toon format is flat lines with (auto)/(unchanged) markers. JSON is a structured object with flat arrays.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/format.go:131-156` -- CascadeEntry and UnchangedEntry types include ParentID field
  - `internal/cli/pretty_formatter.go:189-276` -- cascadeNode struct, FormatCascadeTransition builds tree from ParentID, writeCascadeTree renders recursively with box-drawing characters
  - `internal/cli/transition.go:66-138` -- buildCascadeResult populates ParentID from task.Parent, collects unchanged terminal descendants recursively via involvedIDs set
- Notes: Implementation correctly addresses all three issues from the analysis finding: (1) ParentID added to CascadeEntry/UnchangedEntry, (2) PrettyFormatter builds a tree using cascadeNode and renders with nested box-drawing, (3) buildCascadeResult collects unchanged descendants at all levels by walking all tasks whose parent is in the involvedIDs set. Upward cascades are correctly handled by setting all ParentID values to the primary task ID (line 99), keeping them flat as roots.

TESTS:
- Status: Adequate
- Coverage:
  - `cascade_formatter_test.go:78-103` -- Pretty downward cancel cascade with flat children (spec example 1)
  - `cascade_formatter_test.go:137-165` -- Pretty 3-level hierarchy test matching spec example exactly (parent -> children -> grandchildren with nested box-drawing)
  - `cascade_formatter_test.go:167-193` -- Pretty upward cascade with grandparent chain (flat rendering)
  - `cascade_formatter_test.go:195-219` -- Unchanged terminal grandchildren rendered in nested tree
  - `cascade_formatter_test.go:105-135` -- Mixed cascaded and unchanged children
  - `cascade_formatter_test.go:12-76` -- Toon formatter remains flat with (auto)/(unchanged) markers
  - `cascade_formatter_test.go:222-326` -- JSON formatter remains structured flat arrays
  - `cascade_formatter_test.go:405-452` -- All formatters handle empty cascaded+unchanged
  - `cascade_formatter_test.go:328-403` -- buildCascadeResult tests: ParentID population, recursive unchanged collection
- Notes: Tests directly verify spec examples. All required test scenarios from the analysis task are covered. Tests would fail if tree rendering broke. Not over-tested -- each test covers a distinct scenario.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing, t.Run subtests, "it does X" naming, no testify
- SOLID principles: Good -- cascadeNode is a focused internal type, writeCascadeTree has single responsibility, tree building is cleanly separated from rendering
- Complexity: Low -- tree construction is straightforward map-based parent lookup, recursive rendering is clean
- Modern idioms: Yes -- string builder, map-based indexing, ordered insertion
- Readability: Good -- cascadeNode/writeCascadeTree are self-documenting, buildCascadeResult has clear comments explaining upward vs downward cascade handling
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The baseFormatter still has a stub FormatCascadeTransition (format.go:198) that returns empty string. Since PrettyFormatter overrides it, this is harmless, but it means baseFormatter's stub is dead code for Pretty. Toon also overrides it. This is fine as defensive fallback.
