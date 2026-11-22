# V1

## 1. Blocked Ancestor Ready

---
topic: blocked-ancestor-ready
status: concluded
type: feature
work_type: greenfield
date: 2026-02-20
review_cycle: 1
finding_gate_mode: gated
sources:
  - name: blocked-ancestor-ready
    status: incorporated
---

#### Specification: Blocked Ancestor Ready

##### Problem

Children of blocked parents incorrectly appear as "ready" in `tick ready` and `tick list --ready`. The current ready check uses 3 conditions (open status, no own unclosed blockers, no open children) but never checks whether the task's parent — or any ancestor — is itself dependency-blocked.

###### Example

```
Phase 1 (open)
Phase 2 (open, blocked_by: Phase 1)
  └─ subtask-A (open)    ← shows as "ready" despite Phase 2 being blocked
  └─ subtask-B (open)    ← same
```

The subtasks pass all ready checks because they personally have no blockers — only their parent does.

###### Current Implementation

`internal/cli/query_helpers.go` defines:
- `ReadyConditions()` — 3 conditions: open status, no unclosed blockers, no open children
- `BlockedConditions()` — De Morgan inverse: open status AND (has unclosed blocker OR has open children)
- Both use simple EXISTS/NOT EXISTS subqueries against `dependencies` and `tasks` tables
- No ancestor traversal anywhere

###### Affected Code Paths

| File | Usage |
|------|-------|
| `internal/cli/query_helpers.go` | `ReadyConditions()`, `BlockedConditions()`, `ReadyWhereClause()` |
| `internal/cli/list.go` | Uses conditions for `--ready` and `--blocked` filters |
| `internal/cli/stats.go` | Uses `ReadyWhereClause()` for ready count |

##### Design Decisions

###### Blocker Type: Dependency Blockers Only

A task can be "blocked" in two senses:
1. **Dependency-blocked**: has unclosed entries in the `dependencies` table
2. **Children-blocked**: has open/in-progress children (parent waiting on child work)

Only **dependency blockers** on ancestors propagate down to affect descendant readiness. The "has open children" state is structural — it's the normal state for any parent whose children are the work to be done. A parent with open children isn't externally blocked; it's just waiting for its own subtasks to complete.

If children-blocked propagated, leaf tasks would never be ready since their parent always has open children (the leaf task itself).

###### Traversal Depth: Full Ancestor Chain

The ancestor check walks the **full ancestor chain** to the root, not just the immediate parent.

**Why not immediate parent only:** Intermediate grouping tasks create a gap. Example:
```
Phase 1 (open)
Phase 2 (open, blocked_by: Phase 1)
  └─ Group A (open, no own blockers)
      └─ subtask-X (open)
```
With immediate-parent-only, subtask-X checks Group A (not blocked), so subtask-X incorrectly appears ready despite Phase 2 being dependency-blocked.

**Why full chain is safe:** Recursive CTE is an established pattern in the codebase (`queryDescendantIDs()` in `list.go`). Ancestor chains are typically shallow (2-4 levels), so performance is a non-issue.

##### Implementation

###### New Helper: `ReadyNoBlockedAncestor()`

Add a new helper function in `internal/cli/query_helpers.go` that returns a NOT EXISTS subquery using a recursive CTE to walk the ancestor chain:

```sql
NOT EXISTS (
    WITH RECURSIVE ancestors(id) AS (
        SELECT parent FROM tasks WHERE id = t.id AND parent IS NOT NULL
        UNION ALL
        SELECT t2.parent FROM tasks t2
        JOIN ancestors a ON t2.id = a.id
        WHERE t2.parent IS NOT NULL
    )
    SELECT 1 FROM ancestors a
    JOIN dependencies d ON d.task_id = a.id
    JOIN tasks blocker ON blocker.id = d.blocked_by
    WHERE blocker.status NOT IN ('done', 'cancelled')
)
```

The CTE collects all ancestor IDs by walking `parent` pointers up to the root. The outer query then checks whether any ancestor has an unclosed dependency blocker.

###### Integration Points

**`ReadyConditions()`** — Add `ReadyNoBlockedAncestor()` as the 4th condition. All consumers (`list --ready`, `tick ready`, stats) automatically pick up the change since they compose from `ReadyConditions()`.

**`BlockedConditions()`** — Add the EXISTS inverse to the OR clause:
```sql
OR EXISTS (
    WITH RECURSIVE ancestors(id) AS ( ... )
    SELECT 1 FROM ancestors a
    JOIN dependencies d ON d.task_id = a.id
    JOIN tasks blocker ON blocker.id = d.blocked_by
    WHERE blocker.status NOT IN ('done', 'cancelled')
)
```

This follows the existing pattern: each concern is a separate helper, composed into conditions.

###### Edge Case: Closed Ancestors in the Chain

Don't stop walking at closed/cancelled ancestors. A closed parent with open children is an inconsistency that shouldn't occur in practice, and stopping early adds complexity for no benefit. The CTE walks unconditionally to the root.

##### Test Scenarios

| Scenario | Setup | Expected |
|----------|-------|----------|
| **Child of blocked parent** | Parent blocked by unclosed dep; child is open with no own blockers | Child is NOT ready, IS blocked |
| **Grandchild of blocked grandparent** | Grandparent blocked; intermediate parent has no own blockers; grandchild is open | Grandchild is NOT ready, IS blocked |
| **Intermediate grouping task** | Phase 2 blocked by Phase 1; Group A under Phase 2 has no own blockers; subtask under Group A | Subtask is NOT ready, IS blocked |
| **Ancestor blocker resolved** | Same as above, but blocker is done/cancelled | Descendant IS ready, NOT blocked |
| **No ancestors (root task)** | Root task with no parent, no own blockers, no open children | Remains ready (no ancestor check needed) |
| **Stats count consistency** | Mix of tasks with and without blocked ancestors | `ReadyWhereClause()` counts match `list --ready` output |

##### Dependencies

Prerequisites that must exist before implementation can begin:

###### Notes

- No external dependencies. All required infrastructure (recursive CTE support in SQLite, `query_helpers.go` helper pattern, `dependencies` and `tasks` tables) already exists.
- Implementation is self-contained within `internal/cli/query_helpers.go` and its test file.

## 2. Doctor Validation

---
topic: doctor-validation
status: concluded
type: feature
work_type: greenfield
date: 2026-01-24
sources:
  - name: doctor-command-validation
    status: incorporated
---

#### Specification: Doctor Validation

##### Overview

The `tick doctor` command provides diagnostics for the tick data store. It identifies issues that could cause tick to behave incorrectly - cache staleness, data corruption, invalid references, and constraint violations.

Doctor is **diagnostic only** - it reports problems and suggests remedies but never modifies data. A separate `tick rebuild` command handles cache repair.

###### Design Principles

1. **Report, don't fix** - Doctor diagnoses and suggests; user/agent decides what to run
2. **Human-focused** - Debugging tool for humans; agents don't need to parse diagnostic output
3. **Safety net** - Catches what slipped through write-time validation or got corrupted
4. **Run all checks** - Doctor completes all validations before reporting, never stops early

---

##### Validation Checks

Doctor performs two categories of checks: **errors** (things that break tick) and **warnings** (suspicious but allowed states).

###### Errors

| # | Check | Description |
|---|-------|-------------|
| 1 | Cache staleness | Hash mismatch between JSONL and SQLite cache |
| 2 | JSONL syntax errors | Lines tick cannot load as a task — malformed JSON, or valid JSON of the wrong shape or type |
| 3 | Duplicate IDs | Case-insensitive duplicate detection (tick-ABC123 = tick-abc123) |
| 4 | ID format violations | IDs not matching required format (prefix + 6 hex chars) |
| 5 | Orphaned parent references | Task references non-existent parent |
| 6 | Orphaned dependency references | Task depends on non-existent task |
| 7 | Self-referential dependencies | Task depends on itself |
| 8 | Dependency cycles | Circular dependency chains (A→B→C→A) |
| 9 | Child blocked_by parent | Deadlock condition - child can never become ready |

###### Warnings

| # | Check | Description |
|---|-------|-------------|
| 1 | Parent done with open children | Parent marked done while children still open - allowed but suspicious |

###### Out of Scope

Validation that tick's loader does not itself enforce (valid enum values, value ranges) happens at **write time**, not in doctor. Doctor judges every line the way every tick command loads it, so a passing doctor means every command can open the store; beyond that, it catches corruption and edge cases that slipped through.

---

##### Output Format

Doctor outputs **human-readable text only**. No TOON/JSON variants.

###### Rationale

- Doctor is a debugging/maintenance tool run by humans investigating issues
- Agents use normal operations (`ready`, `start`, `done`) - they don't parse diagnostics
- If cache is stale, tick auto-rebuilds on read anyway
- Adding structured output would be complexity for no real use case

###### Format

```
✓ Cache: OK
✓ JSONL syntax: OK
✓ ID uniqueness: OK
✗ Orphaned reference: tick-a1b2c3 references non-existent parent tick-missing
  → Manual fix required

1 issue found.
```

- `✓` for passing checks
- `✗` for failures, with details and suggested action
- Summary count at end

###### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All checks passed (no errors, warnings allowed) |
| 1 | One or more errors found |

###### Multiple Errors

Doctor lists each error individually. If there are 5 orphaned references, all 5 are shown with their specific details.

---

##### Fix Suggestions

Doctor includes actionable suggestions in its output:

| Issue | Suggestion |
|-------|------------|
| Cache stale | "Run `tick rebuild` to refresh cache" |
| All other errors | "Manual fix required" with explanation of what's wrong |

Doctor never modifies data - it only diagnoses and suggests.

---

##### The `tick rebuild` Command

A separate command for cache repair.

###### Purpose

Rebuilds the SQLite cache from the JSONL source of truth. Resolves cache staleness issues detected by doctor.

###### Behavior

- No flags or options - just rebuilds
- Deletes existing SQLite cache
- Regenerates from JSONL
- If no cache exists, creates it fresh (same as initial build)
- Reports success/failure

###### Usage

```
tick rebuild
```

Doctor suggests this command by name when cache issues are detected.

---

##### Dependencies

Prerequisites that must exist before implementation can begin:

###### Required

| Dependency | Why Blocked | What's Unblocked When It Exists |
|------------|-------------|--------------------------------|
| **tick-core** | Doctor validates against the data schema, ID format, hierarchy rules, and cache structure. Cannot implement validation logic without these definitions. | All doctor checks can be implemented |

###### Notes

- Doctor validates rules defined in tick-core (ID format, hierarchy constraints, cache hash mechanism)
- The `tick rebuild` command depends on the dual-write/freshness system defined in tick-core

##### Corrigenda

> **Corrigendum 2026-09-28** (from `oversized-text-field-bricks-the-store`): "Schema validation (field types, required fields, valid enum values) happens at **write time**, not in doctor" (Out of Scope) and "Malformed JSON lines that can't be parsed" (Errors #2) — corrected: doctor judges every line the way tick loads it, and a line it cannot load as a task — malformed JSON, or valid JSON of the wrong shape or type — fails the JSONL check with its line named; validation the loader does not enforce (enum membership, value ranges) stays at write time. The original boundary let doctor report a store healthy that no command could open: a valid-JSON line tick cannot load (`"priority":"high"`, `null`, `{}`, a whitespace-only line) makes every command fail, while doctor's JSON-only check passed it and its cache advice pointed at a `rebuild` that fails the same way. Agents run doctor to ask whether a store can be trusted, and only a loader-level check answers that.

## 3. Installation

---
topic: installation
status: concluded
type: feature
work_type: greenfield
date: 2026-01-25
sources:
  - name: installation-options
    status: incorporated
---

#### Specification: Installation

Tick is a Go CLI tool requiring cross-platform installation. This specification covers distribution methods, install script behavior, and platform support.

##### Overview

**Installation model**: Global installation (not per-project vendoring). Single version shared across projects, installed to user's system.

**Primary audience**:
- macOS developers (via Homebrew)
- Linux users and ephemeral environments like Claude Code for Web (via install script)
- CI/CD pipelines (via install script)

**Not in scope for v1**: Windows support. Go cross-compiles Windows binaries (available in releases), but no automated install path.

##### Installation Methods

Four methods supported, in priority order:

###### 1. Install Script (Primary)

`curl -fsSL https://raw.githubusercontent.com/{repo}/main/scripts/install.sh | bash`

- Primary documented method
- Essential for constrained/ephemeral environments
- Works anywhere with curl and bash

###### 2. Homebrew (macOS)

```bash
brew tap {owner}/tick
brew install tick
```

- Preferred method for macOS users
- Handles code signing automatically
- Manages updates via `brew upgrade tick`

###### 3. GitHub Releases

Pre-built binaries available for manual download. Required infrastructure (install script pulls from here).

Platforms:
- `darwin-amd64` (macOS Intel)
- `darwin-arm64` (macOS Apple Silicon)
- `linux-amd64`
- `linux-arm64`

###### Release Asset Naming

goreleaser convention: `{binary}_{version}_{os}_{arch}.tar.gz`

Assets per release:
- `tick_X.Y.Z_darwin_amd64.tar.gz`
- `tick_X.Y.Z_darwin_arm64.tar.gz`
- `tick_X.Y.Z_linux_amd64.tar.gz`
- `tick_X.Y.Z_linux_arm64.tar.gz`

Each archive contains the `tick` binary.

###### 4. go install

`go install github.com/{repo}@latest`

- For users with Go toolchain installed
- Available if repository is structured correctly (no extra work)

##### Install Script Behavior

The install script (`scripts/install.sh`) behaves differently based on platform.

###### Linux Behavior

1. **Detect platform**: `uname -s` (OS) and `uname -m` (architecture)
2. **Download binary**: Fetch latest release from GitHub for detected platform
3. **Install location**:
   - `/usr/local/bin` if writable (no sudo required)
   - `~/.local/bin` as fallback (user-writable, XDG-compliant)
4. **Overwrite existing**: Always install latest version (no version checking)
5. **No fallbacks**: If binary download fails, script fails. No `go install` or source build fallback.

###### macOS Behavior

1. **Check for Homebrew**: If `brew` command exists
   - Run `brew tap {owner}/tick && brew install tick`
   - Exit successfully
2. **No Homebrew**: Exit with message:
   ```
   Please install via Homebrew:
   brew tap {owner}/tick && brew install tick
   ```

macOS does not handle direct binary downloads. This avoids code signing complexity - Homebrew handles signing automatically.

###### Design Principles

- **Simple**: No complex fallback chains
- **Idempotent**: Safe to run multiple times
- **Fast**: Important for ephemeral environments where script runs at session start
- **Overwrite by default**: If user runs install script, they want latest version

###### Architecture Mapping

| `uname -m` | Asset arch |
|------------|------------|
| `x86_64` | `amd64` |
| `aarch64` | `arm64` |
| `arm64` | `arm64` |

##### Updates

**No self-update capability**. Tick does not include an `upgrade` command.

Updates handled via the original installation method:

| Method | Update Command |
|--------|---------------|
| Homebrew | `brew upgrade tick` |
| go install | `go install github.com/{repo}@latest` |
| Install script | Re-run the script |

###### Rationale

- Avoids complexity and security concerns of self-update
- Prevents conflicts with package managers (e.g., Homebrew user runs `tick upgrade`, now Homebrew is out of sync)
- Ephemeral environments don't need updates - each session starts fresh with latest version

##### Environment Matrix

| Environment | Method | Notes |
|-------------|--------|-------|
| macOS (developer machine) | Homebrew | Primary dev environment |
| Claude Code for Web | Install script | Run at session start |
| Linux server | Install script or go install | |
| CI/CD | Install script | Fast, idempotent |
| Windows | Manual download | Not officially supported |

##### Dependencies

Prerequisites that must exist before implementation can begin:

###### Required

| Dependency | Why Blocked | What's Unblocked When It Exists |
|------------|-------------|--------------------------------|
| **tick-core (buildable binary)** | Cannot distribute what doesn't exist. goreleaser needs a Go project that compiles successfully. | Install script, Homebrew formula, GitHub releases |

###### Notes

- Homebrew formula requires the GitHub repository URL and release asset naming conventions
- Install script implementation can be written before tick-core is complete, but cannot be tested end-to-end
- goreleaser configuration can be set up early with placeholder build targets

## 4. Migration

---
topic: migration
status: concluded
type: feature
work_type: greenfield
date: 2026-01-25
sources:
  - name: migration-subcommand
    status: incorporated
---

#### Specification: Migration

##### Specification

###### Overview

The `tick migrate` command imports task data from other tools. Users migrating to tick can preserve their existing work without manually recreating tasks.

**Primary use case**: Active project migration - preserving invested work when switching from another task management tool to tick.

**Design principles**:
1. **One-time import** - Not a sync tool; run once, delete the old system
2. **Append-only** - Adds to existing tick data, never modifies or deletes
3. **User responsibility** - Duplicates from re-running are the user's problem, not ours
4. **Plugin architecture** - Providers are self-contained; tick core doesn't manage credentials or source-specific logic

###### Command Interface

```
tick migrate --from <provider> [--dry-run] [--pending-only]
```

**Flags**:

| Flag | Required | Description |
|------|----------|-------------|
| `--from <provider>` | Yes | Source provider to import from (e.g., `beads`) |
| `--dry-run` | No | Preview what would be imported without writing |
| `--pending-only` | No | Import only non-completed tasks (default: import all) |

**Examples**:
```bash
tick migrate --from beads              # Import all tasks from beads
tick migrate --from beads --dry-run    # Preview import
tick migrate --from beads --pending-only  # Import only active work
```

###### Architecture

Migration uses a plugin/strategy pattern:

```
Provider → Normalize → Insert
   ↓           ↓          ↓
 (beads,    (contract)   (tick
  JIRA,                   data)
  Linear)
```

**Provider responsibilities**:
1. Connect to source system (if applicable)
2. Fetch data from source
3. Map source data to normalized format

**Core responsibilities**:
1. Receive normalized data from provider
2. Validate against contract
3. Insert into tick data store

The inserter is provider-agnostic - it receives normalized data and creates tick entries. This separation allows new providers to be added without modifying core logic.

**Authentication**: Delegated entirely to the provider. Each provider handles its own credential needs (env vars, prompts, config files). Tick core doesn't manage credentials. For file-based providers like beads, authentication isn't relevant.

**Normalized format**: The contract mirrors tick's task schema as defined in the tick-core specification. Providers map source data to this format; the core inserter validates and persists it.

###### Data Mapping

**Required fields**: Title only (bare minimum for a valid task)

**Mapping approach**:
- Map all available fields from source to tick equivalents
- Missing data uses sensible defaults or is left empty
- Extra source fields with no tick equivalent are discarded

Most project management tools have more data than tick needs, so mapping is typically straightforward - the challenge is what to omit, not what to invent.

###### Error Handling

**Strategy**: Continue on error, report failures at end.

When a task fails to import:
1. Log the failure with reason
2. Continue processing remaining tasks
3. Report summary at end

No rollback - successfully imported tasks remain even if others fail.

**Unknown provider**: If `--from` specifies an unrecognized provider, exit immediately with an error listing available providers:
```
Error: Unknown provider "xyz"

Available providers:
  - beads
```

###### Output Format

Print each task as imported, then summary at end. No verbose flag needed - show everything by default.

```
Importing from beads...
  ✓ Task: Implement login flow
  ✓ Task: Fix database connection
  ✓ Task: Add unit tests
  ✗ Task: Broken entry (skipped: missing title)

Done: 3 imported, 1 failed
```

**Failure detail** (shown after summary if any failures):
```
Failures:
- Task "foo": Missing required field
- Task "bar": Invalid date format
```

###### Initial Provider: Beads

The first supported provider is **beads** - a task format used in claude-technical-workflows.

**Characteristics**:
- File-based (local filesystem)
- No authentication required
- Source of the primary migration use case

Future providers (JIRA, Linear, etc.) can be added following the same plugin contract.

---

##### Dependencies

Prerequisites that must exist before implementation can begin:

###### Required

| Dependency | Why Blocked | What's Unblocked When It Exists |
|------------|-------------|--------------------------------|
| **tick-core** | Migration inserts tasks into tick's data store. Cannot create tasks without the data layer, schema, and write operations. | All migration functionality - provider plugins, normalization, and insertion. |

###### Notes

- The plugin architecture itself can be designed in parallel with tick-core
- Beads provider implementation requires understanding the beads format (separate from tick dependencies)

## 5. Task Removal

---
topic: task-removal
status: concluded
type: feature
work_type: greenfield
date: 2026-02-18
review_cycle: 3
finding_gate_mode: gated
sources:
  - name: task-removal
    status: incorporated
---

#### Specification: Task Removal

##### Specification

###### Overview

Tick needs a `remove` command that permanently deletes tasks from the JSONL source of truth. This is fundamentally different from `cancel` — cancel marks a task as resolved-but-present; remove erases it entirely.

**What "remove" means at the storage level:** True deletion. The task is filtered from the in-memory slice inside `Store.Mutate()`, and the JSONL file is rewritten without it. The SQLite cache is rebuilt from JSONL after each write, so the task disappears from both stores atomically. No new status is introduced — the task ceases to exist.

**Recovery:** The JSONL file is tracked in Git. Prior state is always recoverable from Git history. No in-app undo mechanism is needed.

###### Command Interface

**Command name:** `remove`

Consistent with the existing vocabulary of full English words (`cancel`, `reopen`, `create`). No alias (`rm`) — aliasing can be added later as a cross-cutting feature.

**Usage:**

```
tick remove <id> [<id>...]
tick remove <id> [<id>...] --force
```

**Arguments:**
- One or more task IDs as positional arguments (bulk removal supported)

**Flags:**
- `--force` / `-f` — Skip interactive confirmation. For AI agents, scripts, and non-interactive use.

**Status restrictions:** None. Tasks in any status (open, in_progress, done, cancelled) can be removed. The confirmation prompt is the safety gate, not status rules.

**Error handling:** If any provided task ID does not exist, the command fails with an error before any removal occurs. No partial removal — either all targets are valid and removed, or none are. This applies to both single and bulk invocations.

**Deduplication:** Duplicate task IDs in arguments are silently deduplicated. This also applies when a task appears both as an explicit argument and as a cascaded descendant of another argument — it is only removed once. The confirmation prompt shows the deduplicated set.

**No arguments:** If no task IDs are provided, the command returns an error: `"task ID is required. Usage: tick remove <id> [<id>...]"`. Consistent with existing commands like `start`, `done`, `cancel`.

###### Confirmation Behavior

Remove is the first truly destructive command in Tick. All other mutations are reversible status transitions. This warrants an interactive confirmation gate.

**Default (no `--force`):** Prompt the user before proceeding. The prompt surfaces the blast radius:
- The target task(s) being removed
- Any children that will be cascade-deleted
- Any dependency references that will be cleaned up from surviving tasks

The user must enter explicit confirmation to proceed.

**Accepted confirmation input:** Case-insensitive `y` or `yes`. Any other input (including empty/Enter) is treated as decline.

The prompt should indicate the expected input, e.g.: `Remove task tick-abc "My task title"? [y/N]`

The `[y/N]` convention (capital N) signals that "no" is the default — pressing Enter without typing anything aborts.

**Prompt destination:** The confirmation prompt (question text) is written to stderr, not stdout. This keeps stdout clean for piped and scripted usage. The abort message ("Aborted.") also goes to stderr, as already specified.

**Abort behavior:** If the user declines (any input other than `y`/`yes`, including empty Enter), the command prints "Aborted." to stderr and exits with code 1. No tasks are modified. This matches the convention where a non-zero exit signals "the requested operation did not happen."

**With `--force`:** Skip the confirmation prompt entirely. The caller accepts full responsibility. Designed for AI agents, scripts, and non-interactive pipelines.

**Design choice:** `--force` rather than TTY auto-detection. The app already injects `IsTTY` for format detection, but behavior should be explicit — always confirm unless `--force` is passed. This avoids ambiguity about what happens in non-TTY contexts without `--force`.

###### Cascade Deletion

Removing a task that has children triggers recursive cascade deletion — the parent and all descendants are removed together.

**Rationale:** Parent-child is structural. Children are subtasks *of* the parent (e.g., "Build authentication" → "Create login form", "Add JWT middleware"). Deleting the parent means abandoning the whole effort. Children don't make sense without the parent in most cases.

**Behavior:**
- When a task with children is removed, all descendants are collected recursively (children, grandchildren, etc.)
- All collected tasks are removed in a single atomic `Store.Mutate()` call
- Dependency references for *all* removed tasks (parent + descendants) are auto-cleaned from surviving tasks

**Confirmation prompt (without `--force`):** Explicitly lists all tasks that will be removed, e.g.:
> "This task has the following children: tick-def, tick-ghi. Removing it will also remove them. Are you sure?"

**With `--force`:** Cascade proceeds silently. The caller accepts the consequences.

**Bulk + cascade interaction:** When multiple IDs are passed and some trigger cascades, all targets plus their descendants are collected, deduplicated, and removed in a single atomic operation. The confirmation prompt lists the full deduplicated set.

**Removing a child:** When a child task (leaf node) is removed, no cleanup is needed on the parent side — the `Parent` field lives on the child, and parents don't maintain a children list. The child is simply filtered out.

###### Dependency Cleanup

When a task is removed, its ID is automatically scrubbed from all surviving tasks' `BlockedBy` arrays. This happens in the same atomic `Store.Mutate()` call as the deletion.

**Rationale:** Removing a blocker removes the block. A task that depended on the deleted task is no longer blocked by it. This is both user-friendly (no manual cleanup) and leaves data clean (no orphaned IDs).

**Implementation approach:** Inside the Mutate callback, after filtering out the removed task(s), iterate remaining tasks and strip all removed IDs from their `BlockedBy` arrays. Since the full task slice is already in memory, identifying affected tasks costs nothing extra.

**Output:** Report which surviving tasks had their dependencies updated. E.g.: "Updated dependencies on tick-def, tick-ghi". This applies to all three formatters (toon, pretty, JSON).

**Interaction with cascade:** When cascade deletion removes multiple tasks, dependency cleanup scrubs *all* removed IDs (parent + descendants) from surviving tasks' `BlockedBy` arrays.

###### Output

After a successful removal (with or without `--force`), output through the Formatter interface:

- Which task(s) were removed (IDs and titles)
- Which cascaded children were removed (if any)
- Which surviving tasks had dependency references cleaned up (if any)

Respects `--quiet` flag (suppress output) consistent with other commands.

All three formatters (toon, pretty, JSON) must support removal output. This will likely require a new Formatter method (e.g., `FormatRemoval`) since removal is a distinct operation — not a status transition, not a dependency change, not a generic message. It combines deletion confirmation with cascade and dependency cleanup reporting.

###### Help Text

The `remove` command's help entry should document:
- Usage and flags
- Cascade behavior (removing a parent removes all descendants)
- That Git history serves as the recovery mechanism for accidental removals

---

##### Dependencies

Prerequisites that must exist before implementation can begin:

###### Notes

- **No blocking dependencies.** The `remove` command builds on existing infrastructure: `Store.Mutate()` for atomic writes, the Task struct with `BlockedBy`/`Parent` fields, the Formatter interface, and the command dispatch pattern in `App.Run()`. All of these already exist.
- The only new concern is stdin reading for the interactive confirmation prompt, which is a standard Go capability (`bufio.Scanner` on `os.Stdin`) — no external dependency required.

## 6. Tick Core

---
topic: tick-core
status: concluded
type: feature
work_type: greenfield
date: 2026-02-09
sources:
  - name: project-fundamentals
    status: incorporated
  - name: data-schema-design
    status: incorporated
  - name: freshness-dual-write
    status: incorporated
  - name: id-format-implementation
    status: incorporated
  - name: hierarchy-dependency-model
    status: incorporated
  - name: cli-command-structure-ux
    status: incorporated
  - name: toon-output-format
    status: incorporated
  - name: tui
    status: incorporated
  - name: task-scoping-by-plan
    status: incorporated
---

#### Specification: Tick Core

##### Specification

###### Vision & Scope

###### What is Tick?

**Tick** is a minimal, deterministic task tracker designed for AI coding agents.

Agents need to know "what should I work on next?" without ambiguity. Existing tools either require manual sync steps (easy to forget, causing data loss), add complexity through daemons and hooks (hard to manage, harder to uninstall), or rely on markdown parsing (non-deterministic, token-expensive at scale).

Tick solves this with a simple model: JSONL as the git-committed source of truth, SQLite as an auto-rebuilding local cache, and a CLI that always returns deterministic results. No sync commands. No daemons. No hooks. Just `tick ready` to get the next task.

**Tagline**: "A minimal, deterministic task tracker for AI coding agents"

###### Primary User

AI coding agents (Claude Code and similar), with humans able to use it directly for oversight and manual intervention.

###### Core Values

- **Minimal simplicity** - Fewer features done well
- **Deterministic queries** - Same input, same output, always
- **Zero sync friction** - No manual sync commands ever

###### Primary Workflow

```
1. PROJECT SETUP
   Human runs: tick init
   Result: .tick/ directory created

2. PLANNING PHASE
   Planning agent (or human) creates tasks:
   - tick create "Setup authentication" --priority 1
   - tick create "Login endpoint" --blocked-by tick-a1b2
   Result: tasks.jsonl populated with work items

3. IMPLEMENTATION LOOP
   Implementation agent (or human):
   a) tick ready        → "What can I work on?"
   b) tick start <id>   → "I'm working on this"
   c) [does the work]
   d) tick done <id>    → "I finished this"
   e) Repeat until tick ready returns nothing

4. PROJECT COMPLETE
   All tasks done. .tick/ can be deleted or kept for reference.
```

**Key principle**: Tick is workflow-agnostic. It's a tracker, not a workflow engine. It doesn't care who creates tasks, who works them, or when. It just answers: "What's ready?"

###### Explicit Non-Goals

| Non-goal | Rationale |
|----------|-----------|
| Archive strategy | YAGNI - single file sufficient for v1 |
| Config file | YAGNI - hardcoded defaults are fine |
| Windows support | Not a priority - macOS and Linux first |
| Multi-agent coordination | Out of scope - single-agent focused |
| Real-time sync | Git is the sync mechanism |
| GUI/web interface | CLI only - agent-first means no visual UI |
| Plugin/extension system | Keep it simple - no hooks, no customization |

###### Success Criteria for v1

| Criterion | Description |
|-----------|-------------|
| All commands implemented | Every command works as specified |
| Fully tested | Comprehensive test coverage |
| Zero sync friction | No manual sync commands ever needed |
| Deterministic queries | Same input = same output, always |
| Sub-100ms operations | Fast enough to not notice |
| Clean uninstall | `rm -rf .tick` removes everything |
| Survives cache deletion | Delete .cache, everything auto-rebuilds |
| Git-friendly | Clean diffs, rare merge conflicts |
| Dogfooded | Used on real projects |

###### Overview

Tick uses a dual-storage architecture:

- **JSONL** (`tasks.jsonl`) - Source of truth, committed to git
- **SQLite** (`.tick/cache.db`) - Query cache, gitignored, auto-rebuilds

This design provides git-friendly storage (JSONL diffs cleanly, one line per task) with fast querying (SQLite for complex filters and joins).

**Key principle**: SQLite is a cache, not a peer. It can always be rebuilt from JSONL. Mismatches self-heal on next read.

###### Task Schema

Each task has 10 fields:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | string | Yes | Generated | Unique identifier (see ID Generation) |
| `title` | string | Yes | - | Brief description of the task |
| `status` | enum | Yes | `open` | One of: `open`, `in_progress`, `done`, `cancelled` |
| `priority` | integer | Yes | `2` | 0 (highest) to 4 (lowest) |
| `description` | string | No | `""` | Extended details, markdown supported |
| `blocked_by` | array | No | `[]` | IDs of tasks that block this one |
| `parent` | string | No | `null` | ID of parent task (organizational grouping) |
| `created` | datetime | Yes | Now | ISO 8601 timestamp |
| `updated` | datetime | Yes | Now | ISO 8601 timestamp, updated on any change |
| `closed` | datetime | No | `null` | ISO 8601 timestamp, set when status becomes `done` or `cancelled` |

###### Field Constraints

- **status**: Only valid transitions enforced by commands (e.g., can't `start` a `done` task)
- **priority**: Integer 0-4 only; invalid values rejected
- **blocked_by**: Must reference existing task IDs; self-references rejected; cycles rejected at write time
- **parent**: Must reference existing task ID; self-references rejected

###### Title and Description Limits

- **title**:
  - Required, non-empty
  - Maximum 500 characters
  - No newlines (single line only)
  - Leading/trailing whitespace trimmed

- **description**:
  - Optional
  - Capped at a finite character limit, enforced on every write path (value set by `oversized-text-field-bricks-the-store`)
  - Newlines and markdown allowed

###### Timestamp Format

All timestamps (`created`, `updated`, `closed`) use ISO 8601 format in UTC:
- Format: `YYYY-MM-DDTHH:MM:SSZ`
- Example: `2026-01-19T10:00:00Z`
- Always UTC (Z suffix), never local time with offset

###### Hierarchy Semantics

- **Parent/child is organizational** - A parent groups related tasks but does not block them
- **blocked_by controls workflow** - A task cannot be worked until its blockers are resolved
- **Cancelled tasks unblock dependents** - When a task is cancelled, tasks blocked by it become unblocked

###### ID Generation

###### Format

Task IDs follow the pattern: `{prefix}-{6 hex chars}`

- **Prefix**: `tick` (hardcoded, no configuration)
- **Random part**: 6 lowercase hexadecimal characters (e.g., `a3f2b7`)
- **Example**: `tick-a3f2b7`

###### Generation

- Use `crypto/rand` from Go stdlib (3 random bytes → 6 hex characters)
- Provides 16.7 million possible IDs (2^24)
- Cryptographically secure, no seeding required

###### Collision Handling

- On collision: retry silently up to 5 times
- If still colliding after 5 retries: return error
- Collisions are practically impossible at expected scale (hundreds to low thousands of tasks)

**Collision error message:**
```
Error: Failed to generate unique ID after 5 attempts - task list may be too large
```

###### Case Sensitivity

- IDs are case-insensitive for matching
- Normalize to lowercase on input (user can type `TICK-A3F2B7`, stored as `tick-a3f2b7`)
- Always output lowercase

###### File Locations & Formats

###### Directory Structure

```
.tick/
├── tasks.jsonl      # Source of truth (git tracked)
├── cache.db         # SQLite cache (gitignored)
└── lock             # Lock file for concurrent access
```

###### JSONL Format

One JSON object per line, no trailing commas, no array wrapper:

```jsonl
{"id":"tick-a1b2","title":"Task title","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
{"id":"tick-c3d4","title":"With all fields","status":"in_progress","priority":1,"description":"Details here","blocked_by":["tick-a1b2"],"parent":"tick-e5f6","created":"2026-01-19T10:00:00Z","updated":"2026-01-19T14:00:00Z"}
{"id":"tick-g7h8","title":"Completed task","status":"done","priority":1,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T16:00:00Z","closed":"2026-01-19T16:00:00Z"}
```

- Optional fields omitted when empty/null (not serialized as `null`)
- `updated` always present (set to `created` value initially)

###### SQLite Schema

```sql
CREATE TABLE tasks (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  priority INTEGER NOT NULL DEFAULT 2,
  description TEXT,
  parent TEXT,
  created TEXT NOT NULL,
  updated TEXT NOT NULL,
  closed TEXT
);

CREATE TABLE dependencies (
  task_id TEXT NOT NULL,
  blocked_by TEXT NOT NULL,
  PRIMARY KEY (task_id, blocked_by)
);

CREATE TABLE metadata (
  key TEXT PRIMARY KEY,
  value TEXT
);

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_priority ON tasks(priority);
CREATE INDEX idx_tasks_parent ON tasks(parent);
```

- `tasks` table mirrors JSONL fields (minus `blocked_by` which is normalized)
- `dependencies` table enables efficient joins for "ready" queries
- `metadata` table stores the JSONL content hash for freshness detection (key: `jsonl_hash`)

###### Synchronization

###### Freshness Detection

SHA256 hash-based comparison:

1. On every operation, read `tasks.jsonl` into memory
2. Compute SHA256 hash of file contents
3. Compare with hash stored in SQLite `metadata` table (key: `jsonl_hash`)
4. If mismatch: rebuild SQLite from JSONL data already in memory (no double-read)

**Trade-off accepted**: Always read full file on every operation. Acceptable for expected file sizes (<1MB, <10ms).

###### Cache Rebuild Triggers

Rebuild SQLite from JSONL when:

1. **Hash mismatch** - Primary case; JSONL was modified externally (e.g., git pull, manual edit)
2. **SQLite file missing** - Hash lookup fails; rebuild from scratch
3. **SQLite query errors** - Delete corrupted cache and rebuild
4. **Explicit command** - `tick rebuild` forces rebuild

The freshness check is the gatekeeper. Everything else is error recovery.

###### Write Operations

###### Mutation Flow

All mutations (create, update, delete) follow this sequence:

1. Acquire exclusive lock on `.tick/lock` file
2. Read `tasks.jsonl`, compute hash, check freshness (rebuild if stale)
3. Apply mutation in memory
4. Write complete new content to temp file
5. `fsync` temp file (flush to disk)
6. `os.Rename(temp, tasks.jsonl)` - atomic on POSIX
7. Update SQLite in single transaction: apply mutation + store new hash
8. Release lock

###### Atomic Rewrite Pattern

Always use full file rewrite for all operations (no append-only mode):

- **Why**: Atomic rename guarantees complete file or no change; append can leave incomplete lines
- **How**: Write to temp file → fsync → rename
- **Performance**: ~500 tasks × ~200 bytes = 100KB; trivially fast

###### Partial Failure Handling

**JSONL-first, SQLite is expendable**:

- If JSONL write succeeds but SQLite fails: log warning, continue
- Next read will detect hash mismatch and rebuild SQLite automatically
- Trade-off accepted: SQLite failure costs one rebuild on next read

The hash update is part of the same SQLite transaction as the data update:
```sql
BEGIN;
INSERT INTO tasks (...) VALUES (...);
UPDATE metadata SET jsonl_hash = 'new_hash';
COMMIT;
```

###### File Locking

Use `github.com/gofrs/flock` for file locking:

- **Write operations**: Exclusive lock (blocks other readers and writers)
- **Lock file**: `.tick/lock` (separate from data files)
- Prevents concurrent access corruption (learned from Taskwarrior issues)

###### Lock Timeout

- **Timeout**: 5 seconds default
- **On timeout**: Return error, do not proceed with operation

**Lock error message:**
```
Error: Could not acquire lock on .tick/lock - another process may be using tick
```

If lock is held for extended periods (e.g., crashed process), user can manually delete `.tick/lock` file.

###### Read Operations

###### Query Flow

All read operations (list, show, ready, etc.) follow this sequence:

1. Acquire shared lock on `.tick/lock` file (allows concurrent reads)
2. Read `tasks.jsonl`, compute hash, check freshness
3. If stale: rebuild SQLite from JSONL data already in memory
4. Query SQLite for requested data
5. Release lock

###### Shared Locking

- **Read operations**: Shared lock (allows other readers, blocks writers)
- Multiple concurrent reads are safe
- A pending write will wait for all readers to finish

###### Hierarchy & Dependency Model

Tick has two distinct relationship types:

- **Hierarchy** (`parent` field): "This task is part of a larger thing" - organizational grouping
- **Dependencies** (`blocked_by` array): "This task can't start until these are done" - workflow sequencing

###### Ready Query Logic

A task is "ready" (workable) only when ALL conditions are met:

1. Status is `open`
2. All `blocked_by` tasks are closed (`done` or `cancelled`)
3. Has no open children (no tasks where `parent` = this task AND status is `open` or `in_progress`)

**The leaf-only rule:**

| Condition | Appears in ready? |
|-----------|-------------------|
| No children (leaf task) | Yes, when unblocked |
| Has open children | No - children are the work |
| All children closed | Yes - becomes a leaf |

**Example:**
```
Auth System (tick-a1b2) - has children → NOT ready
├── Setup Sanctum (tick-c3d4) - done → NOT ready
├── Login endpoint (tick-e5f6) - open, unblocked, no children → READY
└── Logout endpoint (tick-k1l2, blocked_by tick-e5f6) - blocked → NOT ready
```

When Login and Logout are done, Auth System appears in ready (now a "leaf" with no open children). Agent can verify the epic is complete and mark it done.

###### Dependency Validation Rules

| Scenario | Allowed | Reason |
|----------|---------|--------|
| Child blocked_by parent | **No** | Creates deadlock with leaf-only rule |
| Parent blocked_by child | Yes | Valid "do children first" workflow |
| Sibling dependencies | Yes | Normal sequencing within an epic |
| Cross-hierarchy dependencies | Yes | Normal dependency behavior |
| Circular dependencies | **No** | Cycle detection catches these |

**Validation timing**: Validate when adding/modifying `blocked_by` - reject invalid dependencies at write time, before persisting to JSONL.

**Error messages:**
```
Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-epic
       (would create unworkable task due to leaf-only ready rule)

Error: Cannot add dependency - creates cycle: tick-a → tick-b → tick-c → tick-a
```

###### Parent Context in Output

When displaying a task that has a parent (e.g., `tick show`):

- Include parent ID + title (not full description)
- Agent can `tick show <parent_id>` for full context if needed
- Tasks should be self-contained; parent context is a safety net, not required

###### Edge Cases

**Deep nesting**: No depth limit. The leaf-only rule handles naturally - only the deepest incomplete tasks appear in ready.

**Orphaned children** (parent reference points to non-existent task):
- Task remains valid, treated as root-level task
- `tick doctor` flags: "tick-child references non-existent parent tick-deleted"
- No auto-fix - human/agent decides whether to remove parent reference

**Parent done before children**:
- Valid state - children remain workable
- May indicate planning issue, but not enforced by tool
- `tick doctor` warns: "tick-epic is done but has open children"

###### CLI Commands

###### Command Reference

| Command | Action |
|---------|--------|
| `tick init` | Initialize .tick/ directory in current project |
| `tick create "<title>"` | Create a new task |
| `tick update <id>` | Update task fields |
| `tick start <id>` | Mark task as in-progress |
| `tick done <id>` | Mark task as completed successfully |
| `tick cancel <id>` | Mark task as cancelled (not completed) |
| `tick reopen <id>` | Reopen a closed task |
| `tick list` | List tasks with optional filters |
| `tick show <id>` | Show detailed task information |
| `tick ready` | Alias for `list --ready` - show workable tasks |
| `tick blocked` | Alias for `list --blocked` - show blocked tasks |
| `tick dep add <task_id> <blocked_by_id>` | Add dependency (task depends on blocked_by) |
| `tick dep rm <task_id> <blocked_by_id>` | Remove dependency |
| `tick stats` | Show task statistics |
| `tick doctor` | Run diagnostics and validation |
| `tick rebuild` | Force rebuild SQLite cache from JSONL |

###### Init Command

`tick init` creates the `.tick/` directory structure:
- Creates `.tick/` directory
- Creates empty `tasks.jsonl` file
- SQLite cache is created on first operation (not at init)

**If `.tick/` already exists**: Error with message "Tick already initialized in this directory"

**Output**: Confirmation message showing path initialized.

###### Create Command Options

```bash
tick create "<title>" [options]

Options:
  --priority <0-4>           Priority level (default: 2)
  --blocked-by <id,id,...>   Comma-separated dependency IDs
  --blocks <id,id,...>       Tasks this task blocks (updates their blocked_by)
  --parent <id>              Parent task ID
  --description "<text>"     Extended description
```

###### Update Command

```bash
tick update <id> [options]

Options:
  --title "<text>"           New title
  --description "<text>"     New description (use "" to clear)
  --priority <0-4>           New priority level
  --parent <id>              New parent task (use "" to clear)
  --blocks <id,id,...>       Tasks this task blocks (updates their blocked_by)
```

At least one option required. Cannot change `id`, `status`, `created`, or `blocked_by` (use dedicated commands for those).

**Note on `--blocks`**: This is the inverse of `--blocked-by`. Setting `--blocks tick-abc` on task T adds T to tick-abc's `blocked_by` array. The data model remains unchanged - only `blocked_by` is stored.

**Output**: Full task details (same format as `tick show`), TTY-aware.

###### List Command Options

```bash
tick list [options]

Options:
  --ready        Show only ready tasks (unblocked, no open children)
  --blocked      Show only blocked tasks
  --status <s>   Filter by status (open, in_progress, done, cancelled)
  --priority <p> Filter by priority (0-4)
  --parent <id>  Scope to descendants of this task (recursive)
```

The `--parent` flag also applies to `tick ready` and `tick blocked` since they are aliases for `tick list --ready` and `tick list --blocked` respectively.

###### Blocked Query Logic

A task appears in `tick blocked` when:
- Status is `open`
- AND either:
  - Has at least one `blocked_by` task that is not closed (`done` or `cancelled`), OR
  - Has at least one open child (status `open` or `in_progress`)

In other words: `blocked` = open tasks that are not `ready`.

###### Parent Scoping

The `--parent <id>` flag restricts queries to descendants of the specified task. This enables plan-level scoping — create a top-level parent task per plan, add plan tasks as children, then filter queries to that subtree.

**Behavior:**
- Collects all descendants of the specified parent (recursive, not just direct children)
- Applies existing filters (ready, blocked, status, priority) within that set
- The parent task itself is excluded from results naturally (it has open children, so the leaf-only rule filters it out)

**`--parent` is a pre-filter**: It narrows which tasks are considered. The leaf-only and blocked-by rules are post-filters that determine which of those are workable. They compose cleanly with no special cases.

**Example:**
```
Plan: Auth System (tick-p1a2)
├── Login endpoint (tick-e5f6)
│   ├── Validation (tick-g7h8)      ← leaf
│   └── Rate limiting (tick-i9j0)   ← leaf
└── Logout endpoint (tick-k1l2)     ← leaf
```

```bash
tick ready --parent tick-p1a2    # returns tick-g7h8, tick-i9j0, tick-k1l2
tick ready --parent tick-e5f6    # returns tick-g7h8, tick-i9j0
tick ready                       # all ready tasks across all plans
```

**Implementation**: Recursive CTE in SQLite to collect all descendant IDs, then apply existing query filters within that set.

###### Dependency Management

**At creation time:**
```bash
tick create "Login endpoint" --blocked-by tick-a1b2
tick create "Complex task" --blocked-by tick-a1b2,tick-x9y8
```

**Later modifications:**
```bash
tick dep add tick-c3d4 tick-a1b2    # c3d4 now depends on a1b2
tick dep rm tick-c3d4 tick-a1b2     # remove that dependency
```

**Argument order**: Task first, dependency second. Matches the `--blocked-by` flag pattern and reads naturally: "Add to c3d4 a dependency on a1b2."

###### Task Statuses

| Status | Meaning |
|--------|---------|
| `open` | Not started |
| `in_progress` | Being worked on |
| `done` | Completed successfully |
| `cancelled` | Closed without completion |

###### Status Transitions

| Command | From Status | To Status |
|---------|-------------|-----------|
| `start` | `open` | `in_progress` |
| `done` | `open`, `in_progress` | `done` |
| `cancel` | `open`, `in_progress` | `cancelled` |
| `reopen` | `done`, `cancelled` | `open` |

Invalid transitions return an error (e.g., `tick start` on a `done` task).

Note: `reopen` also clears the `closed` timestamp.

###### Error Handling

**Exit codes:**
- `0` = success
- `1` = any error

Agents parse error output for specifics; they don't need to memorize exit code meanings.

**Error output:**
- All errors go to stderr
- Plain text format (human-readable)
- Non-zero exit code signals failure

**Example:**
```
$ tick show tick-xyz
Error: Task 'tick-xyz' not found
$ echo $?
1
```

###### Verbosity Flags

| Flag | Effect |
|------|--------|
| `--quiet` / `-q` | Suppress non-essential output |
| `--verbose` / `-v` | More detail (useful for debugging) |

###### Rebuild Command

`tick rebuild` forces a complete rebuild of the SQLite cache from JSONL, bypassing the freshness check. Use when:
- SQLite appears corrupted
- Debugging cache issues
- After manual JSONL edits (though freshness check should handle this automatically)

Output: Confirmation message showing tasks rebuilt.

###### Mutation Command Output

**`tick create`**: Outputs full task details (same format as `tick show`), TTY-aware.
- With `--quiet`: Outputs only the task ID

**`tick update`**: Outputs full task details (same format as `tick show`), TTY-aware.
- With `--quiet`: Outputs only the task ID

**`tick start/done/cancel/reopen`**: Outputs task ID and status transition
```
tick-a3f2b7: open → in_progress
```
- With `--quiet`: No output

**`tick dep add/rm`**: Outputs confirmation
```
Dependency added: tick-c3d4 blocked by tick-a1b2
Dependency removed: tick-c3d4 no longer blocked by tick-a1b2
```
- With `--quiet`: No output

###### Output Formats

###### Format Selection (TTY Detection)

Output format is determined automatically based on whether stdout is a terminal:

| Condition | Output Format |
|-----------|---------------|
| No TTY (pipe/redirect) | TOON (default for agents) |
| TTY (terminal) | Human-readable table |
| `--toon` flag | Force TOON |
| `--pretty` flag | Force human-readable |
| `--json` flag | Force JSON |

**Why this works**: When agents run commands via Bash tool, stdout is a pipe (not TTY). Agents get TOON automatically without flags. Humans at terminals get readable output naturally.

This is a well-established Unix pattern used by `ls` (colors), `git` (pager), `grep` (colors).

###### TOON Format (Agent Output)

TOON (Token-Oriented Object Notation) is optimized for AI agent consumption - 30-60% token savings over JSON while improving parsing accuracy.

**Structure**: Multi-section approach for complex data. Each section has its own schema header.

**Example - `tick list` output:**
```
tasks[2]{id,title,status,priority}:
  tick-a1b2,Setup Sanctum,done,1
  tick-c3d4,Login endpoint,open,1
```

**Example - `tick show` output:**
```
task{id,title,status,priority,parent,created,updated}:
  tick-a1b2,Setup Sanctum,in_progress,1,tick-e5f6,2026-01-19T10:00:00Z,2026-01-19T14:30:00Z

blocked_by[2]{id,title,status}:
  tick-c3d4,Database migrations,done
  tick-g7h8,Config setup,in_progress

children[0]{id,title,status}:

description: "Full task description here.\nCan be multiple lines."
```

**Example - `tick stats` output:**
```
stats{total,open,in_progress,done,cancelled,ready,blocked}:
  47,12,3,28,4,8,4

by_priority[5]{priority,count}:
  0,2
  1,8
  2,25
  3,7
  4,5
```

**Principles:**
1. Each section has its own schema header - self-documenting
2. Related entities include context (title, status), not just IDs
3. Long text fields are a single TOON-quoted value on the section they belong to
4. Empty arrays shown with zero count: `blocked_by[0]{id,title,status}:`
5. Sections omitted only if the field doesn't exist (vs empty)

**Empty results:**
```
tasks[0]{id,title,status,priority}:
```
Zero count with schema header, no data rows.

###### Human-Readable Format (Terminal Output)

Simple aligned columns. No borders, no colors, no icons. Raw `fmt.Print` output.

**List output:**
```
ID          STATUS       PRI  TITLE
tick-a1b2   done         1    Setup Sanctum
tick-c3d4   in_progress  1    Login endpoint
```

**Show output:**
```
ID:       tick-c3d4
Title:    Login endpoint
Status:   in_progress
Priority: 1
Created:  2026-01-19T10:00:00Z

Blocked by:
  tick-a1b2  Setup Sanctum (done)

Description:
  Implement the login endpoint with validation...
```

**Stats output:**
```
Total:       47

Status:
  Open:        12
  In Progress:  3
  Done:        28
  Cancelled:    4

Workflow:
  Ready:        8
  Blocked:      4

Priority:
  P0 (critical):  2
  P1 (high):      8
  P2 (medium):   25
  P3 (low):       7
  P4 (backlog):   5
```

**Design philosophy**: Minimalist and clean. Human output is secondary to agent output - no TUI libraries, no interactivity.

**Empty results:**
```
No tasks found.
```
Simple message, no headers.

###### JSON Format

Available via `--json` flag for compatibility and debugging. Standard JSON output.

##### Dependencies

Prerequisites that must exist before implementation can begin:

###### Required

None. This is the foundational data layer that other specifications depend on.

###### External Libraries

| Library | Purpose |
|---------|---------|
| `github.com/gofrs/flock` | File locking for concurrent access safety |
| `github.com/mattn/go-sqlite3` | SQLite driver for Go |
| `github.com/toon-format/toon-go` | TOON encoding/decoding for agent output (handles escaping per TOON spec §7.1) |

###### Optional Libraries

| Library | Purpose |
|---------|---------|
| `github.com/natefinch/atomic` | Atomic file writes (alternative to hand-rolling with `os.Rename`) |

###### Notes

- All other dependencies are Go stdlib (`crypto/rand`, `os`, `encoding/json`, `crypto/sha256`)
- This specification can be implemented independently before CLI or workflow features
- Optional libraries will be assessed at implementation for reliability, maintenance, and community support

##### Corrigenda

> **Corrigendum 2026-09-20** (from `free-text-round-trip`): "Long text fields get their own unstructured sections" (TOON output principle 3), and the worked `tick show` example that printed the description as an indented raw block — corrected: a long text field is a single TOON-quoted value on the section it belongs to, so the example now reads `description: "Full task description here.\nCan be multiple lines."`. The unstructured section is the form that made tick's toon output unparseable: it is hand-assembled rather than produced by the encoder, and it obliges a reader to know an indentation-stripping rule the document never states. The quoted value is what the library emits from a Go string, and it is the rule note text already obeyed, so all free text in the output now decodes the same way. The `task{…}` and `stats{…}` scope headers in the same passage are left standing: they are superseded by `free-text-round-trip` §5 rather than corrected here.

> **Corrigendum 2026-09-28** (from `oversized-text-field-bricks-the-store`): "No maximum length" (Task Schema → Title and Description Limits, description) — corrected: the description is capped at a finite character limit, enforced on every write path, its value set by `oversized-text-field-bricks-the-store`. The unbounded description was never unbounded in practice: the store's reader stopped at any task line of 65,536 bytes (Go's default `bufio.Scanner` limit, never chosen), so a description large enough to push its line past that was accepted, written, and then made every command — `rebuild` included — fail until `tasks.jsonl` was edited by hand. That work removes the reader's ceiling, which is what makes the store safe; the cap keeps free text finite as hygiene, the same shape as the existing title and note caps.
