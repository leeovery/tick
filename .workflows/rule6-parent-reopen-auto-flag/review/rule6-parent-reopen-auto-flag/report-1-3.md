TASK: Integration tests for auto flag in JSONL

ACCEPTANCE CRITERIA:
- Integration test confirms create --parent <done-parent> produces auto=true on parent reopen transition in JSONL
- Integration test confirms update --parent reparent triggers auto-completion with auto=true in JSONL
- All tests read JSONL directly via readPersistedTasks (not CLI output) to verify the source of truth
- go test ./... passes with zero failures

STATUS: Complete

SPEC CONTEXT: The spec identifies two system-initiated callers of ApplyWithCascades that incorrectly recorded auto=false: (1) validateAndReopenParent (Rule 6: done parent reopened when child added) and (2) evaluateRule3/autoCompleteParentIfTerminal (Rule 3: parent auto-completed when remaining children all terminal after reparent). Integration tests should verify that after the fix, auto=true flows through the full stack to JSONL for both scenarios.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/create_test.go:1332-1369 (Rule 6 create integration test), internal/cli/update_test.go:1278-1317 (Rule 3 reparent integration test)
- Notes: Both tests are well-structured integration tests that exercise the full CLI pipeline (App.Run -> command handler -> Store.Mutate -> JSONL) and verify the auto flag on the source of truth.

TESTS:
- Status: Adequate
- Coverage:
  - create_test.go:1332 "it records auto=true on parent reopen when creating child under done parent": Sets up a done parent, creates a child under it via CLI, reads JSONL via readPersistedTasks, asserts parent status=open, transition from=done/to=open, auto=true. Correctly exercises Rule 6 end-to-end.
  - update_test.go:1278 "it records auto=true on original parent auto-completion when reparenting away": Sets up in_progress parent with one done child and one open child, reparents the open child to a new parent via CLI, reads JSONL via readPersistedTasks, asserts original parent status=done, transition from=in_progress/to=done, auto=true. Correctly exercises Rule 3 end-to-end.
  - Both tests use readPersistedTasks (storage.ReadJSONL) to read JSONL directly, not CLI output, satisfying the source-of-truth requirement.
  - Both tests would fail if the feature regressed (if ApplySystemTransition were reverted to ApplyUserTransition or if auto were set to false).
  - No over-testing: each integration test makes focused assertions (status, transition count, from/to/auto). No redundant checks.
- Notes: None

CODE QUALITY:
- Project conventions: Followed. Tests use stdlib testing, t.Run subtests, t.Helper on helpers, t.TempDir for isolation, "it does X" naming convention, readPersistedTasks helper pattern consistent with other tests in the file.
- SOLID principles: Good. Tests are focused on a single scenario each (SRP). Test setup is minimal and specific to the scenario being tested.
- Complexity: Low. Straightforward test setup, execution, and assertion flow.
- Modern idioms: Yes. Uses standard Go test patterns, time.Now().UTC().Truncate(time.Second) for consistent time comparison, pointer for Closed field.
- Readability: Good. Test names clearly describe what is being verified. Setup tasks have descriptive titles ("Done parent", "Child done", "Child moving", "New parent"). Assertions include helpful error messages with context.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The for-loop pattern to find a task by ID in persisted tasks (lines 1348-1353 in create_test.go, 1296-1301 in update_test.go) is repeated across integration tests. A findTaskByID helper could reduce boilerplate, but this is a pre-existing pattern not introduced by this task.
