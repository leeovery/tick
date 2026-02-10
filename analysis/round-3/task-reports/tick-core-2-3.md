# Task tick-core-2-3: tick update command

## Task Summary

This task implements `tick update <id>` which modifies task fields after creation. It supports five flags: `--title`, `--description`, `--priority <0-4>`, `--parent <id>`, and `--blocks <id,...>`. At least one flag is required -- no flags produces an error with a list of available options. `--description ""` clears description; `--parent ""` clears parent. `--blocks` adds the current task's ID to the target tasks' `blocked_by` fields and refreshes their `updated` timestamps. Self-referencing parent is rejected. All input IDs are normalized to lowercase. The command outputs full task details on success or just the ID with `--quiet`. Immutable fields (status, id, created, blocked_by) are not exposed as flags.

### Acceptance Criteria (from plan)

1. All five flags work correctly
2. Multiple flags combinable in single command
3. `updated` refreshed on every update
4. No flags produces error with exit code 1
5. Missing/not-found ID produces error with exit code 1
6. Invalid values produce error with exit code 1, no mutation
7. Output shows full task details; `--quiet` outputs ID only
8. Input IDs normalized to lowercase
9. Mutation persisted through storage engine

## Acceptance Criteria Compliance

| Criterion | V4 | V5 |
|-----------|-----|-----|
| All five flags work correctly | PASS -- `--title`, `--description`, `--priority`, `--parent`, `--blocks` all parsed in `parseUpdateArgs` (lines 148-210) and applied in `Mutate` callback (lines 53-112); individual tests for each flag | PASS -- same five flags parsed in `parseUpdateArgs` (lines 143-210) and applied in `Mutate` callback (lines 67-130); individual tests for each flag |
| Multiple flags combinable in single command | PASS -- loop-based parsing handles any combination; tested in `TestUpdate_MultipleFields` with `--title`, `--description`, `--priority` combined | PASS -- same loop-based parsing; tested in subtest `"it updates multiple fields in a single command"` with `--title`, `--description`, `--priority`, `--parent` combined (4 flags vs V4's 3) |
| `updated` refreshed on every update | PASS -- `tasks[idx].Updated = time.Now().UTC().Truncate(time.Second)` at line 115; tested in `TestUpdate_RefreshesUpdatedTimestamp` which verifies `tasks[0].Updated.After(now)` and that `Created` remains unchanged | PASS -- `tasks[idx].Updated = now` at line 130; tested in subtest `"it refreshes updated timestamp on any change"` which verifies `tasks[0].Updated.After(originalUpdated)` with a 1100ms sleep to guarantee difference |
| No flags produces error with exit code 1 | PASS -- `hasAnyFlag()` check at line 219; error message includes `"No flags provided"` and full options list; tested in `TestUpdate_ErrorNoFlags` | PASS -- `hasAnyFlag()` check at line 205; error message `"at least one flag is required. Available: --title, --description, --priority, --parent, --blocks"`; tested in subtest `"it errors when no flags are provided"` |
| Missing/not-found ID produces error with exit code 1 | PASS -- missing: `"Task ID is required. Usage: tick update <id> [options]"` at line 216; not found: `"Task '%s' not found"` at line 47; both tested | PASS -- missing: same message at line 145; not found: same message at line 77; both tested |
| Invalid values produce error with exit code 1, no mutation | PASS -- title validated via `task.ValidateTitle` (line 55); priority via `task.ValidatePriority` (line 67); parent self-ref check (line 80); existence checks for parent and blocks; no-mutation verified in `TestUpdate_ErrorInvalidTitle` which re-reads file | PASS -- title validated via `task.ValidateTitle` before Mutate (line 40); priority via `task.ValidatePriority` before Mutate (line 48); parent via `task.ValidateParent` and `validateIDsExist` inside Mutate (lines 83-90); no-mutation verified in `"it errors on invalid title"` which re-reads file |
| Output shows full task details; `--quiet` outputs ID only | PASS -- quiet: `fmt.Fprintln(a.Stdout, updatedTask.ID)` at line 122; non-quiet: `a.Formatter.FormatTaskDetail(a.Stdout, detail)` at line 126; tested in `TestUpdate_OutputsFullTaskDetails` and `TestUpdate_QuietFlag` | PASS -- quiet: `fmt.Fprintln(ctx.Stdout, updatedTask.ID)` at line 138; non-quiet: `ctx.Fmt.FormatTaskDetail(ctx.Stdout, taskToShowData(updatedTask))` at line 141; tested in corresponding subtests |
| Input IDs normalized to lowercase | PASS -- positional ID: `task.NormalizeID(strings.TrimSpace(arg))` at line 203; parent: `task.NormalizeID(val)` at line 190; blocks: via `parseCommaSeparatedIDs` which normalizes internally; tested in `TestUpdate_NormalizesInputIDs` with uppercase `"TICK-AAA111"` and `"TICK-BBB222"` | PASS -- positional ID: `task.NormalizeID(args[0])` at line 149; parent: `task.NormalizeID(remaining[i])` at line 186; blocks: via `splitCSV` then `normalizeIDs` inside Mutate at line 95; tested in corresponding subtest with `"TICK-AAAAAA"` and `"TICK-BBBBBB"` |
| Mutation persisted through storage engine | PASS -- all changes inside `s.Mutate()` callback; tested in `TestUpdate_PersistsChanges` which reads raw JSONL file and verifies JSON content and cache.db existence | PASS -- all changes inside `store.Mutate()` callback; tested in subtest `"it persists changes via atomic write"` which re-reads tasks from file |

## Implementation Comparison

### Approach

**V4: Method on `App` struct with manual argument parsing**

V4 implements `runUpdate` as a method on `*App` (line 16):

```go
func (a *App) runUpdate(args []string) error {
```

Dispatched via a `case "update":` block in `cli.go`'s `switch` statement (6 lines added):

```go
case "update":
    if err := a.runUpdate(subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```

The implementation flow:
1. Parse args via `a.parseUpdateArgs(args)` returning `(string, *updateFlags, error)` -- note pointer return for flags struct
2. Discover tick dir, open store via `a.openStore(tickDir)`
3. Inside `s.Mutate()`: linear scan for task by ID using `idx` variable, build `existingIDs` map (`map[string]bool`), then validate and apply each flag in sequence
4. All validation (title, priority, parent self-ref, parent existence, blocks existence, dependency cycles) happens **inside** the Mutate callback
5. Output via `a.Formatter.FormatTaskDetail()` or ID-only for quiet

V4 uses `parseCommaSeparatedIDs()` (defined in `create.go` line 199) for blocks parsing and performs self-referencing parent check inline:

```go
if parentVal == id {
    return nil, fmt.Errorf("task %s cannot be its own parent", id)
}
```

**V5: Free function with Context, early validation**

V5 implements `runUpdate` as a package-level function accepting `*Context` (line 34):

```go
func runUpdate(ctx *Context) error {
```

Registered in the `commands` map with a single line added to `cli.go`:

```go
"update": runUpdate,
```

The implementation flow:
1. Parse args via `parseUpdateArgs(ctx.Args)` returning `(string, updateOpts, error)` -- note value return for opts struct
2. **Validate title and priority before opening the store** (lines 39-50) -- early exit on invalid values without touching disk
3. Discover tick dir, open store via `engine.NewStore(tickDir, ctx.storeOpts()...)`
4. Inside `store.Mutate()`: build `existing` map (`map[string]int` -- stores index, not bool), find task, validate parent/blocks existence and cycles, then apply changes
5. Output via `ctx.Fmt.FormatTaskDetail()` or ID-only for quiet

V5 uses shared helpers from `create.go`: `validateIDsExist()` (line 231), `splitCSV()` (line 241), and `normalizeIDs()` (line 254). Self-referencing parent check is delegated to `task.ValidateParent(id, parent)`.

**Key structural differences:**

1. **Validation placement:** V5 validates title and priority **before** opening the store, while V4 validates everything inside the Mutate callback. V5's approach is more efficient -- if the user provides `--title ""`, V5 returns an error without touching the filesystem. V4 would discover the tick dir, open the store, load tasks, and only then fail during mutation. This is a meaningful optimization for error cases.

2. **Flags struct semantics:** V4 returns `*updateFlags` (pointer) from `parseUpdateArgs`; V5 returns `updateOpts` (value). V5's value return is more idiomatic for small structs and avoids nil-pointer concerns. V4's `hasAnyFlag()` is a pointer receiver method; V5's is a value receiver method.

3. **ID existence map type:** V4 uses `map[string]bool` for existence checks. V5 uses `map[string]int` storing the task index. V5's approach is superior because it enables O(1) index lookup when applying `--blocks`, eliminating the need for an inner loop:

   V4 (O(n*m) for n tasks, m blocks):
   ```go
   for i := range tasks {
       for _, blockTarget := range flags.blocks {
           if tasks[i].ID == blockTarget {
               // ...
           }
       }
   }
   ```

   V5 (O(m) for m blocks):
   ```go
   for _, blockID := range opts.blocks {
       bIdx := existing[blockID]
       tasks[bIdx].BlockedBy = append(tasks[bIdx].BlockedBy, id)
       tasks[bIdx].Updated = now
   }
   ```

4. **Duplicate blocked_by prevention:** V4 explicitly checks for duplicates before appending to `blocked_by`:
   ```go
   found := false
   for _, dep := range tasks[i].BlockedBy {
       if dep == id {
           found = true
           break
       }
   }
   if !found {
       tasks[i].BlockedBy = append(tasks[i].BlockedBy, id)
       tasks[i].Updated = now
   }
   ```
   V5 does **not** check for duplicates -- it always appends:
   ```go
   tasks[bIdx].BlockedBy = append(tasks[bIdx].BlockedBy, id)
   tasks[bIdx].Updated = now
   ```
   V4's approach is more defensive. If `tick update X --blocks Y` is run twice, V4 produces `blocked_by: [X]` while V5 produces `blocked_by: [X, X]`. Neither the spec nor the tests cover this edge case, but V4's behavior is clearly more correct.

5. **Blocks ID normalization timing:** V4 normalizes blocks IDs during parsing via `parseCommaSeparatedIDs()`. V5 normalizes inside the Mutate callback via `normalizeIDs()` (line 95). Both produce the same result but V5's later normalization is slightly unusual -- typically all input normalization happens at the parsing boundary.

6. **Parent clearing with NormalizeID:** V5 calls `task.NormalizeID(remaining[i])` on the parent value unconditionally at line 186. For `--parent ""`, `NormalizeID("")` returns `""`, so clearing works. V4 explicitly checks `if val != ""` before normalizing (line 189-190), which is clearer about the intent.

7. **Unknown flag handling:** V5 explicitly rejects unknown flags with `case strings.HasPrefix(arg, "-"):` returning `"unknown flag '%s'"`. V4 does not have this check -- an unknown flag like `--foo` would be treated as a positional argument (the ID), potentially leading to confusing errors downstream. V5's explicit rejection is more user-friendly.

8. **CLI registration:** V4 adds 6 lines to `cli.go` (a full case block with error handling). V5 adds 1 line to the commands map. V5's approach eliminates duplicated boilerplate.

### Code Quality

**Imports:**

V4 imports `"github.com/leeovery/tick/internal/store"`. V5 imports `"github.com/leeovery/tick/internal/engine"`. These are the same storage abstraction with different package names reflecting broader architectural differences between the worktrees. Both also import `"fmt"`, `"strconv"`, `"strings"`, and `"time"` from stdlib, plus `"github.com/leeovery/tick/internal/task"`.

**Store creation:**

V4:
```go
s, err := a.openStore(tickDir)
```

V5:
```go
store, err := engine.NewStore(tickDir, ctx.storeOpts()...)
```

V4 uses a centralized `openStore` method that wires up verbose logging internally. V5 uses the functional options pattern, passing `ctx.storeOpts()` which returns `[]engine.Option`. V5's approach aligns with the skill's guidance to use "functional options or env vars" for configuration.

**Error messages:**

Both versions produce consistent user-facing error messages:
- Missing ID: `"Task ID is required. Usage: tick update <id> [options]"` (identical)
- Not found: `"Task '%s' not found"` (identical)
- Self-referencing parent: V4 `"task %s cannot be its own parent"` (inline); V5 delegates to `task.ValidateParent()` which produces `"task %s cannot be its own parent"` (same message)
- Non-existent parent: V4 `"task '%s' not found (referenced in --parent)"` (inline); V5 `"Task '%s' not found (referenced in --parent)"` via `validateIDsExist()` helper (capitalized "Task")

**No-flags error message:**

V4 provides a detailed multi-line help message:
```
No flags provided. At least one flag is required.

Available options:
  --title "<text>"           New title
  --description "<text>"     New description (use "" to clear)
  --priority <0-4>           New priority level
  --parent <id>              New parent task (use "" to clear)
  --blocks <id,...>          Tasks this task blocks
```

V5 provides a concise single-line message:
```
at least one flag is required. Available: --title, --description, --priority, --parent, --blocks
```

The spec says "error with available options list" -- V4 provides a formatted usage guide; V5 provides a comma-separated list. Both satisfy the requirement. V4's is more helpful for users; V5's is terser but still lists all flags.

**Documentation:**

V4's function doc (line 14-15):
```go
// runUpdate implements the `tick update <id>` command.
// It updates task fields (title, description, priority, parent, blocks) with at least one flag required.
```

V5's function doc (line 31-33):
```go
// runUpdate implements the "tick update" command. It modifies task fields after
// creation. At least one flag is required. On success it outputs the full task
// details (or just the ID with --quiet).
```

V5's struct doc (lines 13-14):
```go
// updateOpts holds parsed optional flags for the update command. Pointer fields
// distinguish "flag provided" from "flag not provided" (nil = not provided).
```

V4's struct doc (line 131):
```go
// updateFlags holds parsed flags for the update command.
```

V5's documentation is more thorough -- it explains the pointer vs nil semantics of the options struct and describes both the success and quiet output behaviors. V4's is adequate but briefer.

**Result variable type:**

V4: `var updatedTask *task.Task` (pointer). Assigned via `updatedTask = &tasks[idx]` (line 117). Accessed via `updatedTask.ID` with pointer dereference.

V5: `var updatedTask task.Task` (value). Assigned via `updatedTask = tasks[idx]` (line 131). Value semantics.

V5's value type is safer -- if the Mutate callback were to error before assignment, V4's `updatedTask` would be nil, risking a nil pointer dereference when outputting. V5's would be a zero-value `Task`, which is still wrong but won't panic. In practice both paths only reach output after a successful Mutate, so this is a theoretical concern, but V5's approach is more defensive.

### Test Quality

#### V4 Test Functions (16 top-level functions, ~22 subtests in worktree)

1. **`TestUpdate_TitleFlag`** (1 subtest)
   - `"it updates title with --title flag"` -- creates task with `Title: "Original title"`, runs `tick update tick-aaa111 --title "Updated title"`, verifies exit 0 and title changed

2. **`TestUpdate_DescriptionFlag`** (1 subtest)
   - `"it updates description with --description flag"` -- runs `--description "New description"`, verifies description set

3. **`TestUpdate_ClearDescription`** (1 subtest)
   - `"it clears description with --description \"\""` -- task starts with `Description: "Has a desc"`, runs `--description ""`, verifies empty string

4. **`TestUpdate_PriorityFlag`** (1 subtest)
   - `"it updates priority with --priority flag"` -- starts with priority 2, runs `--priority 0`, verifies priority is 0

5. **`TestUpdate_ParentFlag`** (1 subtest)
   - `"it updates parent with --parent flag"` -- creates 2 tasks, runs `tick update tick-bbb222 --parent tick-aaa111`, finds child by ID and verifies parent set

6. **`TestUpdate_ClearParent`** (1 subtest)
   - `"it clears parent with --parent \"\""` -- child starts with `Parent: "tick-aaa111"`, runs `--parent ""`, verifies empty parent

7. **`TestUpdate_BlocksFlag`** (1 subtest)
   - `"it updates blocks with --blocks flag"` -- runs `tick update tick-aaa111 --blocks tick-bbb222`, verifies target's `BlockedBy` is `[tick-aaa111]` and target's `Updated` timestamp is refreshed (`target.Updated.After(now)`)

8. **`TestUpdate_MultipleFields`** (1 subtest)
   - `"it updates multiple fields in a single command"` -- runs `--title "New title" --description "New desc" --priority 1`, verifies all 3 fields changed

9. **`TestUpdate_RefreshesUpdatedTimestamp`** (1 subtest)
   - `"it refreshes updated timestamp on any change"` -- verifies `Updated.After(now)` and `Created.Equal(now)` (created unchanged)

10. **`TestUpdate_OutputsFullTaskDetails`** (1 subtest)
    - `"it outputs full task details on success"` -- checks output contains `"tick-aaa111"`, `"Updated"` (title), `"open"` (status), `"task{"` (TOON format header)

11. **`TestUpdate_QuietFlag`** (1 subtest)
    - `"it outputs only task ID with --quiet flag"` -- runs `tick --quiet update tick-aaa111 --title "Updated"`, verifies trimmed output is exactly `"tick-aaa111"`

12. **`TestUpdate_ErrorNoFlags`** (1 subtest)
    - `"it errors when no flags are provided"` -- verifies exit 1, stderr contains `"Error:"`, `"No flags provided"`, and `"--title"`

13. **`TestUpdate_ErrorMissingID`** (1 subtest)
    - `"it errors when task ID is missing"` -- runs `tick update --title "Something"` (no ID), verifies exit 1, `"Task ID is required"`

14. **`TestUpdate_ErrorIDNotFound`** (1 subtest)
    - `"it errors when task ID is not found"` -- runs `tick update tick-nonexist --title "Something"`, verifies exit 1, `"not found"`

15. **`TestUpdate_ErrorInvalidTitle`** (1 parent + 4 table-driven subtests)
    - `"it errors on invalid title (empty/500/newlines)"` -- table: empty `""`, whitespace `"   "`, too long (501 chars), newline `"line one\nline two"`. Each verifies exit 1, stderr non-empty, **and no mutation** (`tasks[0].Title` remains `"Original"`)

16. **`TestUpdate_ErrorInvalidPriority`** (1 parent + 4 table-driven subtests)
    - `"it errors on invalid priority (outside 0-4)"` -- table: `"-1"`, `"5"`, `"100"`, `"abc"`. Verifies exit 1, stderr non-empty

17. **`TestUpdate_ErrorNonExistentParentBlocks`** (1 parent + 2 subtests)
    - `"non-existent parent"` -- `--parent tick-nonexist`, verifies `"not found"`
    - `"non-existent blocks target"` -- `--blocks tick-nonexist`, verifies `"not found"`

18. **`TestUpdate_ErrorSelfReferencingParent`** (1 subtest)
    - `"it errors on self-referencing parent"` -- runs `--parent tick-aaa111` on task `tick-aaa111`, verifies `"cannot be its own parent"`

19. **`TestUpdate_NormalizesInputIDs`** (1 subtest)
    - `"it normalizes input IDs to lowercase"` -- uses `"TICK-AAA111"` and `"TICK-BBB222"`, verifies parent stored as lowercase `"tick-bbb222"`

20. **`TestUpdate_PersistsChanges`** (1 subtest)
    - `"it persists changes via atomic write"` -- reads raw `tasks.jsonl` file, parses JSON, verifies `title == "Persisted title"`, checks `cache.db` exists

21. **`TestUpdate_BlocksRejectsCycle`** (1 subtest)
    - `"it rejects --blocks that would create a dependency cycle"` -- B has `blocked_by=[A]`, runs `tick update tick-bbb222 --blocks tick-aaa111` (would add A's `blocked_by=[B]` creating cycle), verifies exit 1, `"cycle"` in error, no mutation on A's `blocked_by`

22. **`TestUpdate_BlocksRejectsChildBlockedByParent`** (1 subtest)
    - `"it rejects --blocks when child would block its own parent"` -- C has `parent=P`, runs `tick update tick-ppp111 --blocks tick-ccc111` (parent blocking child), verifies exit 1, `"cannot"` in error, no mutation on C's `blocked_by`

**V4 test structure notes:**
- Each scenario is a separate top-level `Test*` function with single `t.Run`
- Task setup uses manual struct construction with explicit timestamps: `task.Task{ID: "tick-aaa111", Title: "...", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now}`
- Fixed timestamp: `time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)` used consistently
- Helpers: `setupInitializedDir(t)`, `setupInitializedDirWithTasks(t, tasks)`, `readTasksFromDir(t, dir)`
- CLI invocation: `app.Run([]string{...})` via `&App{Stdout: &stdout, Stderr: &stderr, Dir: dir}`
- Includes `encoding/json`, `os`, `path/filepath` imports for raw file verification
- Total: 16 top-level functions, ~22 distinct subtests

#### V5 Test Functions (1 top-level function, 21 subtests)

All tests are nested under `TestUpdate(t *testing.T)`:

1. `"it updates title with --title flag"` -- creates task via `task.NewTask("tick-aaaaaa", "Original title")`, updates title to `"New title"`, verifies
2. `"it updates description with --description flag"` -- sets description to `"New desc"`, verifies
3. `"it clears description with --description empty string"` -- starts with `tk.Description = "Old desc"`, clears to `""`, verifies
4. `"it updates priority with --priority flag"` -- sets priority to 0, verifies
5. `"it updates parent with --parent flag"` -- creates parent and child tasks, sets parent, verifies
6. `"it clears parent with --parent empty string"` -- starts with parent set, clears to `""`, verifies
7. `"it updates blocks with --blocks flag"` -- verifies target's `BlockedBy` is `[tick-aaaaaa]`
8. `"it updates multiple fields in a single command"` -- updates 4 flags simultaneously (`--title`, `--description`, `--priority`, `--parent`), verifies all 4
9. `"it refreshes updated timestamp on any change"` -- uses `time.Sleep(1100 * time.Millisecond)` to guarantee timestamp difference, verifies `Updated.After(originalUpdated)`
10. `"it outputs full task details on success"` -- runs with `--pretty` flag, checks for `"tick-aaaaaa"`, `"New output"`, `"ID:"` in output
11. `"it outputs only task ID with --quiet flag"` -- verifies trimmed output is `"tick-aaaaaa"`
12. `"it errors when no flags are provided"` -- checks for `"at least one flag is required"`
13. `"it errors when task ID is missing"` -- runs `tick update` with no args, checks `"Task ID is required"`
14. `"it errors when task ID is not found"` -- runs update on nonexistent task, checks `"not found"`
15. `"it errors on invalid title"` (4 table-driven subtests) -- empty, whitespace, 501 chars, newline. Verifies exit 1, `"Error:"` in stderr, no mutation
16. `"it errors on invalid priority"` (3 table-driven subtests) -- `"-1"`, `"5"`, `"99"`. Verifies exit 1, `"Error:"` in stderr
17. `"it errors on non-existent parent ID"` -- `--parent tick-nonexist`, checks `"not found"`
18. `"it errors on non-existent blocks ID"` -- `--blocks tick-nonexist`, checks `"not found"`
19. `"it errors on self-referencing parent"` -- `--parent tick-aaaaaa` on self, checks `"cannot be its own parent"`
20. `"it normalizes input IDs to lowercase"` -- uses `"TICK-AAAAAA"` and `"TICK-BBBBBB"`, verifies lowercase
21. `"it persists changes via atomic write"` -- re-reads from file, verifies title
22. `"it rejects --blocks that would create a cycle"` -- direct cycle test: C has `blocked_by=[A]`, update C `--blocks A` creates cycle. Verifies exit 1, `"cycle"`, no mutation
23. `"it rejects --blocks that would create an indirect cycle"` -- A blocked_by B, B blocked_by C. Update A `--blocks C` would create chain C->A->B->C. Verifies exit 1, `"cycle"`, no mutation on C
24. `"it accepts valid --blocks dependency"` -- A and B independent, update A `--blocks B`, verifies success and B's `blocked_by=[A]`

**V5 test structure notes:**
- Single top-level `TestUpdate` with all subtests nested
- Task setup via constructor: `task.NewTask("tick-aaaaaa", "...")`, then mutates fields as needed
- Uses `time.Sleep(1100 * time.Millisecond)` for timestamp tests (makes test slower)
- Helpers: `initTickProject(t)`, `initTickProjectWithTasks(t, tasks)`, `readTasksFromFile(t, dir)`
- CLI invocation: `Run([]string{...}, dir, &stdout, &stderr, false)` -- free function
- Imports only `bytes`, `strings`, `testing`, `time`, and `task` (no `encoding/json`, `os`, or `path/filepath`)
- Total: 1 top-level function, ~24 subtests

#### Test Coverage Comparison

| Edge Case | V4 | V5 |
|-----------|-----|-----|
| Update title | Yes | Yes |
| Update description | Yes | Yes |
| Clear description with empty string | Yes | Yes |
| Update priority | Yes | Yes |
| Update parent | Yes | Yes |
| Clear parent with empty string | Yes | Yes |
| Update blocks (single target) | Yes | Yes |
| Target's `blocked_by` updated | Yes | Yes |
| Target's `Updated` timestamp refreshed on --blocks | Yes (`target.Updated.After(now)`) | No (not checked) |
| Multiple flags combined | Yes (3 flags) | Yes (4 flags -- broader) |
| Updated timestamp refreshed | Yes (checks `Created` unchanged too) | Yes (uses sleep-based approach) |
| Full task details output | Yes (checks ID, title, status, priority, TOON format) | Yes (checks ID, title, ID label; uses `--pretty` flag) |
| Quiet outputs ID only | Yes | Yes |
| Error: no flags | Yes (checks "No flags provided" + options list) | Yes (checks "at least one flag is required") |
| Error: missing ID | Yes | Yes |
| Error: not found ID | Yes | Yes |
| Invalid title: empty | Yes | Yes |
| Invalid title: whitespace | Yes | Yes |
| Invalid title: >500 chars | Yes | Yes |
| Invalid title: newlines | Yes | Yes |
| No mutation on invalid title | Yes (verifies original title preserved) | Yes (verifies original title preserved) |
| Invalid priority: negative | Yes (`"-1"`) | Yes (`"-1"`) |
| Invalid priority: too high | Yes (`"5"`) | Yes (`"5"`) |
| Invalid priority: way too high | Yes (`"100"`) | Yes (`"99"`) |
| Invalid priority: non-numeric | Yes (`"abc"`) | No (not tested) |
| Non-existent parent | Yes | Yes |
| Non-existent blocks | Yes | Yes |
| Self-referencing parent | Yes | Yes |
| ID normalization | Yes (uppercase both ID and parent) | Yes (uppercase both ID and parent) |
| Persistence via raw file read | Yes (parses JSONL + checks cache.db) | Yes (re-reads via helper) |
| Raw JSONL format verified | Yes (`json.Unmarshal`, field check) | No (uses helper abstraction) |
| cache.db existence check | Yes | No |
| Cycle detection in --blocks | Yes (direct A<->B cycle) | Yes (direct + indirect 3-node cycle) |
| Indirect cycle detection | No | Yes (A->B->C->A chain) |
| Valid --blocks acceptance test | No | Yes (positive test for non-cyclic blocks) |
| Child blocked by parent rejection | Yes (parent P blocks child C) | No |

**Notable coverage differences:**

- V4 verifies the blocked target's `Updated` timestamp is refreshed when `--blocks` is applied (line `"expected target updated timestamp to be refreshed"`). V5 does not check this, which is a gap since the spec requires "refreshes their `updated`" for blocked targets.
- V4 tests non-numeric priority input (`"abc"`). V5 does not, missing the `strconv.Atoi` error path.
- V4 verifies raw JSONL format in the persistence test, parsing the file directly with `json.Unmarshal` and checking `cache.db` existence. V5 uses the `readTasksFromFile` helper, which is less rigorous but also less coupled to storage internals.
- V5 tests an indirect (3-node) cycle `A->B->C->A` and includes a positive acceptance test for valid blocks. V4 does not have these.
- V4 tests the child-blocked-by-parent constraint (`TestUpdate_BlocksRejectsChildBlockedByParent`). V5 does not cover this case.
- V5's output test uses `--pretty` flag to force pretty-printing format. V4's test checks for TOON format (`"task{"`). Both are valid approaches but test different formatters.
- V4's multi-field test combines 3 flags; V5 combines 4 flags (additionally including `--parent`), making V5's combination test slightly more thorough.

### Skill Compliance

| Constraint | V4 | V5 |
|------------|-----|-----|
| Handle all errors explicitly (no naked returns) | PASS -- all errors from `DiscoverTickDir`, `openStore`, `Mutate`, `ValidateTitle`, `ValidatePriority` checked and returned | PASS -- all errors checked; additionally, title and priority validation errors returned before store opens |
| Write table-driven tests with subtests | PARTIAL -- `TestUpdate_ErrorInvalidTitle` and `TestUpdate_ErrorInvalidPriority` use table-driven patterns; most tests are individual top-level functions | PARTIAL -- `"it errors on invalid title"` and `"it errors on invalid priority"` use table-driven patterns; all others are individual subtests under one parent |
| Document all exported functions, types, and packages | PASS -- `runUpdate`, `updateFlags`, `hasAnyFlag`, `parseUpdateArgs` all documented (unexported but documented) | PASS -- `runUpdate`, `updateOpts`, `hasAnyFlag`, `parseUpdateArgs` all documented with more detail than V4 |
| Propagate errors with fmt.Errorf("%w", err) | PARTIAL -- V4 wraps title error: `fmt.Errorf("invalid title: %w", err)` at line 58; other errors returned directly | PARTIAL -- V5 returns title/priority errors directly (no wrapping); uses shared `validateIDsExist` helper for existence errors |
| No hardcoded configuration | PASS -- no magic values beyond inline error messages | PASS -- no magic values |
| No panic for normal error handling | PASS -- no panics | PASS -- no panics |
| Avoid _ assignment without justification | PASS -- no ignored errors | PASS -- no ignored errors |
| Use functional options or env vars | N/A (V4 uses `a.openStore`) | PASS -- `engine.NewStore(tickDir, ctx.storeOpts()...)` uses functional options pattern |

### Spec-vs-Convention Conflicts

**1. Capitalized error messages**

- **Spec says:** `"No flags is an error"`, `"Task '{id}' not found"` -- capitalized user-facing messages.
- **Go convention:** Error strings should not be capitalized.
- **V4 chose:** Capitalized: `"No flags provided. At least one flag is required."`, `"Task '%s' not found"`, `"Task ID is required. Usage: tick update <id> [options]"`.
- **V5 chose:** Mixed: `"at least one flag is required."` (lowercase, following Go convention), `"Task '%s' not found"` (capitalized, following spec), `"Task ID is required."` (capitalized).
- **Assessment:** V5 is inconsistent -- some messages are lowercase (Go convention) while others are capitalized (spec convention). V4 is consistently capitalized, matching the spec throughout. For user-facing CLI error messages, consistent capitalization is appropriate.

**2. Error message detail for no-flags case**

- **Spec says:** "error with available options list".
- **V4:** Multi-line formatted help with descriptions for each flag.
- **V5:** Single-line comma-separated flag names.
- **Assessment:** Both satisfy the spec. V4 provides more value to the user but produces noisier error output. The spec does not specify format detail.

**3. Title error wrapping**

- **V4:** Wraps with context: `fmt.Errorf("invalid title: %w", err)`. Skill-compliant.
- **V5:** Returns raw error from `task.ValidateTitle`. Less context but simpler.
- **Assessment:** V4 follows the skill constraint more closely. The additional "invalid title:" prefix helps users understand which flag caused the error.

No other spec-vs-convention conflicts identified.

## Diff Stats

| Metric | V4 | V5 |
|--------|-----|-----|
| Files changed (commit) | 5 (cli.go, update.go, update_test.go, 2 docs) | 5 (cli.go, update.go, update_test.go, 2 docs) |
| Lines added (total, commit) | 951 | 671 |
| Lines added (internal/, commit) | 946 | 666 |
| Impl LOC (update.go, commit) | 228 | 216 |
| Test LOC (update_test.go, commit) | 712 | 449 |
| cli.go lines changed | +6 | +1 |
| Impl LOC (update.go, worktree) | 232 | 220 |
| Test LOC (update_test.go, worktree) | 790 | 544 |
| Top-level test functions | 16 | 1 |
| Total test subtests (worktree) | ~22 | ~24 |

## Verdict

**V5 is the slightly better implementation.**

Both versions fully satisfy all 9 acceptance criteria and produce functionally correct behavior. The differences are in code quality, architecture, and test coverage, with each version having distinct advantages.

**V5 advantages:**

1. **Early validation (significant):** V5 validates title and priority **before** opening the store (lines 39-50), avoiding unnecessary filesystem access on invalid input. V4 performs all validation inside the Mutate callback, meaning it opens the store, loads all tasks, and enters the mutation path before discovering a simple input error. This is a meaningful efficiency difference, especially for `--priority abc` or `--title ""` cases.

2. **O(1) index lookup for blocks (moderate):** V5's `map[string]int` storing task indices enables direct index access when applying `--blocks` (`tasks[existing[blockID]]`), while V4's `map[string]bool` requires a separate O(n) scan over all tasks. For small task sets this is negligible, but it is a better algorithmic choice.

3. **Unknown flag handling (moderate):** V5 explicitly rejects unknown flags (`"unknown flag '%s'"`) via `strings.HasPrefix(arg, "-")`. V4 silently treats unknown flags as positional arguments, potentially causing confusing downstream errors.

4. **CLI registration (minor):** V5 adds 1 line to `cli.go`; V4 adds 6 lines with duplicated boilerplate.

5. **Broader cycle testing (moderate):** V5 tests indirect 3-node cycles (A->B->C->A) and includes a positive acceptance test for valid blocks, providing more confidence in cycle detection.

6. **Value semantics (minor):** V5's value-type `updateOpts` return and value-type `updatedTask task.Task` are more idiomatic Go for small structs.

7. **Shared helpers (minor):** V5 reuses `validateIDsExist`, `splitCSV`, and `normalizeIDs` from `create.go`, reducing code duplication. V4 only reuses `parseCommaSeparatedIDs`.

**V4 advantages:**

1. **Duplicate blocked_by prevention (moderate):** V4 checks whether a task ID already exists in the target's `blocked_by` before appending. V5 always appends, which could create duplicates if `--blocks` is run multiple times. While neither the spec nor tests cover this edge case, V4's behavior is more correct.

2. **Blocked target timestamp verification (minor):** V4's blocks test verifies the target's `Updated` timestamp is refreshed. V5 does not check this.

3. **Non-numeric priority test (minor):** V4 tests `"abc"` as a priority value, exercising the `strconv.Atoi` error path. V5 does not.

4. **Child-blocked-by-parent test (minor):** V4 tests the parent-child blocking constraint. V5 does not cover this case.

5. **Consistent error capitalization (minor):** V4 consistently capitalizes user-facing error messages. V5 is inconsistent (lowercase `"at least one flag is required"` vs capitalized `"Task '%s' not found"`).

6. **Title error wrapping (minor):** V4 wraps title validation errors with `fmt.Errorf("invalid title: %w", err)`, providing clearer context about which flag caused the error.

7. **Deeper persistence verification (minor):** V4's persistence test parses raw JSONL and checks `cache.db` existence. V5 uses a helper abstraction.

**Overall assessment:**

V5's early validation pattern is the most significant architectural difference -- it avoids opening the store and loading all tasks when the user provides obviously invalid input (empty title, non-numeric priority). This is a genuine quality improvement that V4 lacks entirely. Combined with the O(1) index lookup, explicit unknown flag rejection, and 1-line CLI registration, V5's implementation code is cleaner and more efficient.

V4's duplicate-prevention in blocks is the strongest counterargument, but it addresses an edge case the spec doesn't require. V4's additional test assertions (blocked target timestamp, non-numeric priority, child-blocked-by-parent) cover useful edge cases, but V5 compensates with indirect cycle testing and a positive acceptance test.

The margin is narrow, but V5's structural improvements in the implementation code give it the edge over V4's incremental test coverage advantages.
