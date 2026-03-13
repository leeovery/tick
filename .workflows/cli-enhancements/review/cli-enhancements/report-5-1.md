TASK: cli-enhancements-6-2 -- Extract query-scan helpers in show.go

ACCEPTANCE CRITERIA:
- All four query-scan blocks in queryShowData use the extracted helpers
- tick show output is unchanged for all cases (deps, children, tags, refs)
- No new exported symbols introduced (helpers are unexported)

STATUS: Complete

SPEC CONTEXT: This is an Analysis Cycle 2 task originating from the duplication agent's finding that queryShowData contained four structurally identical query-scan blocks (~76 lines): blocked_by, children, tags, and refs. Each followed the same pattern (query, iterate rows, scan, append, check rows.Err()) with only the SQL, scan target type, and error prefix differing. The prescribed solution was to extract two helpers: queryStringColumn for tags/refs and queryRelatedTasks for blocked_by/children.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/show.go:170-187` -- `queryStringColumn` helper function
  - `/Users/leeovery/Code/tick/internal/cli/show.go:189-206` -- `queryRelatedTasks` helper function
  - `/Users/leeovery/Code/tick/internal/cli/show.go:108-113` -- blockedBy call site using `queryRelatedTasks`
  - `/Users/leeovery/Code/tick/internal/cli/show.go:117-122` -- children call site using `queryRelatedTasks`
  - `/Users/leeovery/Code/tick/internal/cli/show.go:126-131` -- tags call site using `queryStringColumn`
  - `/Users/leeovery/Code/tick/internal/cli/show.go:135-140` -- refs call site using `queryStringColumn`
- Notes: Implementation matches the prescribed solution exactly. Both helper functions are unexported. The four inline blocks (~76 lines of boilerplate) have been replaced with two helpers (~30 lines) plus four concise call sites. Error wrapping with contextual prefixes ("failed to query dependencies:", etc.) is done at each call site. The notes scanning block (lines 144-164) was correctly left inline since it has a different structure (timestamp parsing, time.Parse conversion) that does not fit the extracted patterns.

TESTS:
- Status: Adequate
- Coverage: This is a pure refactoring task with no behavioral change. The analysis task explicitly states "Existing show tests pass without modification (no behavioral change)." The existing test coverage in `/Users/leeovery/Code/tick/internal/cli/list_show_test.go` includes:
  - "queryShowData populates RelatedTask fields for blockers and children" (line 533) -- directly exercises `queryRelatedTasks` via `queryShowData`
  - Show tests for tags display (lines 507-531)
  - Show tests for refs display (lines 589-611)
  - Show tests for notes display
  - Show tests for dependencies, children, and parent context
- Notes: No new tests needed for a behavioral-preserving refactoring. The existing tests adequately verify that the extracted helpers produce identical results. The `queryShowData` unit test at line 533 is particularly valuable as it directly validates the RelatedTask struct population path through the new `queryRelatedTasks` helper.

CODE QUALITY:
- Project conventions: Followed -- unexported helpers with doc comments, error wrapping with `fmt.Errorf("context: %w", err)`, consistent with existing patterns in the codebase
- SOLID principles: Good -- Single Responsibility (each helper does one thing: execute a query and scan rows into a typed slice), DRY (eliminated ~46 lines of duplicated boilerplate)
- Complexity: Low -- both helpers are linear, straightforward query-scan-append loops
- Modern idioms: Yes -- proper use of `defer rows.Close()`, `rows.Err()` check, nil slice returns
- Readability: Good -- helper names clearly communicate purpose (queryStringColumn, queryRelatedTasks); queryShowData is now significantly more readable with the boilerplate extracted
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `queryRelatedTasks` helper is also indirectly reused via `queryShowData` from `outputMutationResult` in `/Users/leeovery/Code/tick/internal/cli/helpers.go:22`, demonstrating good reuse value from this extraction.
