TASK: Add end-to-end workflow integration test

ACCEPTANCE CRITERIA:
- Test exercises create, ready, start, done, and stats across a multi-task hierarchy with dependencies
- Test verifies correct ready set at multiple points in the workflow
- Test verifies unblocking behavior when a dependency is completed
- Test verifies parent appears in ready after all children are closed
- Test passes reliably

STATUS: Complete

SPEC CONTEXT: The spec defines the primary workflow as init -> create tasks -> implementation loop (ready/start/done) -> project complete. Ready query logic requires: status open, all blocked_by tasks closed, no open children. The leaf-only rule means parents with open children never appear in ready; once all children close, the parent becomes a "leaf" and appears in ready. Cancelled tasks unblock dependents.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/workflow_integration_test.go:1-231
- Notes: Single comprehensive test function `TestWorkflowIntegration` that exercises the full agent workflow. Creates an epic with 4 child tasks in a dependency chain (A -> B -> C, D blocked by B+C), then walks through the entire lifecycle: creating tasks, verifying ready sets at each stage, transitioning through start/done, verifying unblocking, verifying parent becomes ready after all children close, and checking final stats. Uses existing test infrastructure (setupTickProject, runCreate, runReady, runTransition, runStats helpers).

TESTS:
- Status: Adequate
- Coverage: The task itself IS the test. It covers:
  - Create with hierarchy (--parent) and dependencies (--blocked-by): lines 17-50
  - Initial ready verification (only unblocked leaf ready): lines 52-66
  - In-progress task not ready: lines 69-81
  - Unblocking when dependency completed (A done -> B ready): lines 83-100
  - Chain unblocking (B done -> C ready, C done -> D ready): lines 102-131
  - Parent becomes ready when all children closed: lines 133-145
  - Complete epic and verify no tasks ready: lines 147-159
  - Stats verification (all 5 done, 0 open/in_progress/cancelled, 0 ready/blocked): lines 162-198
  - Both presence AND absence assertions at each ready checkpoint
- Notes: Well-structured test with clear step comments. The dependency graph (A -> B -> C, D blocked by B+C) tests both linear chains and multi-blocker scenarios. Helper functions (parseQuietIDs, assertContains, assertNotContains) are clean and appropriately scoped.

CODE QUALITY:
- Project conventions: Followed. Uses existing test infrastructure (App struct with Stdout/Stderr/Getwd/IsTTY). Uses t.Helper() in helper functions. Test file is in internal/cli package consistent with other command tests.
- SOLID principles: Good. Single test function with clear sequential steps. Helper functions have single responsibilities.
- Complexity: Low. Linear test flow, easy to follow step by step.
- Modern idioms: Yes. Uses t.Helper(), t.TempDir() (via setupTickProject), JSON unmarshaling for stats verification.
- Readability: Good. Clear step numbering comments (Steps 1-11), descriptive assertion messages, well-named variables (epicID, childA-D). The dependency structure is documented in comments (lines 38, 45).
- Issues: None significant.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Line 154: The final `runReady(t, dir)` call omits `--quiet`, checking for human-readable "No tasks found.\n" instead. This is a minor inconsistency with the rest of the test which uses `--quiet` + `parseQuietIDs`. Not a bug (the PrettyFormatter returns this for empty results), but slightly inconsistent in style.
- The test does not exercise `tick cancel` to verify cancel-unblocks-dependents behavior. This is a separate concern covered by other tests (blocked_test.go), but could strengthen this integration test. Not required by acceptance criteria.
- The test uses `--json` for stats (line 163) while the rest uses pretty format. This is pragmatic (JSON is easier to assert on) and not a problem.
