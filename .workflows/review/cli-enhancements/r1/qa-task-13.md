TASK: cli-enhancements-3-3 -- Tags display in show output and all formatters

ACCEPTANCE CRITERIA:
- Tags displayed in show output; not displayed in list output
- All three formatters updated for tags in show/detail views

STATUS: Complete

SPEC CONTEXT:
Specification states tags should be displayed in show output but NOT in list output ("List output: not shown (variable-length, would clutter the table)"). The display is in the `FormatTaskDetail` method, not `FormatTaskList`. Edge cases from the plan: task with no tags, task with 10 tags.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/format.go:101-103` -- TaskDetail struct has Tags []string field, populated by show.go
  - `/Users/leeovery/Code/tick/internal/cli/show.go:125-132` -- queryShowData fetches tags from task_tags junction table
  - `/Users/leeovery/Code/tick/internal/cli/show.go:238` -- showDataToTaskDetail passes tags through to TaskDetail
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:91-94` -- ToonFormatter.FormatTaskDetail includes tags section (omitted when empty)
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:193-196` -- buildTagsSection delegates to buildStringListSection
  - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:93-95` -- PrettyFormatter.FormatTaskDetail shows "Tags: tag1, tag2" (omitted when empty)
  - `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:64` -- jsonTaskDetail has Tags []string field (always present as array)
  - `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:87-88,107` -- JSONFormatter.FormatTaskDetail includes tags (non-nil empty slice for empty tags)
- Notes:
  - Tags are correctly NOT included in any FormatTaskList implementation (toonTaskRow, PrettyFormatter.FormatTaskList, jsonTaskListItem all omit tags)
  - ToonFormatter omits tags section entirely when empty (line 92: `if len(detail.Tags) > 0`)
  - PrettyFormatter omits tags line when empty (line 93: `if len(detail.Tags) > 0`)
  - JSONFormatter always includes tags as an array (empty [] when no tags) -- correct for JSON semantics
  - Tags are queried sorted by tag name (`ORDER BY tag` at show.go:128), which gives alphabetical ordering

TESTS:
- Status: Adequate
- Coverage:
  - ToonFormatter: "it displays tags in toon format show output" (line 480, toon_formatter_test.go) verifies count header and indented values; "it omits tags section in toon format when task has no tags" (line 509) verifies empty case
  - PrettyFormatter: "it displays tags in pretty format show output" (line 483, pretty_formatter_test.go) verifies comma-separated format and position after Type; "it omits tags section in pretty format when task has no tags" (line 511) verifies omission
  - JSONFormatter: "it displays tags in json format show output" (line 731, json_formatter_test.go) verifies array with values; "it shows empty tags array in json format when task has no tags" (line 769) verifies empty array (not null)
  - Integration tests in list_show_test.go: "it displays tags in show output" (line 462), "it omits tags section in show output when task has no tags" (line 479), "it displays tags in alphabetical order" (line 496), "it displays all 10 tags when task has maximum tags" (line 513)
  - Both spec edge cases covered: no tags (multiple tests across all formatters) and 10 tags (list_show_test.go:513)
- Notes:
  - There is no explicit test asserting tags do NOT appear in list output (unlike refs which have "it does not show refs in list output" at pretty_formatter_test.go:603). However, the FormatTaskList implementations structurally exclude tags (the list row structs don't have a Tags field), so breakage would require adding tags to the row struct -- which would likely be caught by other tests. This is a minor gap.

CODE QUALITY:
- Project conventions: Followed -- uses t.Run subtests, t.Helper on helpers, stdlib testing only, Formatter interface pattern
- SOLID principles: Good -- each formatter has clear single responsibility for its format; buildTagsSection delegates to shared buildStringListSection (DRY with refs); TaskDetail struct cleanly separates data gathering (show.go) from rendering (formatters)
- Complexity: Low -- simple conditional inclusion of tags section in each formatter
- Modern idioms: Yes -- Go generics used in encodeToonSection[T]; proper nil-safe slice handling in JSONFormatter
- Readability: Good -- consistent formatting patterns across all three formatters; clear section ordering in FormatTaskDetail
- Issues: None

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- Consider adding a test that explicitly verifies tags do not appear in list output (similar to the refs test "it does not show refs in list output" in pretty_formatter_test.go:603). The structural exclusion makes this low risk, but having the test would provide documentation and guard against future regressions.
