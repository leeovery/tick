# Dependency Tree Visualization

Render the dependency graph as a tree in the terminal. All the underlying data already exists — tasks have `BlockedBy` and `Blocks` fields for dependency edges, plus `Parent` for hierarchical relationships. This is purely a presentation concern, not a data model change.

The idea is a command like `tick dep tree <id>` that takes a task and visualizes its full dependency graph as an ASCII tree. You'd see which tasks block it, which tasks those are blocked by in turn, and so on — the full chain laid out hierarchically so you can understand at a glance why something is stuck or what completing a task would unblock.

This could also work in the other direction — showing what a given task blocks downstream. A flag or argument could control the direction: upstream (what blocks me?) vs downstream (what do I unblock?).

The formatter infrastructure is already in place with the `Formatter` interface and its three implementations (toon, pretty, JSON). A tree visualization method could slot in as a new method on the interface, or it could be a standalone rendering function if it doesn't need format-specific variation.

Worth considering whether this should also incorporate the parent/child hierarchy alongside dependency edges, or keep them separate. A combined view could be powerful but risks getting noisy in projects with deep nesting and cross-cutting dependencies.
