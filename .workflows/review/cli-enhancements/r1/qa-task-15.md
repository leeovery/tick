TASK: cli-enhancements-3-5 -- Tag filtering on list/ready/blocked with AND/OR composition

ACCEPTANCE CRITERIA:
- `--tag <comma-separated>` on `list`, `ready`, `blocked`: comma values are AND (task must have all), multiple `--tag` flags are OR
- Filter input normalized (trimmed, lowercased) before matching

STATUS: Complete

SPEC CONTEXT: Tags filtering uses composable AND/OR semantics: `--tag ui,backend --tag api` means "(ui AND backend) OR api". Filter input normalized (trimmed, lowercased) before matching. Invalid kebab-case should error.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/list.go:14-32` -- ListFilter struct with TagGroups field
  - `/Users/leeovery/Code/tick/internal/cli/list.go:73-89` -- parseListFlags `--tag` parsing: splits on comma, normalizes via NormalizeTag, skips empty parts, appends group
  - `/Users/leeovery/Code/tick/internal/cli/list.go:132-138` -- Post-parse validation of each tag in each group against ValidateTag (kebab-case enforcement)
  - `/Users/leeovery/Code/tick/internal/cli/list.go:289-293` -- buildListQuery integrates TagGroups via buildTagFilterSQL
  - `/Users/leeovery/Code/tick/internal/cli/list.go:324-346` -- buildTagFilterSQL generates correct AND/OR SQL using subqueries with `tag IN (...) GROUP BY task_id HAVING COUNT(DISTINCT tag) = ?`
  - `/Users/leeovery/Code/tick/internal/cli/app.go:179-201` -- handleReady and handleBlocked both route through parseListFlags + RunList, ensuring --tag works on all three commands
  - `/Users/leeovery/Code/tick/internal/cli/help.go:61,159,175` -- Help text for list, ready, blocked all document --tag flag
- Notes: Implementation is clean and correct. The SQL approach using `COUNT(DISTINCT tag)` for AND semantics and `OR` across groups is standard and efficient. Normalization happens at parse time (NormalizeTag = trim + lowercase), and validation happens after normalization. Empty tags after normalization are silently dropped, preventing empty-string-in-group edge cases.

TESTS:
- Status: Adequate
- Coverage:
  - Single tag filter (`it filters list by single tag`)
  - AND composition via comma-separated tags (`it filters list by AND (comma-separated tags)`)
  - OR composition via multiple --tag flags (`it filters list by OR (multiple --tag flags)`)
  - Combined AND/OR composition (`it filters list by AND/OR composition (--tag ui,backend --tag api)`)
  - No matching tasks returns empty list (`it returns empty list when no tasks match tag filter`)
  - Invalid kebab-case in filter errors (`it rejects invalid kebab-case tag in filter`)
  - Case normalization (`it normalizes tag filter input to lowercase`)
  - Ready command with tag filter (`it filters ready tasks by tag`)
  - Blocked command with tag filter (`it filters blocked tasks by tag`)
  - Combined with --status filter (`it combines --tag with --status filter`)
  - Combined with --priority filter (`it combines --tag with --priority filter`)
  - Combined with --parent filter (`it combines --tag with --parent filter`)
  - Combined with --count flag (`it combines --tag with --count flag`)
  - No --tag flag returns all tasks (`it returns all tasks when --tag not specified`)
- Notes: All edge cases from the task definition are covered: invalid kebab-case in filter, no matching tasks, single tag value, multiple --tag flags, combined with other filters. Tests are integration-level (run through full App.Run path), which properly exercises the entire stack from flag parsing through SQL execution. Tests are focused and non-redundant -- each test verifies a distinct scenario.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests with "it does X" naming, t.Helper on helpers, IsTTY=true for pretty formatter in tests.
- SOLID principles: Good. buildTagFilterSQL is a clean single-responsibility function. Tag normalization and validation are properly delegated to the task package. ListFilter struct cleanly encapsulates all filter state.
- Complexity: Low. The buildTagFilterSQL function has straightforward iteration with no deep nesting. parseListFlags is a linear switch-case parser.
- Modern idioms: Yes. Proper use of Go string manipulation, SQL parameterization (no injection risk), slice handling.
- Readability: Good. buildTagFilterSQL has a clear docstring explaining semantics. Variable names are descriptive (groupClauses, placeholders, tagGroups). The single-group optimization at line 342 avoids unnecessary parentheses and is well-motivated.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
