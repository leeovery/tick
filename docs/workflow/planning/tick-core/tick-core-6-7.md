---
id: tick-core-6-7
phase: 6
status: pending
created: 2026-02-10
---

# Add explanatory second line to child-blocked-by-parent error

**Problem**: The spec (lines 407-408) defines the child-blocked-by-parent error as a two-line message: "Cannot add dependency - tick-child cannot be blocked by its parent tick-epic" followed by "(would create unworkable task due to leaf-only ready rule)". The implementation in `internal/task/dependency.go:36` only outputs the first line and uses lowercase "cannot" vs the spec's "Cannot".

**Solution**: Update the error message in dependency.go to include the second explanatory line and fix capitalization to match spec.

**Outcome**: Error message matches spec exactly, providing agents and users with the rationale for the constraint.

**Do**:
1. In `internal/task/dependency.go`, find the child-blocked-by-parent error return (around line 36)
2. Update the error message to match spec format: first line "Cannot add dependency - {child} cannot be blocked by its parent {parent}" with capital C
3. Add second line: "(would create unworkable task due to leaf-only ready rule)"
4. Update any tests that assert on the exact error message text

**Acceptance Criteria**:
- Error message matches spec lines 407-408 exactly (two lines, correct capitalization)
- Existing dependency validation tests pass with updated assertions
- The explanatory rationale line is present in the error output

**Tests**:
- Test that child-blocked-by-parent error includes both lines
- Test that error message uses "Cannot" (capital C)
- Test that error message includes the rationale about leaf-only ready rule
