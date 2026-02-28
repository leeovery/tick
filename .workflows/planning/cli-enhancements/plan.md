---
topic: cli-enhancements
status: planning
format: tick
work_type: feature
ext_id: tick-db58ad
specification: ../specification/cli-enhancements/specification.md
spec_commit: 8aae967aad70c57432f9d24dcd226594f295358a
created: 2026-02-28
updated: 2026-02-28
external_dependencies: []
task_list_gate_mode: gated
author_gate_mode: auto
finding_gate_mode: gated
planning:
  phase: 3
  task: ~
---

# Plan: CLI Enhancements

### Phase 1: Partial ID Matching
status: approved
ext_id: tick-bc42b5
approved_at: 2026-02-28

**Goal**: Enable prefix-based task ID resolution so users can reference tasks by shortened hex prefixes across all commands.

**Why this order**: Partial ID matching is foundational — it changes how every command resolves task IDs. Implementing it first means all subsequent features (type, tags, refs, notes) can be developed and tested using short IDs. It also establishes the pattern for centralized ID resolution in the storage layer before new commands and fields are added.

**Acceptance**:
- [ ] `ResolveID(prefix)` method in storage layer queries `WHERE id LIKE 'tick-{prefix}%'` and returns the full ID
- [ ] Both `tick-a3f` and `a3f` input forms accepted; `tick-` prefix stripped if present, input lowercased before matching
- [ ] Exact full-ID match (10-char input `tick-` + 6 hex) returns immediately without prefix collision check
- [ ] Minimum 3 hex chars required for prefix matching; fewer returns a validation error
- [ ] Ambiguous prefix (2+ matches) returns error listing all matching IDs
- [ ] Zero matches returns "not found" error
- [ ] All commands accepting task IDs resolve through `ResolveID`: show, update, start, done, cancel, reopen, dep add/rm, remove, and ID-accepting flags (`--parent`, `--blocked-by`, `--blocks`)

#### Tasks
| ID | Name | Edge Cases | Status | Ext ID |
|----|------|------------|--------|--------|
| cli-enhancements-1-1 | ResolveID method in storage layer | prefix shorter than 3 hex chars, ambiguous prefix matching 2+ tasks, exact full-ID bypass, mixed-case input, tick- prefix stripping | authored | tick-9283bb |
| cli-enhancements-1-2 | Integrate ResolveID into positional ID commands | none | authored | tick-b45af0 |
| cli-enhancements-1-3 | Integrate ResolveID into update and create ID-referencing flags | partial ID resolving to self-reference in --parent or --blocked-by | authored | tick-9540a5 |
| cli-enhancements-1-4 | Integrate ResolveID into dep add/rm | both arguments resolving to same task | authored | tick-376da0 |

### Phase 2: Task Types and List Count
status: approved
ext_id: tick-ccdecb
approved_at: 2026-02-28

**Goal**: Add the `type` field to the Task model with create/update/filter/display support, and add the `--count` flag for capping list results.

**Why this order**: Type is a single-value string field — the simplest new field to add. It touches every layer (domain model, JSONL serialization, SQLite schema, cache rebuild, CLI flags, formatters, list queries) and establishes the full vertical pattern for adding fields. List count is a small, self-contained addition to the list query path that pairs naturally with filtering work.

**Acceptance**:
- [ ] `Task` struct has `Type string` field with `json:"type,omitempty"`
- [ ] SQLite `tasks` table has `type TEXT` column; populated during `Cache.Rebuild()`
- [ ] `--type <value>` on `create` and `update` sets/replaces type; validated against closed set (`bug`, `feature`, `task`, `chore`); case-insensitive input normalized to lowercase
- [ ] `--clear-type` on `update` removes the type; mutually exclusive with `--type`; empty `--type` value errors
- [ ] `--type <value>` on `list`, `ready`, `blocked` filters by single value (normalized before matching)
- [ ] List output includes Type column (ID, Status, Priority, Type, Title); dash (`-`) when unset
- [ ] Show output displays type field
- [ ] All three formatters (ToonFormatter, PrettyFormatter, JSONFormatter) updated
- [ ] `--count N` on `list`, `ready`, `blocked` appends `LIMIT N` to query; value must be >= 1

#### Tasks
| ID | Name | Edge Cases | Status | Ext ID |
|----|------|------------|--------|--------|
| cli-enhancements-2-1 | Add Type field to Task model and JSONL serialization | empty string on --type, mixed-case input, invalid type value, whitespace-only input | authored | tick-5a322f |
| cli-enhancements-2-2 | Add Type column to SQLite schema and Cache.Rebuild | none | authored | tick-9e1481 |
| cli-enhancements-2-3 | Create and update commands with --type and --clear-type flags | --type and --clear-type together on update, empty --type value | authored | tick-811654 |
| cli-enhancements-2-4 | List/ready/blocked filtering by --type | invalid type value in filter, filter with no matching tasks | authored | tick-3357ef |
| cli-enhancements-2-5 | Display Type in list and show output across all formatters | type unset showing as dash in list | authored | tick-2a23a5 |
| cli-enhancements-2-6 | Add --count flag to list/ready/blocked | --count 0, --count negative, --count non-integer, --count larger than result set | authored | tick-3e1ed5 |

### Phase 3: Tags
status: approved
ext_id:
approved_at: 2026-02-28

**Goal**: Add multi-value tags with kebab-case validation, junction table storage, and composable AND/OR filtering on list commands.

**Why this order**: Tags build on the field-addition pattern established in Phase 2 but introduce greater complexity: a junction table (`task_tags`), multi-value comma-separated input with deduplication, and AND/OR filter composition in SQL queries. This progression from simple (type) to complex (tags) is natural.

**Acceptance**:
- [ ] `Task` struct has `Tags []string` field with `json:"tags,omitempty"`
- [ ] SQLite `task_tags(task_id, tag)` junction table created in schema and populated during `Cache.Rebuild()`
- [ ] Tag validation enforces kebab-case regex `[a-z0-9]+(-[a-z0-9]+)*`, max 30 chars per tag, max 10 tags after dedup
- [ ] Input trimmed, lowercased, silently deduplicated before validation
- [ ] `--tags <comma-separated>` on `create` and `update` sets/replaces all tags; `--clear-tags` on `update` removes all; mutually exclusive; empty `--tags` value errors
- [ ] `--tag <comma-separated>` on `list`, `ready`, `blocked`: comma values are AND (task must have all), multiple `--tag` flags are OR
- [ ] Tags displayed in show output; not displayed in list output
- [ ] All three formatters updated for tags in show/detail views

### Phase 4: External References and Notes
status: approved
ext_id:
approved_at: 2026-02-28

**Goal**: Add external reference links and timestamped notes to tasks, with refs on create/update and notes via a new `tick note` subcommand.

**Why this order**: Refs follow the junction table pattern proven in Phase 3 with simpler validation and no filtering. Notes introduce a new subcommand structure (`tick note add/remove`) and a new `Note` data type. Both are show-only display additions. Combining them into one phase avoids two phases that would each be too thin.

**Acceptance**:
- [ ] `Task` struct has `Refs []string` field with `json:"refs,omitempty"`; SQLite `task_refs(task_id, ref)` junction table populated during `Cache.Rebuild()`
- [ ] `--refs <comma-separated>` on `create` and `update` sets/replaces refs; `--clear-refs` on `update` removes all; mutually exclusive; empty `--refs` value errors
- [ ] Ref validation: non-empty, no commas, no whitespace, max 200 chars, max 10 per task, silent dedup
- [ ] `Task` struct has `Notes []Note` field with `json:"notes,omitempty"`; `Note` has `Text string` and `Created time.Time`
- [ ] SQLite `task_notes(task_id, text, created)` table populated during `Cache.Rebuild()`
- [ ] `tick note add <id> <text>` appends a timestamped note; text from remaining args after ID; validates non-empty and max 500 chars
- [ ] `tick note remove <id> <index>` removes by 1-based position; index validated >= 1 and <= note count
- [ ] Adding/removing a note updates the task's `Updated` timestamp
- [ ] Show output displays refs and notes; notes shown chronologically (most recent last) with timestamp format `YYYY-MM-DD HH:MM`
- [ ] All three formatters updated for refs and notes display in detail views
