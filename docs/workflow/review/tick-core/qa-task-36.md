TASK: Consolidate duplicate relatedTask struct into RelatedTask (tick-core-7-5)

ACCEPTANCE CRITERIA:
- The unexported `relatedTask` struct no longer exists in show.go
- `queryShowData` populates `RelatedTask` directly
- `showDataToTaskDetail` no longer has field-by-field conversion loops
- All existing show and format tests pass unchanged

STATUS: Complete

SPEC CONTEXT: The `tick show` command displays detailed task information including blocked_by and children sections with related task context (ID, title, status). The spec requires related entities include context, not just IDs. This refactoring is internal -- it consolidates two structurally identical types without changing any external behavior.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/show.go:14-27` -- `showData` struct uses `[]RelatedTask` for both `blockedBy` and `children` fields
  - `/Users/leeovery/Code/tick/internal/cli/show.go:107-113` -- `queryShowData` scans dependency rows directly into `var r RelatedTask` using `&r.ID, &r.Title, &r.Status`
  - `/Users/leeovery/Code/tick/internal/cli/show.go:128-134` -- `queryShowData` scans child rows directly into `var r RelatedTask` using `&r.ID, &r.Title, &r.Status`
  - `/Users/leeovery/Code/tick/internal/cli/show.go:143-169` -- `showDataToTaskDetail` directly assigns `d.blockedBy` and `d.children` (lines 165-166) with no conversion loops
  - `/Users/leeovery/Code/tick/internal/cli/format.go:86-91` -- `RelatedTask` exported struct definition (single source of truth)
- Notes: No unexported `relatedTask` struct exists anywhere in `/Users/leeovery/Code/tick/internal/`. Grep confirms zero matches. All four acceptance criteria are satisfied.

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/internal/cli/list_show_test.go:461-515` -- Dedicated test "queryShowData populates RelatedTask fields for blockers and children" verifies that `queryShowData` correctly populates `RelatedTask.ID`, `RelatedTask.Title`, and `RelatedTask.Status` for both children and blockers
  - Existing formatter tests (toon, pretty, JSON) all use `RelatedTask` in their test data, providing integration-level coverage that the end-to-end formatting pipeline works with the consolidated type
  - Existing `RunShow` integration tests (e.g., "show with all fields", "show deterministic output") exercise the full path from query through formatting
- Notes: Test coverage is well-balanced -- one focused unit test for the refactored query function, plus existing integration tests that would catch regressions in the formatting pipeline. No over-testing.

CODE QUALITY:
- Project conventions: Followed -- table-driven test style used elsewhere; exported types documented with comments; error handling with `fmt.Errorf` wrapping
- SOLID principles: Good -- DRY improvement (eliminated duplicate type), single responsibility maintained (show.go handles queries, format.go defines types)
- Complexity: Low -- straightforward struct removal and direct assignment; reduced overall complexity by eliminating an unnecessary mapping layer
- Modern idioms: Yes -- idiomatic Go struct usage; direct field scanning aligns with standard `database/sql` patterns
- Readability: Good -- `showData` struct is clearer with `[]RelatedTask` (self-documenting type name); `showDataToTaskDetail` is significantly simpler at 26 lines vs the prior version with conversion loops
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
