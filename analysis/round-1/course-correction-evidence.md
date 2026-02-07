# Course Correction Evidence: V2 vs V3

## V2 Retroactive Improvements

### 1. Task 1-4 (Storage Engine) modified Task 1-2's JSONL code

**Commit:** `352e9d8`
**Files modified from earlier tasks:**
- `internal/storage/jsonl/jsonl.go` (created by task 1-2)
- `internal/storage/jsonl/jsonl_test.go` (created by task 1-2)

**What changed:** V2's task 1-4 executor refactored `ReadTasks()` to delegate to a new `ParseTasks(data []byte)` function. The original `ReadTasks` opened a file and used `bufio.Scanner` on the file handle. The refactored version uses `os.ReadFile` to get raw bytes and passes them to `ParseTasks`, which uses `bufio.Scanner` on a `strings.NewReader`. Two new tests were added for `ParseTasks`.

**Was it an improvement?** Yes. This was a proactive architectural improvement. The store needed to parse JSONL from raw bytes (already in memory after `os.ReadFile` for hash computation), not from a file handle. Without `ParseTasks`, V2 would have had to either read the file twice (like V3 ended up doing) or duplicate parsing logic. V2's executor recognized this and improved the earlier code rather than working around it.

**Nature:** Proactive refactoring to enable cleaner composition. Not a bug fix -- the original code worked -- but a design improvement for the new consumer.

### 2. Task 2-1 (Status Transition) modified Task 1-1's task.go

**Commit:** `a3f0065`
**Files modified from earlier tasks:**
- `internal/task/task.go` (created by task 1-1)
- `internal/task/task_test.go` (created by task 1-1)

**What changed:** Added the `Transition()` method and related status validation logic directly to the existing `task.go` file, alongside 390 lines of tests.

**Was it an improvement?** This is expected behavior -- adding methods to an existing type. Not course correction per se, but demonstrates willingness to modify earlier files.

### 3. Task 2-3 (Update Command) retroactively fixed Task 2-2 and Task 1-6

**Commit:** `477fe0e`
**Files modified from earlier tasks:**
- `internal/cli/create.go` (created by task 1-6): 2 changes
- `internal/cli/create_test.go` (created by task 1-6): 1 new test, 1 assertion added
- `internal/cli/transition.go` (created by task 2-2): error unwrapping fix
- `internal/cli/app.go` (created by task 1-5): added `unwrapMutationError` helper

**What changed -- specifically:**

**(a) `transition.go` -- Error unwrapping fix:**
The task 2-2 executor had written inline error unwrapping for the `"mutation failed:"` prefix:
```go
errMsg := err.Error()
if strings.HasPrefix(errMsg, "mutation failed: ") {
    return fmt.Errorf("%s", strings.TrimPrefix(errMsg, "mutation failed: "))
}
return err
```

Task 2-3's executor replaced this with a shared helper:
```go
return unwrapMutationError(err)
```

And added `unwrapMutationError` to `app.go`:
```go
func unwrapMutationError(err error) error {
    if inner := errors.Unwrap(err); inner != nil {
        return inner
    }
    return err
}
```

This is a genuine improvement: (1) the original string manipulation was fragile, (2) `errors.Unwrap` is the idiomatic Go approach, (3) extracting a shared helper prevents future commands from repeating the same mistake.

**(b) `create.go` -- Duplicate blocked_by fix:**
The task 1-6 executor had written a naive `--blocks` implementation that could append duplicate entries to a target's `blocked_by` array. Task 2-3 added deduplication logic:
```go
alreadyPresent := false
for _, existingID := range modified[i].BlockedBy {
    if task.NormalizeID(existingID) == task.NormalizeID(newTask.ID) {
        alreadyPresent = true
        break
    }
}
```

This is a bug fix applied to earlier code.

**(c) `create.go` -- Applied unwrapMutationError:**
Changed `return err` to `return unwrapMutationError(err)`, applying the same error-handling improvement to the create command that 2-3 created for itself.

**(d) `create_test.go` -- Added retroactive tests:**
Added a test asserting that mutation errors don't expose the `"mutation failed:"` prefix, and a test for the duplicate `--blocks` edge case.

**Was it an improvement?** Unambiguously yes. This is the strongest evidence of V2 cross-task course correction. Task 2-3's executor identified problems in TWO earlier tasks' code, fixed them, and added tests for the fixes -- all while also implementing its own feature. The executor was not asked to do this; it chose to after seeing the patterns while reading the existing code.

### 4. Task 3-1 (Dependency Validation) modified Task 1-1/2-1's task.go

**Commit:** `e1acede`
**Files modified from earlier tasks:**
- `internal/task/task.go` (created by task 1-1, modified by task 2-1)
- `internal/task/task_test.go` (created by task 1-1, modified by task 2-1)

**What changed:** Added `ValidateDependency()` function (105 lines) and `detectCycle()` helper to the existing task.go, with 181 lines of tests.

**Was it an improvement?** This is expected behavior -- adding new functions to an existing package file. No earlier code was modified or improved.

### 5. Task 3-3 (Ready Command) modified Task 1-7's list.go

**Commit:** `819aa39`
**Files modified from earlier tasks:**
- `internal/cli/list.go` (created by task 1-7)
- `internal/cli/app.go` (created by task 1-5)

**What changed:** Added `ReadySQL` constant, `parseListFlags` function, and `listAllSQL` constant to `list.go`. Changed `runList` to accept `args []string` parameter and use the new flag parsing. Added `case "ready": return a.runList([]string{"--ready"})` to app.go.

**Was it an improvement?** This is architectural enhancement. The ready command was designed as a filter on `list`, so task 3-3 refactored `runList` to support flag-driven behavior. The original `runList` was parameter-less; it was changed to accept args and dispatch based on flags. This is a forward-looking improvement, not a bug fix.

### 6. Task 3-4 (Blocked Command) modified Task 3-3's additions to list.go

**Commit:** `b97f76f`
**Files modified from earlier tasks:**
- `internal/cli/list.go` (created by 1-7, modified by 3-3)
- `internal/cli/app.go` (created by 1-5)

**What changed:** Extended the `parseListFlags` to also handle `--blocked`. Converted the `ready bool` return to a `listFlags` struct. Added `BlockedSQL` constant.

**Was it an improvement?** Yes -- the task 3-4 executor recognized that 3-3's `parseListFlags` was too narrow (returning a bare `bool`) and refactored it to return a struct that could accommodate multiple flags. This is genuine code improvement on an earlier task's work.

### 7. Task 3-5 (List Filter Flags) modified Task 3-3/3-4's list.go

**Commit:** `5d325d7`
**Files modified from earlier tasks:**
- `internal/cli/list.go` (created by 1-7, modified by 3-3 and 3-4)

**What changed:** Major refactoring of the SQL constants. Task 3-4 had defined `ReadySQL` and `BlockedSQL` as complete queries. Task 3-5 refactored them into `readyWhere` and `blockedWhere` clause fragments, then reconstituted the full queries via string concatenation. This enabled composing WHERE clauses with additional `--status` and `--priority` filters.

**Was it an improvement?** Yes -- the original SQL constants were monolithic. The refactoring decomposed them into composable fragments, enabling the filter flags to work with ready/blocked as orthogonal filters. The task 3-5 executor actively restructured earlier task decisions.

### 8. Task 4-1 (Formatter Abstraction) modified Task 1-5/1-6's app.go

**Commit:** `e10ae58`
**Files modified from earlier tasks:**
- `internal/cli/app.go` (created by 1-5)
- `internal/cli/app_test.go` (created by 1-5)

**What changed:** Restructured the `App` struct to add `FormatCfg FormatConfig`. Replaced the inline TTY detection and format assignment with `ResolveFormat()`. Changed `parseGlobalFlags` to track individual format flags and detect conflicts. Removed the inline `detectTTY` function and replaced it with `DetectTTY` (exported) from the new formatter package.

**Was it an improvement?** Yes -- significant refactoring of earlier code to accommodate the new formatter system. The old format logic was ad-hoc; the new version is structured and conflict-detecting.

### 9. Task 4-5 (Formatter Integration) modified 12 files from earlier tasks

**Commit:** `82c22b4`
**Files modified from earlier tasks:**
- `internal/cli/app.go` (1-5): added `formatter Formatter` field + `newFormatter()` function
- `internal/cli/create.go` (1-6): replaced `printTaskDetails` with `queryShowData` + formatter
- `internal/cli/dep.go` (3-2): replaced `fmt.Fprintf` with `formatter.FormatDepChange`
- `internal/cli/init.go` (1-5): replaced `fmt.Fprintf` with `formatter.FormatMessage`
- `internal/cli/list.go` (1-7): replaced `fmt.Fprintf` columns with `formatter.FormatTaskList`
- `internal/cli/show.go` (1-7): extracted `queryShowData`, replaced `printShowOutput` with formatter
- `internal/cli/transition.go` (2-2): replaced `fmt.Fprintf` with `formatter.FormatTransition`
- `internal/cli/update.go` (2-3): replaced `printTaskDetails` with `queryShowData` + formatter
- Plus 4 test files updated

**Was it an improvement?** Yes, by design. Task 4-5 was explicitly scoped to replace hardcoded output, but the *quality* of the modifications matters. V2's executor deleted `printTaskDetails` (a utility function created by task 1-6 and reused by 2-3), replaced `printShowOutput` from task 1-7, and extracted `queryShowData` as a reusable function. It removed ~126 lines of old output code and replaced it with clean formatter calls. The executor consolidated two different output paths (show's `printShowOutput` and create/update's `printTaskDetails`) into a single shared `queryShowData` function -- a genuine DRY improvement.

### Summary: V2 Cross-Task Modification Count

| V2 Task | Files Modified from Earlier Tasks | Nature |
|---------|----------------------------------|--------|
| 1-4 (Storage) | jsonl.go, jsonl_test.go (from 1-2) | Proactive refactoring |
| 2-1 (Transitions) | task.go, task_test.go (from 1-1) | Expected: adding to type |
| 2-3 (Update) | create.go, create_test.go (from 1-6), transition.go (from 2-2), app.go (from 1-5) | **Bug fix + DRY improvement** |
| 3-1 (Deps) | task.go, task_test.go (from 1-1/2-1) | Expected: adding to package |
| 3-3 (Ready) | list.go (from 1-7), app.go (from 1-5) | Refactoring for extensibility |
| 3-4 (Blocked) | list.go (from 1-7/3-3), app.go (from 1-5) | Refactoring flag struct |
| 3-5 (Filters) | list.go (from 1-7/3-3/3-4) | Refactoring SQL into fragments |
| 4-1 (Format Abstraction) | app.go, app_test.go (from 1-5) | Structural refactoring |
| 4-5 (Format Integration) | 12 files from 6 earlier tasks | Systematic output replacement |

**Genuine course corrections (not just additive):** Tasks 1-4, 2-3, 3-4, 3-5, 4-1, 4-5 -- **6 tasks** actively improved or refactored code from earlier tasks.

**Bug fixes applied retroactively:** Task 2-3 fixed 2 bugs in earlier code (duplicate blocked_by in create, fragile error unwrapping in transition).

## V3 Retroactive Improvements (Pre-Polish)

### Analysis of V3 Cross-Task Modifications

V3's modification pattern is strikingly different from V2's. Looking at the `--diff-filter=M` output:

| V3 Task | Files Modified from Earlier Tasks | Nature |
|---------|----------------------------------|--------|
| 1-6 (Create) | cli.go (from 1-5) | Expected: adding switch case |
| 1-7 (List/Show) | cli.go (from 1-5), create_test.go (from 1-6) | Expected: adding switch case, test helpers |
| 2-2 (Transitions) | cli.go (from 1-5) | Expected: adding switch case |
| 2-3 (Update) | cli.go (from 1-5) | Expected: adding switch case |
| 3-2 (Dep) | cli.go (from 1-5) | Expected: adding switch case |
| 3-3 (Ready) | cli.go (from 1-5) | Expected: adding switch case |
| 3-4 (Blocked) | cli.go (from 1-5) | Expected: adding switch case |
| 3-5 (Filters) | list.go (from 1-7) | Extending with new flags |
| 4-1 (Format) | cli.go, cli_test.go (from 1-5) | Adding format config |
| 4-5 (Format Integration) | 16 files from earlier tasks | Systematic output replacement |

**Critical difference from V2:**

1. **V3 task 2-3 (Update) did NOT modify `create.go` or `transition.go`.** It only added `update.go`, `update_test.go`, and modified `cli.go` (to add the switch case). V3's update executor did not detect or fix the duplicate blocked_by issue in create, nor did it introduce `unwrapMutationError`. Each V3 task was self-contained.

2. **V3 task 1-4 (Storage) did NOT modify `jsonl.go`.** V3 task 1-4 created only `store.go` and `store_test.go` (plus go.mod/go.sum). It did not refactor the JSONL package. As a result, V3's store reads the file twice -- once via `os.ReadFile` for raw bytes and once via `ReadJSONL(path)` which opens the file again.

3. **V3 task 3-3 (Ready) created a NEW file `ready.go`** instead of extending `list.go`. V2's 3-3 refactored `runList` to support flags; V3 created an entirely separate `runReady` function with its own SQL and output logic. This avoided modifying earlier code but created code duplication.

4. **V3 task 3-4 (Blocked) created a NEW file `blocked.go`** instead of extending `list.go`. Same pattern as 3-3 -- V3 preferred isolation over integration.

5. **V3 task 3-5 (Filters) modified `list.go`** but did NOT refactor ready.go or blocked.go's SQL to share WHERE clause fragments. The filter flags only apply to `tick list`, not to `tick ready` or `tick blocked`. This led to duplication that the V3 polish commit later had to address.

6. **V3's modifications to `cli.go`** across tasks 1-6 through 3-4 were exclusively additive: adding `case "xxx": return a.runXxx(args)` lines. No refactoring, no structural changes. This is the minimum-touch pattern expected when following "match existing conventions."

### V3 Genuine Course Corrections: **Zero (pre-polish)**

Before the polish commit, no V3 task modified or improved the implementation quality of an earlier task's code. Every modification was either:
- Adding a switch case to `cli.go` (the command dispatcher)
- Extending list.go with new filter flags (task 3-5, expected scope)
- Adding `--pretty` flags to earlier tests (task 4-5, necessary for format integration)

V3 never went back to fix a bug, refactor a pattern, or extract a shared helper from earlier code.

### V3 Polish Commit (Post-execution)

**Commit:** `dcdb7a8`
**Files changed:** 9 files, 77 insertions, 258 deletions

The polish commit did what V2's executors had been doing organically:
- Removed dead code (`StubFormatter`, `printTaskDetails`, `DefaultOutputFormat`)
- Extracted `scanTaskRows` and `renderTaskList` shared helpers (DRY)
- Added missing dependency validation to `create --blocked-by` and `update --blocks`
- Cleaned up `blocked.go`, `ready.go`, `list.go` duplication

This is explicit evidence that V3 accumulated technical debt that individual executors never addressed, requiring a dedicated cleanup pass.

## Comparison

### V2: 6 tasks with genuine course corrections, 0 polish commits needed
### V3: 0 tasks with genuine course corrections, 1 polish commit required

V2's stateless executors were paradoxically **more willing** to modify earlier code than V3's context-connected executors. This appears counterintuitive but has a clear explanation:

**V2 executors read the actual code.** Without integration context to tell them "here's how things work," they had to examine the existing files. Upon examination, they found issues and fixed them. The code itself was the context mechanism.

**V3 executors read the integration context.** The context file told them the conventions, patterns, and APIs. They followed those conventions faithfully, which meant replicating whatever patterns were described -- even if those patterns had flaws. The context file said "use `printTaskDetails()` for output" so V3 used it. V2 found `printTaskDetails()` in the code, noticed it was inconsistent with `printShowOutput()`, and consolidated them.

**V3's "match existing conventions" instruction** was specifically designed to prevent drift. But it also prevented improvement. V3 executors were told to match conventions, not to improve them. This created a conservative bias where each executor assumed the existing code was correct and designed its output to fit, rather than questioning whether the existing code was optimal.

**V2's lack of convention guidance** meant each executor had to independently evaluate the codebase. This independence led to some inconsistencies (V2 had a higher V1-style divergence in early tasks) but also led to active improvement when an executor identified something wrong.

The integration context mechanism traded **correction capacity** for **consistency**. V3 was more consistent task-to-task but accumulated latent issues. V2 was less consistent but self-correcting.

## Could V2 Have Recovered from String Timestamps?

This is the key counterfactual: if V2's task 1-1 had chosen `string` for timestamps (as V3 did), would later V2 tasks have independently switched to `time.Time`?

### Evidence For Recovery (YES)

1. **V2 task 1-4 proactively refactored jsonl.go.** The task 1-4 executor modified code from task 1-2 to add `ParseTasks()`. This demonstrates willingness to change foundational code when the executor sees a need.

2. **V2 task 2-3 retroactively fixed 2 bugs in earlier code.** The task 2-3 executor modified `create.go` (task 1-6) and `transition.go` (task 2-2) without being asked. If it found a timestamp issue, it would likely fix it too.

3. **V2 task 3-5 decomposed SQL from earlier tasks.** The executor restructured monolithic SQL constants into composable fragments. This shows willingness to fundamentally restructure earlier decisions.

4. **V2 task 4-1 rewrote the format resolution from earlier code.** The executor didn't just add to `app.go` -- it removed and replaced the old TTY detection and format logic.

### Evidence Against Recovery (NO)

1. **String timestamps aren't obviously "wrong."** Unlike the `"mutation failed:"` error prefix (which is clearly a UX bug) or the missing dedup check (which is clearly a data integrity bug), string timestamps are a reasonable design choice. An executor would have to independently conclude that `time.Time` is *better* and then make the change. This requires more judgment than fixing an obvious bug.

2. **Changing the timestamp type is a breaking change.** Switching from `string` to `time.Time` would require changing the `Task` struct, all serialization code, all comparison code, and all tests. The scope of change is much larger than adding `unwrapMutationError()`. An executor might judge the cost too high for the benefit.

3. **Task 2-3's fixes were small, surgical changes.** Adding `unwrapMutationError` was 5 lines. Adding dedup was 7 lines. A timestamp type change would touch every file that uses `Task`. V2 executors showed willingness to make small cross-task improvements, not large architectural rewrites.

4. **No V2 executor touched timestamp handling.** Tasks 2-1 (transitions), 2-3 (update), and 3-1 (deps) all work with timestamps but none questioned the `time.Time` choice. They used it as-is. If the type had been `string`, they would have used strings as-is too.

### Verdict

**Partial recovery possible, full recovery unlikely.** If V2's task 1-1 had chosen strings, a later task might have added a parsing helper (like `parseTimestamp(s string) time.Time`) to make timestamps more usable -- similar to how task 1-4 added `ParseTasks()` to make JSONL more usable. But it is unlikely that any executor would have changed the fundamental type in the `Task` struct from `string` to `time.Time`, because:
- It would require coordinated changes across too many files
- String timestamps work correctly even if they're less type-safe
- Each executor would independently decide "I'll work with what's here" for something this pervasive

V2's self-correction worked best for **local, clearly-wrong patterns** (bugs, code smell in a specific function). It did not extend to **foundational architectural decisions** that permeate the entire codebase. The initial task's timestamp choice would have stuck regardless of approach.

## Conclusion

V2's stateless executors demonstrated a pattern of organic course correction that V3's context-connected executors did not exhibit. Across 6 tasks, V2 executors actively improved earlier code: refactoring for composability (1-4, 3-5), fixing bugs (2-3), extracting shared helpers (2-3, 4-5), and restructuring APIs (3-4, 4-1). V3 executors, by contrast, treated earlier code as immutable convention and added new code alongside it, creating duplication that required a polish commit to resolve.

This finding has significant implications for multi-agent architecture:

1. **Integration context creates conservatism.** When executors are told "match existing conventions," they stop questioning whether those conventions are optimal. This prevents drift but also prevents improvement.

2. **Code-as-context enables correction.** When executors must read actual code (because they have no summary), they evaluate it critically. This evaluation sometimes leads to beneficial changes that no planning or review step would have caught.

3. **V2's approach is self-healing for local defects.** Bugs and code smells within individual functions/methods are likely to be noticed and fixed by the next executor who reads that code. V3's approach requires explicit review to find these issues.

4. **Neither approach self-heals foundational decisions.** A wrong architectural choice in task 1-1 (like timestamp types) would persist in both V2 and V3, because the scope of change is too large for any individual task executor to justify.

The optimal approach may be a hybrid: use integration context for API contracts and structural patterns (preventing drift), but explicitly instruct executors to "improve code quality in any files you touch" (enabling correction). This would combine V3's consistency with V2's self-healing properties.
