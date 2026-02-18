---
id: tick-core-2-3
phase: 2
status: completed
created: 2026-01-30
---

# tick update command

## Goal

Tasks can be created and transitioned, but fields can't be modified after creation. `tick update <id>` changes title, description, priority, parent, and blocks. At least one flag required — no flags is an error. Immutable fields (status, id, created, blocked_by) have dedicated commands.

## Implementation

- Register `update` subcommand
- Parse positional ID, normalize to lowercase
- Flags: `--title`, `--description`, `--priority <0-4>`, `--parent <id>`, `--blocks <id,...>`
- No flags → error with available options list
- Execute via storage engine `Mutate`:
  1. Look up task by ID
  2. Validate and apply each flag (title trim/500/newlines, priority 0-4, parent exists/no self-ref, blocks IDs exist)
  3. `--description ""` clears description; `--parent ""` clears parent
  4. `--blocks` adds this task to targets' `blocked_by`, refreshes their `updated`
  5. Set `updated` to current UTC
  6. Return modified task list
- Output full task details (like `tick show`); `--quiet` outputs only ID

## Tests

- `"it updates title with --title flag"`
- `"it updates description with --description flag"`
- `"it clears description with --description \"\""`
- `"it updates priority with --priority flag"`
- `"it updates parent with --parent flag"`
- `"it clears parent with --parent \"\""`
- `"it updates blocks with --blocks flag"`
- `"it updates multiple fields in a single command"`
- `"it refreshes updated timestamp on any change"`
- `"it outputs full task details on success"`
- `"it outputs only task ID with --quiet flag"`
- `"it errors when no flags are provided"`
- `"it errors when task ID is missing"`
- `"it errors when task ID is not found"`
- `"it errors on invalid title (empty/500/newlines)"`
- `"it errors on invalid priority (outside 0-4)"`
- `"it errors on non-existent parent/blocks IDs"`
- `"it errors on self-referencing parent"`
- `"it normalizes input IDs to lowercase"`
- `"it persists changes via atomic write"`

## Edge Cases

- No flags: error with options list, exit 1
- Clear description: `--description ""` sets empty string
- Clear parent: `--parent ""` sets null
- Title validation: same as create (trim, 500 max, no newlines)
- Immutable: status/id/created/blocked_by not exposed as flags
- `--blocks` modifies other tasks atomically
- Self-reference parent: error

## Acceptance Criteria

- [ ] All five flags work correctly
- [ ] Multiple flags combinable in single command
- [ ] `updated` refreshed on every update
- [ ] No flags → error with exit code 1
- [ ] Missing/not-found ID → error with exit code 1
- [ ] Invalid values → error with exit code 1, no mutation
- [ ] Output shows full task details; `--quiet` outputs ID only
- [ ] Input IDs normalized to lowercase
- [ ] Mutation persisted through storage engine

## Context

Spec: `tick update <id>` with --title, --description, --priority, --parent, --blocks. At least one required. Cannot change id/status/created/blocked_by. `--blocks` is inverse of `--blocked-by`. Output like `tick show`.

Specification reference: `docs/workflow/specification/tick-core.md`
