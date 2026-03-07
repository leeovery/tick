TASK: Human-readable formatter -- list, show, stats output (tick-core-4-3)

ACCEPTANCE CRITERIA:
- [ ] Implements full Formatter interface
- [ ] List matches spec format -- aligned columns with header
- [ ] Empty list -> "No tasks found."
- [ ] Show matches spec -- aligned labels, omitted empty sections
- [ ] Stats three groups with right-aligned numbers
- [ ] Priority P0-P4 labels always present
- [ ] Long titles truncated in list
- [ ] All output matches spec examples

STATUS: Complete

SPEC CONTEXT: The spec (section "Human-Readable Format") defines aligned columns with no borders/colors/icons. List output uses columns ID, STATUS, PRI, TITLE. Show output uses key-value pairs with aligned labels plus optional sections (Blocked by, Children, Description). Stats output has four groups: Total, Status (Open/In Progress/Done/Cancelled), Workflow (Ready/Blocked), Priority (P0-P4 with labels). Empty results show "No tasks found." with no headers. Design philosophy: "Minimalist and clean. Human output is secondary to agent output."

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:1-167
- Notes:
  - PrettyFormatter struct embeds baseFormatter for shared FormatTransition and FormatDepChange (line 14-16)
  - Compile-time interface check at line 19
  - FormatTaskList (lines 24-70): Dynamic column widths computed from data. Gutter spacing of 3 for ID, 2 for STATUS/PRI. Empty input returns "No tasks found." Titles truncated via truncateTitle helper.
  - FormatTaskDetail (lines 74-121): Key-value format with aligned labels. Parent shown with title when available. Blocked by/Children/Description sections omitted when empty.
  - FormatStats (lines 127-154): Four groups (Total, Status, Workflow, Priority) with right-aligned numbers. All P0-P4 labels present. Fixed-width format strings ensure alignment.
  - FormatMessage (lines 157-159): Plain passthrough.
  - truncateTitle (lines 162-167): Truncates at 50 chars (maxListTitleLen constant at line 10) with "..." suffix.
  - All methods produce output matching the spec examples character-for-character.

TESTS:
- Status: Adequate
- Coverage:
  - "it formats list with aligned columns" -- verifies exact output matches spec example (tick-a1b2/done + tick-c3d4/in_progress)
  - "it aligns with variable-width data" -- tests dynamic column widths with variable-length IDs and statuses (in_progress vs open)
  - "it shows No tasks found for empty list" -- empty slice
  - "it shows No tasks found for nil list" -- nil input (bonus edge case)
  - "it formats show with all sections" -- full TaskDetail with parent, blockers, children, description
  - "it omits empty sections in show" -- no blockers/children/description, verifies absence of section headers
  - "it includes closed timestamp when present in show" -- Closed field rendering
  - "it formats stats with all groups right-aligned" -- exact expected output with non-zero values (matches spec example)
  - "it shows zero counts in stats" -- all zeros, verifies all rows present
  - "it renders P0-P4 priority labels" -- checks all 5 label strings present
  - "it truncates long titles in list" -- 80-char title, verifies "..." suffix and <= 50 char limit
  - "it does not truncate in show" -- full title preserved in show output
  - "it formats transition as plain text" -- inherited baseFormatter method
  - "it formats dep change as plain text" -- inherited baseFormatter method (add + remove)
  - "it formats message as plain text passthrough" -- FormatMessage
- Notes: Tests are well-structured, focused, and cover all acceptance criteria plus edge cases. No over-testing detected. Each test verifies a distinct behavior. The "closed timestamp" and "nil list" tests go beyond the task's listed tests but are valuable additions, not redundant.

CODE QUALITY:
- Project conventions: Followed. Table-driven patterns not used for single-case tests, which is appropriate here since each test has unique setup/assertions. Exported types documented. File follows project structure conventions.
- SOLID principles: Good. Single responsibility (PrettyFormatter only handles pretty formatting). Open/closed via Formatter interface. Dependency inversion via Formatter interface. baseFormatter composition for shared behavior is clean.
- Complexity: Low. FormatTaskList has straightforward linear passes (compute widths, render). FormatStats uses fixed format strings. No complex branching.
- Modern idioms: Yes. strings.Builder for string concatenation. fmt.Fprintf with width specifiers. Compile-time interface check. Embedding for composition.
- Readability: Good. Clear method names, good comments explaining alignment arithmetic (lines 125-126). Constants extracted (maxListTitleLen). Helper function (truncateTitle) keeps FormatTaskList clean.
- Issues: None significant.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The stats format widths are hardcoded with magic numbers in format strings (e.g., %8d, %10d, %2d). This works for the current label set but would require manual recalculation if labels change. Acceptable for the current scope since the labels are spec-defined and unlikely to change.
- The `truncateTitle` function uses byte length (`len(title)`) rather than rune count. For ASCII-only titles (expected in a task tracker), this is fine. For Unicode titles, it could truncate mid-rune. Very low risk given the domain.
