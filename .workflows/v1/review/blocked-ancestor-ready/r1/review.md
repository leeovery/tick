# Implementation Review: Blocked Ancestor Ready

**Plan**: blocked-ancestor-ready
**QA Verdict**: Approve

## Summary

Clean, well-executed implementation. All 3 tasks across 2 phases are complete with no blocking issues. The recursive CTE ancestor check matches the specification exactly, integrates seamlessly into the existing ReadyConditions/BlockedConditions composition pattern, and is thoroughly tested at both unit and integration levels. The analysis-driven refactoring (Phase 2) eliminated ~25 lines of SQL duplication via a neat `negateNotExists()` helper.

## QA Verification

### Specification Compliance

Implementation aligns with specification exactly:
- `ReadyNoBlockedAncestor()` returns NOT EXISTS with recursive CTE walking full ancestor chain
- Only dependency blockers propagate (not children-blocked state), per spec decision
- CTE walks unconditionally to root (doesn't stop at closed ancestors), per spec edge case handling
- All integration points (`list --ready`, `list --blocked`, `tick ready`, stats) automatically pick up the change via composition

### Plan Completion
- [x] Phase 1 acceptance criteria met (all 8 criteria verified)
- [x] Phase 2 acceptance criteria met (BlockedConditions composed from helpers, no SQL duplication)
- [x] All 3 tasks completed
- [x] No scope creep — changes confined to `query_helpers.go` and test files as planned

### Code Quality

No issues found. Highlights:
- **DRY**: `negateNotExists()` ensures blocked conditions are mechanically derived from ready conditions — impossible to drift
- **Low complexity**: Each helper is a pure string return with no branching
- **Readability**: Clear doc comments, self-documenting function names, well-formatted SQL
- **Security**: All SQL is static string composition, no injection risk
- **Performance**: Recursive CTE on shallow ancestor chains (2-4 levels typical) is a non-issue for CLI task tool dataset sizes

### Test Quality

Tests adequately verify requirements:
- **Unit tests** (query_helpers_test.go): Verify structural properties — condition count, composition from helpers, no hand-written SQL leaking in
- **Integration tests** (ready_test.go, blocked_test.go): Cover all 6 spec scenarios through real SQL execution
- **Stats consistency**: Cross-cutting test verifies `ready + blocked = open`
- **Balance**: No over-testing detected. Each test targets a distinct scenario

### Required Changes

None.

## Recommendations

1. **Minor gap**: The cancelled-blocker variant for ancestor resolution is only tested indirectly (the `done` variant is tested explicitly in ready_test.go:480). The `status NOT IN ('done', 'cancelled')` clause covers both, and cancellation is tested at the direct-blocker level elsewhere, so risk is very low. Consider adding an explicit cancelled-ancestor test if the test file is revisited for other reasons.

2. **Cross-file test helpers**: The stats consistency test in blocked_test.go relies on `runReady` and `runStats` defined in other test files (same package). This is idiomatic Go but worth noting for discoverability.
