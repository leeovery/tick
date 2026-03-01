TASK: cli-enhancements-2-6 -- Add --count flag to list/ready/blocked

ACCEPTANCE CRITERIA:
- `--count N` on `list`, `ready`, `blocked` appends `LIMIT N` to query; value must be >= 1

STATUS: Complete

SPEC CONTEXT: The specification defines a `--count N` flag to cap the number of results returned, translating to a SQL LIMIT clause, with the constraint that value must be >= 1 (zero or negative values error). The flag applies to `list`, `ready`, and `blocked` commands.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/list.go:28-31` -- `Count int` and `HasCount bool` fields on `ListFilter` struct
  - `/Users/leeovery/Code/tick/internal/cli/list.go:90-101` -- Flag parsing in `parseListFlags`, handles missing value and non-integer input
  - `/Users/leeovery/Code/tick/internal/cli/list.go:140-142` -- Post-parse validation enforcing `>= 1`
  - `/Users/leeovery/Code/tick/internal/cli/list.go:313-316` -- SQL LIMIT clause appended in `buildListQuery` when `HasCount` is true
  - `/Users/leeovery/Code/tick/internal/cli/app.go:184,197` -- `ready` and `blocked` handlers reuse `parseListFlags`, so `--count` is automatically supported
  - `/Users/leeovery/Code/tick/internal/cli/help.go:65,161,177` -- Help text includes `--count` for all three commands
- Notes: Clean implementation. Uses the `HasCount` boolean pattern consistent with `HasPriority` for optional flags. The LIMIT is applied after ORDER BY, ensuring deterministic results. The parameterized `LIMIT ?` approach prevents SQL injection.

TESTS:
- Status: Adequate
- Coverage:
  - Happy path: limits list results to N (`--count 2` with 3 tasks returns 2) -- line 607
  - Count exceeds result set: returns all tasks gracefully (`--count 100` with 3 tasks) -- line 633
  - Error: `--count 0` produces error with "must be >= 1" -- line 653
  - Error: `--count -1` (negative) produces error with "must be >= 1" -- line 666
  - Error: `--count abc` (non-integer) produces error with "invalid count" -- line 679
  - Error: `--count` without value produces "--count requires a value" -- line 692
  - Ready command: `ready --count 2` limits ready results -- line 705
  - Blocked command: `blocked --count 2` limits blocked results -- line 724
  - Combination: `--type bug --count 2` filters then limits -- line 744
  - Combination: `--tag ui --count 2` (in tag_filter_test.go:278) -- cross-filter composition
  - Baseline: no `--count` returns all results -- line 774
- Notes: All four edge cases from the plan (--count 0, --count negative, --count non-integer, --count larger than result set) are covered. Tests verify both exit codes and error message content. The combination tests confirm LIMIT applies after WHERE filtering. Test count is appropriate -- not over-tested, each test covers a distinct scenario.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib `testing`, `t.Run()` subtests with "it does X" naming, `t.TempDir()` for isolation, `t.Helper()` on helpers. Flag pattern matches existing `--priority` with `HasPriority`/`HasCount` boolean.
- SOLID principles: Good. `parseListFlags` has single responsibility (parse+validate), `buildListQuery` composes SQL independently. The LIMIT logic is a clean addition to the existing query builder without modifying other filter paths.
- Complexity: Low. The `--count` handling adds a simple case branch, one validation check, and a conditional LIMIT clause. No increase in cyclomatic complexity worth noting.
- Modern idioms: Yes. Parameterized SQL query (`LIMIT ?`), `strconv.Atoi` for parsing, `fmt.Errorf` for error messages.
- Readability: Good. The `HasCount` flag pattern makes intent clear (distinguish "not provided" from "provided as 0"). Comments on the struct fields explain purpose.
- Issues: None.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- (none)
