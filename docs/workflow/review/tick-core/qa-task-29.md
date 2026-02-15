TASK: Extract shared helpers for --blocks application and ID parsing (tick-core-6-5)

ACCEPTANCE CRITERIA:
- No inline comma-separated ID parsing loops remain in create.go or update.go
- No inline --blocks application loops remain in create.go or update.go
- Both helpers are called from both create and update
- All existing create and update tests pass

STATUS: Complete

SPEC CONTEXT: The spec defines --blocked-by and --blocks flags on create (comma-separated IDs) and --blocks on update. IDs are case-insensitive, normalized to lowercase on input. The --blocks flag is an inverse: setting --blocks tick-abc on task T adds T to tick-abc's blocked_by array. The update command cannot change blocked_by directly (use dep add/rm).

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/helpers.go:44-77
- `parseCommaSeparatedIDs` at helpers.go:44-54: splits on comma, trims whitespace, normalizes via task.NormalizeID, filters empty values
- `applyBlocks` at helpers.go:59-77: iterates tasks, matches by normalized blockID, appends sourceID to BlockedBy (with duplicate prevention), sets Updated timestamp
- create.go uses `parseCommaSeparatedIDs` at lines 53 and 59 (--blocked-by and --blocks), `applyBlocks` at line 148
- update.go uses `parseCommaSeparatedIDs` at line 77 (--blocks), `applyBlocks` at line 185
- No inline `strings.Split` on comma exists in create.go or update.go
- No inline BlockedBy append loops exist in create.go or update.go
- Notes: update.go has only one parseCommaSeparatedIDs call because the spec says blocked_by cannot be changed via update (only --blocks is available). This is correct.

TESTS:
- Status: Adequate
- Coverage:
  - TestParseCommaSeparatedIDs (helpers_test.go:13-71): 7 subtests covering single ID, multiple IDs, whitespace, empty string, only commas/whitespace, lowercase normalization, trailing comma
  - TestApplyBlocks (helpers_test.go:73-201): 7 subtests covering basic append, Updated timestamp, non-existent blockIDs (no-op), duplicate prevention, case-insensitive blockID matching, case-insensitive existing dep detection, multiple blockIDs
- All task-specified tests are covered:
  - parseCommaSeparatedIDs with single ID: yes
  - parseCommaSeparatedIDs with multiple IDs: yes
  - parseCommaSeparatedIDs with whitespace: yes
  - parseCommaSeparatedIDs with empty strings: yes
  - parseCommaSeparatedIDs normalizes to lowercase: yes
  - applyBlocks appends sourceID to matching tasks' BlockedBy: yes
  - applyBlocks sets Updated timestamp on modified tasks: yes
  - applyBlocks with non-existent blockIDs (no-op): yes
- Additional tests for duplicate prevention and case-insensitive matching are valuable, not over-tested
- Notes: Tests use table-driven pattern for parseCommaSeparatedIDs and subtest pattern for applyBlocks, both idiomatic Go

CODE QUALITY:
- Project conventions: Followed -- table-driven tests, explicit error handling, no ignored errors
- SOLID principles: Good -- SRP (each helper does one thing), functions are small and focused
- Complexity: Low -- parseCommaSeparatedIDs is a simple loop; applyBlocks has O(n*m*k) nested loops (tasks * blockIDs * existing deps) but this is acceptable given expected task counts (hundreds)
- Modern idioms: Yes -- range over index for slice mutation, proper string handling
- Readability: Good -- clear function names, doc comments on both helpers, intent is obvious
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The duplicate-prevention logic in applyBlocks (lines 63-70) was also planned as a separate task tick-core-8-1. Having it already in the shared helper is fine but the Phase 8 task may be redundant now.
