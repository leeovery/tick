# Ideas

Feature ideas for Tick, collected from comparing with [Ticket](https://github.com/wedow/ticket) and general brainstorming. Each item needs discussion/specification via the workflow system before implementation.

## Planned

Features to spec and build:

### Dependency Tree Visualization
Render the dependency graph as a tree in the terminal. All the data already exists — this is purely a formatter addition. `tick dep tree <id>` or similar.

### Auto-Cascade Parent Status
When a child task is started, automatically set its parent to `in_progress`, recursively up the ancestor chain. Currently parents just sit as `open` until manually started, even when children are actively being worked. Needs discussion — implications for explicit vs implicit status transitions, undo behavior, and whether this should apply to `done`/`cancel` cascading downward too.

## Considered / Future

Ideas worth tracking but not building now:

### Assignee
Not unreasonable, but Tick is single-user by design. JSONL conflicts on concurrent multi-user edits (especially updates to existing lines) make this problematic without rethinking the storage model. Revisit if/when multi-user becomes a goal.

### Event-Sourced Storage
Append-only mutation log where task state is derived from replaying events. Would solve multi-user JSONL conflicts since you never update lines, only append. Significant architectural change — future consideration if multi-user is pursued.

## Rejected

### Plugin System
Over-engineering for Tick's scope. Adds maintenance burden and complexity without proportional value for a focused CLI tool.

### Symmetric Links
"Related" without directionality is too vague to be actionable. Dependencies and parent/child already cover meaningful relationships.

### Design/Acceptance Fields
Too rigid as dedicated fields. Notes and description already serve this purpose with more flexibility.
