# Discussion: Dep Tree Visualization

## Context

Tick tracks task dependencies via `BlockedBy` and `Blocks` fields, plus `Parent` for hierarchical relationships. All the data exists but there's no way to visualize the dependency graph. The idea is a command like `tick dep tree <id>` that renders the full dependency graph as an ASCII tree in the terminal, showing upstream blockers and downstream unblocks.

This is purely a presentation concern -- no data model changes needed. The formatter infrastructure (`Formatter` interface with toon/pretty/JSON implementations) is already in place.

### References

- Inbox idea: `.workflows/.inbox/.archived/ideas/2026-03-21--dep-tree-visualization.md`

## Questions

- [ ] What should the command structure and UX look like?
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
