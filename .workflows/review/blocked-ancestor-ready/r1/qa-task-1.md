TASK: Add ReadyNoBlockedAncestor helper and integrate into ReadyConditions

ACCEPTANCE CRITERIA:
- ReadyNoBlockedAncestor() helper returns a NOT EXISTS subquery with recursive CTE that walks the full ancestor chain checking for unclosed dependency blockers
- ReadyConditions() includes the ancestor check as the 4th condition
- Child of a dependency-blocked parent does not appear in ready results and does appear in blocked results
- Grandchild of a dependency-blocked grandparent (with clean intermediate parent) does not appear in ready results
- Descendant behind an intermediate grouping task (no own blockers) under a blocked ancestor is correctly excluded from ready
- When the ancestor's blocker is resolved (done/cancelled), descendants become ready again
- Root tasks with no parent remain unaffected by the ancestor check
- Stats ready count matches list --ready output for mixed scenarios with blocked ancestors

STATUS: Complete

SPEC CONTEXT: The specification identifies that children of blocked parents incorrectly appear as "ready" because the current 3-condition ready check (open status, no own unclosed blockers, no open children) never checks whether any ancestor is dependency-blocked. The fix requires a recursive CTE walking the full ancestor chain (not just immediate parent) because intermediate grouping tasks create gaps. Only dependency blockers propagate -- children-blocked state does not propagate since leaf tasks would never be ready otherwise.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/query_helpers.go:28-46 (ReadyNoBlockedAncestor), :51-58 (ReadyConditions), :62-80 (BlockedConditions)
- Notes:
  - ReadyNoBlockedAncestor() returns a NOT EXISTS subquery with a recursive CTE that walks parent pointers to root. The CTE collects all ancestor IDs, then the outer query checks if any ancestor has an unclosed dependency blocker. This matches the spec's SQL exactly.
  - ReadyConditions() returns 4 conditions: open status, no unclosed blockers, no open children, no blocked ancestor. The ancestor check is the 4th condition as specified.
  - BlockedConditions() uses negateNotExists() to derive EXISTS versions from all three ReadyNo*() helpers, composing them into an OR clause. This is the De Morgan inverse pattern from the spec.
  - Integration points: list.go:203-204 uses ReadyConditions() for --ready filter; list.go:207-208 uses BlockedConditions() for --blocked filter; stats.go:79 uses ReadyWhereClause() for ready count. All consumers automatically pick up the new ancestor check.
  - No drift from the plan or specification.

TESTS:
- Status: Adequate
- Coverage:
  - Unit tests in query_helpers_test.go:
    - Verifies ReadyNoBlockedAncestor() returns non-empty string with NOT EXISTS and WITH RECURSIVE (:23-34)
    - Verifies ReadyConditions() returns exactly 4 conditions in correct order (:36-53)
    - Verifies BlockedConditions() includes ancestor blocker in OR clause (:69-77)
    - Verifies BlockedConditions() derives subqueries from ReadyNo*() helpers via negation (:79-107)
    - Verifies BlockedConditions() contains exactly 3 EXISTS occurrences (no hand-written SQL) (:109-129)
  - Integration tests in ready_test.go:
    - Child of dependency-blocked parent excluded from ready (:403-426) -- covers AC #3
    - Grandchild of dependency-blocked grandparent excluded (:428-452) -- covers AC #4
    - Descendant behind intermediate grouping task excluded (:454-478) -- covers AC #5
    - Descendant included when ancestor blocker resolved (done) (:480-500) -- covers AC #6
    - Root task with no parent unaffected (:502-517) -- covers AC #7
  - Integration tests in blocked_test.go:
    - Child of dependency-blocked parent appears in blocked (:311-330) -- covers AC #3 blocked side
    - Grandchild of blocked grandparent appears in blocked (:332-353) -- covers AC #4 blocked side
    - Descendant behind intermediate grouping task appears in blocked (:355-376) -- covers AC #5 blocked side
    - Descendant excluded from blocked when blocker resolved (:378-398) -- covers AC #6 blocked side
    - Stats count consistency test (:400-451) -- covers AC #8
  - Tests would fail if the feature broke (removing the ancestor check would cause descendants of blocked parents to appear in ready and not in blocked).
- Notes: Test coverage is thorough without being redundant. Each test targets a distinct scenario from the spec. The stats consistency test verifies ready + blocked = open, which is a good cross-cutting check. No over-testing detected.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run() subtests, t.Helper() on helpers, fmt.Errorf with %w wrapping, functional composition pattern for SQL helpers.
- SOLID principles: Good. Single responsibility: each ReadyNo*() helper handles exactly one concern. Open/closed: new conditions are added by composing new helpers, not modifying existing SQL strings. DI pattern maintained through App struct.
- Complexity: Low. The helper functions are pure string returns with no branching. The recursive CTE is the natural pattern for ancestor traversal in SQL. negateNotExists() is a simple string transformation.
- Modern idioms: Yes. Clean Go with appropriate use of string building. No unnecessary abstraction.
- Readability: Good. Each helper has a clear doc comment explaining its purpose and assumption (outer query aliases tasks as "t"). The SQL is well-formatted with consistent indentation. The negateNotExists helper is self-explanatory and keeps BlockedConditions clean.
- Security: No SQL injection risk -- all SQL is static string composition with no user input interpolation.
- Performance: The recursive CTE walks ancestor chains which are typically shallow (2-4 levels as noted in spec). This is a subquery per task row, but for the expected dataset sizes of a CLI task tool, this is a non-issue. The CTE pattern is already established in the codebase (queryDescendantIDs in list.go).
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The cancelled-blocker variant for ancestor resolution is not explicitly tested (only done is tested in the "ancestor blocker resolved" ready_test.go scenario). The existing test for direct blocker cancellation covers the cancelled path at the dependency-check level, and the ancestor CTE reuses the same `status NOT IN ('done', 'cancelled')` clause, so the risk is very low. This is a minor gap, not a blocking issue.
