# Investigation: Unknown Flags Silently Ignored

## Symptoms

### Problem Description

**Expected behavior:**
All commands should reject unrecognised flags with a clear error message, e.g. `Error: unknown flag --blocks. Run 'tick help dep' for usage.`

**Actual behavior:**
Unknown flags are silently discarded. Any argument starting with `-` that isn't a known flag is stripped without warning, which can mislead users into thinking a flag had effect when it didn't.

### Manifestation

- No error messages produced when unknown flags are passed
- Commands execute with the unknown flag silently dropped
- User intent can be misinterpreted — e.g. `tick dep add tick-aaa --blocks tick-bbb` silently ignores `--blocks` (only valid on `create`/`update`) and treats it as `tick dep add tick-aaa tick-bbb`

### Reproduction Steps

1. Run any tick command with an unknown flag, e.g. `tick dep add tick-aaa --blocks tick-bbb`
2. Observe: command succeeds without error
3. The `--blocks` flag was silently ignored; the command behaved as if it was `tick dep add tick-aaa tick-bbb`

**Reproducibility:** Always

### Impact

- **Severity:** Low
- **Scope:** All commands affected
- **Business impact:** User confusion, potential for unintended task relationships

---

## Analysis

### Initial Hypotheses

Flag parsing across commands likely uses a pattern that strips unknown flags rather than rejecting them. Need to trace how flags are parsed in the CLI layer.

### Code Trace

The bug exists at two layers: global dispatch and per-command parsing.

**Layer 1: Global flag parsing — `app.go:317-339`**

`parseArgs()` loops through args. Known global flags (`--quiet`, `--verbose`, `--toon`, `--pretty`, `--json`, `--help`) are extracted via `applyGlobalFlag()`. Unknown flags appearing *before* the subcommand are silently skipped (line 328-330):

```
if strings.HasPrefix(arg, "-") {
    // Unknown flag before subcommand — skip
    continue
}
```

After the subcommand is found, all remaining args (including unknown flags) are passed through to the command handler in `rest`.

**Layer 2: Per-command flag parsing — every command**

Each command hand-rolls its own flag parsing with the same pattern of silently skipping unknown flags:

| File | Lines | Pattern |
|------|-------|---------|
| `dep.go` | 40-43 | `if strings.HasPrefix(arg, "-") { continue }` — strips ALL flags when extracting positional args |
| `create.go` | 93-94 | `case strings.HasPrefix(arg, "-"):` — switch fallthrough, comment: "global flags already extracted" |
| `update.go` | 111-112 | Same as create |
| `remove.go` | 29-30 | `case strings.HasPrefix(arg, "-"):` — comment: "Skip unknown flags." |
| `note.go` | 43-44 | `if strings.HasPrefix(arg, "-") { continue }` — same as dep |
| `list.go` | 34-102 | Switch on known flags; unknown flags fall through silently |
| `show.go` | — | Takes `args[0]` as task ID; any flags ignored |
| `transition.go` | — | Takes `args[0]` as task ID; any flags ignored |

**The dep add example in detail:**

1. `tick dep add tick-aaa --blocks tick-bbb`
2. `parseArgs()` passes `["add", "tick-aaa", "--blocks", "tick-bbb"]` to `handleDep()`
3. `handleDep()` strips `"add"`, calls `RunDepAdd()` with `["tick-aaa", "--blocks", "tick-bbb"]`
4. `parseDepArgs()` loops: `"tick-aaa"` → positional, `"--blocks"` → skipped (starts with `-`), `"tick-bbb"` → positional
5. Result: two positional args found, no error. `--blocks` silently vanished.

### Root Cause

There is no unified flag parser with validation. Every command hand-rolls its own arg parsing and silently ignores anything starting with `-` that isn't a known flag. There is no mechanism to:
- Define the set of valid flags for a command
- Error on unknown flags
- Share validation logic across commands

The comments ("global flags already extracted") are misleading — global parsing is also non-validating.

### Contributing Factors

- Hand-rolled parsing in every command rather than a shared flag parsing utility
- The `strings.HasPrefix(arg, "-")` skip pattern was used from the start as a "safe" way to handle unknown args
- No test coverage for unknown flag rejection (tests only verify known flags work)

### Why It Wasn't Caught

- Tests verify happy paths (known flags produce correct behavior) but never assert that unknown flags are rejected
- The silent-skip pattern doesn't cause crashes or errors — it's a UX problem, not a runtime problem
- Each command was built independently, copying the same skip pattern

### Blast Radius

**Directly affected:**
- All commands: `create`, `update`, `list`, `show`, `dep add/remove`, `remove`, `note add/remove`, `start`, `done`, `cancel`, `reopen`, `stats`, `doctor`, `init`

**Potentially affected:**
- Any future commands that copy the existing pattern

---

## Fix Direction

*To be determined after findings review*

---

## Notes

- Referenced from `bugs.md` as BUG-1
- General CLI parsing concern — not specific to any single command
