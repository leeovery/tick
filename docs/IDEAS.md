# Ideas

Feature ideas for Tick, collected from comparing with [Ticket](https://github.com/wedow/ticket) and general brainstorming. Each item needs discussion/specification via the workflow system before implementation.

## Planned

Features to spec and build:

### Tags
Categorization via string labels. `--tags ui,backend` on create/update, `--tag` filter on list. Simple `[]string` field on Task. High value, low complexity.

### Partial ID Matching
Allow `tick show a3f` to resolve to `tick-a3f2b1`. Prefix matching on the hex portion. Error on ambiguous match (multiple hits). Pure DX improvement.

### Notes
Timestamped text entries appended to a task. Not multi-user "comments" — just a log of context, decisions, progress. Keeps information with the task instead of relying on memory. Particularly useful in AI-assisted workflows where context is ephemeral.

### Task Types
`bug`, `feature`, `task`, `chore` (maybe `epic`). String field with validation. Useful for filtering and stats. Keeps things categorizable without over-structuring.

### External References
`[]string` field for cross-system links: `gh-123`, `JIRA-456`, URLs. Low effort, occasionally very useful for connecting Tick tasks to PRs/issues.

### Dependency Tree Visualization
Render the dependency graph as a tree in the terminal. All the data already exists — this is purely a formatter addition. `tick dep tree <id>` or similar.

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
