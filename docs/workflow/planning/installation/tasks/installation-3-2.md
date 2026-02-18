---
id: installation-3-2
phase: 3
status: completed
created: 2026-02-14
---

# Extract step-search helper in release_test.go

**Problem**: Four test cases in .github/workflows/release_test.go (lines 210-285) repeat the same nested loop pattern (iterate jobs, iterate steps, check strings.Contains on Uses field) with minor field variations. Each block is ~10 lines of near-identical traversal code.

**Solution**: Extract a local findStepByUses helper within release_test.go that encapsulates the nested loop search pattern.

**Outcome**: Each test case calls the helper in 2-3 lines instead of repeating 10 lines of loop code. Future step-search tests follow the same concise pattern.

**Do**:
1. Add a `findStepByUses(w workflow, usesSubstring string) (step, bool)` helper function in release_test.go (unexported, file-local)
2. Replace each of the 4 nested loop blocks with a call to findStepByUses
3. Each test case checks the returned step's specific fields (With, Env, etc.)
4. Run release_test.go tests to verify no regressions

**Acceptance Criteria**:
- A single findStepByUses helper exists in release_test.go
- The 4 test cases that previously had inline loops now call the helper
- No change to what is being asserted, only how the step is located

**Tests**:
- All existing tests in .github/workflows/release_test.go pass unchanged
