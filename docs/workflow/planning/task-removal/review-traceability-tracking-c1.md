---
status: in-progress
created: 2026-02-19
cycle: 1
phase: Traceability Review
topic: Task Removal
---

# Review Tracking: Task Removal - Traceability

## Findings

### 1. Confirmation prompt does not surface dependency cleanup info

**Type**: Incomplete coverage
**Spec Reference**: Confirmation Behavior — "The prompt surfaces the blast radius: ... Any dependency references that will be cleaned up from surviving tasks"
**Plan Reference**: Phase 3 / task-removal-3-4 (tick-9e0c27) — Integrate cascade into removal flow with confirmation prompt
**Change Type**: add-to-task

**Details**:
The specification explicitly requires the confirmation prompt to surface three categories of blast radius information: (1) target tasks being removed, (2) children that will be cascade-deleted, and (3) dependency references that will be cleaned up from surviving tasks. Task 3-4 extends the confirmation prompt to show targets and cascaded descendants (items 1 and 2), but does not include any mention of surviving tasks whose BlockedBy arrays will be cleaned (item 3). No other task addresses this either. An implementer following the current plan would produce a prompt that omits dependency cleanup preview, diverging from the spec.

**Current**:
```
## Do

1. In internal/cli/remove.go, inside RunRemove, after the all-or-nothing ID validation loop and before the confirmation prompt / filtering logic, call collectDescendants(removeSet, tasks) and reassign the result to removeSet.

2. Update the pre-prompt task lookup to collect information for all IDs in the expanded removeSet. Build two collections:
   - targetTasks: tasks explicitly passed as arguments (original ids slice)
   - cascadedTasks: tasks added by collectDescendants (in removeSet but not in original ids set)

3. Update the confirmation prompt logic (non-force path) to display cascade blast radius on stderr:
   - Single target, no cascade: keep existing format -- Remove task tick-abc "Title"? [y/N]
   - Single target with cascade: show target, list cascaded descendants, then prompt
   - Multiple targets: list all targets, then additional cascaded descendants
   Prompt must surface every task that will be removed and distinguish descendants from explicit targets.

4. Ensure filtering step uses expanded removeSet for keeping/removing tasks, recording RemovalResult.Removed, and stripping from surviving tasks' BlockedBy arrays.

5. In internal/cli/remove_test.go, add tests for cascade integration.

## Acceptance Criteria

- [ ] Removing a parent task also removes all transitive descendants in a single Store.Mutate call
- [ ] Removing a leaf (child) task does not remove its parent or siblings
- [ ] Confirmation prompt (without --force) lists all tasks including cascaded descendants
- [ ] Prompt distinguishes explicitly targeted tasks from cascaded descendants
- [ ] --force with cascade proceeds silently without any prompt
- [ ] RemovalResult.Removed includes targets and all cascaded descendants
- [ ] Dependency cleanup scrubs all removed IDs (targets + descendants) from surviving tasks' BlockedBy
- [ ] RemovalResult.DepsUpdated reflects tasks whose BlockedBy was cleaned of any expanded-set ID
- [ ] Prompt text and abort message written to stderr
- [ ] Single target with no children retains existing simple prompt format
- [ ] All existing remove tests continue to pass

## Tests

- "it removes parent and all descendants when removing a parent with --force"
- "it removes 3-level hierarchy (parent -> child -> grandchild) with --force"
- "it does not remove parent when removing a child with --force"
- "it does not remove siblings when removing a child with --force"
- "it shows descendants in confirmation prompt when removing parent without --force"
- "it proceeds with cascade removal when user confirms with y"
- "it aborts cascade removal when user declines"
- "it skips prompt entirely with --force for cascade removal"
- "it cleans BlockedBy references for all cascaded descendant IDs on surviving tasks"
- "it reports all cascade-removed tasks in RemovalResult.Removed"
- "it reports dep-updated tasks in RemovalResult.DepsUpdated for cascade-removed IDs"
- "it retains simple prompt format for single target with no children"
- "it writes cascade prompt to stderr not stdout"
```

**Proposed**:
```
## Do

1. In internal/cli/remove.go, inside RunRemove, after the all-or-nothing ID validation loop and before the confirmation prompt / filtering logic, call collectDescendants(removeSet, tasks) and reassign the result to removeSet.

2. Update the pre-prompt task lookup to collect information for all IDs in the expanded removeSet. Build three collections:
   - targetTasks: tasks explicitly passed as arguments (original ids slice)
   - cascadedTasks: tasks added by collectDescendants (in removeSet but not in original ids set)
   - affectedDepTasks: surviving tasks (not in removeSet) that have any removeSet ID in their BlockedBy arrays — these will have dependency references cleaned up

3. Update the confirmation prompt logic (non-force path) to display the full blast radius on stderr:
   - Single target, no cascade, no dep impact: keep existing format -- Remove task tick-abc "Title"? [y/N]
   - When cascade applies: show target, list cascaded descendants
   - When surviving tasks have dependency references: list which surviving tasks will have dependency references cleaned (e.g., "Will update dependencies on tick-def, tick-ghi")
   - Multiple targets: list all targets, then additional cascaded descendants, then affected dependencies
   Prompt must surface every task that will be removed, distinguish descendants from explicit targets, and show which surviving tasks will have dependency references cleaned.

4. Ensure filtering step uses expanded removeSet for keeping/removing tasks, recording RemovalResult.Removed, and stripping from surviving tasks' BlockedBy arrays.

5. In internal/cli/remove_test.go, add tests for cascade integration.

## Acceptance Criteria

- [ ] Removing a parent task also removes all transitive descendants in a single Store.Mutate call
- [ ] Removing a leaf (child) task does not remove its parent or siblings
- [ ] Confirmation prompt (without --force) lists all tasks including cascaded descendants
- [ ] Prompt distinguishes explicitly targeted tasks from cascaded descendants
- [ ] Confirmation prompt (without --force) lists surviving tasks whose dependency references will be cleaned
- [ ] --force with cascade proceeds silently without any prompt
- [ ] RemovalResult.Removed includes targets and all cascaded descendants
- [ ] Dependency cleanup scrubs all removed IDs (targets + descendants) from surviving tasks' BlockedBy
- [ ] RemovalResult.DepsUpdated reflects tasks whose BlockedBy was cleaned of any expanded-set ID
- [ ] Prompt text and abort message written to stderr
- [ ] Single target with no children and no dep impact retains existing simple prompt format
- [ ] All existing remove tests continue to pass

## Tests

- "it removes parent and all descendants when removing a parent with --force"
- "it removes 3-level hierarchy (parent -> child -> grandchild) with --force"
- "it does not remove parent when removing a child with --force"
- "it does not remove siblings when removing a child with --force"
- "it shows descendants in confirmation prompt when removing parent without --force"
- "it shows affected dependency tasks in confirmation prompt"
- "it proceeds with cascade removal when user confirms with y"
- "it aborts cascade removal when user declines"
- "it skips prompt entirely with --force for cascade removal"
- "it cleans BlockedBy references for all cascaded descendant IDs on surviving tasks"
- "it reports all cascade-removed tasks in RemovalResult.Removed"
- "it reports dep-updated tasks in RemovalResult.DepsUpdated for cascade-removed IDs"
- "it retains simple prompt format for single target with no children and no dep impact"
- "it writes cascade prompt to stderr not stdout"
```

**Resolution**: Pending
**Notes**:
The spec's "Confirmation Behavior" section lists three blast radius items the prompt must surface. The plan covers two (targets and cascade children) but omits the third (dependency references on surviving tasks). This applies to all prompt scenarios — single, cascade, and bulk. The proposed fix adds a pre-scan for affected dependency tasks and includes them in the prompt output.

---
