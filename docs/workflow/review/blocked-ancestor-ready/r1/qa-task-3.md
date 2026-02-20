TASK: Compose BlockedConditions() from ReadyNo*() helpers instead of duplicating SQL

ACCEPTANCE CRITERIA:
- BlockedConditions() composes from ReadyNo*() helpers (ReadyNoUnclosedBlockers, ReadyNoOpenChildren, ReadyNoBlockedAncestor) instead of duplicating SQL
- SQL duplication between ReadyConditions() and BlockedConditions() is eliminated

STATUS: Complete

SPEC CONTEXT: The specification defines BlockedConditions() as the De Morgan inverse of ReadyConditions(): open status AND (has unclosed blocker OR has open children OR has dependency-blocked ancestor). The analysis cycle 1 duplication report identified that BlockedConditions() contained three independently-written EXISTS subqueries whose SQL bodies were character-for-character identical to the NOT EXISTS subqueries returned by the ReadyNo*() helpers -- roughly 25 lines of duplicated SQL that could drift independently.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/query_helpers.go:62-80
- Notes: The refactoring introduces a `negateNotExists()` helper (line 62-64) that converts "NOT EXISTS (...)" to "EXISTS (...)" by stripping the leading "NOT " prefix via `strings.TrimPrefix`. `BlockedConditions()` (lines 70-80) then composes its OR clause by calling `negateNotExists()` on each of the three `ReadyNo*()` helpers. This is a clean mechanical derivation -- the blocked conditions are guaranteed to stay in sync with the ready conditions since they share the exact same SQL bodies. The approach matches the analysis report's recommendation precisely. No SQL literals appear in `BlockedConditions()` beyond the `t.status = 'open'` check.

TESTS:
- Status: Adequate
- Coverage:
  - "BlockedCondition returns open AND negation of ready subconditions" (query_helpers_test.go:55-67): verifies structure (2 conditions, status check)
  - "BlockedConditions includes ancestor blocker in OR clause" (query_helpers_test.go:69-77): verifies the ancestor CTE is present
  - "BlockedConditions derives subqueries from ReadyNo helpers" (query_helpers_test.go:79-107): key test -- iterates all three ReadyNo*() helpers, strips "NOT " prefix, and asserts the resulting "EXISTS (...)" substring appears in the OR clause. This directly validates composition rather than duplication.
  - "BlockedConditions contains no SQL literals beyond status check" (query_helpers_test.go:109-129): counts exactly 3 EXISTS occurrences in the OR clause, catching any hand-written SQL additions.
  - Integration tests in blocked_test.go cover the behavioral correctness (ancestor-blocked tasks appear in blocked, resolved blockers remove from blocked, stats consistency).
- Notes: The test suite is well-balanced. The "derives subqueries from ReadyNo helpers" test is the most important -- it would fail if someone replaced the composition with hand-written SQL that doesn't match the helpers character-for-character. The "contains no SQL literals" test provides an additional guard against accidental additions. No over-testing detected; each test verifies a distinct property.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, helper pattern consistent with codebase.
- SOLID principles: Good. negateNotExists() has a single responsibility (string prefix removal). BlockedConditions() delegates to the same helpers as ReadyConditions(), satisfying open/closed -- adding a new ready condition automatically propagates to blocked if the same pattern is followed.
- Complexity: Low. negateNotExists is a one-liner. BlockedConditions is 10 lines with clear intent.
- Modern idioms: Yes. strings.TrimPrefix is the correct idiomatic choice over manual string slicing.
- Readability: Good. The function comment on BlockedConditions (lines 66-69) explicitly states "De Morgan inverse" and "derived from the ReadyNo*() helpers", making the design intent clear. The negateNotExists name is self-documenting.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The negateNotExists() helper is unexported and tightly coupled to the "NOT EXISTS" prefix convention of the ReadyNo*() helpers. This is appropriate -- it's an internal implementation detail. If a future helper returned a condition that doesn't start with "NOT EXISTS" (e.g., a plain boolean expression), it would silently pass through TrimPrefix unchanged. The test "derives subqueries from ReadyNo helpers" guards against this scenario by verifying the "NOT " prefix is actually stripped.
