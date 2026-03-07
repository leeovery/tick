TASK: Add CascadeResult type and FormatCascadeTransition to Formatter interface

ACCEPTANCE CRITERIA:
- FormatCascadeTransition method added to Formatter interface
- Non-cascade single-task transitions still use existing FormatTransition with no visual regression
- Unchanged terminal children appear in all cascade output formats
- Edge cases: empty cascaded list, empty unchanged list, both empty

STATUS: Complete

SPEC CONTEXT: The specification defines a FormatCascadeTransition method on the Formatter interface that receives the primary transition result plus cascade changes and unchanged terminal children. Three output formats: Toon (flat lines with "(auto)" and "(unchanged)" markers), Pretty (box-drawing tree characters), JSON (structured object with "transition", "cascaded", "unchanged" keys). Existing FormatTransition remains for non-cascade single-task transitions.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/cli/format.go:130-177 -- CascadeEntry, UnchangedEntry, CascadeResult types and FormatCascadeTransition in Formatter interface
  - /Users/leeovery/Code/tick/internal/cli/format.go:198 -- baseFormatter stub returning ""
  - /Users/leeovery/Code/tick/internal/cli/format.go:243 -- StubFormatter stub returning ""
  - /Users/leeovery/Code/tick/internal/cli/toon_formatter.go:145-158 -- ToonFormatter implementation
  - /Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:199-256 -- PrettyFormatter implementation with tree rendering
  - /Users/leeovery/Code/tick/internal/cli/json_formatter.go:290-322 -- JSONFormatter implementation
  - /Users/leeovery/Code/tick/internal/cli/helpers.go:102-108 -- outputTransitionOrCascade routing logic
  - /Users/leeovery/Code/tick/internal/cli/transition.go:66-108 -- buildCascadeResult constructor
- Notes: All three formatters implement FormatCascadeTransition. The CascadeResult type includes TaskID, TaskTitle, OldStatus, NewStatus, Cascaded ([]CascadeEntry), and Unchanged ([]UnchangedEntry). CascadeEntry and UnchangedEntry both include ParentID for tree construction. The outputTransitionOrCascade helper correctly falls back to FormatTransition when cr is nil or has no cascaded entries, preserving non-cascade behavior. Note that baseFormatter provides a stub FormatCascadeTransition returning "" -- Toon and Pretty formatters override this with their own implementations, which is correct.

TESTS:
- Status: Adequate
- Coverage:
  - /Users/leeovery/Code/tick/internal/cli/cascade_formatter_test.go -- Tests all three formatters (Toon, Pretty, JSON) with downward cascades, upward cascades, single entry, mixed cascaded+unchanged, 3-level hierarchy, grandchild tree rendering, empty arrays for all three formatters, and buildCascadeResult with ParentID population and unchanged terminal descendant collection
  - /Users/leeovery/Code/tick/internal/cli/format_test.go:370-427 -- Interface satisfaction, stub returns empty, empty CascadeResult handled by all formatters
  - /Users/leeovery/Code/tick/internal/cli/format_test.go:429-451 -- CascadeResultStruct field verification
  - /Users/leeovery/Code/tick/internal/cli/helpers_test.go:296-380 -- outputTransitionOrCascade tested for nil cr, empty cascades (falls back to FormatTransition), non-empty cascades (uses FormatCascadeTransition), and identical output to inline FormatTransition pattern
- Notes: Edge cases from the task are well covered: empty cascaded list (JSON test at line 300), empty unchanged list (Toon upward test at line 37), both empty (TestAllFormattersCascadeEmptyArrays at line 405). Unchanged terminal children tested in all formats. Non-cascade regression tested via outputTransitionOrCascade helper tests. Test balance is appropriate -- no redundant or over-tested areas.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, "it does X" naming, error wrapping, interface-based design.
- SOLID principles: Good. CascadeResult is a clean data transfer type. Formatter interface extended minimally with one new method. baseFormatter provides default stub following open/closed principle. buildCascadeResult separates construction from rendering.
- Complexity: Low. Toon formatter is straightforward iteration. Pretty formatter uses a clear tree-building algorithm with cascadeNode type and recursive writeCascadeTree. JSON formatter maps to typed structs then marshals.
- Modern idioms: Yes. strings.Builder for Pretty, make with capacity for JSON slices, compile-time interface checks.
- Readability: Good. Types are well-documented with comments. Helper function outputTransitionOrCascade has clear routing logic. The ParentID field on entries enables tree construction without requiring the formatter to know about task hierarchy.
- Issues: None significant.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- The baseFormatter.FormatCascadeTransition stub returns "" but is overridden by both ToonFormatter and PrettyFormatter with their own implementations. This is fine, but the comment on line 197 saying "stub for text-based formatters" is slightly misleading since both text formatters override it. Minor documentation inaccuracy.
