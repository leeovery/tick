AGENT: duplication
FINDINGS:
- FINDING: Duplicate dep tree dispatch smoke test across test files
  SEVERITY: medium
  FILES: internal/cli/note_test.go:569-582, internal/cli/dep_tree_test.go:52-64
  DESCRIPTION: TestNoteTreeRejection/"it preserves dep tree dispatch" (note_test.go) is functionally identical to TestDepTreeWiring/"it dispatches dep tree without error" (dep_tree_test.go). Both set up a tick project, run `tick --pretty dep tree`, and assert exit code 0. These were written by separate task executors (task 1-1 added dep_tree_test.go tests, task 1-2 added note_test.go tests) and independently verify the same behavior.
  RECOMMENDATION: Remove "it preserves dep tree dispatch" from TestNoteTreeRejection in note_test.go. The canonical test for dep tree dispatch already lives in dep_tree_test.go where it belongs. The note_test.go regression suite should only verify note-specific behavior (the tree rejection tests and flag error test are sufficient).

- FINDING: Duplicate dep tree flag validation test across test files
  SEVERITY: medium
  FILES: internal/cli/note_test.go:584-593, internal/cli/dep_tree_test.go:34-43
  DESCRIPTION: TestNoteTreeRejection/"it preserves dep tree flag validation" (note_test.go) tests `ValidateFlags("dep tree", []string{"--unknown"}, commandFlags)` and asserts the error references "dep tree". TestDepTreeWiring/"it rejects unknown flag on dep tree" (dep_tree_test.go) tests the exact same call with the same unknown flag and asserts the same error. Both were independently authored by different task executors.
  RECOMMENDATION: Remove "it preserves dep tree flag validation" from TestNoteTreeRejection in note_test.go. dep_tree_test.go already covers flag validation for the "dep tree" command. The note-side regression only needs to verify that `tick note tree --foo` does NOT produce a "note tree" error (which "it does not reference note tree in flag error" already covers).

SUMMARY: Two tests in note_test.go (added by the regression-test task) duplicate existing dep_tree_test.go coverage added by the fix task. Both verify dep tree dispatch and flag validation that already have canonical tests in dep_tree_test.go. Consolidating means removing the two redundant subtests from TestNoteTreeRejection.
