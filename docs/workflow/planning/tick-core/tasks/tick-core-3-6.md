---
id: tick-core-3-6
phase: 3
status: completed
created: 2026-02-09
---

# Parent scoping — --parent flag with recursive descendant CTE

## Problem

Agents often work within a single plan (a top-level parent task representing a feature or epic). Without scoping, `tick ready`, `tick blocked`, and `tick list` return tasks from all plans, forcing agents to mentally filter irrelevant work. There is no way to restrict queries to a subtree of the task hierarchy.

## Solution

Add a `--parent <id>` flag to `tick list` (and by extension `tick ready` and `tick blocked`, since they are aliases for `tick list --ready` and `tick list --blocked`). The flag uses a recursive CTE in SQLite to collect all descendant IDs of the specified parent, then applies existing query filters (ready, blocked, status, priority) within that narrowed set. The parent task itself is excluded from results naturally by the leaf-only rule (it has open children).

## Outcome

`tick list --parent <id>`, `tick ready --parent <id>`, and `tick blocked --parent <id>` return only tasks that are descendants of the specified parent, with all existing filters composing cleanly on top. No special-case logic — `--parent` is a pre-filter that narrows the candidate set before post-filters (leaf-only, blocked-by, status, priority) are applied.

## Do

1. **Add recursive descendant CTE query function** — Write a function (e.g., `DescendantIDs(parentID string) ([]string, error)`) that executes a recursive CTE against the SQLite `tasks` table using the `parent` column. The CTE starts from the given parent ID and recursively collects all tasks whose `parent` matches any already-collected ID. Return the list of descendant IDs (excluding the parent itself).

   ```sql
   WITH RECURSIVE descendants(id) AS (
     SELECT id FROM tasks WHERE parent = ?
     UNION ALL
     SELECT t.id FROM tasks t
     JOIN descendants d ON t.parent = d.id
   )
   SELECT id FROM descendants;
   ```

2. **Add `--parent` flag to the list command** — Register a `--parent` string flag on `tick list`. Since `tick ready` and `tick blocked` are aliases for `tick list --ready` and `tick list --blocked`, the flag becomes available on all three commands automatically.

3. **Validate the parent ID** — When `--parent` is provided, verify the task exists. If it does not exist, return an error: `Error: Task '<id>' not found`. Normalize the ID to lowercase before lookup (case-insensitive matching, consistent with existing ID handling).

4. **Integrate as pre-filter** — When `--parent` is set, first collect descendant IDs via the CTE, then restrict the existing query (ready, blocked, list) to only consider tasks whose ID is in that descendant set. This can be done by adding a `WHERE id IN (...)` clause or by passing the descendant ID set to the existing query functions as an optional filter parameter. The key design point: `--parent` narrows the input set; all other filters (ready conditions, blocked conditions, `--status`, `--priority`) apply unchanged within that set.

5. **Ensure composition with all existing filters** — Verify that `--parent` composes with `--ready`, `--blocked`, `--status`, and `--priority` as AND conditions. No special cases needed — the pre-filter/post-filter design handles this naturally.

## Acceptance Criteria

- [ ] `tick list --parent <id>` returns only descendants of the specified parent (recursive, all levels)
- [ ] `tick ready --parent <id>` returns only ready tasks within the descendant set
- [ ] `tick blocked --parent <id>` returns only blocked tasks within the descendant set
- [ ] Parent task itself is excluded from results (leaf-only rule filters it out naturally when it has open children)
- [ ] Non-existent parent ID returns error: `Error: Task '<id>' not found`
- [ ] Parent with no descendants returns empty result (`No tasks found.`, exit 0)
- [ ] Deep nesting (3+ levels) collects all descendants recursively
- [ ] `--parent` composes with `--status` filter (AND)
- [ ] `--parent` composes with `--priority` filter (AND)
- [ ] `--parent` composes with `--ready` flag (AND)
- [ ] `--parent` composes with `--blocked` flag (AND)
- [ ] Case-insensitive parent ID matching (e.g., `TICK-A1B2` treated as `tick-a1b2`)
- [ ] `--quiet` outputs IDs only within the scoped set

## Tests

- `"it returns all descendants of parent (direct children)"`
- `"it returns all descendants recursively (3+ levels deep)"`
- `"it excludes parent task itself from results"`
- `"it returns empty result when parent has no descendants"`
- `"it errors with 'Task not found' for non-existent parent ID"`
- `"it returns only ready tasks within parent scope with tick ready --parent"`
- `"it returns only blocked tasks within parent scope with tick blocked --parent"`
- `"it combines --parent with --status filter"`
- `"it combines --parent with --priority filter"`
- `"it combines --parent with --ready and --priority"`
- `"it combines --parent with --blocked and --status"`
- `"it handles case-insensitive parent ID"`
- `"it excludes tasks outside the parent subtree"`
- `"it outputs IDs only with --quiet within scoped set"`
- `"it returns 'No tasks found.' when descendants exist but none match filters"`

## Edge Cases

- **Non-existent parent ID**: Error, not empty result — the parent must exist for the query to be meaningful.
- **Parent with no descendants**: Valid parent exists but has no children at any level. Returns empty result with `No tasks found.`, exit 0.
- **Deep nesting (3+ levels)**: Recursive CTE must traverse the full depth. E.g., grandparent → parent → child → grandchild — querying grandparent returns all three descendants.
- **Parent task excluded naturally**: The parent itself is not in the descendant set (CTE starts from children of the parent). Even if it were, the leaf-only rule would exclude it when it has open children. When all children are closed, the parent becomes a leaf and would be ready — but it is still not a descendant of itself.
- **Filters reduce to empty**: `--parent tick-p1 --status done` when no descendants are done returns `No tasks found.`, exit 0 — not an error.
- **`--parent` on `tick ready` vs `tick blocked` vs `tick list`**: All three use the same pre-filter mechanism. `tick ready --parent X` = `tick list --ready --parent X`. `tick blocked --parent X` = `tick list --blocked --parent X`.

## Context

> From the specification — Parent Scoping section:
>
> The `--parent <id>` flag restricts queries to descendants of the specified task. This enables plan-level scoping — create a top-level parent task per plan, add plan tasks as children, then filter queries to that subtree.
>
> `--parent` is a pre-filter: It narrows which tasks are considered. The leaf-only and blocked-by rules are post-filters that determine which of those are workable. They compose cleanly with no special cases.
>
> Implementation: Recursive CTE in SQLite to collect all descendant IDs, then apply existing query filters within that set.
>
> The SQLite schema already has `CREATE INDEX idx_tasks_parent ON tasks(parent)` — the parent column is indexed, supporting efficient CTE traversal.

Specification reference: `docs/workflow/specification/tick-core.md`
