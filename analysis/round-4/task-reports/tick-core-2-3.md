# Task tick-core-2-3: tick update command

## Task Summary

This task implements the `tick update <id>` command, which modifies task fields after creation. It supports five flags: `--title`, `--description`, `--priority`, `--parent`, and `--blocks`. At least one flag is required (no flags is an error). The positional ID and all reference IDs are normalized to lowercase. `--description ""` clears the description; `--parent ""` clears the parent. `--blocks` adds the current task's ID to the targets' `blocked_by` lists and refreshes their `updated` timestamps. On success, the command outputs full task details (like `tick show`); `--quiet` outputs only the ID. All mutations go through the storage engine's `Mutate` callback.

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|----|----|
| All five flags work correctly | PASS | PASS |
| Multiple flags combinable in single command | PASS | PASS |
| `updated` refreshed on every update | PASS | PASS |
| No flags -> error with exit code 1 | PASS | PASS |
| Missing/not-found ID -> error with exit code 1 | PASS | PASS |
| Invalid values -> error with exit code 1, no mutation | PASS | PASS |
| Output shows full task details; `--quiet` outputs ID only | PASS | PASS |
| Input IDs normalized to lowercase | PASS | PASS |
| Mutation persisted through storage engine | PASS | PASS |

Both versions satisfy all acceptance criteria.

## Implementation Comparison

### Approach

**V5** (`internal/cli/update.go`, 220 lines) uses the `Context`-based dispatch pattern. The `runUpdate` function receives a `*Context` carrying `WorkDir`, `Stdout`, `Quiet`, `Args`, and a `Fmt` formatter. It delegates arg parsing to `parseUpdateArgs` which returns the normalized ID and an `updateOpts` struct:

```go
// V5 update.go lines 31-35 -- Context-based dispatch
func runUpdate(ctx *Context) error {
    id, opts, err := parseUpdateArgs(ctx.Args)
    if err != nil {
        return err
    }
```

V5's `parseUpdateArgs` returns three values `(string, updateOpts, error)` -- the ID is returned separately from opts:

```go
// V5 update.go lines 152-159 -- ID is first positional arg, returned separately
func parseUpdateArgs(args []string) (string, updateOpts, error) {
    var opts updateOpts
    if len(args) == 0 {
        return "", opts, fmt.Errorf("Task ID is required. Usage: tick update <id> [options]")
    }
    id := task.NormalizeID(args[0])
    remaining := args[1:]
```

The parser rejects unknown flags and unexpected positional args explicitly:

```go
// V5 update.go lines 207-211
case strings.HasPrefix(arg, "-"):
    return "", opts, fmt.Errorf("unknown flag '%s'", arg)
default:
    return "", opts, fmt.Errorf("unexpected argument '%s'", arg)
```

V5 registers the command in the `commands` map in `cli.go`:

```go
// V5 cli.go line 66
"update": runUpdate,
```

For output, V5 captures the full `task.Task` from the Mutate callback and formats it via `ctx.Fmt.FormatTaskDetail`:

```go
// V5 update.go lines 141-148
if ctx.Quiet {
    fmt.Fprintln(ctx.Stdout, updatedTask.ID)
    return nil
}
return ctx.Fmt.FormatTaskDetail(ctx.Stdout, taskToShowData(updatedTask))
```

This uses `taskToShowData` (defined in `show.go`) which converts a `task.Task` to a `*showData` struct. Importantly, this conversion does **not** enrich related tasks (blockedBy, children) with titles/statuses since that would require additional DB queries. The output shows dependency IDs only, not their full context.

**V6** (`internal/cli/update.go`, 202 lines) uses the `App` struct pattern with an exported `RunUpdate` function that takes explicit parameters:

```go
// V6 update.go lines 93-94 -- explicit params, no Context struct
func RunUpdate(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
    opts, err := parseUpdateArgs(args)
```

V6's `parseUpdateArgs` embeds the ID inside the `updateOpts` struct rather than returning it separately:

```go
// V6 update.go lines 15-22 -- ID embedded in opts
type updateOpts struct {
    id          string
    title       *string
    description *string
    priority    *int
    parent      *string
    blocks      []string
}
```

```go
// V6 update.go lines 31-32 -- returns single opts struct
func parseUpdateArgs(args []string) (updateOpts, error) {
    var opts updateOpts
```

V6's parser silently skips unknown flags (treating them as global flags already extracted) and captures the first positional argument as the task ID:

```go
// V6 update.go lines 78-84
case strings.HasPrefix(arg, "-"):
    // Unknown flag -- skip (global flags already extracted)
default:
    // Positional argument: task ID (first one wins)
    if opts.id == "" {
        opts.id = task.NormalizeID(arg)
    }
```

This is a more lenient parser -- it silently ignores flags like `--quiet` that were already consumed by the global parser, whereas V5 would reject them as unknown.

For the `--parent` flag, V6 normalizes inline with an explicit empty-string check:

```go
// V6 update.go lines 67-71
v := args[i]
if v != "" {
    v = task.NormalizeID(strings.TrimSpace(v))
}
opts.parent = &v
```

V5 normalizes unconditionally:

```go
// V5 update.go lines 196-197
v := task.NormalizeID(remaining[i])
opts.parent = &v
```

This means V5 passes `""` through `NormalizeID` (which lowercases the empty string to `""`), which is functionally identical but V6 is more explicit about the intent.

For output, V6 refactored `show.go` to extract a reusable `queryShowData` function (also used by `RunShow`), and created a shared `outputMutationResult` helper in `helpers.go`:

```go
// V6 update.go lines 199-201 -- uses shared output helper
return outputMutationResult(store, updatedID, fc, fmtr, stdout)
```

```go
// V6 helpers.go lines 16-30 -- shared output function
func outputMutationResult(store *storage.Store, id string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error {
    if fc.Quiet {
        fmt.Fprintln(stdout, id)
        return nil
    }
    data, err := queryShowData(store, id)
    if err != nil {
        return err
    }
    detail := showDataToTaskDetail(data)
    fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))
    return nil
}
```

This means V6's update output performs a **full SQL query** after the mutation to fetch enriched task data (including blocked_by titles/statuses, children, parent title) -- identical to what `tick show` produces. V5 uses in-memory conversion which only has IDs, not enriched context.

**Key architectural difference: ID lookup strategy.** V5 builds a `map[string]int` (ID to index) for O(1) task lookup and block application:

```go
// V5 update.go lines 69-72
existing := make(map[string]int, len(tasks))
for i, t := range tasks {
    existing[t.ID] = i
}
```

```go
// V5 update.go lines 126-129 -- uses index for direct access
bIdx := existing[blockID]
tasks[bIdx].BlockedBy = append(tasks[bIdx].BlockedBy, id)
tasks[bIdx].Updated = now
```

V6 builds a `map[string]bool` for existence checks and uses linear iteration for task lookup and block application:

```go
// V6 update.go lines 131-134
idSet := make(map[string]bool, len(tasks))
for _, t := range tasks {
    idSet[t.ID] = true
}
```

```go
// V6 update.go lines 156-158 -- linear search for target task
for i := range tasks {
    if tasks[i].ID != opts.id {
        continue
    }
```

V6 delegates block application to a separate `applyBlocks` function in `helpers.go`:

```go
// V6 helpers.go lines 59-76 -- dedicated block application function
func applyBlocks(tasks []task.Task, sourceID string, blockIDs []string, now time.Time) {
    for i := range tasks {
        for _, blockID := range blockIDs {
            if tasks[i].ID == blockID {
                alreadyPresent := false
                for _, dep := range tasks[i].BlockedBy {
                    if dep == sourceID {
                        alreadyPresent = true
                        break
                    }
                }
                if !alreadyPresent {
                    tasks[i].BlockedBy = append(tasks[i].BlockedBy, sourceID)
                    tasks[i].Updated = now
                }
            }
        }
    }
}
```

V6's `applyBlocks` has **deduplication logic** -- it checks if `sourceID` is already in `BlockedBy` before appending. V5 does not deduplicate and will add duplicate entries.

**Cycle detection and ValidateDependency:** Both versions call `task.ValidateDependency` for `--blocks` validation, but with different placement relative to mutation:

V5 validates **before** applying the blocks:

```go
// V5 update.go lines 95-106 -- validation before mutation
if opts.blocks != nil {
    blocks := normalizeIDs(opts.blocks)
    opts.blocks = blocks
    if err := validateIDsExist(existing, blocks, "--blocks"); err != nil {
        return nil, err
    }
    for _, blockID := range blocks {
        if err := task.ValidateDependency(tasks, blockID, id); err != nil {
            return nil, err
        }
    }
}
```

V6 applies the blocks **first**, then validates:

```go
// V6 update.go lines 184-193 -- apply blocks, then validate
if len(opts.blocks) > 0 {
    applyBlocks(tasks, opts.id, opts.blocks, now)
    // Validate dependencies (cycle detection + child-blocked-by-parent) against full task list.
    for _, blockID := range opts.blocks {
        if err := task.ValidateDependency(tasks, blockID, opts.id); err != nil {
            return nil, err
        }
    }
}
```

V6's approach of applying first then validating is notable because `ValidateDependency` performs BFS/DFS on the task list's `BlockedBy` fields. By applying the new edges first, V6 validates against the **post-mutation** graph state. If validation fails, the Mutate callback returns an error and the entire mutation is rolled back (the storage engine does not persist). This is arguably more correct -- it checks whether the resulting graph state is valid, not just whether the new edge in isolation creates a cycle.

V5 validates pre-mutation, which means it checks the edge against the **pre-mutation** graph. This is also correct for simple cases but could theoretically miss issues where the combined mutation creates a problem.

**Self-referencing parent validation:** V5 delegates to `task.ValidateParent`:

```go
// V5 update.go lines 84-86
if err := task.ValidateParent(id, parent); err != nil {
    return nil, err
}
```

V6 performs the self-reference check inline:

```go
// V6 update.go lines 139-141
if *opts.parent == opts.id {
    return nil, fmt.Errorf("task %s cannot be its own parent", opts.id)
}
```

V5's approach is more DRY (reuses the shared validation function). V6 duplicates the check inline.

**Title validation:** V5's `task.ValidateTitle` returns `(string, error)` -- it trims and returns the trimmed value:

```go
// V5 update.go lines 38-44
if opts.title != nil {
    validated, err := task.ValidateTitle(*opts.title)
    if err != nil {
        return err
    }
    opts.title = &validated
}
```

V6 separates trimming and validation into two functions:

```go
// V6 update.go lines 108-113
if opts.title != nil {
    trimmed := task.TrimTitle(*opts.title)
    if err := task.ValidateTitle(trimmed); err != nil {
        return err
    }
}
```

V6 calls `TrimTitle` again during application (line 163), which is redundant but harmless:

```go
// V6 update.go line 163
tasks[i].Title = task.TrimTitle(*opts.title)
```

### Code Quality

**Go idioms:**

- Both versions follow idiomatic Go: early returns, `defer store.Close()`, explicit error handling.
- V5's `parseUpdateArgs` returning `(string, updateOpts, error)` uses multiple return values idiomatically. V6's embedding the ID in the struct is also clean.
- V5 correctly rejects unknown flags; V6 silently skips them, which is more robust in the context of global flag stripping but loses strict validation.

**Error message casing:**

```go
// V5 update.go line 156
return "", opts, fmt.Errorf("Task ID is required. Usage: tick update <id> [options]")
```

```go
// V6 update.go line 101
return fmt.Errorf("task ID is required. Usage: tick update <id> [options]")
```

V5 capitalizes "Task" matching spec text. V6 uses lowercase following Go convention. Same pattern for the not-found error:

```go
// V5 update.go line 78
return nil, fmt.Errorf("Task '%s' not found", id)
// V6 update.go line 181
return nil, fmt.Errorf("task '%s' not found", opts.id)
```

**Imports:**

- V5: `fmt`, `strconv`, `strings`, `time`, `engine`, `task` (6 imports)
- V6: `fmt`, `io`, `strconv`, `strings`, `time`, `task` (6 imports, no storage -- uses `openStore` helper)

**Function visibility:**

- V5: `runUpdate` is unexported, consistent with V5's pattern of package-private handlers accessed via the commands map.
- V6: `RunUpdate` is exported, independently callable.

**Code reuse:**

- V5 reuses `validateIDsExist`, `splitCSV`, `normalizeIDs` from `create.go` and `task.ValidateParent` from the task package.
- V6 introduces shared helpers: `openStore`, `outputMutationResult`, `parseCommaSeparatedIDs`, `applyBlocks` in `helpers.go`. This is a more modular approach -- the helpers are purpose-built and reusable across create and update commands.

**show.go refactoring (V6 only):**

V6's commit includes a refactoring of `show.go` to extract `queryShowData` into a standalone function. The diff shows 31 lines of change: the query logic was moved out of `RunShow` into `queryShowData`, and the return type changed from inline usage to a return value. This enables both `RunShow` and `RunUpdate` to share the exact same query path for full task detail output. V5 did not touch `show.go`.

### Test Quality

**V5** (`internal/cli/update_test.go`, current worktree: 544 lines, commit diff: 449 lines) has one top-level `TestUpdate` function with subtests. The current V5 worktree has 22 subtests (3 extra cycle/dependency tests added in later commits). The commit-level V5 had 19 subtests. Tests invoke the full CLI via `Run([]string{"tick", ...}, dir, &stdout, &stderr, false)`.

**V6** (`internal/cli/update_test.go`, current worktree: 595 lines, commit diff: 483 lines) has one top-level `TestUpdate` function with subtests. The current V6 worktree has 20 subtests (3 extra beyond the 17 spec tests). Tests use a dedicated `runUpdate` helper.

**Complete test listing (commit-level, V5):**

| # | Test Name | Line |
|---|-----------|------|
| 1 | `it updates title with --title flag` | 14 |
| 2 | `it updates description with --description flag` | 30 |
| 3 | `it clears description with --description empty string` | 47 |
| 4 | `it updates priority with --priority flag` | 65 |
| 5 | `it updates parent with --parent flag` | 82 |
| 6 | `it clears parent with --parent empty string` | 107 |
| 7 | `it updates blocks with --blocks flag` | 132 |
| 8 | `it updates multiple fields in a single command` | 156 |
| 9 | `it refreshes updated timestamp on any change` | 195 |
| 10 | `it outputs full task details on success` | 216 |
| 11 | `it outputs only task ID with --quiet flag` | 239 |
| 12 | `it errors when no flags are provided` | 256 |
| 13 | `it errors when task ID is missing` | 271 |
| 14 | `it errors when task ID is not found` | 285 |
| 15 | `it errors on invalid title` (table-driven: empty, whitespace, 501 chars, newline) | 299 |
| 16 | `it errors on invalid priority` (table-driven: -1, 5, 99) | 333 |
| 17 | `it errors on non-existent parent ID` | 360 |
| 18 | `it errors on non-existent blocks ID` | 375 |
| 19 | `it errors on self-referencing parent` | 390 |
| 20 | `it normalizes input IDs to lowercase` | 405 |
| 21 | `it persists changes via atomic write` | 429 |

The current V5 worktree additionally has (from later commits):
| 22 | `it rejects --blocks that would create a cycle` | 450 |
| 23 | `it rejects --blocks that would create an indirect cycle` | 480 |
| 24 | `it accepts valid --blocks dependency` | 520 |

**Complete test listing (commit-level, V6):**

| # | Test Name | Line |
|---|-----------|------|
| 1 | `it updates title with --title flag` | 29 |
| 2 | `it updates description with --description flag` | 50 |
| 3 | `it clears description with --description empty string` | 68 |
| 4 | `it updates priority with --priority flag` | 86 |
| 5 | `it updates parent with --parent flag` | 104 |
| 6 | `it clears parent with --parent empty string` | 129 |
| 7 | `it updates blocks with --blocks flag` | 154 |
| 8 | `it updates multiple fields in a single command` | 183 |
| 9 | `it refreshes updated timestamp on any change` | 222 |
| 10 | `it outputs full task details on success` | 244 |
| 11 | `it outputs only task ID with --quiet flag` | 274 |
| 12 | `it errors when no flags are provided` | 292 |
| 13 | `it errors when task ID is missing` | 312 |
| 14 | `it errors when task ID is not found` | 324 |
| 15 | `it errors on invalid title (empty/500/newlines)` (table-driven: empty, whitespace, 501 chars, newline) | 336 |
| 16 | `it errors on invalid priority (outside 0-4)` (table-driven: -1, 5, 100) | 366 |
| 17 | `it errors on non-existent parent/blocks IDs` (table-driven: parent, blocks) | 395 |
| 18 | `it errors on self-referencing parent` | 423 |
| 19 | `it normalizes input IDs to lowercase` | 439 |
| 20 | `it persists changes via atomic write` | 464 |

The current V6 worktree additionally has (from later commits):
| 21 | `it rejects --blocks that would create child-blocked-by-parent dependency` | 486 |
| 22 | `it does not duplicate blocked_by when --blocks with existing dependency` | 522 |
| 23 | `it rejects --blocks that would create a cycle` | 556 |

**Test fixture construction:**

V5 uses `task.NewTask(id, title)`:

```go
// V5 update_test.go line 14
tk := task.NewTask("tick-aaaaaa", "Original title")
dir := initTickProjectWithTasks(t, []task.Task{tk})
```

V6 constructs explicit struct literals with a pre-computed `now`:

```go
// V6 update_test.go lines 30-33
now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
tasks := []task.Task{
    {ID: "tick-aaa111", Title: "Old title", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
}
dir, tickDir := setupTickProjectWithTasks(t, tasks)
```

V6's approach is more explicit -- every field is visible at the call site. V5 relies on `NewTask` defaults.

**Test helper:**

V6 has a dedicated `runUpdate` helper that constructs a minimal `App`:

```go
// V6 update_test.go lines 14-26
func runUpdate(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
    t.Helper()
    var stdoutBuf, stderrBuf bytes.Buffer
    app := &App{
        Stdout: &stdoutBuf,
        Stderr: &stderrBuf,
        Getwd:  func() (string, error) { return dir, nil },
        IsTTY:  true,
    }
    fullArgs := append([]string{"tick", "update"}, args...)
    code := app.Run(fullArgs)
    return stdoutBuf.String(), stderrBuf.String(), code
}
```

V5 uses the full `Run` function in each test, declaring `var stdout, stderr bytes.Buffer` repeatedly.

**Timestamp test:**

V5 uses `time.Sleep(1100 * time.Millisecond)` to ensure the timestamp differs:

```go
// V5 update_test.go lines 200-201
// Small delay to ensure timestamp differs
time.Sleep(1100 * time.Millisecond)
```

V6 uses a fixed past `time.Date` and asserts `After`:

```go
// V6 update_test.go lines 223-235
now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
...
if !persisted[0].Updated.After(now) {
    t.Errorf("updated = %v, should be after %v", persisted[0].Updated, now)
}
```

V6's approach avoids real-time delays, making the test faster and deterministic. V6 additionally verifies `Created` was not modified:

```go
// V6 update_test.go lines 239-241
if !persisted[0].Created.Equal(now) {
    t.Errorf("created = %v, should remain %v", persisted[0].Created, now)
}
```

**Output detail test:**

V5 checks for `ID:`, task ID, and updated title:

```go
// V5 update_test.go lines 228-236
if !strings.Contains(output, "tick-aaaaaa") { ... }
if !strings.Contains(output, "New output") { ... }
if !strings.Contains(output, "ID:") { ... }
```

V6 checks for `ID:`, `Status:`, `Priority:`, task ID, and updated title:

```go
// V6 update_test.go lines 257-271
if !strings.Contains(stdout, "ID:") { ... }
if !strings.Contains(stdout, "tick-aaa111") { ... }
if !strings.Contains(stdout, "Updated task") { ... }
if !strings.Contains(stdout, "Status:") { ... }
if !strings.Contains(stdout, "Priority:") { ... }
```

V6 verifies more output fields, providing better coverage of the "full task details" requirement.

**Non-existent parent/blocks tests:**

V5 has two separate subtests: `it errors on non-existent parent ID` (line 360) and `it errors on non-existent blocks ID` (line 375).

V6 combines these into a single table-driven test: `it errors on non-existent parent/blocks IDs` (line 395) with two sub-cases.

**Blocks test:**

V6's blocks test adds `--title` as an additional flag and also verifies the target's updated timestamp was refreshed:

```go
// V6 update_test.go lines 162, 178-179
_, stderr, exitCode := runUpdate(t, dir, "tick-aaa111", "--blocks", "tick-bbb222", "--title", "Blocker updated")
...
if !target.Updated.After(now) {
    t.Error("target's updated timestamp should be refreshed")
}
```

V5's blocks test only passes `--blocks` and checks `blocked_by` but not the target's timestamp.

**No-mutation verification on invalid title:**

V5 verifies that invalid title does not mutate the task:

```go
// V5 update_test.go lines 324-327
tasks := readTasksFromFile(t, dir)
if tasks[0].Title != "Valid title" {
    t.Errorf("title should not have changed, got %q", tasks[0].Title)
}
```

V6 does not verify no-mutation on invalid title -- it only checks exit code and error message.

### Skill Compliance

| Constraint | V5 | V6 |
|-----------|----|----|
| Handle all errors explicitly (no naked returns) | PASS | PASS |
| Document all exported functions/types/packages | PASS (all unexported, still documented) | PASS (`RunUpdate` is exported and documented) |
| Write table-driven tests with subtests | PASS (title/priority tests are table-driven) | PASS (title/priority/parent-blocks tests are table-driven) |
| Propagate errors with `fmt.Errorf("%w", err)` | PARTIAL -- most errors are plain strings | PARTIAL -- same; `handleUpdate` wraps Getwd with `%w` |
| MUST NOT ignore errors | PASS | PASS |
| MUST NOT use panic for normal error handling | PASS | PASS |
| MUST NOT hardcode configuration | PASS | PASS |
| Use `gofmt` compatible formatting | PASS | PASS |
| Run race detector on tests | Not verified at commit level | Not verified at commit level |

### Spec-vs-Convention Conflicts

1. **Error message capitalization**: V5 uses `"Task ID is required"` and `"Task '%s' not found"` (capital T, matching spec text). V6 uses `"task ID is required"` and `"task '%s' not found"` (lowercase, Go convention). Since `Error: ` is prepended by the dispatcher, V6 is more Go-idiomatic but deviates from spec wording.

2. **Unknown flag handling**: V5 rejects unknown flags with an explicit error (`"unknown flag '%s'"`). V6 silently skips them, assuming they were already handled by global flag parsing. This is a design philosophy difference -- V5 is stricter, V6 is more tolerant.

3. **`hasAnyFlag` vs `hasChanges` naming**: V5 names the method `hasAnyFlag()` which describes the check accurately. V6 names it `hasChanges()` which is slightly misleading -- it checks whether any *flag was provided*, not whether any *change would result*. The semantic difference is minor.

4. **`--blocks` nil vs empty slice check**: V5 checks `opts.blocks != nil` to distinguish "flag not provided" from "flag provided with empty value." V6 checks `len(opts.blocks) > 0`. This means V6's `hasChanges()` returns false if `--blocks ""` results in an empty slice after parsing, while V5's `hasAnyFlag()` would return true if `opts.blocks` was set to an empty non-nil slice. In practice, the CSV parser always produces at least one element, so this is unlikely to cause a behavioral difference.

5. **Output enrichment**: The spec says "Output full task details (like `tick show`)." V5's output via `taskToShowData` converts the in-memory task to show format but does not enrich relationships with titles/statuses. V6's output via `queryShowData` performs a full SQL query identical to `tick show`, producing enriched output with dependency titles and children. V6 is more spec-compliant on this point.

## Diff Stats

| Metric | V5 | V6 |
|--------|----|----|
| Files changed (commit) | 5 | 6 |
| Total lines added (commit) | 671 | 745 |
| `update.go` lines (commit) | 216 | 227 |
| `update.go` lines (worktree) | 220 | 202 |
| `update_test.go` lines (commit) | 449 | 483 |
| `update_test.go` lines (worktree) | 544 | 595 |
| Dispatcher change | +1 line in `cli.go` | +11 lines in `app.go` (case + `handleUpdate` method) |
| Show.go refactoring | None | +31/-15 lines (extracted `queryShowData`) |
| Helper functions introduced | 0 (reuses `validateIDsExist`, `splitCSV`, `normalizeIDs` from create.go) | 3 (`outputMutationResult`, `parseCommaSeparatedIDs`, `applyBlocks` in helpers.go) |
| Subtests (commit) | 21 (+ 7 table-driven sub-cases) | 20 (+ 9 table-driven sub-cases) |
| Subtests (worktree, incl. later commits) | 24 (+ 7 table-driven sub-cases) | 23 (+ 9 table-driven sub-cases) |
| Test helper functions added | 0 | 1 (`runUpdate`, 13 lines) |
| `time.Sleep` in tests | Yes (1.1s) | No (deterministic timestamps) |
| Deduplication in `--blocks` | No | Yes (`applyBlocks` checks `alreadyPresent`) |

## Verdict

Both implementations satisfy all 9 acceptance criteria and produce functionally correct behavior. The core logic -- parsing flags into an `updateOpts` struct with pointer fields for optionality, validating inputs before opening the store, mutating via the storage engine's `Mutate` callback, and outputting via formatter -- is structurally the same.

**V5 strengths:**
- Stricter argument parsing: rejects unknown flags explicitly, catches unexpected positional args (more defensive)
- Reuses existing shared validation helpers (`validateIDsExist`, `ValidateParent`) rather than inlining checks
- O(1) task lookup via `map[string]int` index map vs V6's linear iteration
- No-mutation assertion on invalid title test (verifies data was not corrupted)
- Separate, clearly named error tests for non-existent parent and non-existent blocks (easier to diagnose failures)
- Pre-mutation dependency validation (validates edges before they are applied)

**V6 strengths:**
- Dedicated `runUpdate` test helper eliminates repeated `bytes.Buffer` boilerplate (DRYer tests)
- Deterministic timestamp testing with `time.Date` (no `time.Sleep`, faster tests)
- Verifies `Created` timestamp is not modified during update
- Richer output assertion: checks `Status:` and `Priority:` fields in addition to `ID:` and title
- Blocks test also verifies target's `updated` timestamp was refreshed
- `applyBlocks` deduplication prevents duplicate `blocked_by` entries (robustness improvement)
- `outputMutationResult` shared helper used by both create and update, with full SQL enrichment matching `tick show` output (more spec-compliant)
- `queryShowData` refactoring of `show.go` improves code reuse across commands
- Post-mutation dependency validation catches issues in the resulting graph state
- Table-driven combined test for non-existent parent/blocks IDs (more concise)
- `handleUpdate` wraps Getwd error with `%w` (proper error chain)

**Winner: V6.** V6 makes stronger architectural choices: the `show.go` refactoring and shared `outputMutationResult` helper produce truly spec-compliant "like `tick show`" output with enriched relationship data, while V5's `taskToShowData` is a lossy conversion that omits dependency titles and children statuses. The `applyBlocks` deduplication logic is a real robustness improvement that prevents data corruption on repeated `--blocks` invocations. The test suite, while covering the same spec-mandated cases, is faster (no `time.Sleep`), more thorough on output assertions, and verifies immutability of `Created`. V5's advantages (stricter flag parsing, O(1) lookup, no-mutation assertion on invalid title) are real but lower impact. The spec-compliance gap on output enrichment is the decisive factor.
