# Discussion: Dep Tree Visualization

## Context

Tick tracks task dependencies via `BlockedBy` (stored on task) plus `Parent` for hierarchical relationships. There is no explicit `Blocks` field — the reverse relationship is computed via query. All the data exists but there's no way to visualize the dependency graph.

This is purely a presentation concern — no data model changes needed. The formatter infrastructure (`Formatter` interface with toon/pretty/JSON implementations) is already in place. The existing `tick dependency add/remove` subcommand pattern means `tick dependency tree` fits naturally.

### References

- Inbox idea: `.workflows/.inbox/.archived/ideas/2026-03-21--dep-tree-visualization.md`

## Questions

- [x] What should the command structure and UX look like?
      - Subcommand name and arguments
      - Direction control: upstream vs downstream vs both
      - Default behavior when no direction specified
- [ ] How should the tree be rendered?
      - ASCII art style and characters
      - How to show task status, priority, and other metadata inline
      - Handling of circular or complex graph structures
- [ ] How should this integrate with the existing formatter system?
      - New method on `Formatter` interface vs standalone renderer
      - What each format (toon/pretty/JSON) should output
- [ ] What about the parent/child hierarchy?
      - Should dep tree include parent/child relationships alongside dependency edges?
      - Combined view vs separate views
      - Risk of noise in deeply nested projects
- [ ] How should edge cases be handled?
      - Tasks with no dependencies
      - Very wide or very deep graphs
      - Terminal width constraints

---

*Each question above gets its own section below. Check off as completed.*

---

## What should the command structure and UX look like?

### Context
Need a command to visualize dependency relationships. `tick dependency` already exists with `add`/`remove` subcommands, so `tree` slots in naturally as a third subcommand.

### Options Considered

**Option A: `tick tree <id>` — new top-level command**
- Pros: Short, simple
- Cons: New command namespace, doesn't group with existing dep commands

**Option B: `tick deps <id>`**
- Pros: Explicit about content
- Cons: Abbreviation inconsistent with existing `dependency` command name

**Option C: `tick dependency tree [id]`**
- Pros: Groups with existing `dependency add/remove`, established subcommand pattern
- Cons: Longer to type

### Journey
Initially considered all three. The existing `tick dependency add/remove` pattern made Option C the natural fit — it's not introducing a new convention, it's extending an existing one. Options A and B would create parallel ways to interact with dependency data.

Key realisation: the command should work with and without an ID argument. No ID = full project dependency graph. With ID = focused view centred on that task.

### Decision
**`tick dependency tree [id]`** — extends the existing dependency subcommand pattern. Optional ID argument controls scope.

**Two modes:**
1. **Full graph** (`tick dependency tree`) — shows all tasks that participate in dependency relationships. Each task lists what it blocks. Includes a summary line (chains, longest chain, blocked count).
2. **Focused view** (`tick dependency tree <id>`) — walks both directions from the target task. "Blocked by" section walks upstream (what blocks me, what blocks those). "Blocks" section walks downstream (what I unblock, what those unblock). Full transitive depth — no artificial cap.

**Open question:** Whether tasks with zero dependencies should appear in the full graph. Leaning towards omitting them (they're not interesting in a dependency view) but feels slightly incomplete. Parking this — will revisit after seeing it in practice.

---
