TASK: Add blocked-ancestor EXISTS condition to BlockedConditions

ACCEPTANCE CRITERIA:
- BlockedConditions() includes the EXISTS inverse of the ancestor check in its OR clause
- Child of a dependency-blocked parent does not appear in ready results and does appear in blocked results
- Grandchild of a dependency-blocked grandparent (with clean intermediate parent) does not appear in ready results
- When the ancestor's blocker is resolved (done/cancelled), descendants become ready again
- Stats ready count matches list --ready output for mixed scenarios with blocked ancestors

STATUS: Complete

SPEC CONTEXT: The specification defines BlockedConditions() as the De Morgan inverse of ReadyConditions(). Specifically, blocked = open AND (has unclosed blocker OR has open children OR has dependency-blocked ancestor). The EXISTS inverse of ReadyNoBlockedAncestor() must be added to the OR clause. The spec also requires that stats.Blocked is derived as open - ready, ensuring consistency. The blocked filter is consumed by `list.go` (line 208) and indirectly by stats (via ready count subtraction on stats.go line 85).

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/query_helpers.go:60-80
- Notes: BlockedConditions() at line 70 composes from all three ReadyNo*() helpers via `negateNotExists()` (line 62-64), which strips the "NOT " prefix to convert NOT EXISTS to EXISTS. The three parts (unclosed blockers, open children, blocked ancestor) are joined with OR into a single disjunctive condition. This is clean, DRY, and matches the spec exactly. The `negateNotExists()` helper is a nice utility that avoids duplicating SQL between ready and blocked conditions. The implementation follows the spec's prescribed pattern: each concern is a separate helper, composed into conditions.

TESTS:
- Status: Adequate
- Coverage:
  - Unit tests in query_helpers_test.go:
    - "BlockedCondition returns open AND negation of ready subconditions" (line 55): verifies 2 conditions, status = open, non-empty OR clause
    - "BlockedConditions includes ancestor blocker in OR clause" (line 69): verifies the OR clause contains "ancestors" CTE reference
    - "BlockedConditions derives subqueries from ReadyNo helpers" (line 79): verifies all three ReadyNo*() helpers are negated and present in the OR clause -- this is the key structural test ensuring DRY composition
    - "BlockedConditions contains no SQL literals beyond status check" (line 109): counts exactly 3 EXISTS occurrences, catching hand-written SQL
  - Integration tests in blocked_test.go:
    - "it returns child of dependency-blocked parent in blocked" (line 311): child of blocked parent appears in blocked output
    - "it returns grandchild of dependency-blocked grandparent in blocked" (line 332): grandchild with clean intermediate parent appears in blocked
    - "it returns descendant behind intermediate grouping task under blocked ancestor in blocked" (line 355): tests the grouping-task gap scenario
    - "it excludes descendant from blocked when ancestor blocker resolved" (line 378): verifies done blocker removes descendants from blocked
    - "it maintains stats count consistency with blocked ancestors" (line 400): verifies stats.ready matches list --ready count AND ready + blocked = open
  - Edge cases from task specification:
    - Grandchild appears in blocked: covered (line 332)
    - Blocker resolution removes from blocked: covered (line 378)
    - Stats count consistency: covered (line 400)
- Notes: Test coverage is thorough and well-balanced. The unit tests verify structural composition (not just "it works") which catches regressions if someone rewrites BlockedConditions() with hand-crafted SQL. The integration tests exercise real SQL execution through the full app stack. The stats consistency test (line 400) is particularly valuable -- it verifies both that stats.ready matches list --ready count and that ready + blocked = open, catching any asymmetry between the ready and blocked conditions.

CODE QUALITY:
- Project conventions: Followed. stdlib testing only, t.Run() subtests, t.Helper() on helpers, fmt.Errorf with %w wrapping pattern (not directly applicable here but consistent), DI via struct fields in test helpers.
- SOLID principles: Good. Single responsibility: negateNotExists() does one thing; BlockedConditions() composes helpers rather than duplicating logic. Open/closed: adding a new ReadyNo*() helper automatically propagates to both ReadyConditions() and BlockedConditions() if added to both. The composition pattern means the blocked condition is always the exact inverse of the ready condition.
- Complexity: Low. BlockedConditions() is 9 lines. negateNotExists() is 2 lines. No conditionals, no loops, linear flow.
- Modern idioms: Yes. strings.TrimPrefix for safe prefix removal, slice composition, clean function composition.
- Readability: Good. The function names are self-documenting (negateNotExists, BlockedConditions). The doc comment on BlockedConditions() explains the De Morgan relationship. The composition pattern makes it immediately clear that blocked = NOT ready.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The stats consistency test at blocked_test.go:400 uses `runReady` and `runStats` which are defined in separate test files. This cross-file dependency is fine for Go (same package), but worth noting that the test relies on helpers from ready_test.go and stats_test.go.
