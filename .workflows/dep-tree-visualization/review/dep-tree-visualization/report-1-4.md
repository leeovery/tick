TASK: Pretty Formatter FormatDepTree

ACCEPTANCE CRITERIA:
- Full graph shows root tasks at top level with box-drawing indentation for blocked tasks
- Full graph ends with summary line: "{N} chains, longest: {M}, {B} blocked"
- Focused shows "Blocked by:" header with upstream tree when task has blockers
- Focused shows "Blocks:" header with downstream tree when task blocks others
- Asymmetric focused view omits empty sections
- Each tree line shows {id}  {title} ({status})
- Long titles truncated with "..." to fit available width
- Box-drawing characters correctly nested
- Diamond dependencies appear as duplicate entries
- All tests pass

STATUS: Complete

SPEC CONTEXT: The specification requires PrettyFormatter to render dependency trees with box-drawing characters (mid/last/pipe connectors), inline metadata per task (ID + title truncated to fit + status), diamond dependency duplication (no dedup), labeled focused-view sections ("Blocked by:" / "Blocks:"), and asymmetric section omission. Title truncation must account for indentation + ID + status, using 80-char default width.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:299-406
- Notes:
  - `FormatDepTree` (line 307-312) dispatches to `formatFullDepTree` or `formatFocusedDepTree` based on `result.Target != nil` rather than a Mode string field. This is a minor positive drift from the plan (which mentioned `result.Mode`) -- the nil-pointer check is more idiomatic Go.
  - `formatFullDepTree` (line 316-332) correctly iterates roots, renders each with box-drawing children, separates roots with blank lines, and appends the pre-formatted summary string.
  - `formatFocusedDepTree` (line 337-358) renders the target task header, then conditionally renders "Blocked by:" and "Blocks:" sections only when non-empty.
  - `writeDepTreeTaskLine` (line 362-365) renders `{prefix}{id}  {title} ({status})` with depth-aware title truncation.
  - `writeDepTreeNodes` (line 377-386) delegates to the shared generic `writeTree` helper with `depTreeStyle`.
  - `truncateDepTreeTitle` (line 391-406) computes available width as `80 - (depth*4 + 27)`, floors at `depTreeMinTitle` (10), and truncates with "..." when needed.
  - `depTreeStyle` (line 368-373) uses correct box-drawing characters with 4-char-per-level indentation.
  - The shared `writeTree[T any]` generic helper (line 23-42) is reused from cascade transition rendering -- good DRY adherence.
  - Edge case: focused view with no dependencies renders task info + message (line 341-344).

TESTS:
- Status: Adequate
- Coverage: All 13 planned tests are present in `TestPrettyFormatDepTree` (line 745-1097), plus 1 additional well-motivated test for the no-dependencies edge case. Tests cover:
  - Full graph: single chain, multiple roots, diamond duplication, deep chain indentation, summary line
  - Focused: both sections, only blocked-by, only blocks, asymmetric omission, target header
  - Title truncation with ellipsis
  - Box-drawing character verification
  - Task ID and status presence in each line
  - No-deps focused view with message
- Notes:
  - Tests construct `DepTreeResult` directly (no store needed) -- appropriately unit-level.
  - Most tests use exact string comparison, which is thorough and will catch regressions.
  - The asymmetric test (line 955) checks absence of "Blocked by:" which properly validates omission behavior.
  - The truncation test (line 969) verifies both presence of "..." and absence of the full long title.
  - No over-testing observed -- each test covers a distinct scenario.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, strings.Builder for output, same box-drawing patterns as existing cascade formatter.
- SOLID principles: Good. Single responsibility (each helper does one thing), open for extension (writeTree generic handles both cascade and dep tree), proper separation between mode dispatch and rendering.
- Complexity: Low. Clear linear code paths, simple recursive delegation via writeTree.
- Modern idioms: Yes. Uses Go generics for writeTree, proper io.Writer interface usage.
- Readability: Good. Well-commented functions, clear naming (depTreeLineWidth, depTreeMinTitle constants), self-documenting structure.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The truncation overhead calculation at line 392-394 assumes a fixed ID length of 11 chars ("tick-XXXXXX") and max status length of 11 chars ("in_progress"). These are reasonable assumptions for the current system but are hardcoded rather than derived from the actual task data. This is fine given the spec says "Tick projects won't realistically hit problematic depths" and the depTreeMinTitle floor provides graceful degradation.
