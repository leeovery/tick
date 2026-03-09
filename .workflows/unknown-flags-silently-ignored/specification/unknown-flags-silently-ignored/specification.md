# Specification: Unknown Flags Silently Ignored

## Specification

## Problem

Unknown flags passed to any `tick` command are silently ignored. Arguments starting with `-` that aren't recognised are stripped without warning, which can mislead users into thinking a flag had effect.

**Example:** `tick dep add tick-aaa --blocks tick-bbb` silently ignores `--blocks` (only valid on `create`/`update`) and treats it as `tick dep add tick-aaa tick-bbb`.

## Requirements

1. All commands must reject unrecognised flags with a clear error message
2. Known global flags (`--quiet`, `--verbose`, `--toon`, `--pretty`, `--json`, `--help`) must not be rejected by command-level validation
3. The fix must cover all commands: `create`, `update`, `list`, `show`, `dep add/remove`, `remove`, `note add/remove`, `start`, `done`, `cancel`, `reopen`, `stats`, `doctor`, `init`

## Design

### Approach: Command-Exported Flags + Central Validation

Each command exports its set of valid flags. The dispatcher validates args against that set before invoking the handler. Flag knowledge stays with the command, validation logic is written once centrally.

### Flow

1. `parseArgs()` strips global flags, identifies subcommand, collects `rest`
2. Look up valid flags for the identified command, validate `rest` against them — error on any unknown flag
3. Dispatch to command handler (which can now assume all flags are valid)

### Rationale

- **Inline validation** (rejected): replacing the silent skip with `fmt.Errorf` in every command duplicates validation logic and requires every new command to remember to add it
- **Central flag registry** (rejected): a `map[string][]string` in `app.go` creates a second place to maintain flag knowledge alongside the parsers
- **Command-exported flags + central validation** (chosen): flag knowledge lives with the command, validation written once. No duplication of either flag definitions or validation logic

### Cleanup

The existing `strings.HasPrefix(arg, "-")` silent-skip logic in each command's parser can be removed — unknown flags are caught before the handler is called.

## Command Flag Inventory

### Global Flags

Accepted by all commands, stripped before dispatch:

| Flag | Short |
|------|-------|
| `--quiet` | `-q` |
| `--verbose` | `-v` |
| `--toon` | — |
| `--pretty` | — |
| `--json` | — |
| `--help` | `-h` |

### Per-Command Flags

| Command | Accepted Flags |
|---------|---------------|
| `init` | *(none)* |
| `create` | `--priority`, `--description`, `--blocked-by`, `--blocks`, `--parent`, `--type`, `--tags`, `--refs` |
| `update` | `--title`, `--description`, `--priority`, `--parent`, `--clear-description`, `--type`, `--clear-type`, `--tags`, `--clear-tags`, `--refs`, `--clear-refs`, `--blocks` |
| `list` | `--ready`, `--blocked`, `--status`, `--priority`, `--parent`, `--type`, `--tag`, `--count` |
| `show` | *(none)* |
| `start` | *(none)* |
| `done` | *(none)* |
| `cancel` | *(none)* |
| `reopen` | *(none)* |
| `ready` | same as `list` minus `--ready` |
| `blocked` | same as `list` minus `--blocked` |
| `dep add` | *(none)* |
| `dep rm` | *(none)* |
| `note add` | *(none)* |
| `note remove` | *(none)* |
| `remove` | `--force` / `-f` |
| `stats` | *(none)* |
| `doctor` | *(none)* |

## Error Behavior

When an unknown flag is encountered, the command must fail immediately with a clear error message.

**Error message format:**

```
Error: unknown flag "{flag}" for "{command}". Run 'tick help {command}' for usage.
```

**Examples:**
- `tick dep add tick-aaa --blocks tick-bbb` → `Error: unknown flag "--blocks" for "dep add". Run 'tick help dep' for usage.`
- `tick show --verbose tick-abc123` → *(no error — `--verbose` is a known global flag)*
- `tick list --unknown` → `Error: unknown flag "--unknown" for "list". Run 'tick help list' for usage.`

**Exit code:** Non-zero (consistent with existing error handling)

**Unknown flags before subcommand:** `parseArgs()` currently silently skips unknown flags appearing before the subcommand is identified. These must also be rejected with: `Error: unknown flag "{flag}". Run 'tick help' for usage.`

## Testing

1. Each command rejects an unknown flag with the correct error message
2. Global flags (`--verbose`, `--json`, etc.) are not rejected by command-level validation
3. The specific `dep add --blocks` scenario from the bug report is covered
4. Short flags (`-x`) are rejected as well as long flags (`--unknown`)
5. Unknown flags before the subcommand are rejected

## Dependencies

No dependencies. This bugfix modifies existing CLI flag parsing logic entirely within `internal/cli/`. No external systems, data, or infrastructure are required.

---

## Working Notes

[Optional - capture in-progress discussion if needed]
