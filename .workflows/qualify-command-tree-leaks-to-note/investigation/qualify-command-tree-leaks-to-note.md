# Investigation: Qualify Command Tree Leaks To Note

## Symptoms

### Problem Description

**Expected behavior:**
`tick note tree` should produce an error indicating "tree" is not a valid subcommand of "note". Flag validation should not reference "note tree" as if it were a real command.

**Actual behavior:**
`qualifyCommand` in `internal/cli/app.go` shares the `"tree"` case across both `"dep"` and `"note"` parent commands. This means `tick note tree` gets qualified as `"note tree"`, which isn't registered in `commandFlags`. When flags are passed (e.g., `tick note tree --foo`), the error says `unknown flag "--foo" for "note tree"`, implying "note tree" is a real command that just doesn't accept that flag.

### Manifestation

- Misleading error message: `unknown flag "--foo" for "note tree"` suggests the command exists
- `handleNote` eventually returns "unknown note sub-command 'tree'" without flags, which is correct but the flag validation path is wrong

### Reproduction Steps

1. Run `tick note tree --foo`
2. Observe error: `unknown flag "--foo" for "note tree"` (misleading)
3. Run `tick note tree` (without flags)
4. Observe error: `unknown note sub-command 'tree'` (correct but inconsistent path)

**Reproducibility:** Always

### Impact

- **Severity:** Low
- **Scope:** Unlikely user path
- **Business impact:** Poor UX for edge case — misleading error messages

### References

- Surfaced during dep-tree-visualization review
- `qualifyCommand` switch in `internal/cli/app.go`

---

## Analysis

### Initial Hypotheses

The `qualifyCommand` switch groups `"add"`, `"remove"`, and `"tree"` together for both `"dep"` and `"note"` parents. `"add"` and `"remove"` are valid for both, but `"tree"` only applies to `dep`.

### Code Trace

**Entry point:**
`internal/cli/app.go:100` — `qualifyCommand(subcmd, subArgs)` called before flag validation.

**Execution path:**
1. `app.go:366-380` — `qualifyCommand("note", ["tree", "--foo"])` enters function
2. `app.go:367` — passes gate: `subcmd == "note"`
3. `app.go:374-376` — `sub = "tree"` matches shared case `"add", "remove", "tree"` → returns `("note tree", ["--foo"])`
4. `app.go:101` — `ValidateFlags("note tree", ["--foo"], commandFlags)` called
5. `flags.go:117` — `cmdFlags := flags["note tree"]` → key doesn't exist, `cmdFlags` is nil map
6. `flags.go:136` — nil map lookup `cmdFlags["--foo"]` returns zero value, `ok` is false → error: `unknown flag "--foo" for "note tree"`

**Without flags:** `tick note tree` (no flags) → `ValidateFlags` loop body never executes (no flag args) → passes validation → reaches `handleNote` → `note.go:33` correctly returns `"unknown note sub-command 'tree'"`

**Key files involved:**
- `internal/cli/app.go:366-380` — `qualifyCommand` with shared switch case
- `internal/cli/flags.go:32-91` — `commandFlags` registry (no `"note tree"` entry)
- `internal/cli/flags.go:116-147` — `ValidateFlags` consumes the qualified command name
- `internal/cli/note.go:14-35` — `handleNote` with correct subcommand validation

### Root Cause

`qualifyCommand` unconditionally qualifies `"tree"` as a subcommand of both `"dep"` and `"note"` parents, but `"tree"` is only a valid subcommand of `"dep"`. The switch case at `app.go:375` groups `"add"`, `"remove"`, `"tree"` without checking which parent command is active.

**Why this happens:**
The function was written when `"dep"` was the only two-level command, or `"add"` and `"remove"` were the only shared subcommands. When `"tree"` was added for `dep tree`, it was grouped into the same case for simplicity, but `"note"` also passes the parent check at line 367.

### Contributing Factors

- `qualifyCommand` checks parent identity (`dep` or `note`) at line 367 but then uses a single undifferentiated switch for all subcommands — no parent-specific branching within the switch
- `"add"` and `"remove"` genuinely are shared, so the grouped pattern worked until `"tree"` was added (tree is dep-only)
- `commandFlags` acts as a secondary validator but only for flags, not for command existence — the qualified command name is trusted

### Why It Wasn't Caught

- No test for `qualifyCommand("note", ["tree", ...])` — existing tests only cover the `dep` parent
- `flag_validation_test.go:136-139` lists `noFlagCommands` and `flag_validation_test.go:191` lists global-flag commands — neither includes `"note tree"` (correctly, since it's not in `commandFlags`), so there's no test that would surface the mismatch
- The drift-detection test checks `commandFlags` against the help registry, not against `qualifyCommand`'s output space
- The error path still "works" in a loose sense — the user gets an error either way — so it's unlikely to be noticed in manual testing

### Blast Radius

**Directly affected:**
- `tick note tree` with any flags — produces misleading error message
- `tick note tree` without flags — works correctly (falls through to `handleNote`)

**Potentially affected:**
- None — the `"add"` and `"remove"` cases are legitimately shared between `dep` and `note`, so no other subcommands leak

---

## Fix Direction

*To be filled after root cause validation and findings review*

---

## Notes

- The fix is straightforward: scope `"tree"` to only qualify under `"dep"`, not `"note"`
- Two approaches: (a) split the switch into parent-specific cases, or (b) add a parent check inside the `"tree"` case
- Should add a test for `qualifyCommand("note", ["tree"])` to verify it falls through to default
