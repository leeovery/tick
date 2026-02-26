<div align="center">

# Tick

**Task management for agentic engineering**

A CLI that gives AI agents deterministic, token-efficient task tracking,
<br>without the complexity of full project management tools.

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://go.dev)

[Install](#install) · [Quick Start](#quick-start) · [Commands](#commands) · [Output Formats](#output-formats) · [Why Tick?](#why-tick)

</div>

---

Tick is a lightweight CLI for tracking tasks, dependencies, and status transitions inside your project. It stores everything in a plain JSONL file (human-readable, git-friendly) with a SQLite cache for fast queries. Run `tick init`, and you're set.

It's built to be used by AI agents as much as by humans. Output auto-switches between a token-efficient format for agents and clean tables for terminals, so the same commands work in both contexts.

## Why Tick?

Claude, Cursor, and other AI coding agents need a way to track tasks across sessions. The built-in approaches have problems:

- **TodoWrite / in-context lists** — lost between sessions, no persistence, no dependency tracking
- **Markdown files** — no structure, agents parse them inconsistently, output is verbose
- **Beads / full PM tools** — too much complexity for a coding session, heavy overhead

Tick sits in between: structured enough for agents to reason about reliably, simple enough that it doesn't get in the way.

### Key differences

| | Tick | TodoWrite | Markdown | Beads |
|---|---|---|---|---|
| Persists across sessions | Yes | No | Yes | Yes |
| Dependencies & blockers | Yes | No | No | Yes |
| Token-efficient output | TOON (30-60% savings) | N/A | No | No |
| Deterministic format | Yes | Varies | No | Yes |
| Complexity | Low | Minimal | Minimal | High |
| Setup | `tick init` | None | None | Config |

## Install

**macOS**

```bash
brew install leeovery/tools/tick
```

**Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/leeovery/tick/main/scripts/install.sh | bash
```

**Go**

```bash
go install github.com/leeovery/tick/cmd/tick@latest
```

## Quick Start

```bash
tick init                           # create .tick/ in your project
tick create "Build auth module"     # create a task
tick create "Write tests" --priority 1 --blocked-by tick-a1b2
tick list                           # see all tasks
tick ready                          # tasks with no blockers
tick start tick-a1b2                # open → in_progress
tick done tick-a1b2                 # in_progress → done
```

## Commands

### `init`

Initialize a new tick project in the current directory. Creates a `.tick/` directory with an empty `tasks.jsonl` file.

```bash
tick init
```

### `create`

Create a new task. Returns the full task detail on success.

```bash
tick create <title> [flags]
```

| Flag | Type | Default | Description |
|---|---|---|---|
| `--priority` | `0-4` | `2` | `0` critical, `1` high, `2` medium, `3` low, `4` backlog |
| `--description` | string | | Task description (supports multi-line) |
| `--parent` | ID | | Make this a subtask of another task |
| `--blocked-by` | IDs | | Comma-separated list of tasks this depends on |
| `--blocks` | IDs | | Comma-separated list of tasks this blocks |

```bash
tick create "Build auth module"
tick create "Critical fix" --priority 0
tick create "Write tests" --blocked-by tick-a1b2,tick-c3d4
tick create "Login endpoint" --parent tick-a1b2
```

### `list`

List tasks with optional filters. Results are sorted by priority (ascending), then creation date.

```bash
tick list [flags]
```

| Flag | Type | Default | Description |
|---|---|---|---|
| `--status` | string | | Filter by status: `open`, `in_progress`, `done`, `cancelled` |
| `--priority` | `0-4` | | Filter by priority level |
| `--parent` | ID | | Show descendants of a task |
| `--ready` | bool | `false` | Show only ready tasks (open, no unresolved blockers, no open children, no dependency-blocked ancestor) |
| `--blocked` | bool | `false` | Show only blocked tasks (open with unresolved blockers, open children, or dependency-blocked ancestor) |

`--ready` and `--blocked` are mutually exclusive.

```bash
tick list                           # all tasks
tick list --status open             # filter by status
tick list --priority 0              # only critical tasks
tick list --parent tick-a1b2        # descendants of a task
```

### `ready`

Alias for `tick list --ready`. Shows tasks that are open, have no unresolved blockers, no open children, and no dependency-blocked ancestor.

```bash
tick ready
```

### `blocked`

Alias for `tick list --blocked`. Shows tasks that are open but waiting on dependencies, have open children, or have an ancestor with unresolved blockers.

```bash
tick blocked
```

### `show`

Display full detail for a single task, including blockers, children, and description.

```bash
tick show <task-id>
```

### `update`

Modify one or more fields on an existing task. At least one flag is required.

```bash
tick update <task-id> [flags]
```

| Flag | Type | Description |
|---|---|---|
| `--title` | string | Set a new title |
| `--description` | string | Set or replace the description |
| `--clear-description` | bool | Remove the description (mutually exclusive with `--description`) |
| `--priority` | `0-4` | Change priority level |
| `--parent` | ID | Set or change the parent task (pass empty string to clear) |
| `--blocks` | IDs | Comma-separated list of tasks this blocks |

```bash
tick update tick-a1b2 --title "Revised title" --priority 1
tick update tick-a1b2 --parent tick-c3d4
```

### `start` / `done` / `cancel` / `reopen`

Transition a task between statuses.

```bash
tick start  <task-id>               # open → in_progress
tick done   <task-id>               # in_progress → done
tick cancel <task-id>               # any → cancelled
tick reopen <task-id>               # done/cancelled → open
```

`done` and `cancel` set a closed timestamp. `reopen` clears it.

### `remove`

Permanently delete one or more tasks. Removing a parent cascades to all descendants. Dependency references on surviving tasks are automatically cleaned up.

```bash
tick remove <id> [<id>...] [flags]
```

| Flag | Type | Description |
|---|---|---|
| `--force, -f` | bool | Skip confirmation prompt |

```bash
tick remove tick-a1b2                  # remove with confirmation
tick remove tick-a1b2 tick-c3d4 -f     # remove multiple, skip prompt
```

Since `tasks.jsonl` is tracked in git, accidental removals can be recovered from history.

### `dep`

Manage task dependencies. Tick validates all dependency changes and prevents cycles, self-references, and children blocked by their own parent.

```bash
tick dep add <task-id> <blocked-by-id>
tick dep rm  <task-id> <blocked-by-id>
```

```bash
tick dep add tick-a1b2 tick-c3d4    # tick-a1b2 is now blocked by tick-c3d4
tick dep rm  tick-a1b2 tick-c3d4    # remove that dependency
```

### `stats`

Show aggregate task counts grouped by status, workflow state (ready/blocked), and priority.

```bash
tick stats
```

### `doctor`

Run diagnostic checks against your task data. Read-only, never modifies data.

```bash
tick doctor
```

Checks for: JSONL syntax errors, invalid IDs, duplicates, orphaned references, self-referential dependencies, dependency cycles, parent/child constraint violations, and cache staleness.

### `rebuild`

Force a full SQLite cache rebuild from the JSONL source file, bypassing the freshness check.

```bash
tick rebuild
```

### `version`

Print the tick version and exit.

```bash
tick version
```

### `help`

Show usage information. With no argument, lists all commands and global flags. With a command name, shows detailed help including flags.

```bash
tick help                           # list all commands
tick help create                    # detailed help for create
tick help --all                     # full reference of all commands and flags
tick create --help                  # same as tick help create
tick -h                             # same as tick help
```

### `migrate`

Import tasks from external tools.

```bash
tick migrate --from <provider> [flags]
```

| Flag | Type | Default | Description |
|---|---|---|---|
| `--from` | string | *required* | Provider to import from (currently: `beads`) |
| `--dry-run` | bool | `false` | Preview what would be imported without persisting |
| `--pending-only` | bool | `false` | Only import tasks not yet migrated |

```bash
tick migrate --from beads
tick migrate --from beads --dry-run --pending-only
```

## Output Formats

Tick auto-detects the context and picks the right format:

| Context | Default format | Override |
|---|---|---|
| Terminal (TTY) | `--pretty` | `--toon`, `--json` |
| Pipe / agent | `--toon` | `--pretty`, `--json` |

<table>
<tr>
<td>

**Agent / pipe** (TOON)
```
$ tick list
tasks[3]{id,title,status,priority}:
  tick-a1b2,Auth middleware,in_progress,1
  tick-f3e4,Write tests,open,2
  tick-d5c6,Update docs,open,3
```

</td>
<td>

**Terminal** (Pretty)
```
$ tick list
ID          STATUS        PRI   TITLE
tick-a1b2   in_progress   1     Auth middleware
tick-f3e4   open          2     Write tests
tick-d5c6   open          3     Update docs
```

</td>
</tr>
</table>

### TOON (Token-Oriented Object Notation)

Designed for AI consumption. Schema is declared once in the header; rows are compact CSV-like lines. Uses 30-60% fewer tokens than equivalent JSON.

```
tasks[2]{id,title,status,priority}:
  tick-a1b2,Setup auth,done,1
  tick-c3d4,Login endpoint,open,1
```

```
task{id,title,status,priority,created,updated}:
  tick-a1b2,Setup auth,in_progress,1,"2026-01-19T10:00:00Z","2026-01-19T14:30:00Z"

blocked_by[1]{id,title,status}:
  tick-c3d4,Database migrations,done

children[0]{id,title,status}:

description:
  Full task description here.
  Can be multiple lines.
```

### Pretty

Clean aligned columns for terminals. No borders, no colors, no icons.

```
ID          STATUS        PRI   TITLE
tick-a1b2   in_progress   1     Setup auth
tick-c3d4   open          1     Login endpoint
```

### JSON

Standard 2-space indented JSON with snake_case keys.

```json
[
  {
    "id": "tick-a1b2",
    "title": "Setup auth",
    "status": "in_progress",
    "priority": 1
  }
]
```

## Storage

Tick stores data in a `.tick/` directory at your project root:

- `tasks.jsonl` — append-only source of truth (one JSON object per line, human-editable, git-friendly)
- `cache.db` — SQLite cache (auto-rebuilt when JSONL changes, do not commit)
- `lock` — file lock for safe concurrent access

Add to `.gitignore`:

```
.tick/cache.db
.tick/lock
```

## Global Flags

```
--help, -h        Show help (tick --help or tick <command> --help)
--quiet, -q       Minimal output (IDs only where applicable)
--verbose, -v     Debug logging to stderr
--toon            Force TOON format
--pretty          Force pretty format
--json            Force JSON format
```

## License

MIT
