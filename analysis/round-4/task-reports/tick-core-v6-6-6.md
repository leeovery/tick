# Task tick-core-6-6: Add End-to-End Workflow Integration Test (V6 Only -- Analysis Phase 6)

## Note

This is an analysis refinement task that only exists in V6. Standalone quality assessment.

## Task Summary

The task adds a single comprehensive integration test (`TestWorkflowIntegration`) that exercises the primary agent workflow end-to-end: creating a parent/epic task with child tasks that have inter-dependencies, verifying the ready set at each step, transitioning tasks through their lifecycle, confirming unblocking behavior, and validating final stats. The goal is to catch seam issues between mutation and query paths that unit tests for individual commands would miss.

## V6 Implementation

### Architecture & Design

The test is placed in `internal/cli/workflow_integration_test.go` within the `cli` package, which is the appropriate location given that it exercises CLI-level commands (create, ready, start, done, stats) through the existing `App` runner infrastructure.

The test follows a clear 11-step sequential workflow:

1. Initialize project via `setupTickProject`
2. Create epic (parent task)
3. Create 4 child tasks with a dependency chain: A (free), B (blocked by A), C (blocked by B), D (blocked by B and C)
4. Verify initial ready set (only A)
5. Start A, verify B still blocked while A is in_progress, then complete A
6. Verify B unblocked after A done
7. Complete B, verify C unblocked
8. Complete C, verify D unblocked
9. Complete D, verify epic becomes ready (all children closed)
10. Complete epic, verify no tasks ready
11. Verify stats JSON reflects 5 tasks all done, 0 ready, 0 blocked

The dependency graph is well designed -- it exercises both linear chaining (A->B->C) and fan-in (B+C->D), covering the two most common dependency topologies. The test also verifies both positive (task IS ready) and negative (task is NOT ready) assertions at each step, which is critical for catching false-positive bugs.

Reuse of existing test helpers (`setupTickProject`, `runCreate`, `runReady`, `runTransition`, `runStats`) is excellent -- no infrastructure duplication.

### Code Quality

**Strengths:**

- Clear step-by-step comments (Steps 1-11) make the test narrative easy to follow
- Descriptive assertion messages include both the expectation and the reason (e.g., `"child B should NOT be ready (A still in_progress)"`)
- Consistent use of `t.Fatalf` for setup/precondition failures and `t.Errorf` for assertion failures -- the test stops early on infrastructure failures but continues collecting assertion failures
- The `parseQuietIDs`, `assertContains`, and `assertNotContains` helpers are clean, focused, and properly use `t.Helper()` for correct line reporting
- All error paths capture and report both stderr and exit code

**Minor observations:**

- `parseQuietIDs` uses `strings.Split` on newline, which is correct but relies on `--quiet` output ending with a trailing newline. This coupling is acceptable since the test validates behavior through the CLI contract.
- The `assertContains`/`assertNotContains` helpers are defined in this file rather than in the existing `helpers_test.go`. This is defensible since they are simple and specific to ID-list checking, but could arguably live alongside other shared test helpers.
- The stats verification at lines 174-197 uses raw `map[string]interface{}` with `float64` comparisons (JSON number default). This is standard Go JSON unmarshalling pattern but is slightly verbose. A struct-based unmarshal would be more type-safe, though the current approach is perfectly functional for a test.
- The `--quiet` flag is used consistently for ready queries to get machine-parseable ID-only output, which is the right approach for programmatic assertions.

### Test Coverage

The test directly satisfies all acceptance criteria from the task plan:

| Acceptance Criterion | Covered |
|---|---|
| Exercises create, ready, start, done, and stats across multi-task hierarchy with dependencies | Yes -- all 5 commands used |
| Verifies correct ready set at multiple points in the workflow | Yes -- 7 separate ready checks (steps 4, 5a, 6, 7, 8, 9, 10) |
| Verifies unblocking behavior when a dependency is completed | Yes -- steps 6, 7, 8, 9 each verify a specific unblocking |
| Verifies parent appears in ready after all children are closed | Yes -- step 9 |
| Test passes reliably | Yes -- no time-dependent logic, no goroutines, deterministic |

**Edge cases tested:**
- In-progress blocker does NOT unblock dependents (step 5 -- important distinction)
- Fan-in dependency: D requires both B and C done (step 8 verifies D still blocked when only B done)
- Epic/parent with open children is never ready (checked at steps 4, 6, 7, 8)
- Empty ready set after all tasks done (step 10)
- Stats verification covers all status counts and workflow counts (step 11)

**Not tested (acceptable omissions for a single integration test):**
- `cancel` command (only `done` transitions used) -- covered by unit tests
- `reopen` command -- covered by unit tests
- `update` command -- covered by unit tests
- `--blocks` flag on create (only `--blocked-by` used) -- covered by unit tests
- Multiple independent tasks at same level (only chain/fan-in topology)

### Spec Compliance

The test exercises exactly the "primary agent workflow" described in the task plan: init -> create tasks with dependencies/hierarchy -> ready -> transition -> ready -> complete all -> stats. The sequence matches the acceptance criteria point-for-point.

The test uses the CLI-level `App.Run` mechanism (via helper wrappers), meaning it exercises the full stack from argument parsing through store operations to formatted output. This is a true integration test, not a mock-based approximation.

### golang-pro Skill Compliance

| Requirement | Status |
|---|---|
| Table-driven tests with subtests | N/A -- single sequential workflow test, not amenable to table-driven structure |
| Document exported functions | Yes -- `parseQuietIDs`, `assertContains`, `assertNotContains` all have doc comments |
| Handle all errors explicitly | Yes -- every command invocation checks exit code and fails with stderr context |
| `t.Helper()` on test helpers | Yes -- both `assertContains` and `assertNotContains` use `t.Helper()` |
| No ignored errors | Yes -- no `_` assignments without justification (stdout discarded only when not needed) |

The `parseQuietIDs` function, while unexported (lowercase `p`), has a doc comment, which is good practice even though not strictly required by the skill for unexported functions.

## Quality Assessment

### Strengths

1. **Comprehensive workflow coverage**: The test exercises the exact multi-step workflow that individual unit tests cannot -- the interaction between create, ready queries, state transitions, and stats across a realistic task hierarchy with dependencies.
2. **Both positive and negative assertions**: At every checkpoint, the test asserts which tasks SHOULD and SHOULD NOT be in the ready set. This catches both false negatives and false positives.
3. **Clean infrastructure reuse**: Leverages existing test helpers (`setupTickProject`, `runCreate`, `runReady`, `runTransition`, `runStats`) without duplication.
4. **Well-designed dependency graph**: The A->B->C chain plus B+C->D fan-in covers the two primary dependency patterns in a compact 4-task setup.
5. **In-progress blocker check**: Step 5 explicitly verifies that an in-progress blocker does not unblock dependents, catching a subtle semantic difference between "done" and "started".
6. **Deterministic and reliable**: No timing dependencies, no concurrency, no external resources. The test will pass consistently.

### Weaknesses

1. **No `cancel` path coverage**: The test only uses `done` transitions. Adding one cancelled child (e.g., cancel child D instead of completing it) would also verify that cancellation unblocks the parent, increasing workflow variety with minimal code.
2. **Stats verification is somewhat fragile**: Using `map[string]interface{}` with float64 comparisons couples the test to JSON unmarshalling internals. A typed struct would be slightly more robust.
3. **Helper placement**: `parseQuietIDs`, `assertContains`, `assertNotContains` could be moved to `helpers_test.go` to be shared with other test files if similar ID-list assertions are needed elsewhere.
4. **Single monolithic test function**: At 200 lines, the test is long. While this is inherent to integration testing (steps are sequential and stateful), the lack of subtests means a failure at step 9 still requires reading through all earlier steps to understand context. Named sub-sections via comments help but `t.Run` would improve output.

### Overall Quality Rating

**Excellent**

The test precisely fulfills its stated purpose: a single comprehensive integration test that exercises the primary agent workflow end-to-end and catches seam issues between mutation and query paths. The dependency graph design is thoughtful, the assertion strategy is thorough (both presence and absence checks), and the code is clean, well-commented, and deterministic. The minor weaknesses (no cancel path, monolithic structure) are acceptable trade-offs for a focused integration test that complements the existing extensive unit test suite.
