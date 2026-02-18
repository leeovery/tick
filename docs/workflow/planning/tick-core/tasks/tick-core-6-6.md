---
id: tick-core-6-6
phase: 6
status: completed
created: 2026-02-10
---

# Add end-to-end workflow integration test

**Problem**: Tests cover individual commands well but no test exercises the full agent workflow: init -> create tasks with dependencies/hierarchy -> ready (verify correct tasks) -> transition -> ready (verify unblocking). The closest tests are parent_scope_test.go and individual command tests, but none chain multiple mutations and verify emergent behavior of the ready/blocked query logic across a realistic multi-step workflow.

**Solution**: Add one integration test that exercises the primary workflow end-to-end, catching seam issues between mutation and query paths.

**Outcome**: Confidence that the cross-command integration works correctly for the primary agent workflow described in the spec.

**Do**:
1. Create a test (in `internal/cli/` or an appropriate integration test location) that exercises this sequence:
   - Create a parent/epic task
   - Create child tasks with inter-dependencies (some blocked-by others)
   - Call `tick ready` and verify only the correct unblocked leaf tasks appear
   - Transition a blocker to done
   - Call `tick ready` and verify the previously-blocked task now appears
   - Transition remaining tasks
   - Verify the parent/epic task eventually appears in ready (all children closed)
   - Mark parent done
   - Verify `tick stats` reflects the final state correctly
2. Use the existing test infrastructure (tmpdir setup, Store creation, command runners)
3. Assert on both the correct presence AND absence of tasks in ready results at each step

**Acceptance Criteria**:
- Test exercises create, ready, start, done, and stats across a multi-task hierarchy with dependencies
- Test verifies correct ready set at multiple points in the workflow
- Test verifies unblocking behavior when a dependency is completed
- Test verifies parent appears in ready after all children are closed
- Test passes reliably

**Tests**:
- The task itself is a test -- one comprehensive integration test covering the full workflow
