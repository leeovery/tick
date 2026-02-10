---
id: tick-core-7-2
phase: 7
status: pending
created: 2026-02-10
---

# Add relationship context to create command output

**Problem**: The spec (line 631) states create output should be "full task details (same format as tick show), TTY-aware." The implementation constructs a TaskDetail with only the raw Task struct and empty BlockedBy/Children slices (create.go line 183), without querying SQLite for relationship context (blocker titles/statuses, parent title). If a task is created with `--blocked-by` or `--parent`, those relationships will not appear in the output. In contrast, `tick update` (update.go lines 214-220) correctly calls `queryShowData` to populate all relationship context before formatting.

**Solution**: After the Mutate call succeeds in RunCreate, call `queryShowData(store, createdTask.ID)` to retrieve full relationship context (matching the approach used by RunUpdate), then pass that populated TaskDetail to `FormatTaskDetail`.

**Outcome**: Create output is truly "same format as tick show" -- blocked_by entries show with titles and statuses, parent title is included, and children (if any exist due to --blocks creating reverse relationships) are shown.

**Do**:
1. In `internal/cli/create.go` RunCreate, after the successful Mutate call and before the output block
2. Replace the manual `TaskDetail{Task: createdTask}` construction (line 183) with a call to `queryShowData(store, createdTask.ID)`
3. Use the returned showData to build the TaskDetail via `showDataToTaskDetail` (same approach as update.go)
4. Ensure the store is available at that point in the function (it should be, from the openStore call)
5. Handle the error case from queryShowData appropriately

**Acceptance Criteria**:
- `tick create "test" --blocked-by tick-abc` output includes the blocker's title and status in the blocked_by section
- `tick create "test" --parent tick-abc` output includes the parent's title
- Create output matches show output for the same task (when viewed immediately after creation)
- `--quiet` mode still outputs only the task ID (no change)
- All existing create tests pass

**Tests**:
- Test create with --blocked-by shows blocker title and status in output
- Test create with --parent shows parent title in output
- Test create with --blocks shows the created task's relationship context
- Test create with --quiet still outputs only the task ID
- Test create without relationships still produces correct output (empty blocked_by/children sections)
