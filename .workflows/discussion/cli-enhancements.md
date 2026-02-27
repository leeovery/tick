---
topic: cli-enhancements
status: in-progress
work_type: feature
date: 2026-02-27
---

# Discussion: CLI Enhancements

## Context

Six feature additions bundled as one feature, all from the IDEAS.md planned list. These are additive enhancements to the existing CLI with no architectural impact — new fields on Task, new flags on commands, and one new subcommand.

1. **List Count/Limit** — `--count N` flag on `list`/`ready`/`blocked` to cap results (LIMIT clause)
2. **Partial ID Matching** — prefix matching on hex portion of task IDs, error on ambiguous match
3. **Task Types** — `bug`, `feature`, `task`, `chore` string field with validation + filtering
4. **External References** — `[]string` field for cross-system links (`gh-123`, `JIRA-456`, URLs)
5. **Tags** — `[]string` field with `--tags` on create/update, `--tag` filter on list
6. **Notes** — timestamped text entries appended to a task (subcommand)

### References

- [IDEAS.md](../../IDEAS.md) — source of all six items

## Questions

- [x] How should new fields (tags, type, refs) be stored in JSONL and cached in SQLite?
- [x] What's the right UX for partial ID matching — where does resolution happen?
- [x] How should Notes work as a subcommand — add/list/show?
- [x] Should tags and type be settable at creation only, or also via update?
- [ ] How should filtering work for tags and type on list commands?
- [ ] What validation rules apply to task types and tags?

---

*Each question above gets its own section below. Check off as concluded.*

---

## How should new fields (tags, type, refs) be stored in JSONL and cached in SQLite?

### Context
Three new fields need storage: tags (`[]string`), type (`string`), and refs (`[]string`). Need to decide JSONL representation and SQLite caching strategy.

### Options Considered

**Option A: Junction tables for slice fields** (like `dependencies` for `blocked_by`)
- `task_tags(task_id, tag)` and `task_refs(task_id, ref)` tables in SQLite
- Proper relational filtering: `WHERE EXISTS (SELECT 1 FROM task_tags ...)`
- Consistent with existing `blocked_by` → `dependencies` pattern

**Option B: JSON/comma-delimited column in tasks table**
- Single TEXT column storing serialized data
- Simpler schema, fewer joins
- Filtering is messy (LIKE or json_each)

### Decision

**Option A — junction tables.** Follows the established `blocked_by` → `dependencies` pattern exactly.

- **JSONL**: JSON arrays on the Task struct with `omitempty`. Empty slices omitted entirely. Simple, flat, human-readable.
- **SQLite**: Junction tables `task_tags(task_id, tag)` and `task_refs(task_id, ref)` populated during `Cache.Rebuild()` by iterating the slice fields — same as `dependencies` today.
- **Type field**: Simple `TEXT` column on `tasks` table, same as `status`. No junction table needed.

JSONL stays flat and simple as source of truth. SQLite gets relational structure for fast filtered queries.

---

## What's the right UX for partial ID matching — where does resolution happen?

### Context
Currently IDs are resolved by exact match (case-insensitive via `NormalizeID`). Every command that takes an ID does its own lookup. Partial matching would let users type `a3f` instead of `tick-a3f2b1`.

### Options Considered

**Option A: Storage layer resolution via SQLite**
- `ResolveID(prefix)` method querying `WHERE id LIKE 'tick-{prefix}%'`
- Centralized — all commands call it once to get the full ID, then proceed as normal
- Read operation, fits naturally as a Store.Query

**Option B: Task/helper layer — pure function scanning `[]Task`**
- `ResolvePrefix(tasks, prefix)` used inside Mutate closures
- Requires in-memory task list, scattered across commands

### Decision

**Option A — storage layer resolution via SQLite.**

- Both `tick-a3f` and `a3f` work — strip `tick-` prefix if present before matching
- Exact full-ID match takes priority: if input matches a complete ID, return immediately without checking for prefix collisions
- Prefix matching only kicks in when input is shorter than a full ID (< 10 chars: `tick-` + 6 hex)
- On ambiguity (2+ matches): return error listing the matching IDs so user can be more specific
- On zero matches: return "not found" error
- Centralized in one place — commands resolve first, then proceed with the full ID

---

## How should Notes work as a subcommand — add/list/show?

### Context
Notes are timestamped text entries appended to a task — a log of context, decisions, progress. Need to decide subcommand structure, mutability, and display.

### Options Considered

**Implicit default: `tick note <id> <text>`**
- Simplest possible UX for adding
- But inconsistent if there are other subcommands like remove

**Explicit subcommands: `tick note add` / `tick note remove`**
- Consistent — symmetric verbs
- Matches full-word convention used everywhere else (`remove`, `create`, `cancel`)

**Append-only vs mutable**
- Append-only matches JSONL philosophy, but too restrictive for typo corrections
- Full edit adds complexity for little gain
- Delete-only is the sweet spot — remove mistakes, re-add if needed

### Journey
Initially considered `tick note <id> <text>` as the simplest approach with notes displayed via `tick show`. Then discussed mutability — started with append-only to match JSONL philosophy, but agreed that's too restrictive when typos happen. Settled on allowing delete but not edit. Once `remove` was introduced, the implicit-add felt inconsistent — symmetry demands explicit `add` verb. Confirmed all existing commands use full words (no abbreviations like `rm`).

### Decision

**Explicit subcommands with full words:**
- `tick note add <id> <text>` — append a timestamped note
- `tick note remove <id> <index>` — remove by 1-based position
- `tick show <id>` — displays notes chronologically (most recent last)
- No `tick note list` — view notes via `show`. Can add later if needed.

**Storage:**
- Task struct gets `Notes []Note` where `Note` is `{Text string, Created time.Time}`
- JSONL: JSON array with `omitempty`
- SQLite: `task_notes(task_id, text, created)` table

**Display in show:**
```
Notes:
  2026-02-27 10:00  Started investigating the auth flow
  2026-02-27 14:30  Root cause found — token refresh race condition
```

---

## Should tags and type be settable at creation only, or also via update?

### Context
Need to decide lifecycle of new fields — create-only or mutable via update? And if mutable, how to handle clearing.

### Journey
Straightforward that both create and update should support all three fields (tags, type, refs). The interesting question was clearing. The `--clear-description` pattern already exists — it was added specifically because empty `--description ""` was erroring, and agents were accidentally erasing descriptions. Same protective philosophy applies here: empty values on `--tags`/`--refs`/`--type` should error, clearing requires an explicit flag.

### Decision

**All fields settable on both create and update. Replace semantics.**

- `--tags`, `--refs`, `--type` on create: set initial values
- `--tags`, `--refs`, `--type` on update: replace entire value
- Empty value on any of them: error (protective against accidental erasure)
- `--clear-tags`, `--clear-refs`, `--clear-type`: explicit clearing flags
- Mutually exclusive: `--tags` and `--clear-tags` can't be used together (same as `--description` / `--clear-description`)

---
