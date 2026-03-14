TASK: Integration tests for auto flag in JSONL

ACCEPTANCE CRITERIA:
- Integration test confirms create --parent <done-parent> produces auto=true on parent reopen transition in JSONL
- Integration test confirms update --parent reparent triggers auto-completion with auto=true in JSONL
- All tests read JSONL directly via readPersistedTasks (not CLI output) to verify the source of truth
- go test ./... passes with zero failures

STATUS: Complete

SPEC CONTEXT: The spec identifies two system-initiated callers that previously hardcoded Auto: false incorrectly: (1) validateAndReopenParent (Rule 6: reopens done parent when child added), and (2) autoCompleteParentIfTerminal (Rule 3 via reparent: auto-completes parent when remaining children are terminal). Integration tests should verify auto=true on the primary target's TransitionRecord by reading JSONL directly.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/create_test.go:1332-1369, internal/cli/update_test.go:1278-1317
- Notes: Both integration tests are correctly placed as subtests within the existing TestCreate and TestUpdate test functions. They exercise the full CLI path (App.Run) and verify results from the JSONL source of truth via readPersistedTasks.

TESTS:
- Status: Adequate
- Coverage:
  - "it records auto=true on parent reopen when creating child under done parent" (create_test.go:1332): Sets up a done parent, creates a child under it via CLI, reads JSONL, asserts parent status is open, transition is from done->open, and auto=true. Covers Rule 6 end-to-end.
  - "it records auto=true on original parent auto-completion when reparenting away" (update_test.go:1278): Sets up an in-progress parent with two children (one done, one open), reparents the open child away via CLI, reads JSONL, asserts original parent auto-completes to done with auto=true. Covers Rule 3 via reparent end-to-end.
  - Both tests read JSONL via readPersistedTasks (storage.ReadJSONL), not CLI output -- satisfying the source-of-truth requirement.
  - Tests are complementary to existing tests: the pre-existing "it reopens done parent when adding child" test (create_test.go:1070) verifies the reopen behavior but NOT the auto flag; the pre-existing "it triggers Rule 3 on original parent when reparenting away" test (update_test.go:1031) verifies status change and CLI output but NOT the auto flag. No redundancy.
  - Tests would fail if the feature broke (auto flag reverted to false), since they assert auto=true explicitly.
- Notes: No over-testing detected. Each test is focused on the specific auto flag behavior that was the bugfix target.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, t.Helper on helpers, t.TempDir for isolation, "it does X" naming convention.
- SOLID principles: Good. Tests are focused on a single responsibility (verifying auto flag). Setup uses existing shared helpers (setupTickProjectWithTasks, readPersistedTasks, runCreate, runUpdate).
- Complexity: Low. Each test is a straightforward arrange-act-assert pattern.
- Modern idioms: Yes. Uses Go test conventions properly.
- Readability: Good. Test names clearly describe what they verify. Setup is concise and the assertions are well-labeled with descriptive error messages.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The task lookup pattern (for-loop over persisted to find by ID) is repeated across both new tests and the existing tests. A findTaskByID helper could reduce this boilerplate, but this is a pre-existing pattern throughout the test files and not introduced by this task.
