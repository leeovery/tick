# Task tick-core-2-3: tick update Command

## Task Summary

Implement `tick update <id>` to modify mutable task fields after creation. The command accepts a positional task ID and five optional flags: `--title`, `--description`, `--priority <0-4>`, `--parent <id>`, and `--blocks <id,...>`. At least one flag is required; providing no flags is an error. Input IDs are normalized to lowercase. Immutable fields (status, id, created, blocked_by) are not exposed as flags.

Validation rules: title must be trimmed, max 500 chars, no newlines; priority must be 0-4; parent must exist and not self-reference; blocks IDs must exist. `--description ""` clears description; `--parent ""` clears parent. `--blocks` adds the updated task's ID to target tasks' `blocked_by` and refreshes their `updated` timestamps. The source task's `updated` is always refreshed. Output shows full task details (like `tick show`); `--quiet` outputs only ID. All changes persisted via the storage engine's atomic `Mutate`.

### Acceptance Criteria (from plan)

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

| Criterion | V2 | V4 |
|-----------|-----|-----|
| All five flags work correctly | PASS -- title, description, priority, parent, blocks all implemented with dedicated tests | PASS -- all five flags implemented with dedicated tests |
| Multiple flags combinable | PASS -- test `"it updates multiple fields in a single command"` verifies title+priority+description | PASS -- test `"it updates multiple fields in a single command"` verifies title+priority+description |
| `updated` refreshed on every update | PASS -- test checks `tk["updated"] != "2026-01-19T10:00:00Z"` | PASS -- test checks `tasks[0].Updated.After(now)` with stronger typed comparison; also verifies `Created` unchanged |
| No flags -> error exit 1 | PASS -- test verifies error returned containing `--title` and `--description` | PASS -- test verifies exit code 1, stderr contains "No flags provided" and "--title" |
| Missing/not-found ID -> error exit 1 | PASS -- separate tests for missing ID and not-found ID | PASS -- separate tests for missing ID and not-found ID, checks exit code 1 |
| Invalid values -> error exit 1, no mutation | PASS -- separate tests for empty/500/newline title, priority 5/-1, non-existent parent/blocks, self-ref; one title test verifies no mutation | PASS -- table-driven tests for 4 title cases and 4 priority cases, each with no-mutation verification for title |
| Full task details output; `--quiet` ID only | PASS -- tests verify output contains ID, title, status; quiet test verifies exact ID output | PASS -- tests verify output contains "ID:", "Title:", "Status:", "Priority:"; quiet test verifies exact ID output |
| Input IDs normalized to lowercase | PASS -- test uses `TICK-AAA111` and `TICK-BBB222`, verifies lowercase parent | PASS -- test uses `TICK-AAA111` and `TICK-BBB222`, verifies lowercase parent |
| Mutation persisted via storage engine | PASS -- test reads back from JSONL file directly | PASS -- test reads raw JSONL and also verifies `cache.db` existence |

## Implementation Comparison

### Approach

Both versions create a new `update.go` file in `internal/cli/` and register the `update` subcommand in the main dispatch. The core flow is identical: parse args, validate, find task via `Mutate`, apply changes, output results.

**Architecture Differences:**

V2 uses a `func (a *App) Run(args []string) error` signature that returns errors directly to the caller. The dispatch in `app.go` is:
```go
case "update":
    return a.runUpdate(cmdArgs)
```

V4 uses a `func (a *App) Run(args []string) int` signature returning exit codes. The dispatch in `cli.go` is:
```go
case "update":
    if err := a.runUpdate(subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```

This means V4 handles error-to-stderr formatting at the dispatch level, while V2 relies on the caller.

**Flag Parsing:**

V2 uses `parseUpdateArgs` as a package-level function that returns `(*updateFlags, error)`. The ID is extracted first (checking for `--` prefix to detect missing ID), then flags are parsed in a `switch` statement. The `updateFlags` struct uses boolean sentinels (`titleProvided`, `descriptionProvided`, `parentProvided`) alongside the string values:

```go
// V2 updateFlags
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

V4 uses `parseUpdateArgs` as a method `(a *App) parseUpdateArgs(args []string) (string, *updateFlags, error)` returning the ID separately. The `updateFlags` struct uses pointer-based optionality for all fields:

```go
// V4 updateFlags
type updateFlags struct {
    title       *string
    description *string
    priority    *int
    parent      *string
    blocks      []string
}
```

V4's pointer-based approach is more idiomatic Go for optional values -- it eliminates the need for separate `*Provided` booleans, reducing the struct from 8 fields to 5. The `hasAnyFlag()` methods differ accordingly:

```go
// V2
func (f *updateFlags) hasAnyFlag() bool {
    return f.titleProvided || f.descriptionProvided || f.priority != nil || f.parentProvided || len(f.blocks) > 0
}

// V4
func (f *updateFlags) hasAnyFlag() bool {
    return f.title != nil || f.description != nil || f.priority != nil || f.parent != nil || f.blocks != nil
}
```

V4's parser is slightly more flexible: it does not require the ID to be the first positional argument -- any non-flag argument encountered first is treated as the ID. V2 strictly requires the first argument to be the ID (if it starts with `--`, the ID is considered missing). V4 also delegates comma-separated blocks parsing to an existing `parseCommaSeparatedIDs` helper, while V2 inlines this logic with `strings.Split`.

**No-flags error message:**

V2 provides a terse single-line message:
```go
return fmt.Errorf("No update flags provided. Use at least one of: --title, --description, --priority, --parent, --blocks")
```

V4 provides a formatted multi-line help text:
```go
return fmt.Errorf("No flags provided. At least one flag is required.\n\nAvailable options:\n  --title \"<text>\"           New title\n  --description \"<text>\"     New description (use \"\" to clear)\n  --priority <0-4>           New priority level\n  --parent <id>              New parent task (use \"\" to clear)\n  --blocks <id,...>          Tasks this task blocks")
```

V4's error is significantly more helpful to users, providing a mini-usage guide when invoked incorrectly.

**Validation Strategy:**

V2 performs all validation upfront before any mutation (validate-then-apply pattern):
```go
// V2: Validate all flags first
if flags.titleProvided {
    if _, err := task.ValidateTitle(flags.title); err != nil {
        return nil, fmt.Errorf("invalid title: %w", err)
    }
}
if flags.priority != nil {
    if err := task.ValidatePriority(*flags.priority); err != nil {
        return nil, err
    }
}
if flags.parentProvided && flags.parent != "" {
    if err := task.ValidateParent(tasks[idx].ID, flags.parent); err != nil {
        return nil, err
    }
    if !existingIDs[flags.parent] {
        return nil, fmt.Errorf(...)
    }
}
// ... then apply mutations
```

V4 interleaves validation and application for each flag:
```go
// V4: Validate and apply per-flag
if flags.title != nil {
    validTitle, err := task.ValidateTitle(*flags.title)
    if err != nil {
        return nil, fmt.Errorf("invalid title: %w", err)
    }
    tasks[idx].Title = validTitle
}
if flags.description != nil {
    tasks[idx].Description = *flags.description
}
if flags.priority != nil {
    if err := task.ValidatePriority(*flags.priority); err != nil {
        return nil, err
    }
    tasks[idx].Priority = *flags.priority
}
```

V2's approach is genuinely better here: by validating everything before mutating anything, it guarantees that a failed validation on a later flag (e.g., invalid priority) doesn't leave a partial mutation in the task slice (e.g., title already changed). While the Mutate callback's error would prevent persistence, V2 is architecturally cleaner. However, V2 calls `ValidateTitle` twice (once for validation, once to get the cleaned value), which is wasteful:
```go
// V2 calls ValidateTitle twice
if flags.titleProvided {
    if _, err := task.ValidateTitle(flags.title); err != nil { ... }
}
// ...later...
if flags.titleProvided {
    cleanTitle, _ := task.ValidateTitle(flags.title)  // second call
    tasks[idx].Title = cleanTitle
}
```

**Self-reference parent check:**

V2 uses a shared `task.ValidateParent()` function:
```go
if err := task.ValidateParent(tasks[idx].ID, flags.parent); err != nil {
    return nil, err
}
```

V4 does the check inline:
```go
if parentVal == id {
    return nil, fmt.Errorf("task %s cannot be its own parent", id)
}
```

V2's use of a shared validator function is better for DRY -- the same validation is likely used in create.

**Error unwrapping:**

V2 introduced a shared `unwrapMutationError()` function in `app.go` and also refactored `transition.go` and `create.go` to use it, replacing ad-hoc string prefix stripping:

```go
func unwrapMutationError(err error) error {
    if inner := errors.Unwrap(err); inner != nil {
        return inner
    }
    return err
}
```

V4 does not unwrap mutation errors -- it returns the raw error from `s.Mutate`:
```go
if err != nil {
    return err
}
```

V2's approach is genuinely better: it ensures clean user-facing errors without "mutation failed:" prefixes, and consolidates what was previously duplicated logic.

**Blocks duplicate handling:**

Both versions prevent adding duplicate entries to `blocked_by`. V2 uses normalized ID comparison:
```go
if task.NormalizeID(existingID) == task.NormalizeID(sourceID) {
    alreadyPresent = true
    break
}
```

V4 uses direct string comparison:
```go
if dep == id {
    found = true
    break
}
```

V2's approach is more defensive (case-insensitive dedup), while V4 relies on IDs already being normalized before reaching this point.

**Additional V2 changes:**

V2 also modified `create.go` to add duplicate-skip logic for `--blocks` (preventing the same task from appearing twice in `blocked_by`), added a test for this in `create_test.go`, added a test verifying no "mutation failed:" prefix in errors, and refactored `transition.go` error handling. V4 made no changes outside `update.go`, `update_test.go`, and `cli.go`.

### Code Quality

**Naming:**

Both versions use clear, idiomatic names. V2's `parseUpdateArgs` is a package-level function; V4's is a method on `*App`. V2's `updateFlags` struct is more explicit but verbose (8 fields vs 5). V4's pointer-based flags are more idiomatic Go.

**Error handling:**

V2 consistently unwraps mutation errors via a shared helper. V4 returns raw errors from Mutate, which may expose internal "mutation failed:" prefixes to users. This is a quality gap in V4.

V2 returns unknown-flag errors:
```go
default:
    return nil, fmt.Errorf("unknown flag %q for update command", arg)
```

V4 returns unexpected-argument errors:
```go
default:
    if id == "" {
        id = task.NormalizeID(strings.TrimSpace(arg))
    } else {
        return "", nil, fmt.Errorf("unexpected argument '%s'", arg)
    }
```

V4's approach is slightly better because it differentiates between "this is the ID" and "this is unexpected", allowing the ID to appear at any position among arguments.

**DRY:**

V2 reuses `task.ValidateParent()` for self-reference checks and introduces `unwrapMutationError()` shared across three commands. V4 inlines the self-reference check and does not share error-unwrapping logic.

V2 refactors existing code (create.go, transition.go) while implementing the new feature. This is extra scope but improves overall codebase consistency.

**Type safety:**

V4's pointer-based optionality (`*string`, `*int`) is more type-safe than V2's boolean sentinels. With V2's pattern, a developer could forget to set `titleProvided = true` when setting `title`, creating a subtle bug. V4's nil-check pattern makes this impossible.

### Test Quality

**V2 Test Functions (1 top-level function, 22 subtests):**

`TestUpdateCommand` (single top-level, all subtests nested):
1. `"it updates title with --title flag"` -- verifies JSONL title change
2. `"it updates description with --description flag"` -- verifies description set
3. `"it clears description with --description empty string"` -- checks key omitted from JSONL
4. `"it updates priority with --priority flag"` -- verifies priority=0
5. `"it updates parent with --parent flag"` -- verifies parent set
6. `"it clears parent with --parent empty string"` -- checks key omitted from JSONL
7. `"it updates blocks with --blocks flag"` -- verifies blocked_by + target updated
8. `"it updates multiple fields in a single command"` -- title+priority+description combined
9. `"it refreshes updated timestamp on any change"` -- checks timestamp changed
10. `"it outputs full task details on success"` -- checks output contains ID, title, status
11. `"it outputs only task ID with --quiet flag"` -- exact ID match
12. `"it errors when no flags are provided"` -- checks --title and --description in message
13. `"it errors when task ID is missing"` -- checks "Task ID is required"
14. `"it errors when task ID is not found"` -- checks "not found" + task ID in error
15. `"it errors on invalid title (empty after trim)"` -- whitespace-only title; verifies no mutation
16. `"it errors on invalid title (over 500 chars)"` -- 501 chars; checks "title" and "500" in error
17. `"it errors on invalid title (contains newlines)"` -- newline in title; checks "newline"
18. `"it errors on invalid priority (outside 0-4)"` -- tests 5 and -1
19. `"it errors on non-existent parent ID"` -- checks "tick-nonexist" in error
20. `"it errors on non-existent blocks ID"` -- checks "tick-nonexist" in error
21. `"it errors on self-referencing parent"` -- checks "cannot be its own parent"
22. `"it normalizes input IDs to lowercase"` -- TICK-AAA111 -> tick-aaa111
23. `"it persists changes via atomic write"` -- reads JSONL directly
24. `"it silently skips duplicate when --blocks target already has source in blocked_by"` -- dedup test
25. `"it updates multiple targets with comma-separated --blocks"` -- 2 targets, both blocked_by updated
26. `"it applies --blocks combined with --title atomically"` -- combined blocks+title

Total: 26 subtests. Additional create_test.go test: `"it silently skips duplicate when --blocks target already has source in blocked_by"`.

**V4 Test Functions (16 top-level functions, subtests within):**

1. `TestUpdate_TitleFlag` -> `"it updates title with --title flag"` -- typed task.Task comparison
2. `TestUpdate_DescriptionFlag` -> `"it updates description with --description flag"`
3. `TestUpdate_ClearDescription` -> `"it clears description with --description \"\""` -- checks `tasks[0].Description != ""`
4. `TestUpdate_PriorityFlag` -> `"it updates priority with --priority flag"` -- typed int comparison
5. `TestUpdate_ParentFlag` -> `"it updates parent with --parent flag"` -- typed string comparison
6. `TestUpdate_ClearParent` -> `"it clears parent with --parent \"\""` -- typed string comparison
7. `TestUpdate_BlocksFlag` -> `"it updates blocks with --blocks flag"` -- typed BlockedBy slice + Updated.After(now)
8. `TestUpdate_MultipleFields` -> `"it updates multiple fields in a single command"`
9. `TestUpdate_RefreshesUpdatedTimestamp` -> `"it refreshes updated timestamp on any change"` -- also verifies Created unchanged
10. `TestUpdate_OutputsFullTaskDetails` -> `"it outputs full task details on success"` -- checks "ID:", "Title:", "Status:", "Priority:"
11. `TestUpdate_QuietFlag` -> `"it outputs only task ID with --quiet flag"`
12. `TestUpdate_ErrorNoFlags` -> `"it errors when no flags are provided"` -- checks exit code 1, "No flags provided", "--title"
13. `TestUpdate_ErrorMissingID` -> `"it errors when task ID is missing"` -- checks exit code 1, "Task ID is required"
14. `TestUpdate_ErrorIDNotFound` -> `"it errors when task ID is not found"` -- checks exit code 1, "not found"
15. `TestUpdate_ErrorInvalidTitle` -> table-driven: "empty", "whitespace only", "too long" (501), "contains newline"; each verifies exit code 1, error on stderr, and no-mutation (title remains "Original")
16. `TestUpdate_ErrorInvalidPriority` -> table-driven: "negative" (-1), "too high" (5), "way too high" (100), "not a number" (abc); each verifies exit code 1, error on stderr
17. `TestUpdate_ErrorNonExistentParentBlocks` -> two subtests: "non-existent parent", "non-existent blocks target"
18. `TestUpdate_ErrorSelfReferencingParent` -> `"it errors on self-referencing parent"`
19. `TestUpdate_NormalizesInputIDs` -> `"it normalizes input IDs to lowercase"`
20. `TestUpdate_PersistsChanges` -> `"it persists changes via atomic write"` -- reads raw JSONL + checks cache.db exists

Total: 20 top-level test functions, approximately 24 unique test cases (including table-driven subtests).

**Test Coverage Differences:**

| Edge Case | V2 | V4 |
|-----------|-----|-----|
| Blocks duplicate skip (already in blocked_by) | TESTED | NOT TESTED |
| Comma-separated --blocks (multiple targets) | TESTED | NOT TESTED |
| --blocks combined with --title atomically | TESTED | NOT TESTED |
| Invalid priority "not a number" (e.g., "abc") | NOT TESTED (tests 5 and -1 only) | TESTED (table-driven) |
| Invalid priority "way too high" (100) | NOT TESTED | TESTED |
| Invalid title empty string "" | NOT TESTED (tests whitespace only) | TESTED (table-driven) |
| Created timestamp unchanged after update | NOT TESTED | TESTED |
| cache.db existence after update | NOT TESTED | TESTED |
| No-mutation verification for invalid title | 1 test (whitespace) | 4 tests (all title cases) |

V2 has better coverage of blocks-related edge cases (3 extra tests for dedup, comma-separated, and combined operations). V4 has better coverage of validation edge cases (more title and priority variants via table-driven tests) and a stronger persistence test (verifies cache.db).

**Test Style:**

V2 uses a single `TestUpdateCommand` function with all subtests nested. Tests use raw JSONL strings, `NewApp()` constructor, and `readTaskByID`/`readTasksJSONL` helpers that return `map[string]interface{}` -- assertions use untyped JSON comparisons like `tk["title"] != "New title"` and `int(tk["priority"].(float64)) != 0`.

V4 uses separate top-level test functions per logical group (`TestUpdate_TitleFlag`, `TestUpdate_ErrorInvalidTitle`, etc.). Tests construct `task.Task` structs directly and use `setupInitializedDirWithTasks` + `readTasksFromDir` helpers that return `[]task.Task` -- assertions are strongly typed: `tasks[0].Title != "Updated title"`, `tasks[0].Priority != 0`, `target.Updated.After(now)`.

V4's test approach is genuinely better:
- Strongly typed assertions catch more bugs at compile time
- Separate top-level functions allow running individual test groups in isolation
- Table-driven tests for validation are more systematic and easier to extend
- Using `task.Task` structs directly avoids fragile JSON type assertions like `int(tk["priority"].(float64))`

V2's test approach is better for:
- Testing the actual serialization format (JSONL) since it reads raw JSON
- Coverage breadth of `--blocks` behavior

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed (Go only) | 6 | 3 |
| Lines added (Go only) | 919 (+) / 10 (-) | 946 (+) / 0 (-) |
| Impl LOC (update.go) | 244 | 228 |
| Test LOC (update_test.go) | 609 | 712 |
| Test functions (top-level) | 1 | 16 (20 including grouped) |
| Test subtests (total) | 26 | ~24 |
| Additional files modified | app.go, create.go, create_test.go, transition.go | cli.go only |

## Verdict

**V2 is the better implementation overall**, though V4 has notable strengths in specific areas.

V2's advantages:
1. **Validate-then-apply pattern** -- All validation happens before any mutation, ensuring atomicity at the application level. V4 interleaves validation and mutation, which, while functionally safe due to Mutate's transactional nature, is architecturally weaker.
2. **Error unwrapping** -- V2 introduces the shared `unwrapMutationError()` helper and applies it across update, create, and transition commands. V4 does not unwrap mutation errors, potentially exposing "mutation failed:" prefixes to users.
3. **Broader --blocks coverage** -- V2 tests duplicate skip, comma-separated targets, and combined blocks+title operations. V4 tests none of these.
4. **Codebase improvements** -- V2 refactors existing code (create.go dedup, transition.go error handling) alongside the new feature, improving overall quality.
5. **DRY with shared validators** -- V2 reuses `task.ValidateParent()` rather than inlining the self-reference check.

V4's advantages:
1. **Pointer-based optionality** -- More idiomatic Go, fewer struct fields, impossible to have desync between value and "provided" boolean.
2. **Richer no-flags error message** -- Provides a formatted usage guide vs. V2's single-line list.
3. **Table-driven validation tests** -- More systematic and extensible. Tests more edge cases for invalid titles and priorities.
4. **Strongly typed test assertions** -- Using `task.Task` structs eliminates fragile JSON type assertions.
5. **cache.db persistence verification** -- Tests both JSONL content and SQLite cache existence.
6. **Flexible ID positioning** -- V4's parser allows the ID anywhere among arguments, not just first position.

The deciding factors are V2's validate-then-apply pattern (genuinely safer architecture), its error unwrapping improvement (user-facing quality), and its broader test coverage of the `--blocks` feature (which is the most complex part of this task). V4's test ergonomics and flag struct design are better, but these are style preferences rather than correctness concerns.
