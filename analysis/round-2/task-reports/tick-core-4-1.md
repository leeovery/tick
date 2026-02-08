# Task tick-core-4-1: Formatter Abstraction & TTY-Based Format Selection

## Task Summary

Define a `Formatter` interface, implement TTY detection, resolve format from flags vs auto-detection with conflict handling, and wire the chosen formatter into CLI dispatch. This is the foundation for tasks 4-2 through 4-4.

Required deliverables:
- `Formatter` interface with methods: `FormatTaskList`, `FormatTaskDetail`, `FormatTransition`, `FormatDepChange`, `FormatStats`, `FormatMessage`
- `Format` enum: `FormatToon`, `FormatPretty`, `FormatJSON`
- `DetectTTY()`: `os.Stdout.Stat()` -> check `ModeCharDevice`; stat failure -> default non-TTY
- `ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY)`: >1 flag -> error; 1 flag -> that format; 0 flags + TTY -> Pretty; 0 flags + no TTY -> Toon
- `FormatConfig` struct: Format, Quiet, Verbose -- passed to all handlers
- Stub formatter as placeholder
- `--verbose` to stderr only, never contaminates stdout

**Acceptance Criteria:**
1. Formatter interface covers all command output types
2. Format enum with 3 constants
3. TTY detection works correctly
4. ResolveFormat handles all flag/TTY combos
5. Conflicting flags -> error
6. FormatConfig wired into CLI dispatch
7. Verbose to stderr only
8. Stat failure handled gracefully

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| Formatter interface covers all command output types | PASS -- 6 methods: `FormatTaskList`, `FormatTaskDetail`, `FormatTransition`, `FormatDepChange`, `FormatStats`, `FormatMessage` | PASS -- Same 6 methods with slightly different signatures (see Approach section) |
| Format enum with 3 constants | PASS -- `FormatTOON`, `FormatPretty`, `FormatJSON` as `OutputFormat string` constants | PASS -- `FormatToon`, `FormatPretty`, `FormatJSON` as `Format int` iota constants |
| TTY detection works correctly | PASS -- `DetectTTY()` -> `DetectTTYFrom()` with injectable stat function | PASS -- `DetectTTY(w io.Writer)` checks `*os.File` type assertion then `Stat()` |
| ResolveFormat handles all flag/TTY combos | PASS -- Tested for all 3 flags x 2 TTY states + defaults | PASS -- Tested for each flag override + defaults |
| Conflicting flags -> error | PASS -- Returns error with descriptive message | PASS -- Returns error with descriptive message |
| FormatConfig wired into CLI dispatch | PASS -- `App.FormatCfg` set before subcommand dispatch, tested via `App.Run` integration | PASS -- `FormatCfg` added to App struct, built after flag parsing; also `Formatter` resolved. Note: FormatCfg was later removed from V4 codebase (visible in worktree) suggesting it was superseded |
| Verbose to stderr only | PASS -- `logVerbose()` on App writes to stderr; tested with stdout/stderr capture | PARTIAL -- `VerboseLogger` exists from prior task, `vlog` wired in `Run()`. The task commit does not add new verbose-to-stderr tests; relies on prior task's `verbose_test.go` |
| Stat failure handled gracefully | PASS -- Tested via `DetectTTYFrom` with error-returning stat function | PASS -- Tested via closed `*os.File` whose `Stat()` returns error |

## Implementation Comparison

### Approach

**File naming:**
- V2: `formatter.go` / `formatter_test.go` (new), modifies `app.go` / `app_test.go`
- V4: `format.go` / `format_test.go` (new), modifies `cli.go` / `cli_test.go`

**Format type:**

V2 uses string-based format constants (pre-existing `OutputFormat string` type):
```go
// V2 app.go (pre-existing, not changed by this task)
type OutputFormat string
const (
    FormatTOON   OutputFormat = "toon"
    FormatPretty OutputFormat = "pretty"
    FormatJSON   OutputFormat = "json"
)
```

V4 replaces the string type with an `int` iota enum (moves definition from `cli.go` to `format.go`):
```go
// V4 format.go:9-18
type Format int
const (
    FormatToon Format = iota
    FormatPretty
    FormatJSON
)
```
V4's `iota`-based approach is more idiomatic Go for enumerations. It prevents invalid string values, is more compact, and is more efficient for comparison. However, it requires a `formatName()` helper for human-readable output, which V4 provides (lines 86-97). V2's string-based approach is self-documenting but less type-safe.

**TTY detection architecture:**

V2 uses a two-layer approach with an injectable stat function:
```go
// V2 formatter.go:109-127
func DetectTTY() bool {
    return DetectTTYFrom(func() (FileInfo, error) {
        fi, err := os.Stdout.Stat()
        if err != nil { return nil, err }
        return fi, nil
    })
}

func DetectTTYFrom(statFn func() (FileInfo, error)) bool {
    fi, err := statFn()
    if err != nil { return false }
    return fi.Mode()&os.ModeCharDevice != 0
}
```
V2 also defines a `FileInfo` interface and `errStatFailure` sentinel for testing. The `DetectTTY()` function always checks `os.Stdout`, making it a package-level global dependency. `DetectTTYFrom` is exported specifically for test injection.

V4 takes a simpler approach -- `DetectTTY` accepts an `io.Writer` parameter:
```go
// V4 format.go:101-111
func DetectTTY(w io.Writer) bool {
    f, ok := w.(*os.File)
    if !ok { return false }
    info, err := f.Stat()
    if err != nil { return false }
    return (info.Mode() & os.ModeCharDevice) != 0
}
```
V4's design is genuinely better: it avoids a global `os.Stdout` dependency, makes the function testable by simply passing a `bytes.Buffer` (non-TTY) or `*os.File` (pipe = non-TTY, terminal = TTY), and does not need the extra `DetectTTYFrom` indirection or `FileInfo` interface. The `io.Writer` parameter approach is a natural fit since the App already holds `Stdout io.Writer`.

**ResolveFormat:**

Both implementations are structurally identical -- count flags, error if >1, return the matching format or auto-detect from TTY. The only difference is the error message text:
- V2: `"only one format flag allowed: --toon, --pretty, or --json"`
- V4: `"only one format flag allowed; got multiple of --toon, --pretty, --json"`

**parseGlobalFlags integration:**

V2 collects boolean flags during parsing and calls `ResolveFormat` at two points -- when a non-flag argument is encountered (subcommand found) and after the loop ends (no subcommand):
```go
// V2 app.go:111-146
func (a *App) parseGlobalFlags(args []string) (string, []string, error) {
    var toonFlag, prettyFlag, jsonFlag bool
    for i := 0; i < len(args); i++ {
        // ...flag cases...
        default:
            format, err := ResolveFormat(toonFlag, prettyFlag, jsonFlag, DetectTTY())
            // ...
            return arg, args[i+1:], nil
        }
    }
    format, err := ResolveFormat(toonFlag, prettyFlag, jsonFlag, DetectTTY())
    // ...
}
```
Note V2 calls `DetectTTY()` (the parameterless version that checks `os.Stdout` directly) inside `parseGlobalFlags`. This means TTY detection happens during flag parsing.

V4 calls `DetectTTY(a.Stdout)` earlier in `Run()` and stores the result in `a.IsTTY`, then `parseGlobalFlags` uses `a.IsTTY`:
```go
// V4 cli.go:38
a.IsTTY = DetectTTY(a.Stdout)
// ...
// V4 cli.go:163
format, err := ResolveFormat(toonFlag, prettyFlag, jsonFlag, a.IsTTY)
```
V4's approach avoids calling `DetectTTY` twice (V2 potentially calls it at both the `default` case and the post-loop site). V4 also consolidates the `ResolveFormat` call to a single post-loop site by using `i = len(args)` to break from the loop on the first non-flag argument, then falling through to the shared `ResolveFormat` call.

**FormatConfig wiring:**

V2 adds `FormatCfg FormatConfig` to the App struct and populates it in `Run()`:
```go
// V2 app.go:70-71
a.FormatCfg = a.formatConfig()
```
With a helper method:
```go
// V2 app.go:183-189
func (a *App) formatConfig() FormatConfig {
    return FormatConfig{
        Format:  a.config.OutputFormat,
        Quiet:   a.config.Quiet,
        Verbose: a.config.Verbose,
    }
}
```

V4 inlines the construction directly:
```go
// V4 cli.go (from diff):
a.FormatCfg = FormatConfig{
    Format:  a.OutputFormat,
    Quiet:   a.Quiet,
    Verbose: a.Verbose,
}
```
V4 does not have a separate `formatConfig()` method -- the construction is straightforward enough to inline.

**Formatter resolution:**

V2 resolves the concrete formatter in `Run()` via `newFormatter(a.FormatCfg.Format)` and stores it as `a.formatter`:
```go
// V2 app.go:75
a.formatter = newFormatter(a.FormatCfg.Format)
```

V4 does the same but names it `resolveFormatter` and stores on the exported `a.Formatter`:
```go
// V4 cli.go:51
a.Formatter = resolveFormatter(a.OutputFormat)
```

Both return `*ToonFormatter{}`, `*PrettyFormatter{}`, or `*JSONFormatter{}`. This is technically beyond the scope of task 4-1 (which specifies a stub formatter only), but both versions include it because they already have concrete formatters from later tasks in the branch.

**Formatter interface signatures:**

V2:
```go
FormatTaskList(w io.Writer, tasks []TaskRow) error
FormatTaskDetail(w io.Writer, data *showData) error
FormatTransition(w io.Writer, id string, oldStatus, newStatus task.Status) error
FormatDepChange(w io.Writer, action, taskID, blockedByID string) error
FormatStats(w io.Writer, stats interface{}) error
FormatMessage(w io.Writer, message string) error
```

V4:
```go
FormatTaskList(w io.Writer, rows []listRow, quiet bool) error
FormatTaskDetail(w io.Writer, detail TaskDetail) error
FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error
FormatStats(w io.Writer, stats StatsData) error
FormatMessage(w io.Writer, msg string) error
```

Key differences:
1. V2's `FormatTaskList` takes `[]TaskRow` (a V2-defined struct); V4 takes `[]listRow` (internal type) and a `quiet bool` parameter.
2. V2's `FormatTaskDetail` takes `*showData` (pointer to existing internal type); V4 takes `TaskDetail` (a new, dedicated value type with full field definitions).
3. V2's `FormatTransition` uses `task.Status` (domain type); V4 uses plain `string`.
4. V2's `FormatDepChange` has parameters `(action, taskID, blockedByID)`; V4 has `(taskID, blockedByID, action, quiet)` -- different order plus `quiet` flag.
5. V2's `FormatStats` takes `interface{}`; V4 takes `StatsData` (a concrete struct with fields).

V4's approach is stronger on type safety for `FormatStats` (concrete `StatsData` vs `interface{}`) and `FormatTaskDetail` (fully-specified `TaskDetail` struct). V2's use of `task.Status` in `FormatTransition` preserves domain types, which is arguably better than V4's `string`. V4's inclusion of `quiet bool` in `FormatTaskList` and `FormatDepChange` pushes quiet-awareness into the formatter, while V2 keeps it external.

**Stub formatter:**

V2's `StubFormatter` has real implementations that produce readable output:
```go
// V2 formatter.go:59-104
func (f *StubFormatter) FormatTaskList(w io.Writer, tasks []TaskRow) error {
    if len(tasks) == 0 {
        _, err := fmt.Fprintln(w, "No tasks found.")
        return err
    }
    _, err := fmt.Fprintf(w, "%-12s%-12s%-4s%s\n", "ID", "STATUS", "PRI", "TITLE")
    // ... iterates and formats each row
}
```

V4's `StubFormatter` is truly a stub -- all methods return `nil` with no output:
```go
// V4 format.go:60-82
func (f *StubFormatter) FormatTaskList(w io.Writer, rows []listRow, quiet bool) error {
    return nil
}
```
V4's approach is more appropriate for a "placeholder" since task 4-1 specifies a stub that will be replaced. V2's stub does useful work but will be entirely replaced in later tasks anyway.

**Verbose logging in this task:**

V2 adds a `logVerbose` method to `App` that delegates to `VerboseLogger` (or falls back to direct stderr write):
```go
// V2 app.go:196-204
func (a *App) logVerbose(msg string) {
    if a.verbose != nil {
        a.verbose.Log(msg)
        return
    }
    if a.config.Verbose {
        fmt.Fprintf(a.stderr, "verbose: %s\n", msg)
    }
}
```

V4 does not add a `logVerbose` wrapper; it uses `a.vlog.Log(...)` directly in `Run()`:
```go
// V4 cli.go:52
a.vlog.Log("format resolved to %s", formatName(a.OutputFormat))
```
V4's `VerboseLogger.Log` accepts format strings (`format string, args ...interface{}`), while V2's takes a plain `string`.

### Code Quality

**Go idioms:**
- V4's `Format int` with `iota` is more idiomatic Go for enumerations than V2's `OutputFormat string`.
- V4's `DetectTTY(w io.Writer)` is more idiomatic dependency injection than V2's `DetectTTYFrom(statFn)` callback approach.
- V4's `VerboseLogger.Log(format string, args ...interface{})` supports printf-style formatting, while V2's `Log(msg string)` requires callers to pre-format with `fmt.Sprintf`.

**Naming:**
- V2 uses `FormatTOON` (all caps); V4 uses `FormatToon` (Go convention for initialisms would suggest `FormatTOON`, but `Toon` is not an acronym -- it's a format name, so `FormatToon` is arguably more correct).
- V2 names the file `formatter.go`; V4 names it `format.go`. Both are reasonable.
- V2 uses `TaskRow` (exported); V4 uses `listRow` (unexported). V4 is better here since it's an internal type.
- V2 defines `showData` as a pre-existing type used by `FormatTaskDetail`; V4 defines a new `TaskDetail` struct with fully specified fields -- a cleaner separation.

**Error handling:**
Both handle errors identically in `ResolveFormat`. V2's `logVerbose` has defensive nil-checking for the `VerboseLogger`, which V4 avoids by ensuring `vlog` is always created in `Run()`.

**DRY:**
V2 calls `ResolveFormat` at two code paths in `parseGlobalFlags` (inside the loop default case and after the loop). V4 consolidates to a single call after the loop by using `i = len(args)` to break, which is more DRY.

**Type safety:**
V4 is stronger: `StatsData` struct vs `interface{}` for `FormatStats`, `TaskDetail` with named fields vs `*showData` pointer, `Format int` vs `OutputFormat string`.

### Test Quality

**V2 test functions (in `formatter_test.go`, 280 lines):**

1. `TestDetectTTY` (4 subtests):
   - `"it detects TTY vs non-TTY"` -- calls `DetectTTY()` in pipe environment, expects false
   - `"it defaults to non-TTY on stat failure"` -- uses `DetectTTYFrom` with error-returning stat function
   - `"it returns true when ModeCharDevice is set"` -- uses `fakeFileInfo{mode: os.ModeCharDevice}`
   - `"it returns false when ModeCharDevice is not set"` -- uses `fakeFileInfo{mode: 0}`

2. `TestResolveFormat` (3 subtests with table-driven sub-subtests):
   - `"it defaults to Toon when non-TTY, Pretty when TTY"` -- table: 2 cases (non-TTY->Toon, TTY->Pretty)
   - `"it returns correct format for each flag override"` -- table: 6 cases (each flag x 2 TTY states)
   - `"it errors when multiple format flags set"` -- table: 4 cases (all pairwise conflicts + all three)

3. `TestFormatConfig` (1 subtest with table):
   - `"it propagates quiet and verbose in FormatConfig"` -- table: 4 cases testing struct construction

4. `TestFormatterInterface` (1 subtest):
   - `"it has a stub formatter that satisfies the Formatter interface"` -- compile-time check only

5. `TestAppConflictingFormatFlags` (1 subtest):
   - `"it errors when multiple format flags set via App.Run"` -- integration test via `App.Run`

6. `TestAppFormatConfigWiring` (2 subtests):
   - `"it wires FormatConfig into App from config"` -- runs with `--quiet --verbose --toon`, checks `formatConfig()`
   - `"it stores FormatConfig on App during dispatch"` -- runs with `--toon`, checks `App.FormatCfg`

7. `TestVerboseStderr` (2 subtests):
   - `"it writes verbose output to stderr not stdout"` -- captures stdout/stderr, calls `logVerbose`
   - `"it does not write verbose output when verbose is disabled"` -- verifies no output

Helper type: `fakeFileInfo` (implements `FileInfo` interface for TTY test injection)

Also in `app_test.go` (1 line changed): Updated `detectTTY` -> `DetectTTY` call.

**V4 test functions (in `format_test.go`, 193 lines as committed):**

1. `TestDetectTTY` (3 subtests):
   - `"it detects non-TTY when stdout is not an os.File"` -- passes `bytes.Buffer`
   - `"it detects non-TTY for pipe file descriptor"` -- creates `os.Pipe`, passes write end
   - `"it defaults to non-TTY on stat failure"` -- creates temp file, closes it, passes closed `*os.File`

2. `TestResolveFormat` (5 subtests):
   - `"it defaults to Toon when non-TTY"` -- single assertion
   - `"it defaults to Pretty when TTY"` -- single assertion
   - `"it returns correct format for --toon flag override"` -- single assertion
   - `"it returns correct format for --pretty flag override"` -- single assertion
   - `"it returns correct format for --json flag override"` -- single assertion
   - `"it errors when multiple format flags set"` -- table: 4 cases

3. `TestFormatter` (1 subtest):
   - `"it provides a stub formatter that implements the interface"` -- calls all 6 methods, verifies no panic

4. `TestCLI_ConflictingFormatFlags` (1 subtest):
   - `"it errors when multiple format flags set via CLI"` -- integration via `App.Run`, checks exit code and stderr

5. `TestCLI_FormatConfigWired` (1 subtest, in commit diff):
   - `"it wires FormatConfig into App after flag parsing"` -- runs with `--quiet --verbose --json`, checks `FormatCfg` fields

Also in `cli_test.go` (20 lines changed): Updated TTY detection test to use `Run()` flow, updated format constant names.

**Test coverage differences:**

| Edge case | V2 | V4 |
|-----------|-----|-----|
| TTY positive case (ModeCharDevice) | Tested via `fakeFileInfo` | Not directly tested (cannot simulate TTY in tests) |
| Non-os.File writer | Not applicable (V2 uses os.Stdout directly) | Tested (`bytes.Buffer`) |
| Pipe file descriptor | Not directly tested | Tested (`os.Pipe`) |
| Stat failure | Tested via injected error function | Tested via closed `*os.File` |
| Each flag override x TTY state | 6 combinations tested | 3 flag overrides tested (one TTY state each) |
| FormatConfig propagation (quiet+verbose) | Tested with 4 struct combinations | Tested via App.Run integration |
| Verbose to stderr | 2 dedicated tests | No tests in this task commit (relies on prior VerboseLogger tests) |
| StubFormatter method calls | Compile-time check only | All 6 methods called (verifies no panic) |

V2 has more comprehensive test coverage within its own task commit: it tests the TTY positive case (ModeCharDevice set), has 6 flag override combinations (each flag x both TTY states), and includes dedicated verbose-to-stderr tests. V4 cannot test the positive TTY case (a limitation of its design, though in practice this is fine since you can't create a real TTY in tests), tests fewer flag override combinations, and relies on prior tasks for verbose testing.

V2 uses more table-driven tests (nested tables in `TestResolveFormat`). V4 uses individual subtests for flag overrides and only uses tables for conflicting flags.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 4 | 4 |
| Lines added | 488 | 363 |
| Lines removed | 24 | 49 |
| New file: impl LOC | 163 (`formatter.go`) | 135 (`format.go`) |
| New file: test LOC | 280 (`formatter_test.go`) | 193 (`format_test.go`) |
| Modified file changes | 67 lines in `app.go`, 2 in `app_test.go` | 64 lines in `cli.go`, 20 in `cli_test.go` |
| Test functions (in task commit) | 7 top-level (13 subtests total) | 5 top-level (11 subtests total) |

## Verdict

**V4 is the better implementation for this task**, though both meet all acceptance criteria.

V4's advantages are structural and idiomatic:

1. **Type design**: `Format int` with `iota` is more idiomatic Go than `OutputFormat string`. The `TaskDetail` and `StatsData` structs with concrete fields provide stronger type safety than V2's `*showData` and `interface{}`.

2. **TTY detection**: `DetectTTY(w io.Writer)` is a cleaner API than V2's `DetectTTY()` + `DetectTTYFrom(statFn)` two-function approach. V4 avoids the global `os.Stdout` dependency, needs no `FileInfo` interface or `errStatFailure` sentinel, and is naturally testable.

3. **DRY flag resolution**: V4 consolidates the `ResolveFormat` call to a single post-loop site in `parseGlobalFlags` using `i = len(args)` to break. V2 has two `ResolveFormat` call sites (inside the loop and after it).

4. **Formatter interface**: V4's signatures push `quiet bool` into formatters that need it (`FormatTaskList`, `FormatDepChange`) and use concrete types (`StatsData`) instead of `interface{}`. The parameter ordering in V4 (`taskID, blockedByID, action`) is also more natural.

5. **Smaller delta**: V4 achieves the same functionality in 363 added lines vs V2's 488, indicating less ceremony.

V2's advantages are in test coverage: it tests the positive TTY case via `fakeFileInfo`, has 6 flag override combinations vs V4's 3, and includes dedicated verbose-to-stderr tests within this task commit. V2's `StubFormatter` also produces actual output, though this is of debatable value since it will be replaced.

The type safety and idiomatic design improvements in V4 outweigh V2's testing edge, particularly because V2's extra TTY-positive test is enabled by a design choice (injectable stat function) that introduces unnecessary complexity.
