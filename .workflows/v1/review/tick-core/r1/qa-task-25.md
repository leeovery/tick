TASK: Add dependency validation to create and update --blocked-by/--blocks

ACCEPTANCE CRITERIA:
- `tick create --blocked-by <parent-id>` on a child task returns child-blocked-by-parent error
- `tick create --blocked-by` that would create a cycle returns cycle detection error
- `tick create --blocks <child-id>` on a parent task returns child-blocked-by-parent error
- `tick update --blocks <child-id>` on a parent task returns child-blocked-by-parent error
- `tick create --blocks` that would create a cycle returns cycle detection error
- No invalid dependency graphs can be persisted through any write path

STATUS: Complete

SPEC CONTEXT: Spec line 403 states: "Validate when adding/modifying blocked_by - reject invalid dependencies at write time, before persisting to JSONL." The two prohibited patterns are child-blocked-by-parent (creates deadlock with leaf-only ready rule) and circular dependencies. All write paths (create, update, dep add) must enforce these rules.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/create.go:153-163` -- validates `--blocked-by` via `task.ValidateDependencies()` and `--blocks` via `task.ValidateDependency()` within the Mutate callback, after new task is appended to the full task list
  - `/Users/leeovery/Code/tick/internal/cli/update.go:184-193` -- validates `--blocks` via `task.ValidateDependency()` within the Mutate callback, after `applyBlocks` modifies the in-memory task list
  - `/Users/leeovery/Code/tick/internal/task/dependency.go:12-32` -- `ValidateDependency()` and `ValidateDependencies()` implement cycle detection via DFS and child-blocked-by-parent check
- Notes:
  - In `create.go`, validation runs after `applyBlocks` has modified the in-memory task list and the new task has been appended, ensuring the full graph state is visible to the validator. Errors returned from the Mutate callback prevent persistence.
  - In `update.go`, same pattern: `applyBlocks` modifies in-memory state, then validation runs against the modified state. Errors prevent persistence.
  - The `update.go` `--blocked-by` path is correctly excluded from this task because spec says blocked_by cannot be changed via update (use `tick dep add/rm` instead).

TESTS:
- Status: Adequate
- Coverage:
  - `create_test.go:537-557` -- create `--blocked-by` child-blocked-by-parent rejection: verifies exit code 1, error mentions "parent", no new task persisted
  - `create_test.go:559-597` -- create `--blocked-by` cycle rejection via combined `--blocked-by` + `--blocks`: verifies exit code 1, error mentions "cycle", no new task persisted
  - `create_test.go:599-606` -- create `--blocks` child-blocked-by-parent: SKIPPED with valid reasoning (architecturally impossible since new task gets random ID that no existing child references as parent)
  - `create_test.go:608-640` -- create `--blocks` cycle rejection: 3-node cycle via `--blocked-by taskA --blocks taskB` where taskA blocked by taskB; verifies exit code 1, error mentions "cycle", no new tasks persisted
  - `create_test.go:780-827` -- valid dependencies through `--blocked-by` and `--blocks` work correctly
  - `update_test.go:486-520` -- update `--blocks` child-blocked-by-parent rejection: verifies exit code 1, error mentions "parent", child's blocked_by not modified
  - `update_test.go:556-594` -- update `--blocks` cycle rejection: 2-node cycle; verifies exit code 1, error mentions "cycle", target's blocked_by not modified
- Notes:
  - All 7 required test scenarios from the task spec are covered (one correctly skipped with justification).
  - Tests verify both the error condition AND the non-persistence of invalid state, which is thorough.
  - The cycle test for create `--blocked-by` (line 559) necessarily involves `--blocks` as well since a pure create `--blocked-by` cannot create a cycle (new task has no inbound edges from existing tasks). The test comment explains this clearly.

CODE QUALITY:
- Project conventions: Followed -- table-driven tests where applicable, Go idioms, proper error propagation
- SOLID principles: Good -- validation logic lives in `task.ValidateDependency()` (single responsibility), called from multiple write paths (DRY), the interface is clean with `ValidateDependency` for single and `ValidateDependencies` for batch
- Complexity: Low -- validation calls are straightforward, placed at the correct point in the Mutate callback after state is fully constructed
- Modern idioms: Yes -- idiomatic Go error handling, proper use of closures within Mutate
- Readability: Good -- clear comments explain the validation placement; the create.go code at lines 153-163 has a comment "Validate dependencies (cycle detection + child-blocked-by-parent) against full task list" that makes intent clear
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The skipped test at `create_test.go:599-606` is well-justified but could be documented more concisely. The current Skip message is adequate.
- In `create.go` the `applyBlocks` call at line 148 modifies tasks before the new task is appended at line 151, while validation at lines 153-163 runs after the append. This ordering is correct: `applyBlocks` modifies existing tasks' blocked_by, then the new task is added, then validation sees the full graph. The code flow is sound but could benefit from a brief comment explaining why this ordering matters.
