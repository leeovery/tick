TASK: auto-cascade-parent-status-3-2 — Implement FormatCascadeTransition for Toon, Pretty, and JSON formatters

ACCEPTANCE CRITERIA:
- Pretty format renders cascade tree with box-drawing characters, unchanged terminal children shown
- Toon format renders flat lines with (auto) and (unchanged) markers
- JSON format renders structured object with transition, cascaded, and unchanged keys
- Edge cases: deeply nested tree in Pretty, mixed cascaded and unchanged children, single cascade entry

STATUS: Complete

SPEC CONTEXT: The specification (CLI Display section) defines three output formats for cascade transitions. Pretty uses box-drawing tree characters with a "Cascaded:" header, showing nested parent-child relationships. Toon uses flat lines with (auto) and (unchanged) markers. JSON outputs a structured object with transition, cascaded, and unchanged keys where cascaded/unchanged are always arrays (never null).

IMPLEMENTATION:
- Status: Implemented
- Location:
  - Toon: /Users/leeovery/Code/tick/internal/cli/toon_formatter.go:143-158
  - Pretty: /Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:196-276
  - JSON: /Users/leeovery/Code/tick/internal/cli/json_formatter.go:288-322
  - Types: /Users/leeovery/Code/tick/internal/cli/format.go:130-156 (CascadeEntry, UnchangedEntry, CascadeResult)
  - Interface: /Users/leeovery/Code/tick/internal/cli/format.go:175-176
- Notes:
  - Toon: Correctly renders flat lines, primary transition first, then cascaded with (auto), then unchanged with (unchanged). Matches spec exactly.
  - Pretty: Uses cascadeNode tree struct with ParentID-based tree construction. writeCascadeTree recursively renders with box-drawing characters (├─, └─, │). Deeply nested hierarchies supported via recursive rendering with proper prefix management. Matches spec examples.
  - JSON: Uses dedicated jsonCascadeTransition, jsonCascadeEntry, jsonUnchangedEntry structs with correct JSON tags. Pre-allocates slices with `make([], 0, len(...))` to ensure empty arrays render as `[]` not `null`. Matches spec JSON structure exactly.
  - All three formatters return empty string for empty TaskID (guard clause).
  - baseFormatter has a stub returning empty string (not used by any concrete formatter since all three override).

TESTS:
- Status: Adequate
- Coverage:
  - Toon: 3 tests — downward cancel cascade, upward start cascade, single cascade entry. All edge cases from task covered.
  - Pretty: 5 tests — downward cancel with tree, mixed cascaded/unchanged, 3-level hierarchy (deeply nested), upward grandparent chain, unchanged terminal grandchildren in tree. Deeply nested edge case explicitly tested.
  - JSON: 2 tests — full structured object verification (transition/cascaded/unchanged keys), empty cascaded array renders as [] not null.
  - Cross-formatter: 1 test verifying all formatters handle both-empty cascaded+unchanged gracefully.
  - format_test.go: compile-time interface checks, stub returns empty, empty CascadeResult returns empty for all formatters.
  - buildCascadeResult: 3 tests covering ParentID population, recursive unchanged collection, direct children unchanged ParentID.
- Notes: Good coverage of all edge cases specified in the task. Tests verify actual output strings (Pretty) and parsed JSON structure (JSON). No over-testing observed — each test covers a distinct scenario.

CODE QUALITY:
- Project conventions: Followed — stdlib testing only, t.Run subtests, "it does X" naming, no external test libs.
- SOLID principles: Good — each formatter has single responsibility, interface segregation via Formatter interface, tree rendering extracted to writeCascadeTree helper.
- Complexity: Low — Toon is straightforward iteration, Pretty uses clean tree-building pattern (map + ordered IDs + parent attachment), JSON maps to dedicated structs.
- Modern idioms: Yes — strings.Builder for Pretty, pre-allocated slices for JSON null-safety, clean recursive tree rendering.
- Readability: Good — clear separation between tree construction and rendering in Pretty, well-commented code, descriptive variable names.
- Issues: None significant.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The baseFormatter.FormatCascadeTransition stub (format.go:198) returns empty string but is never called since all three concrete formatters override it. This was flagged in analysis cycle 2 and accepted as consistent with the existing baseFormatter pattern. No action needed.
