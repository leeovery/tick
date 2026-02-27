---
topic: cli-enhancements
status: in-progress
type: feature
date: 2026-02-27
review_cycle: 0
finding_gate_mode: gated
sources:
  - name: cli-enhancements
    status: pending
---

# Specification: CLI Enhancements

## Specification

### Partial ID Matching

Allow users to reference tasks by a prefix of the hex portion instead of the full `tick-XXXXXX` ID.

**Resolution rules:**
- Both `tick-a3f` and `a3f` accepted — strip `tick-` prefix if present before matching
- Exact full-ID match takes priority: if input matches a complete ID (10 chars: `tick-` + 6 hex), return immediately without checking for prefix collisions
- Prefix matching only for inputs shorter than a full ID
- Minimum 3 hex chars required for prefix matching (prevents overly broad matches)
- Ambiguity (2+ matches): error listing the matching IDs
- Zero matches: "not found" error

**Implementation location:**
- Storage layer — `ResolveID(prefix)` method querying `WHERE id LIKE 'tick-{prefix}%'`
- Centralized: all commands resolve first, then proceed with the full ID
- Applies everywhere an ID is accepted: positional args, `--parent`, `--blocked-by`, `--blocks`

### Task Types

A string field on Task classifying the kind of work.

**Allowed values:** `bug`, `feature`, `task`, `chore` (closed set — anything else errors).

**Validation:**
- Case-insensitive input, trimmed, stored lowercase
- Validated on create and update

**CLI flags:**
- `--type <value>` on `create` and `update` — sets or replaces the type
- `--clear-type` on `update` — explicitly removes the type
- `--type` and `--clear-type` are mutually exclusive
- Empty value on `--type` errors (protective against accidental erasure)

**Filtering:**
- `--type <value>` on `list`, `ready`, `blocked` — single value filter only
- No comma-separated, no multiple flags (keeps comma semantics consistent with tags where comma = AND; AND is meaningless for a single-value field)

**Storage:**
- JSONL: string field with `omitempty`
- SQLite: `TEXT` column on `tasks` table

**Display:**
- List output: shown as a column — ID, Status, Priority, Type, Title. Dash (`-`) when not set.
- Show output: displayed with other fields

### Tags

A `[]string` field on Task for user-defined labels.

**Validation:**
- Strict kebab-case: `[a-z0-9]+(-[a-z0-9]+)*`
- No spaces, no commas, no leading/trailing hyphens, no double hyphens
- Input trimmed and lowercased (normalize, don't error on case)
- Max 30 chars per tag
- Max 10 tags per task (validated after deduplication)
- Silently deduplicate on input (e.g. `--tags ui,backend,ui` stores `[ui, backend]`)

**CLI flags:**
- `--tags <comma-separated>` on `create` and `update` — sets or replaces all tags
- `--clear-tags` on `update` — explicitly removes all tags
- `--tags` and `--clear-tags` are mutually exclusive
- Empty value on `--tags` errors (protective against accidental erasure)

**Filtering:**
- `--tag <comma-separated>` on `list`, `ready`, `blocked` — comma-separated values are AND (task must have all listed tags)
- Multiple `--tag` flags are OR (task matches any group)
- Composable: `--tag ui,backend --tag api` means "(ui AND backend) OR api"

**Storage:**
- JSONL: JSON array with `omitempty` (empty slices omitted entirely)
- SQLite: junction table `task_tags(task_id, tag)` — follows the established `blocked_by` → `dependencies` pattern. Populated during `Cache.Rebuild()`.

**Display:**
- List output: not shown (variable-length, would clutter the table)
- Show output: displayed with other fields

---

## Working Notes
