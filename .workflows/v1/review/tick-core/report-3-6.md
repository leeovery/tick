TASK: tick stats command (tick-core-5-1)

ACCEPTANCE CRITERIA:
- [ ] StatsQuery returns correct counts by status, priority, workflow
- [ ] All 5 priority levels always present (P0-P4)
- [ ] Ready/blocked counts match Phase 3 query semantics
- [ ] Empty project returns all zeros with full structure
- [ ] TOON format matches spec example
- [ ] Pretty format matches spec example with right-aligned numbers
- [ ] JSON format nested with correct keys
- [ ] --quiet suppresses all output

STATUS: Complete

SPEC CONTEXT: The spec defines `tick stats` as producing three output formats. TOON uses two sections: `stats{total,open,in_progress,done,cancelled,ready,blocked}:` for summary counts, and `by_priority[5]{priority,count}:` for the priority breakdown. Pretty uses four visual groups (Total, Status, Workflow, Priority) with right-aligned numbers and P0-P4 labels (critical/high/medium/low/backlog). JSON is a nested object with `total`, `by_status`, `workflow`, and `by_priority` keys. Ready/blocked semantics reuse the Phase 3 query logic: ready = open + no unclosed blockers + no open children; blocked = open - ready.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/stats.go:1-95` -- RunStats function, StatsQuery logic
  - `/Users/leeovery/Code/tick/internal/cli/format.go:102-112` -- Stats struct definition
  - `/Users/leeovery/Code/tick/internal/cli/query_helpers.go:60-65` -- ReadyWhereClause used by stats
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:91-115` -- TOON FormatStats
  - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:127-154` -- Pretty FormatStats
  - `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:165-190` -- JSON FormatStats
  - `/Users/leeovery/Code/tick/internal/cli/app.go:84-85,175-182` -- CLI wiring
- Notes:
  - StatsQuery executes 4 SQL queries within a single `store.Query` callback: total count, group by status, group by priority, and ready count. Blocked is derived as `Open - Ready` which correctly implements the spec semantics.
  - Ready count reuses the shared `ReadyWhereClause()` from `query_helpers.go`, ensuring consistency with Phase 3 ready query logic.
  - `ByPriority` is a `[5]int` array (zero-initialized by Go), so absent priorities automatically appear as 0.
  - `--quiet` returns nil immediately before any DB work, producing no output -- correct per spec ("stats has no mutation ID to return").
  - All three formatter implementations produce output matching the spec examples.

TESTS:
- Status: Adequate
- Coverage:
  - `TestStats` in `/Users/leeovery/Code/tick/internal/cli/stats_test.go` contains 8 subtests matching all 8 planned tests:
    1. "it counts tasks by status correctly" -- 7 tasks across all 4 statuses, verifies JSON counts
    2. "it counts ready and blocked tasks correctly" -- tests blocked-by-dep, parent-with-open-child, leaf ready; verifies ready=2, blocked=2
    3. "it includes all 5 priority levels even at zero" -- only priority-2 tasks, verifies all 5 entries present with correct counts
    4. "it returns all zeros for empty project" -- empty project, verifies total=0, all statuses=0, workflow=0, 5 priority entries all 0
    5. "it formats stats in TOON format" -- verifies exact TOON header/row format including section separation
    6. "it formats stats in Pretty format with right-aligned numbers" -- verifies group headers and right-aligned number formatting
    7. "it formats stats in JSON format with nested structure" -- verifies all 4 top-level keys and correct nested values
    8. "it suppresses output with --quiet" -- verifies empty stdout
  - Formatter-specific tests also exist in:
    - `/Users/leeovery/Code/tick/internal/cli/toon_formatter_test.go:250-308` -- TOON stats with spec example values + zero-count priority rows
    - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter_test.go:195-284` -- Pretty stats exact format match, zero counts, P0-P4 labels
    - `/Users/leeovery/Code/tick/internal/cli/json_formatter_test.go:360-457` -- JSON stats nested structure, zero-count priority
  - Edge cases covered: empty project, all-same-status (implicitly via tests), zero-count priorities, ready/blocked with dependency and hierarchy combinations.
  - Tests use real store/SQLite (via `setupTickProjectWithTasks`), not mocks -- tests would fail if the feature broke.
- Notes: Test coverage is thorough without being redundant. The stats_test.go tests verify end-to-end behavior, while the formatter-specific tests isolate formatting logic. No over-testing observed.

CODE QUALITY:
- Project conventions: Followed. Table-driven style where appropriate. Test helpers use `t.Helper()`. Exported functions documented. Error handling with `fmt.Errorf("%w", err)`.
- SOLID principles: Good. Single responsibility -- RunStats handles orchestration, formatters handle rendering, query_helpers handle SQL. Formatter interface cleanly abstracts output format. Stats struct is a simple data transfer object.
- Complexity: Low. RunStats is straightforward: 4 SQL queries, populate struct, format, print. No branching beyond the status switch. Blocked count derived arithmetically rather than via another query -- clean and efficient.
- Modern idioms: Yes. Uses generics for `encodeToonSingleObject[T any]` and `encodeToonSection[T any]`. `[5]int` array type leverages Go's zero-value initialization for always-present priorities.
- Readability: Good. Code is self-documenting with clear variable names and inline comments explaining each SQL query section. The `stats.Blocked = stats.Open - stats.Ready` derivation is a particularly clean choice.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `readyQuery` construction in stats.go:79 uses string concatenation with embedded literal whitespace. This works but the raw string with `\n\t\t\t` is slightly less readable than a multi-line raw string. Trivial and cosmetic only.
