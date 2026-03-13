TASK: cli-enhancements-2-5 -- Display Type in list and show output across all formatters

ACCEPTANCE CRITERIA:
- List output includes Type column (ID, Status, Priority, Type, Title); dash (`-`) when unset
- Show output displays type field
- All three formatters (ToonFormatter, PrettyFormatter, JSONFormatter) updated

STATUS: Complete

SPEC CONTEXT:
The specification (Display section under Task Types) states: "List output: shown as a column -- ID, Status, Priority, Type, Title. Dash (`-`) when not set." and "Show output: displayed with other fields." The task's listed edge case is "type unset showing as dash in list."

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:24-78` -- PrettyFormatter.FormatTaskList includes TYPE column in header and rows; `typeOrDash()` helper at line 190-195 renders "-" when empty
  - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:82-149` -- PrettyFormatter.FormatTaskDetail renders `Type:     <value>` using `typeOrDash()` at line 91
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:22-28` -- toonTaskRow struct includes `Type string` field with `toon:"type"` tag
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:61-76` -- ToonFormatter.FormatTaskList maps Type field into list rows
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:144-178` -- buildTaskSection conditionally includes type in show schema (line 154-156)
  - `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:17-23` -- jsonTaskListItem includes `Type string` with `json:"type"` tag
  - `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:27-39` -- JSONFormatter.FormatTaskList maps Type into JSON list items
  - `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:58-74` -- jsonTaskDetail includes `Type string` with `json:"type"` tag
  - `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:79-120` -- JSONFormatter.FormatTaskDetail maps Type into JSON detail object
  - `/Users/leeovery/Code/tick/internal/cli/list.go:163-218` -- RunList SQL query selects `t.type` at line 307 and maps it into task.Task.Type at lines 215-218
  - `/Users/leeovery/Code/tick/internal/cli/show.go:74-76` -- queryShowData SQL selects `type` column; mapped into showData.taskType at lines 84-86
  - `/Users/leeovery/Code/tick/internal/cli/show.go:210-240` -- showDataToTaskDetail maps taskType into task.Task.Type at line 219
- Notes: Column order in Pretty list header is ID, STATUS, PRI, TYPE, TITLE -- matches the spec's required order (ID, Status, Priority, Type, Title). Dynamic column widths computed for TYPE column. The Toon formatter includes Type in the list schema (always present, empty string when unset). The JSON formatter always includes type key (empty string when unset, not omitted). The Pretty formatter uses dash for unset in both list and show. The Toon formatter omits type from show schema when empty (dynamic schema approach), which is a reasonable design choice consistent with how parent/closed are handled.

TESTS:
- Status: Adequate
- Coverage:
  - **ToonFormatter list with type**: `toon_formatter_test.go:392-418` -- verifies type value appears in list rows and schema header
  - **ToonFormatter show with type set**: `toon_formatter_test.go:420-449` -- verifies type in schema header and row value
  - **ToonFormatter show with type empty**: `toon_formatter_test.go:451-478` -- verifies type is omitted from schema when empty
  - **PrettyFormatter list with type column**: `pretty_formatter_test.go:371-404` -- verifies TYPE column exists between PRI and TITLE in header, rows contain type values
  - **PrettyFormatter dash for unset type in list**: `pretty_formatter_test.go:406-432` -- edge case: verifies `-` in type column when unset
  - **PrettyFormatter show with type**: `pretty_formatter_test.go:434-460` -- verifies `Type:     bug` in show output
  - **PrettyFormatter show with dash for unset type**: `pretty_formatter_test.go:462-481` -- verifies `Type:     -` when unset
  - **JSONFormatter list with type**: `json_formatter_test.go:707-729` -- verifies type key present with correct values
  - **JSONFormatter list with unset type**: `json_formatter_test.go:827-853` -- verifies type key exists as empty string when unset
  - **JSONFormatter show with type**: `json_formatter_test.go:800-825` -- verifies type key in detail JSON
  - **Integration test list with type column in header**: `list_show_test.go:45-78` -- end-to-end test verifying aligned columns including TYPE
  - **Integration test show with type set**: `list_show_test.go:677-693` -- end-to-end show with type=bug
  - **Integration test show with type unset (dash)**: `list_show_test.go:695-711` -- end-to-end show with unset type shows `-`
  - **Integration test type display after create**: `list_show_test.go:713-724` -- verifies type in post-mutation output
  - **Integration test type display after update**: `list_show_test.go:726-743` -- verifies type in post-mutation output
  - Pre-existing formatter tests also indirectly validate type presence (e.g., "it formats list with aligned columns" at `pretty_formatter_test.go:15-29` includes TYPE in the expected header string)
- Notes: Tests cover all three formatters for both list and show, plus the key edge case (unset type showing as dash/empty string). Integration tests exercise the full storage-to-display pipeline. The tests are focused and not over-tested -- each subtest verifies a distinct behavior.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing, t.Run subtests, "it does X" naming, t.Helper on helpers, error wrapping with %w, handler signature pattern
- SOLID principles: Good -- `typeOrDash()` helper has single responsibility; Formatter interface respected; formatters each handle type in their own style without leaking implementation details
- Complexity: Low -- the type display logic is simple string field rendering; `typeOrDash()` is a straightforward ternary-style helper
- Modern idioms: Yes -- Go generics used in `encodeToonSection[T any]()`, struct tags for serialization, consistent `strings.Builder` usage
- Readability: Good -- `typeOrDash` function name is self-documenting; column width computation is clear; consistent formatting patterns across all three formatters
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
