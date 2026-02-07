# Task tick-core-2-3: tick update command

## Task Summary

Implement `tick update <id>` which changes title, description, priority, parent, and blocks fields on an existing task. At least one flag is required -- providing no flags is an error. Immutable fields (status, id, created, blocked_by) have dedicated commands and are not exposed as update flags.

**Flags:** `--title`, `--description`, `--priority <0-4>`, `--parent <id>`, `--blocks <id,...>`

**Key behaviors:**
- Positional ID normalized to lowercase
- `--description ""` clears description; `--parent ""` clears parent
- `--blocks` adds this task to targets' `blocked_by`, refreshes their `updated`
- Title validation: trim, max 500, no newlines
- Priority validation: 0-4
- Parent validation: exists, no self-ref
- Blocks validation: all IDs exist
- `updated` refreshed to current UTC on any change
- Output full task details (like `tick show`); `--quiet` outputs only ID
- Mutation persisted through storage engine

**Acceptance Criteria:**
1. All five flags work correctly
2. Multiple flags combinable in single command
3. `updated` refreshed on every update
4. No flags -> error with exit code 1
5. Missing/not-found ID -> error with exit code 1
6. Invalid values -> error with exit code 1, no mutation
7. Output shows full task details; `--quiet` outputs ID only
8. Input IDs normalized to lowercase
9. Mutation persisted through storage engine

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| 1. All five flags work | PASS - all 5 flags implemented and tested | PASS - all 5 flags implemented and tested | PASS - all 5 flags implemented and tested |
| 2. Multiple flags combinable | PASS - test "updates multiple fields in a single command" | PASS - test "it updates multiple fields in a single command" | PASS - test "it updates multiple fields in a single command" |
| 3. `updated` refreshed | PASS - `t.Updated = now` set on every mutation | PASS - `tasks[idx].Updated = now` set on every mutation | PASS - `tasks[targetIdx].Updated = now` set on every mutation |
| 4. No flags -> error exit 1 | PASS - returns error "At least one flag is required" | PASS - returns error "No update flags provided" | PASS - writes to stderr "At least one flag is required" with available flags list |
| 5. Missing/not-found ID -> error exit 1 | PASS - tested for both missing and not-found | PASS - tested for both missing and not-found | PASS - tested for both missing and not-found |
| 6. Invalid values -> error, no mutation | PASS - validates before store open (title, priority, self-ref parent) | PASS - validates inside Mutate before applying mutations; also verifies no mutation in test | PASS - validates before store open; V3 explicitly checks empty title separately |
| 7. Output: full details / --quiet ID only | PASS - `a.printTaskBasic(updatedTask)` / `--quiet` prints ID | PASS - `a.printTaskDetails(updatedTask)` / `--quiet` prints ID | PASS - `a.printTaskDetails(updatedTask)` / `--quiet` prints ID |
| 8. IDs normalized to lowercase | PASS - `task.NormalizeID()` on task ID, parent, and blocks IDs | PASS - `task.NormalizeID()` on task ID, parent, and blocks IDs | PASS - `task.NormalizeID()` on task ID; `normalizeIDs()` helper for blocks |
| 9. Mutation persisted | PASS - tested via file read-back | PASS - tested via `readTasksJSONL` read-back | PASS - tested via `readTasksFromDir` and cross-verification with `tick show` |

## Implementation Comparison

### Approach

All three versions follow the same fundamental pattern: parse arguments, validate inputs, open storage, execute a `store.Mutate()` callback that finds the target task, applies changes, and returns the modified task list. The differences lie in structural organization, error handling strategy, and return type conventions.

#### V1: Single-function monolith (`internal/cli/update.go`, 175 LOC)

V1 uses the simplest approach: a single `cmdUpdate` method that does everything inline. It returns `error` (the `App.Run` dispatcher converts errors to exit codes).

```go
func (a *App) cmdUpdate(workDir string, args []string) error {
    // ...
    var (
        titleFlag       *string
        descriptionFlag *string
        priorityFlag    *int
        parentFlag      *string
        blocksFlag      []string
    )
```

Flag parsing uses raw pointer variables (`*string`, `*int`) declared inline. The function signature takes `workDir string` as a parameter rather than reading from the App struct.

Key structural choice: V1 validates title and parent **before** opening the store, but validates parent-existence and blocks-existence **inside** the Mutate callback. This is efficient -- early validation avoids unnecessary I/O for clearly invalid input.

The task lookup inside Mutate uses a `map[string]*task.Task`:
```go
taskMap := make(map[string]*task.Task, len(tasks))
for i := range tasks {
    taskMap[tasks[i].ID] = &tasks[i]
}
```

V1 accesses the task via pointer through the map, mutating the slice element in-place. This is clean but relies on the map values being pointers into the original slice.

V1 uses `time.Now().UTC().Truncate(time.Second)` for timestamps, which produces a `time.Time` value. The `Updated` field is set as a `time.Time` (not a string).

#### V2: Separated flags struct + parseUpdateArgs (`internal/cli/update.go`, 244 LOC)

V2 introduces a named `updateFlags` struct with explicit `*Provided` booleans:

```go
type updateFlags struct {
    id                  string
    title               string
    titleProvided       bool
    description         string
    descriptionProvided bool
    priority            *int
    parent              string
    parentProvided      bool
    blocks              []string
}
```

This is more explicit than V1's pointer approach. It separates "was the flag given" from "what value was given" using boolean flags for string types. For priority, it still uses `*int` since nil/non-nil is sufficient.

V2 extracts a standalone `parseUpdateArgs` function and a `hasAnyFlag()` method:
```go
func (f *updateFlags) hasAnyFlag() bool {
    return f.titleProvided || f.descriptionProvided || f.priority != nil || f.parentProvided || len(f.blocks) > 0
}
```

V2 returns `error` from `runUpdate`. Errors from the Mutate callback are unwrapped with `unwrapMutationError()` -- a shared helper V2 also added to `app.go` and retroactively applied to `transition.go` and `create.go`:

```go
func unwrapMutationError(err error) error {
    if inner := errors.Unwrap(err); inner != nil {
        return inner
    }
    return err
}
```

V2 performs **all validation inside the Mutate callback** -- including title validation and priority validation. This means invalid input still triggers a store open and lock acquisition before the validation error is returned. However, this guarantees atomicity: validation sees the actual current state.

V2 also adds duplicate detection for `--blocks`:
```go
alreadyPresent := false
for _, existingID := range tasks[i].BlockedBy {
    if task.NormalizeID(existingID) == task.NormalizeID(sourceID) {
        alreadyPresent = true
        break
    }
}
```

V2 calls `task.ValidateTitle` twice -- once for validation, once for the cleaned value:
```go
if flags.titleProvided {
    if _, err := task.ValidateTitle(flags.title); err != nil {
        return nil, fmt.Errorf("invalid title: %w", err)
    }
}
// ... later ...
if flags.titleProvided {
    cleanTitle, _ := task.ValidateTitle(flags.title)
    tasks[idx].Title = cleanTitle
}
```

This double-call is slightly wasteful but separates the validation phase from the mutation phase clearly.

V2 also detects if the first argument starts with `--`, treating that as a missing ID:
```go
if strings.HasPrefix(args[0], "--") {
    return nil, fmt.Errorf("Task ID is required. Usage: tick update <id>")
}
```

#### V3: Exported flags struct + int return codes (`internal/cli/update.go`, 253 LOC)

V3 uses an exported `UpdateFlags` struct with pointer types (same pattern as V1 but in a struct):

```go
type UpdateFlags struct {
    Title       *string  // nil means not provided
    Description *string  // nil means not provided, empty string means clear
    Priority    *int     // nil means not provided
    Parent      *string  // nil means not provided, empty string means clear
    Blocks      []string // IDs this task blocks
}
```

V3's `runUpdate` returns `int` (exit code) instead of `error`. It writes errors directly to `a.Stderr`:
```go
func (a *App) runUpdate(args []string) int {
    // ...
    if err != nil {
        fmt.Fprintf(a.Stderr, "Error: %s\n", err)
        return 1
    }
```

This is a fundamentally different error-handling pattern. V1 and V2 return errors and let the dispatcher format them. V3 handles formatting internally. V3's `parseUpdateArgs` receives the full `args` including "tick" and "update" (index 0 and 1), while V1 and V2 receive only post-subcommand args.

V3 validates title, priority, and self-referencing parent **before** opening the store. It explicitly checks for empty title after trimming:
```go
if flags.Title != nil {
    trimmed := task.TrimTitle(*flags.Title)
    if trimmed == "" {
        fmt.Fprintf(a.Stderr, "Error: Title cannot be empty\n")
        return 1
    }
    if err := task.ValidateTitle(trimmed); err != nil {
        // ...
    }
}
```

V3 uses a `normalizeIDs()` helper for blocks IDs (reused from elsewhere in the codebase).

V3 uses `time.Now().UTC().Format(time.RFC3339)` for timestamps -- storing as a **string** rather than `time.Time`. This differs from V1's `Truncate(time.Second)` approach.

V3 has duplicate detection for `--blocks` (like V2) but without NormalizeID during the check:
```go
found := false
for _, existing := range tasks[i].BlockedBy {
    if existing == taskID {
        found = true
        break
    }
}
```

V3 provides a detailed options list when no flags are given:
```go
fmt.Fprintf(a.Stderr, "Error: At least one flag is required. Available flags:\n")
fmt.Fprintf(a.Stderr, "  --title \"<text>\"        New title\n")
// ...5 lines of flag descriptions
```

V3 also adds the "update" command to the help/usage output in `cli.go`:
```go
fmt.Fprintln(a.Stdout, "  update  Update task fields")
```

### Code Quality

**Naming:**
- V1: uses `cmdUpdate` (matches existing pattern like `cmdShow`, `cmdTransition`). Local variables like `titleFlag`, `descriptionFlag`.
- V2: uses `runUpdate` (matches its codebase's convention). The `updateFlags` struct is unexported (lowercase) -- appropriate since it's internal. Uses `hasAnyFlag()` method.
- V3: uses `runUpdate`. The `UpdateFlags` struct is **exported** (uppercase) -- unnecessary since it's only used within the `cli` package. The `parseUpdateArgs` function is also unexported.

**Error handling:**
- V1: returns `error`, relies on dispatcher. Clean but errors from Mutate may include "mutation failed:" prefix.
- V2: returns `error` but adds `unwrapMutationError()` to strip the "mutation failed:" wrapper. This is the most thoughtful approach. V2 also applies this fix retroactively to `create.go` and `transition.go`.
- V3: returns `int` exit code, writes "Error: ..." to stderr directly. This is self-contained but means the function both formats and returns -- mixing concerns.

**DRY:**
- V1: most compact at 175 LOC. No code duplication.
- V2: 244 LOC. Calls `ValidateTitle` twice (once to check, once to get cleaned value). Introduces `unwrapMutationError` as a reusable helper. Adds dedup check for `--blocks`.
- V3: 253 LOC. Separates empty-title check from `ValidateTitle` call, which is slightly redundant if `ValidateTitle` already handles empty. Has its own dedup logic for blocks.

**Type safety:**
- V1: raw pointer variables. Works but loses the grouping/documentation that a struct provides.
- V2: struct with `*Provided` booleans. Most explicit about intent. The `hasAnyFlag()` method is a clean abstraction.
- V3: struct with pointer types. Standard Go pattern. But the exported type is over-scoped.

**Blocks deduplication:**
- V1: **Does NOT check for duplicates.** If `--blocks` targets a task that already has the source in `blocked_by`, it will append a duplicate entry. This is a bug.
- V2: Checks for duplicates using `NormalizeID()` comparison. Most robust.
- V3: Checks for duplicates using direct string comparison. Works but could miss case-sensitivity edge cases if IDs weren't already normalized.

**Timestamp format:**
- V1: `time.Now().UTC().Truncate(time.Second)` -- stores as `time.Time`
- V2: `time.Now().UTC().Truncate(time.Second)` -- stores as `time.Time` (same as V1; the `Updated` field type is `time.Time` with custom marshaling)
- V3: `time.Now().UTC().Format(time.RFC3339)` -- stores as `string`. This is a different approach that depends on the `Updated` field being a `string` type in V3's codebase.

**Scope of changes beyond update.go:**
- V1: Touched only `cli.go` (2 lines to add case "update") + `update.go` + `update_test.go`. Minimal footprint.
- V2: Touched 6 non-doc files: `app.go` (added `unwrapMutationError`), `create.go` (dedup fix + unwrap), `create_test.go` (dedup test + unwrap test), `transition.go` (unwrap), `update.go`, `update_test.go`. Broadest scope -- improves the codebase beyond the task requirements.
- V3: Touched `cli.go` (3 lines: case + usage), `update.go`, `update_test.go`, plus doc files. Added the "update" command to the help output.

### Test Quality

#### V1 Test Functions (17 tests in `update_test.go`, 308 LOC)

```
TestUpdateCommand/
  updates title with --title flag
  updates description with --description flag
  clears description with empty --description
  updates priority with --priority flag
  updates parent with --parent flag
  clears parent with empty --parent
  updates blocks with --blocks flag
  updates multiple fields in a single command
  refreshes updated timestamp on any change
  outputs full task details on success
  outputs only task ID with --quiet flag
  errors when no flags are provided
  errors when task ID is missing
  errors when task ID is not found
  errors on invalid title
  errors on invalid priority
  errors on non-existent parent ID
  errors on self-referencing parent
  normalizes input IDs to lowercase
  persists changes via atomic write
```

V1 tests use `initTickDir`/`createTask`/`runCmd` helpers, which actually invoke the full CLI pipeline. Tests verify results by reading JSONL files directly with `os.ReadFile`. The test for "errors on invalid title" only tests empty string -- does NOT test 500-char limit or newlines.

**V1 edge cases covered:** empty title, priority=5, non-existent parent, self-referencing parent, uppercase ID normalization, clear description, clear parent.

**V1 edge cases NOT covered:** title over 500 chars, title with newlines, non-existent blocks ID, negative priority, duplicate --blocks entries, multiple comma-separated blocks, blocks + title atomicity, blocks target timestamp refresh.

#### V2 Test Functions (22 tests in `update_test.go`, 609 LOC)

```
TestUpdateCommand/
  it updates title with --title flag
  it updates description with --description flag
  it clears description with --description empty string
  it updates priority with --priority flag
  it updates parent with --parent flag
  it clears parent with --parent empty string
  it updates blocks with --blocks flag
  it updates multiple fields in a single command
  it refreshes updated timestamp on any change
  it outputs full task details on success
  it outputs only task ID with --quiet flag
  it errors when no flags are provided
  it errors when task ID is missing
  it errors when task ID is not found
  it errors on invalid title (empty after trim)
  it errors on invalid title (over 500 chars)
  it errors on invalid title (contains newlines)
  it errors on invalid priority (outside 0-4)
  it errors on non-existent parent ID
  it errors on non-existent blocks ID
  it errors on self-referencing parent
  it normalizes input IDs to lowercase
  it persists changes via atomic write
  it silently skips duplicate when --blocks target already has source in blocked_by
  it updates multiple targets with comma-separated --blocks
  it applies --blocks combined with --title atomically
```

V2 tests use `setupTickDirWithContent`/`NewApp()`/`readTaskByID` helpers. Tests set up JSONL content directly (inline strings), which is more explicit about initial state. V2 verifies "no mutation" on validation failure in the empty-title test. V2 also tests negative priority (-1).

**V2 edge cases covered (beyond V1):** title over 500 chars, title with newlines, non-existent blocks ID, negative priority, duplicate --blocks dedup, multiple comma-separated blocks, blocks + title atomicity, empty-after-trim title, verification of no-mutation on error.

**V2 also adds tests outside update_test.go:** `create_test.go` gets a test for duplicate --blocks dedup and mutation-error prefix checking.

#### V3 Test Functions (20 tests in `update_test.go`, 643 LOC)

```
TestUpdateCommand/
  it updates title with --title flag
  it updates description with --description flag
  it clears description with --description ""
  it updates priority with --priority flag
  it updates parent with --parent flag
  it clears parent with --parent ""
  it updates blocks with --blocks flag
  it updates multiple fields in a single command
  it refreshes updated timestamp on any change
  it outputs full task details on success
  it outputs only task ID with --quiet flag
  it errors when no flags are provided
  it errors when task ID is missing
  it errors when task ID is not found
  it errors on invalid title (empty/500/newlines)
  it errors on invalid priority (outside 0-4)
  it errors on non-existent parent/blocks IDs
  it errors on self-referencing parent
  it normalizes input IDs to lowercase
  it persists changes via atomic write
  it refreshes target task updated timestamp when --blocks is used
```

V3 tests use `setupTickDir`/`setupTask`/`setupTaskFull`/`readTasksFromDir` helpers with `bytes.Buffer` for stdout/stderr. Tests construct `App` structs directly. V3 bundles multiple sub-assertions into single tests (e.g., "empty/500/newlines" is one test with 4 sub-checks; "parent/blocks IDs" tests both in one).

V3 uniquely tests: persistence verified via a second `app.Run([]string{"tick", "show", ...})` call (round-trip verification), and target task updated timestamp refresh as a dedicated test.

**V3 edge cases covered (beyond V1):** title over 500 chars, title with newlines, whitespace-only title, non-existent blocks ID, verification that invalid priority doesn't mutate, blocks target timestamp refresh.

**V3 edge cases NOT covered (that V2 has):** duplicate --blocks dedup, multiple comma-separated blocks, blocks + title atomicity, negative priority.

#### Test Coverage Gap Analysis

| Edge Case | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| Empty title | YES | YES | YES |
| Whitespace-only title | NO | YES | YES |
| Title > 500 chars | NO | YES | YES |
| Title with newlines | NO | YES | YES |
| Priority = 5 (above range) | YES | YES | YES |
| Priority = -1 (below range) | NO | YES | NO |
| Non-existent parent ID | YES | YES | YES |
| Non-existent blocks ID | NO | YES | YES |
| Self-referencing parent | YES | YES | YES |
| Duplicate --blocks dedup | NO | YES | NO |
| Comma-separated --blocks | NO | YES | NO |
| --blocks + --title atomicity | NO | YES | NO |
| Blocks target timestamp refresh | NO | YES (inline) | YES (dedicated) |
| No-mutation verification on error | NO | YES (title) | YES (priority) |
| Persistence round-trip via show | NO | NO | YES |
| Uppercase ID normalization | YES | YES | YES |
| Clear description | YES | YES | YES |
| Clear parent | YES | YES | YES |

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed (code only) | 3 | 6 | 4 |
| Lines added (code only) | 485 | 918 | 899 |
| Lines removed (code only) | 0 | 10 | 0 |
| Impl LOC (update.go) | 175 | 244 | 253 |
| Test LOC (update_test.go) | 308 | 609 | 643 |
| Test functions | 17 | 22 | 20 |

Note: V2 also modified `app.go` (+14), `create.go` (+13/-3), `create_test.go` (+38), `transition.go` (+1/-7). V3 also modified `cli.go` (+3). V1 only modified `cli.go` (+2).

## Verdict

**V2 is the best implementation.**

**Evidence:**

1. **Most complete test coverage:** V2 has 22 test functions covering the widest range of edge cases -- including duplicate --blocks dedup, comma-separated --blocks, blocks+title atomicity, negative priority, and all three title validation failures as separate tests. V2 is the only version to test all of these.

2. **Blocks deduplication:** V1 has a **bug** -- it does not check for duplicate entries when appending to `blocked_by`. V2 and V3 both handle this, but only V2 tests it and V2's implementation uses `NormalizeID()` during the comparison for robustness.

3. **Codebase-wide improvements:** V2 introduced `unwrapMutationError()` as a shared helper and retroactively fixed error messages in `create.go` and `transition.go`. This cross-cutting improvement demonstrates awareness of the broader codebase, and addresses a real UX problem (users seeing "mutation failed:" prefixes).

4. **Clean separation of concerns:** V2's `parseUpdateArgs` + `updateFlags` struct + `hasAnyFlag()` method provides the cleanest factoring. The explicit `*Provided` booleans make the distinction between "not provided" and "provided as empty" crystal clear for string types.

5. **Validation-before-mutation inside Mutate:** V2 validates all inputs (including title validation) inside the Mutate callback but before applying any mutations. This guarantees that validation sees the actual current state of the data, avoiding TOCTOU issues. V1 and V3 validate title/priority before opening the store, which is slightly more efficient but less correct in theory.

**V3 is a close second** -- it has good test coverage (20 tests), the blocks dedup fix, and the unique round-trip persistence test via `tick show`. However, its exported `UpdateFlags` type is over-scoped, it lacks the codebase-wide `unwrapMutationError` improvement, it returns `int` exit codes instead of `error` (mixing formatting with logic), and it misses several edge cases that V2 covers (duplicate blocks, comma-separated blocks, atomicity).

**V1 is the weakest** -- while it is the most concise (175 impl LOC), it has a bug (no blocks deduplication), the fewest tests (17), and misses significant edge cases (title length/newlines, non-existent blocks IDs, negative priority). It also doesn't address the "mutation failed:" error prefix issue.
