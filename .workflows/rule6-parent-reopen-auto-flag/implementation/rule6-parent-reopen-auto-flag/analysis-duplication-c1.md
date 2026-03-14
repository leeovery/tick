AGENT: duplication
FINDINGS:
- FINDING: Repeated transition record assertion block in apply_cascades_test.go
  SEVERITY: medium
  FILES: internal/task/apply_cascades_test.go:129-140, internal/task/apply_cascades_test.go:145-153, internal/task/apply_cascades_test.go:198-222, internal/task/apply_cascades_test.go:328-352, internal/task/apply_cascades_test.go:366-385, internal/task/apply_cascades_test.go:538-551, internal/task/apply_cascades_test.go:566-577
  DESCRIPTION: The same 4-assertion block (check Transitions length, check From, check To, check Auto) is repeated 10+ times across subtests with only the expected values and task index varying. Each instance is 8-10 lines of identical structure. This emerged from a single executor writing many subtests that each verify transition records.
  RECOMMENDATION: Extract a test helper like `assertTransition(t *testing.T, task Task, index int, from, to Status, auto bool)` in apply_cascades_test.go. This reduces each 8-10 line block to a single call while preserving assertion clarity.
- FINDING: Identical transition record assertion in create_test.go and update_test.go integration tests
  SEVERITY: low
  FILES: internal/cli/create_test.go:1357-1368, internal/cli/update_test.go:1305-1316
  DESCRIPTION: Both integration tests verify a transition record with the same 4-check pattern (Transitions length, From, To, Auto). These were written by separate executors for Rule 6 (create) and Rule 3 (update) scenarios. The pattern is identical modulo variable names and expected values.
  RECOMMENDATION: This is only 2 instances across separate test files, so it falls below the Rule of Three threshold. No action needed now, but if future tests add more transition-inspecting integration tests, extract a shared helper in a test_helpers_test.go file.
SUMMARY: The main duplication is the transition record assertion block repeated 10+ times in apply_cascades_test.go. A small test helper would eliminate significant boilerplate. The cross-file integration test duplication is only 2 instances and not yet worth extracting.
