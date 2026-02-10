---
id: tick-core-6-5
phase: 6
status: pending
created: 2026-02-10
---

# Extract shared helpers for --blocks application and ID parsing

**Problem**: Two patterns are duplicated between create.go and update.go. (1) The --blocks application loop (iterate tasks, match by blockID, append new task ID to BlockedBy, set Updated) is structurally identical in both files (~10 lines each). (2) Comma-separated ID parsing with normalize (`strings.Split` -> range -> `NormalizeID(TrimSpace(id))` -> append if non-empty) appears three times across two files.

**Solution**: Extract `applyBlocks(tasks []task.Task, sourceID string, blockIDs []string, now time.Time)` and `parseCommaSeparatedIDs(s string) []string` as shared helper functions. Both create.go and update.go call these instead of inlining the logic.

**Outcome**: Block application and ID parsing logic exist in one place. Changes to either pattern (e.g., adding validation) only need to be made once.

**Do**:
1. Create a helper function `parseCommaSeparatedIDs(s string) []string` in an appropriate shared file (e.g., `internal/cli/helpers.go` or similar)
2. The function splits on comma, trims whitespace, normalizes IDs, and filters empty values
3. Replace the three inline parsing instances in create.go and update.go with calls to this helper
4. Create a helper function `applyBlocks(tasks []task.Task, sourceID string, blockIDs []string, now time.Time)` in the same or appropriate file
5. The function iterates tasks, matches by blockID, appends sourceID to BlockedBy, sets Updated timestamp
6. Replace the inline --blocks loops in create.go and update.go with calls to this helper

**Acceptance Criteria**:
- No inline comma-separated ID parsing loops remain in create.go or update.go
- No inline --blocks application loops remain in create.go or update.go
- Both helpers are called from both create and update
- All existing create and update tests pass

**Tests**:
- Test parseCommaSeparatedIDs with single ID, multiple IDs, whitespace, empty strings
- Test parseCommaSeparatedIDs normalizes to lowercase
- Test applyBlocks correctly appends sourceID to matching tasks' BlockedBy
- Test applyBlocks sets Updated timestamp on modified tasks
- Test applyBlocks with non-existent blockIDs (no-op)
