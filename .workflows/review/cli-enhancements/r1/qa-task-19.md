TASK: cli-enhancements-4-4 -- Refs display in show output and all formatters

ACCEPTANCE CRITERIA:
- Show output displays refs
- All three formatters updated for refs display in detail views
- Refs not shown in list output

STATUS: Complete

SPEC CONTEXT:
The specification states refs are a `[]string` field for cross-system links (gh-123, JIRA-456, URLs). Display rules: "List output: not shown" and "Show output: displayed with other fields". Phase 4 acceptance also requires: "Show output displays refs" and "All three formatters updated for refs display in detail views" and "Refs not shown in list output". Edge cases from the plan: task with no refs, task with 10 refs.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/format.go:102-103` -- `TaskDetail` struct includes `Refs []string` field
  - `/Users/leeovery/Code/tick/internal/cli/show.go:29` -- `showData` struct includes `refs []string`
  - `/Users/leeovery/Code/tick/internal/cli/show.go:134-141` -- `queryShowData` queries refs from `task_refs` junction table via `queryStringColumn` helper
  - `/Users/leeovery/Code/tick/internal/cli/show.go:236` -- `showDataToTaskDetail` maps `d.refs` to `TaskDetail.Refs`
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:96-99` -- ToonFormatter: refs section included when non-empty, omitted when empty
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:199-201` -- `buildRefsSection` delegates to `buildStringListSection`
  - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:126-131` -- PrettyFormatter: refs displayed as "Refs:" section with indented entries, only when non-empty
  - `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:65` -- `jsonTaskDetail` includes `Refs []string` with `json:"refs"` tag (always present as array)
  - `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:90-91` -- `FormatTaskDetail` populates refs with `make([]string, 0, ...)` ensuring `[]` not `null`
- Notes: All three formatters handle refs correctly. Toon omits when empty, Pretty omits when empty, JSON always includes as `[]` (never null). Refs are NOT included in any `FormatTaskList` implementation -- the `toonTaskRow`, `jsonTaskListItem`, and Pretty list columns all exclude refs. Implementation fully matches spec and acceptance criteria.

TESTS:
- Status: Adequate
- Coverage:
  - ToonFormatter:
    - `/Users/leeovery/Code/tick/internal/cli/toon_formatter_test.go:530-557` -- "it displays refs in toon show output": verifies `refs[2]:` header and indented values for 2 refs
    - `/Users/leeovery/Code/tick/internal/cli/toon_formatter_test.go:559-578` -- "it omits refs section in toon format when task has no refs": verifies no refs section appears
  - PrettyFormatter:
    - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter_test.go:541-573` -- "it displays refs in pretty show output": verifies "Refs:" section with indented entries, position after Type
    - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter_test.go:575-601` -- "it displays all 10 refs when task has maximum": verifies all 10 refs appear (10-ref edge case)
    - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter_test.go:603-620` -- "it does not show refs in list output": verifies refs absent from list output (explicitly tests the "refs not in list" criterion)
    - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter_test.go:622-641` -- "it omits refs section in pretty when no refs": verifies no Refs section when empty
  - JSONFormatter:
    - `/Users/leeovery/Code/tick/internal/cli/json_formatter_test.go:870-906` -- "it displays refs in json show output": verifies refs array with 2 values
    - `/Users/leeovery/Code/tick/internal/cli/json_formatter_test.go:908-937` -- "it shows empty refs array in json when no refs": verifies `[]` not null
  - Integration (via CLI):
    - `/Users/leeovery/Code/tick/internal/cli/list_show_test.go:589-611` -- "it displays refs in show output when task has refs": end-to-end via `runShow` with Pretty formatter
    - `/Users/leeovery/Code/tick/internal/cli/list_show_test.go:613-629` -- "it omits refs section in show when task has no refs": end-to-end empty case
- Notes: Both edge cases from the plan are covered: no refs (empty/omitted tests across all formatters) and 10 refs (Pretty formatter max-refs test). The "refs not shown in list" acceptance criterion is explicitly tested in the Pretty formatter test. Toon and JSON list formatters inherently exclude refs by their struct definitions (no refs field in `toonTaskRow` or `jsonTaskListItem`), which is implicitly verified by existing list tests that would fail if extra fields appeared. Test coverage is thorough without being redundant.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, "it does X" naming pattern, t.Helper on test helpers.
- SOLID principles: Good. Each formatter has single responsibility. The `buildStringListSection` helper (shared by tags and refs in toon) follows DRY. The `queryStringColumn` helper in show.go avoids duplicating SQL scanning logic for tags and refs.
- Complexity: Low. Refs display logic is straightforward conditional inclusion in each formatter. No complex branching.
- Modern idioms: Yes. Uses generics in `encodeToonSection[T any]`. `make([]string, 0, len(...))` pattern ensures non-nil slices for JSON marshaling.
- Readability: Good. Code is self-documenting. Formatter sections are clearly commented. The pattern mirrors the existing tags implementation, making it easy to follow.
- Issues: None identified.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The Toon and JSON formatters do not have explicit "refs not in list" tests. This is implicitly safe because their list structs (`toonTaskRow`, `jsonTaskListItem`) exclude refs fields, but a paranoid reviewer might want symmetry with the Pretty formatter test. Not blocking since the struct-level exclusion is definitive.
