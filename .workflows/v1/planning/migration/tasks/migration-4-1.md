---
id: migration-4-1
phase: 4
status: completed
created: 2026-02-15
---

# Fix beads provider to distinguish absent priority from priority zero

**Problem**: The `beadsIssue` struct declares `Priority int` (line 29 of `internal/migrate/beads/beads.go`). When a beads JSON line omits the `priority` field entirely, Go's zero value (0) is used. `mapToMigratedTask` then unconditionally sets `Priority: &priority` (line 129), telling the engine "priority was explicitly set to 0." The `StoreTaskCreator` only applies tick's default priority (2) when `mt.Priority == nil`, so omitted-priority beads tasks get priority 0 instead of tick's default 2. The spec says "Missing data uses sensible defaults."

**Solution**: Change `beadsIssue.Priority` from `int` to `*int` (`Priority *int \`json:"priority"\``). In `mapToMigratedTask`, only set `MigratedTask.Priority` when the beads issue pointer is non-nil, leaving it nil otherwise so the engine applies tick's default priority of 2.

**Outcome**: When a beads JSON line omits the priority field, the resulting `MigratedTask.Priority` is nil, causing the `StoreTaskCreator` to apply tick's default priority of 2. When a beads JSON line explicitly sets priority to 0, the value is preserved as `*int` pointing to 0.

**Do**:
1. In `internal/migrate/beads/beads.go` line 33, change `Priority int` to `Priority *int` in the `beadsIssue` struct (keep the json tag as `json:"priority"`).
2. In `mapToMigratedTask` (same file), change the priority assignment from unconditional `priority := issue.Priority` / `Priority: &priority` to conditional: only set `MigratedTask.Priority` when `issue.Priority != nil`.
3. Update tests in `internal/migrate/beads/beads_test.go`:
   - Add a test case verifying that when the JSON line omits the priority field entirely, `MigratedTask.Priority` is nil.
   - Existing tests that set `priority:0` explicitly should continue to produce a non-nil `*int` pointing to 0.
   - The `mapToMigratedTask` unit test at line 459 uses a `beadsIssue` literal with `Priority: 0` -- this needs updating to use a `*int` value (`Priority: intPtr(0)` or similar helper).
4. Verify all existing tests still pass after the change.

**Acceptance Criteria**:
- `beadsIssue.Priority` field type is `*int`
- JSON line with `"priority":0` produces `MigratedTask.Priority` pointing to 0
- JSON line with `"priority":3` produces `MigratedTask.Priority` pointing to 3
- JSON line omitting the `priority` field produces `MigratedTask.Priority == nil`
- All existing beads provider tests pass
- New test covers the absent-priority case

**Tests**:
- Unit test: parse a beads JSON line that omits the priority field entirely; assert `MigratedTask.Priority == nil`
- Unit test: parse a beads JSON line with `"priority":0`; assert `MigratedTask.Priority != nil && *MigratedTask.Priority == 0`
- Unit test: existing priority mapping tests (0-3) continue to pass with non-nil pointers
- Integration: a beads file with one task omitting priority and one task with explicit priority 0 should produce correct results through the full engine (omitted gets default 2, explicit 0 stays 0)
