TASK: tick-core-4-2 -- TOON formatter -- list, show, stats output

ACCEPTANCE CRITERIA:
- [ ] Implements full Formatter interface
- [ ] List output matches spec TOON format exactly
- [ ] Show output multi-section with dynamic schema
- [ ] blocked_by/children always present, description conditional
- [ ] Stats produces summary + 5-row by_priority
- [ ] Escaping handled by toon-go
- [ ] All output matches spec examples

STATUS: Complete

SPEC CONTEXT: The specification defines TOON (Token-Oriented Object Notation) as the agent-facing output format, with 30-60% token savings over JSON. Key rules: schema headers with counts for arrays, single-object scope (no count) for task/stats, dynamic schema (parent/closed omitted when null), blocked_by/children always present even at [0], description omitted when empty, multiline description as indented lines, empty results as zero-count with schema header. Non-structured outputs (transition, dep, message) are plain text. Escaping per TOON spec section 7.1 handled by toon-go library.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/toon_formatter.go:1-202
- Notes:
  - ToonFormatter struct embeds baseFormatter (provides FormatTransition, FormatDepChange)
  - Compile-time interface verification at line 19: `var _ Formatter = (*ToonFormatter)(nil)`
  - FormatTaskList (line 54): Handles zero-count with correct schema header, delegates encoding to toon-go via encodeToonSection generic helper
  - FormatTaskDetail (line 71): Multi-section output with dynamic schema via buildTaskSection; blocked_by/children always present; description conditional on non-empty
  - FormatStats (line 92): Two sections -- stats summary as single-object scope, by_priority always 5 rows (0-4)
  - FormatMessage (line 118): Plain text passthrough
  - buildTaskSection (line 123): Dynamic schema using toon.Field slice -- parent included only when non-empty, closed only when non-nil
  - buildRelatedSection (line 157): Zero-count returns hardcoded schema header; non-zero delegates to toon-go
  - buildDescriptionSection (line 169): Splits on newline, prepends 2-space indent to each line
  - encodeToonSection (line 182): Generic helper wrapping toon-go MarshalString
  - encodeToonSingleObject (line 194): Generic helper that strips "[1]" for single-object scope format
  - All acceptance criteria are met

TESTS:
- Status: Adequate
- Coverage:
  - "it formats list with correct header count and schema" -- verifies 2-task list with correct header, count, and row data (line 15)
  - "it formats zero tasks as empty section" -- zero count with schema header (line 41)
  - "it formats zero tasks from nil slice as empty section" -- nil slice edge case (line 50)
  - "it formats show with all sections" -- 4 sections (task, blocked_by, children, description), verifies header schemas, row data, and description indentation (line 59)
  - "it omits parent and closed from schema when null" -- dynamic schema without parent/closed fields (line 120)
  - "it renders blocked_by and children with count 0 when empty" -- both sections present at [0] (line 148)
  - "it omits description section when empty" -- exactly 3 sections when no description (line 172)
  - "it renders multiline description as indented lines" -- 3-line description with 2-space indent (line 198)
  - "it escapes commas in titles" -- verifies toon-go quoting for comma-containing values (line 235)
  - "it formats stats with all counts" -- stats summary header and data row (line 249)
  - "it formats by_priority with 5 rows including zeros" -- header, count, and zero-value rows (line 278)
  - "it formats transition as plain text" -- via inherited baseFormatter (line 312)
  - "it formats dep change as plain text" -- add and remove variants (line 321)
  - "it formats message as plain text" -- passthrough verification (line 335)
  - "it includes closed in show schema when present" -- closed field in dynamic schema (line 344)
  - "it includes both parent and closed in show schema when both present" -- full dynamic schema (line 374)
  - All 11 planned tests are covered, plus 5 additional useful edge cases (nil slice, closed-only schema, both parent+closed schema, transition, dep change, message)
  - Tests verify behavior, not implementation details
  - Tests would fail if the feature broke

CODE QUALITY:
- Project conventions: Followed -- table-driven subtests pattern used (though most tests are individual subtests which is fine for output format verification), exported functions documented, error handling explicit, compile-time interface check present
- SOLID principles: Good -- single responsibility (ToonFormatter only handles TOON rendering), open/closed (Formatter interface allows new formats), baseFormatter composition avoids duplication for shared plain-text methods
- Complexity: Low -- each method is straightforward; helper functions (buildTaskSection, buildRelatedSection, buildDescriptionSection, encodeToonSection, encodeToonSingleObject) keep cyclomatic complexity low
- Modern idioms: Yes -- Go generics used for encodeToonSection and encodeToonSingleObject helpers, struct tags for toon serialization, strings.Builder for efficient string construction
- Readability: Good -- clear separation of concerns, well-named helper functions, comments explain non-obvious patterns (e.g., stripping "[1]" for single-object scope)
- Issues: None significant

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Timestamp quoting discrepancy: The spec examples show timestamps unquoted (e.g., `2026-01-19T10:00:00Z`) but the toon-go library quotes them (e.g., `"2026-01-19T10:00:00Z"`), likely because colons trigger TOON escaping rules. The task spec says "Escaping handled by toon-go" so the library's behavior is authoritative. The spec examples appear to be simplified illustrations. This is acceptable but worth documenting if questioned.
- The `buildTaskSection` helper (line 123-154) uses a somewhat brittle approach of encoding as a 1-element array then stripping "[1]" via string replacement. This works but couples to the toon-go output format. The same pattern is used in `encodeToonSingleObject`. If toon-go ever changes its output format (e.g., spacing around "[1]"), this would break silently. Low risk given the library is stable.
- The `type toonRelatedRow struct` at line 30 duplicates the `RelatedTask` struct fields. The type conversion on line 163 (`toonRelatedRow(r)`) works because the field names and types match, but this is somewhat fragile. Plan Phase 6 (tick-core-6-4) already identifies formatter duplication for consolidation.
