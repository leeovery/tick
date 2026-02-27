TASK: Prevent duplicate blocked_by entries in applyBlocks

ACCEPTANCE CRITERIA:
- applyBlocks does not create duplicate entries in BlockedBy when called with a sourceID already present
- Existing non-duplicate behavior is unchanged
- The Updated timestamp is only modified when a new dependency is actually added

STATUS: Complete

SPEC CONTEXT: The spec defines `blocked_by` as an array of task IDs (implying uniqueness). `dep add` explicitly rejects duplicates with an error. The `--blocks` flag on `create` and `update` calls `applyBlocks` which previously lacked duplicate checking, allowing silent duplicates. This fix aligns `--blocks` with `dep add` semantics.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/helpers.go:59-77
- Notes: The `applyBlocks` function now iterates `tasks[i].BlockedBy` to check if `sourceID` is already present (using `task.NormalizeID` for case-insensitive comparison). If found, skips the append and does not update the timestamp. Only appends and sets `Updated` when the dependency is genuinely new. Implementation matches acceptance criteria exactly. No drift from plan.

TESTS:
- Status: Adequate
- Coverage:
  - Unit test "it skips duplicate when sourceID already in BlockedBy" (helpers_test.go:129-149): verifies slice length unchanged, no duplicate, Updated timestamp unchanged. Directly matches AC.
  - Unit test "it appends sourceID to matching tasks BlockedBy" (helpers_test.go:74-91): verifies new sourceID is appended. Matches AC for non-duplicate behavior.
  - Unit test "it sets Updated timestamp on modified tasks" (helpers_test.go:93-110): verifies Updated is set only when a new dep is added.
  - Unit test "it detects existing dep case-insensitively" (helpers_test.go:165-178): edge case -- existing dep stored as uppercase, sourceID lowercase; still detected as duplicate.
  - Unit test "it is a no-op with non-existent blockIDs" (helpers_test.go:112-127): edge case.
  - Unit test "it handles multiple blockIDs" (helpers_test.go:180-200): multi-target behavior.
  - Integration test "it does not duplicate blocked_by when --blocks with existing dependency" (update_test.go:522-554): end-to-end via `tick update` with `--blocks` where dep already exists; verifies persisted JSONL has no duplicates.
  - Missing: The task requested an integration test for `tick create "A" --blocks T1` where A already blocks T1, but this scenario is architecturally impossible -- a newly created task cannot have previously established blocks. The update integration test covers the real-world scenario. Not a gap.
- Notes: Tests are well-focused and not over-tested. Each test verifies a distinct aspect. Would fail if the feature broke.

CODE QUALITY:
- Project conventions: Followed. Table-driven subtests used where appropriate. Exported function documented with godoc comment.
- SOLID principles: Good. Single responsibility -- applyBlocks does one thing. No unnecessary coupling.
- Complexity: Low. Simple nested loop with early break on duplicate detection. Clear control flow.
- Modern idioms: Yes. Uses `range` with index for slice mutation, `NormalizeID` for case-insensitive comparison (consistent with rest of codebase).
- Readability: Good. Comment on function updated to reflect skip behavior. Variable name `alreadyPresent` is self-documenting.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The duplicate check is O(n) per BlockedBy slice, but since BlockedBy slices are expected to be very small (typically <10 items), this is fine. No performance concern.
- The `dep add` path returns an error on duplicates while `applyBlocks` silently skips. This behavioral difference is intentional per the task description ("silently skips") and makes sense since `--blocks` may target multiple tasks and a partial error would be more complex to handle.
