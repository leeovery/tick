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

### Creating tasks

```bash
tick create "Task title"
tick create "Critical fix" --priority 0
tick create "Subtask" --parent tick-a1b2
tick create "Blocked work" --blocked-by tick-a1b2,tick-c3d4
tick create "Blocking work" --blocks tick-f5e6
tick create "With details" --description "Multi-line\ndescription"
```

Priority levels: `0` critical, `1` high, `2` medium (default), `3` low, `4` backlog.

### Listing and filtering

```bash
tick list                           # all tasks
tick list --status open             # filter by status
tick list --priority 0              # filter by priority
tick list --parent tick-a1b2        # show descendants
tick ready                          # open, no blockers, no open children
tick blocked                        # open, has unresolved blockers
```

### Viewing details

```bash
tick show tick-a1b2                 # full detail with blockers and children
tick stats                          # counts by status, priority, workflow
```

### Status transitions

```bash
tick start tick-a1b2                # open → in_progress
tick done tick-a1b2                 # in_progress → done
tick cancel tick-a1b2               # any → cancelled
tick reopen tick-a1b2               # done/cancelled → open
```

### Updating tasks

```bash
tick update tick-a1b2 --title "New title"
tick update tick-a1b2 --priority 1
tick update tick-a1b2 --description "Updated description"
tick update tick-a1b2 --parent tick-c3d4
```

### Dependencies

```bash
tick dep add tick-a1b2 tick-c3d4    # tick-a1b2 blocked by tick-c3d4
tick dep rm tick-a1b2 tick-c3d4     # remove dependency
```

Tick prevents dependency cycles, self-references, and children blocked by their own parent.

### Maintenance

```bash
tick doctor                         # diagnose data issues
tick rebuild                        # force SQLite cache rebuild
tick migrate --from beads           # import from external tools
tick migrate --from beads --dry-run # preview without changes
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
- `.store` — SQLite cache (auto-rebuilt when JSONL changes, do not commit)
- `lock` — file lock for safe concurrent access

Add to `.gitignore`:

```
.tick/.store
.tick/lock
```

## Global Flags

```
--quiet, -q       Minimal output (IDs only where applicable)
--verbose, -v     Debug logging to stderr
--toon            Force TOON format
--pretty          Force pretty format
--json            Force JSON format
```

## License

MIT
