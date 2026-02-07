# Task tick-core-4-1: Formatter abstraction & TTY-based format selection

## Task Summary

Define a `Formatter` interface, implement TTY detection, resolve format from flags vs auto-detection with conflict handling, and wire the chosen formatter into CLI dispatch. Foundation for tasks 4-2 through 4-4.

### Required Components
- `Formatter` interface with methods: `FormatTaskList`, `FormatTaskDetail`, `FormatTransition`, `FormatDepChange`, `FormatStats`, `FormatMessage`
- `Format` enum: `FormatToon`, `FormatPretty`, `FormatJSON`
- `DetectTTY()`: `os.Stdout.Stat()` -> check `ModeCharDevice`. Stat failure -> default non-TTY
- `ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY)`: >1 flag -> error; 1 flag -> that format; 0 flags + TTY -> Pretty; 0 flags + no TTY -> Toon
- `FormatConfig` struct: Format, Quiet, Verbose -- passed to all handlers
- Stub formatter as placeholder (concrete formatters in 4-2 through 4-4)
- `--verbose` to stderr only, never contaminates stdout

### Acceptance Criteria
1. Formatter interface covers all command output types
2. Format enum with 3 constants
3. TTY detection works correctly
4. ResolveFormat handles all flag/TTY combos
5. Conflicting flags -> error
6. FormatConfig wired into CLI dispatch
7. Verbose to stderr only
8. Stat failure handled gracefully

### Edge Cases
- Stat failure -> non-TTY default, no panic
- Conflicting flags -> error before dispatch
- Verbose orthogonal to format -- stderr only
- Quiet orthogonal -- doesn't change format selection

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| 1. Formatter interface covers all command output types | PASS - 6 methods: FormatTaskList, FormatTaskDetail, FormatTransition, FormatDepChange, FormatStats, FormatMessage | PASS - Same 6 methods, uses existing types (`*showData`, `task.Status`) | PASS - Same 6 methods, returns `string` instead of writing to `io.Writer` |
| 2. Format enum with 3 constants | PASS - `Format int` iota: FormatToon=0, FormatPretty=1, FormatJSON=2 | PASS - `OutputFormat string`: FormatTOON="toon", FormatPretty="pretty", FormatJSON="json" (reuses existing type) | PASS - `Format string`: FormatToon="toon", FormatPretty="pretty", FormatJSON="json" |
| 3. TTY detection works correctly | PASS - `DetectTTY(f *os.File) bool` checks ModeCharDevice | PASS - `DetectTTY() bool` with injectable `DetectTTYFrom` for testing | PASS - `DetectTTY(w io.Writer) bool` checks writer is `*os.File` then ModeCharDevice |
| 4. ResolveFormat handles all flag/TTY combos | PASS - All combos tested | PASS - All combos tested | PASS - All combos tested |
| 5. Conflicting flags -> error | PASS - Returns error when count > 1 | PASS - Returns error when count > 1, wired into `parseGlobalFlags` | PASS - Returns error when count > 1, wired into `Run()` via `NewFormatConfig` |
| 6. FormatConfig wired into CLI dispatch | FAIL - No CLI integration; standalone file only | PASS - `App.FormatCfg` set in `Run()`, `formatConfig()` helper, integration tests | PASS - `app.formatConfig` set in `Run()` via `NewFormatConfig`, integration tests |
| 7. Verbose to stderr only | FAIL - No verbose helper implemented | PASS - `logVerbose()` writes to stderr, tested | PASS - `WriteVerbose()` writes to stderr with `[verbose]` prefix, tested |
| 8. Stat failure handled gracefully | PASS - nil file returns false, stat error returns false | PASS - `DetectTTYFrom` with error-returning stat function tested | PASS - Closed file test, nil writer test |

## Implementation Comparison

### Approach

**V1** takes a pure greenfield approach, creating a single new file `internal/cli/format.go` with no integration into the existing CLI. It defines its own Format type as an `int` iota enum and creates rich data structs (TaskListItem, TaskDetail, TransitionData, DepChangeData, StatsData) as parameters for the Formatter interface. No existing code is modified.

```go
// V1: Self-contained Format type (int iota)
type Format int
const (
    FormatToon Format = iota
    FormatPretty
    FormatJSON
)
```

```go
// V1: DetectTTY takes explicit *os.File parameter
func DetectTTY(f *os.File) bool {
    if f == nil {
        return false
    }
    fi, err := f.Stat()
    if err != nil {
        return false
    }
    return fi.Mode()&os.ModeCharDevice != 0
}
```

**V2** integrates deeply into the existing `app.go`, modifying `parseGlobalFlags` to track individual format booleans, wiring `ResolveFormat` into the parse flow, and adding `FormatCfg` as a field on `App`. It reuses the existing `OutputFormat string` type from `app.go` and references existing types (`*showData`, `task.Status`) in the Formatter interface. New code goes in `formatter.go`; modifications go in `app.go`.

```go
// V2: Reuses existing OutputFormat string type from app.go
type FormatConfig struct {
    Format  OutputFormat
    Quiet   bool
    Verbose bool
}
```

```go
// V2: DetectTTY() uses injectable stat function for testability
func DetectTTY() bool {
    return DetectTTYFrom(func() (FileInfo, error) {
        fi, err := os.Stdout.Stat()
        // ...
    })
}

func DetectTTYFrom(statFn func() (FileInfo, error)) bool {
    fi, err := statFn()
    if err != nil { return false }
    return fi.Mode()&os.ModeCharDevice != 0
}
```

```go
// V2: Format resolution integrated into parseGlobalFlags
func (a *App) parseGlobalFlags(args []string) (string, []string, error) {
    var toonFlag, prettyFlag, jsonFlag bool
    // ...parse flags...
    format, err := ResolveFormat(toonFlag, prettyFlag, jsonFlag, DetectTTY())
    if err != nil { return "", nil, err }
    a.config.OutputFormat = format
    return arg, args[i+1:], nil
}
```

**V3** also modifies the existing CLI (`cli.go`), adding `formatConfig FormatConfig` to `App` and individual flag booleans to `GlobalFlags`. It introduces a `NewFormatConfig` constructor that combines `ResolveFormat` + `DetectTTY` + quiet/verbose into one call. The Formatter interface returns `string` instead of writing to an `io.Writer`, and all data types are defined fresh. Adds context documentation in `tick-core-context.md`.

```go
// V3: Formatter returns strings instead of writing to io.Writer
type Formatter interface {
    FormatTaskList(data *TaskListData) string
    FormatTaskDetail(data *TaskDetailData) string
    FormatTransition(taskID, oldStatus, newStatus string) string
    FormatDepChange(action, taskID, blockedByID string) string
    FormatStats(data *StatsData) string
    FormatMessage(msg string) string
}
```

```go
// V3: DetectTTY takes io.Writer, checks if it's *os.File
func DetectTTY(w io.Writer) bool {
    if w == nil { return false }
    f, ok := w.(*os.File)
    if !ok { return false }
    info, err := f.Stat()
    if err != nil { return false }
    return info.Mode()&os.ModeCharDevice != 0
}
```

```go
// V3: NewFormatConfig bundles format resolution + TTY detection
func NewFormatConfig(toonFlag, prettyFlag, jsonFlag, quiet, verbose bool, stdout io.Writer) (FormatConfig, error) {
    isTTY := DetectTTY(stdout)
    format, err := ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY)
    if err != nil { return FormatConfig{}, err }
    return FormatConfig{Format: format, Quiet: quiet, Verbose: verbose}, nil
}
```

### Key Structural Differences

| Aspect | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Format type | `int` iota | `OutputFormat string` (existing) | `Format string` (new) |
| DetectTTY signature | `DetectTTY(f *os.File) bool` | `DetectTTY() bool` (hardcoded os.Stdout) | `DetectTTY(w io.Writer) bool` |
| Formatter io pattern | Writes to `io.Writer` parameter | Writes to `io.Writer` parameter | Returns `string` |
| Formatter data types | Fresh structs (TaskListItem, etc.) | References existing types (`*showData`, `task.Status`) | Fresh structs (TaskListData, etc.) |
| CLI integration | None | Modified `app.go` parseGlobalFlags + Run | Modified `cli.go` Run + ParseGlobalFlags |
| Verbose helper | None | `logVerbose(msg string)` | `WriteVerbose(format string, args ...interface{})` |
| FormatConfig constructor | None | `formatConfig()` method on App | `NewFormatConfig()` standalone function |

### Code Quality

**V1: Naming & Idioms**

V1 uses clean Go naming. `Format int` with iota is idiomatic but loses self-documenting string values in debugging/logging. The `DetectTTY` function takes an explicit `*os.File` parameter, which is testable but less flexible than accepting `io.Writer`. The data structs are comprehensive but entirely self-contained -- they duplicate concepts that already exist elsewhere in the codebase (e.g., `TaskDetail` vs the existing `showData` in V2's context).

```go
// V1: Rich but standalone data structs (format.go lines 23-77)
type TaskDetail struct {
    ID          string
    Title       string
    Status      string
    Priority    int
    Description string
    Parent      *RelatedTask
    Created     string
    Updated     string
    Closed      string
    BlockedBy   []RelatedTask
    Children    []RelatedTask
}
```

The StubFormatter produces actual human-readable output, which is useful for development:
```go
// V1: StubFormatter has meaningful output
func (s *StubFormatter) FormatTaskList(w io.Writer, tasks []TaskListItem) error {
    for _, t := range tasks {
        if _, err := fmt.Fprintf(w, "%s %s %d %s\n", t.ID, t.Status, t.Priority, t.Title); err != nil {
            return err
        }
    }
    return nil
}
```

**V2: Naming & Integration**

V2 reuses the existing `OutputFormat string` type, maintaining consistency. The naming `FormatTOON` (all caps) follows the existing codebase convention. The `DetectTTYFrom` pattern with injectable stat function is clever for testing but adds a custom `FileInfo` interface and `errStatFailure` sentinel that feel over-engineered:

```go
// V2: Custom FileInfo interface for stat injection (formatter.go lines 18-20)
type FileInfo interface {
    Mode() os.FileMode
}
var errStatFailure = errors.New("stat failure")
```

V2 references existing types in the Formatter interface (`*showData`, `task.Status`), which creates tighter coupling but ensures consistency. The `FormatCfg` field is exported on `App` (`App.FormatCfg`), which exposes internal state:

```go
// V2: Exported field on App (app.go)
type App struct {
    config    Config
    FormatCfg FormatConfig  // exported -- leaks internal state
    workDir   string
    // ...
}
```

**V3: Naming & Design**

V3's `Format string` type with `FormatToon` (mixed case, not `FormatTOON`) is clean and Go-idiomatic. The `NewFormatConfig` constructor is a well-designed convenience function. The `WriteVerbose` method uses `fmt.Sprintf` formatting with `[verbose]` prefix, which is more flexible than V2's `logVerbose`:

```go
// V3: WriteVerbose with format string support (cli.go)
func (a *App) WriteVerbose(format string, args ...interface{}) {
    if !a.formatConfig.Verbose { return }
    msg := fmt.Sprintf(format, args...)
    fmt.Fprintf(a.Stderr, "[verbose] %s\n", msg)
}
```

V3's Formatter interface returns `string` instead of writing to `io.Writer`. This is simpler but less efficient for large outputs and doesn't follow the standard Go io pattern:

```go
// V3: Returns string -- simpler but doesn't stream
type Formatter interface {
    FormatTaskList(data *TaskListData) string
    FormatTaskDetail(data *TaskDetailData) string
    // ...
}
```

V3's `StubFormatter` returns empty strings, which is the correct "stub" behavior but provides no useful feedback during development:

```go
// V3: Stub returns empty strings
func (f *StubFormatter) FormatTaskList(data *TaskListData) string { return "" }
```

**Error Messages**

| Version | Conflicting flags error message |
|---------|-------------------------------|
| V1 | `"only one of --toon, --pretty, --json may be specified"` |
| V2 | `"only one format flag allowed: --toon, --pretty, or --json"` |
| V3 | `"cannot specify multiple format flags (--toon, --pretty, --json)"` |

### Test Quality

#### V1 Test Functions (file: `internal/cli/format_test.go`, 151 LOC)

Top-level functions:
1. `TestDetectTTY`
   - `"it detects TTY vs non-TTY"` -- tests os.Stdout in pipe environment
   - `"it defaults to non-TTY on stat failure"` -- tests nil file
2. `TestResolveFormat` (table-driven)
   - `"it defaults to Toon when non-TTY"`
   - `"it defaults to Pretty when TTY"`
   - `"it returns Toon for --toon flag override"`
   - `"it returns Pretty for --pretty flag override"`
   - `"it returns JSON for --json flag override"`
   - `"it errors when multiple format flags set (toon+pretty)"`
   - `"it errors when multiple format flags set (toon+json)"`
   - `"it errors when multiple format flags set (pretty+json)"`
   - `"it errors when all three format flags set"`
3. `TestFormatConfig`
   - `"it propagates quiet and verbose in FormatConfig"`
4. `TestFormatEnum`
   - `"it has three format constants"`
5. `TestFormatterInterface`
   - `"it defines Formatter interface covering all command output types"` -- compile-time check

**Total: 5 top-level test functions, 6 subtests (11 total including nested table entries)**

#### V2 Test Functions (file: `internal/cli/formatter_test.go`, 280 LOC)

Top-level functions:
1. `TestDetectTTY`
   - `"it detects TTY vs non-TTY"` -- pipe environment
   - `"it defaults to non-TTY on stat failure"` -- injectable stat function
   - `"it returns true when ModeCharDevice is set"` -- fakeFileInfo
   - `"it returns false when ModeCharDevice is not set"` -- fakeFileInfo
2. `TestResolveFormat`
   - `"it defaults to Toon when non-TTY, Pretty when TTY"` (2 sub-cases)
   - `"it returns correct format for each flag override"` (6 sub-cases: toon/TTY, toon/nonTTY, pretty/nonTTY, pretty/TTY, json/TTY, json/nonTTY)
   - `"it errors when multiple format flags set"` (4 sub-cases: toon+pretty, toon+json, pretty+json, all three)
3. `TestFormatConfig`
   - `"it propagates quiet and verbose in FormatConfig"` (4 sub-cases: all false, quiet only, verbose only, both)
4. `TestFormatterInterface`
   - `"it has a stub formatter that satisfies the Formatter interface"`
5. `TestAppConflictingFormatFlags`
   - `"it errors when multiple format flags set via App.Run"`
6. `TestAppFormatConfigWiring`
   - `"it wires FormatConfig into App from config"` -- tests quiet, verbose, toon via Run
   - `"it stores FormatConfig on App during dispatch"` -- verifies FormatCfg after Run
7. `TestVerboseStderr`
   - `"it writes verbose output to stderr not stdout"`
   - `"it does not write verbose output when verbose is disabled"`

Helper: `fakeFileInfo` struct implementing `FileInfo` interface.

**Total: 7 top-level test functions, 18 subtests (25 total including nested)**

#### V3 Test Functions (file: `internal/cli/format_test.go`, 547 LOC)

Top-level functions:
1. `TestFormat`
   - `"it has three format constants: Toon, Pretty, JSON"` -- checks string values
2. `TestDetectTTY`
   - `"it detects non-TTY for bytes.Buffer"`
   - `"it detects non-TTY for nil writer"`
   - `"it defaults to non-TTY on stat failure"` -- closed file test
3. `TestResolveFormat`
   - `"it defaults to Toon when non-TTY and no flags"`
   - `"it defaults to Pretty when TTY and no flags"`
   - `"it returns Toon when --toon flag is set"` -- with TTY override
   - `"it returns Pretty when --pretty flag is set"` -- with non-TTY override
   - `"it returns JSON when --json flag is set"`
   - `"it errors when --toon and --pretty both set"` -- checks exact error message
   - `"it errors when --toon and --json both set"`
   - `"it errors when --pretty and --json both set"`
   - `"it errors when all three format flags set"`
4. `TestFormatConfig`
   - `"it propagates format, quiet, and verbose"`
   - `"it has zero values for fields when not set"`
5. `TestNewFormatConfig`
   - `"it creates config from flags with TTY detection"`
   - `"it propagates quiet and verbose flags"`
   - `"it returns error when multiple format flags set"`
6. `TestFormatter`
   - `"it defines interface with all required methods"`
7. `TestStubFormatter`
   - `"it implements FormatTaskList"`
   - `"it implements FormatTaskDetail"`
   - `"it implements FormatTransition"`
   - `"it implements FormatDepChange"`
   - `"it implements FormatStats"`
   - `"it implements FormatMessage"`
8. `TestCLIFormatConflictHandling`
   - `"it errors before dispatch when multiple format flags set"` -- verifies stderr, exit code, empty stdout
   - `"it errors with --toon and --json"` -- via App.Run
   - `"it errors with --pretty and --json"` -- via App.Run
   - `"it allows single format flag to proceed"` -- verifies no error
9. `TestFormatConfigWiredIntoApp`
   - `"it stores format config in App after flag parsing"` -- checks JSON format
   - `"it sets quiet in format config"` -- via Run
   - `"it sets verbose in format config"` -- via Run
10. `TestVerboseOutput`
    - `"it writes verbose output to stderr, not stdout"` -- via App.Run --verbose
    - `"verbose is orthogonal to format selection"` -- --verbose + --json
    - `"quiet is orthogonal to format selection"` -- --quiet + --toon
11. `TestWriteVerbose`
    - `"it writes to stderr when verbose is enabled"` -- checks exact output
    - `"it does nothing when verbose is disabled"` -- checks both stdout and stderr empty
    - `"it formats messages with arguments"` -- tests variadic formatting

**Total: 11 top-level test functions, 38 subtests (49 total including nested)**

#### Test Coverage Comparison

| Test Area | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| TTY pipe detection | Yes | Yes | Yes |
| TTY nil/failure | nil file | Injectable stat error | nil writer + closed file |
| TTY positive (ModeCharDevice) | No | Yes (fakeFileInfo) | No |
| TTY non-char-device | No | Yes (fakeFileInfo mode=0) | No |
| TTY bytes.Buffer | No | No | Yes |
| ResolveFormat defaults | Yes | Yes | Yes |
| ResolveFormat each flag | Yes | Yes (x2 with TTY/nonTTY) | Yes |
| Conflicting flags (all combos) | Yes (4 combos) | Yes (4 combos) | Yes (4 combos) |
| Exact error message check | No | Yes | Yes |
| FormatConfig propagation | 1 case | 4 cases | 2 cases + zero value |
| Format enum values | Uniqueness check | No | String value check |
| Formatter interface compile check | Yes | Yes | Yes |
| StubFormatter each method | No | No | Yes (all 6 methods) |
| NewFormatConfig | N/A | N/A | Yes (3 tests) |
| CLI integration (conflict flags via Run) | No | Yes (1 test) | Yes (3 tests) |
| CLI FormatConfig wiring | No | Yes (2 tests) | Yes (3 tests) |
| Verbose to stderr | No | Yes (2 tests) | Yes (3 tests) |
| Verbose with format args | No | No | Yes |
| Quiet orthogonal to format | No | No | Yes |
| Verbose orthogonal to format | No | No | Yes |

**Tests unique to V1:** Format enum uniqueness check (distinct int values).
**Tests unique to V2:** Positive TTY detection via fakeFileInfo (ModeCharDevice set/unset).
**Tests unique to V3:** StubFormatter method coverage (all 6), NewFormatConfig, quiet/verbose orthogonality, WriteVerbose format args, zero-value FormatConfig.
**Tests in all 3:** TTY pipe detection, ResolveFormat defaults, flag overrides, conflicting flags, FormatConfig propagation, Formatter interface compile check.

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed | 2 | 6 | 7 |
| Lines added | 341 | 491 | 817 |
| Impl LOC (new format file) | 190 | 163 | 213 |
| Impl LOC (CLI modifications) | 0 | ~44 net changes in app.go | ~35 net changes in cli.go |
| Test LOC | 151 | 280 | 547 |
| Top-level test functions | 5 | 7 | 11 |
| Total test cases (incl. subtests) | 11 | 25 | 49 |

## Verdict

**V2 is the best implementation.**

**Against V1:** V1 fails two acceptance criteria -- no CLI integration (criterion 6) and no verbose helper (criterion 7). It is a standalone library with no wiring into the actual application. While its code is clean and the data structs are thorough, the task explicitly requires "FormatConfig wired into CLI dispatch" and "verbose to stderr only." V1 delivers neither.

**Against V3:** V3 is the most thorough in testing (547 LOC, 49 test cases) and covers edge cases no other version touches (StubFormatter methods, quiet/verbose orthogonality, WriteVerbose format args). However, V3's Formatter interface design is questionable: returning `string` instead of writing to `io.Writer` breaks the standard Go streaming pattern and will cause issues for large task lists or JSON output. The `WriteVerbose` with `[verbose]` prefix is opinionated -- it bakes in a presentation detail that may not be desirable for all output formats.

**Why V2 wins:**
1. **Full acceptance criteria compliance** -- passes all 8 criteria
2. **Best integration design** -- reuses existing `OutputFormat` type rather than introducing a parallel type system, references existing `showData` and `task.Status` types in the Formatter interface, showing awareness of the broader codebase
3. **Testable TTY detection** -- the `DetectTTYFrom` injection pattern is the only version that can positively test TTY=true via `fakeFileInfo{mode: os.ModeCharDevice}`
4. **Standard io patterns** -- Formatter writes to `io.Writer`, following Go conventions
5. **Clean CLI integration** -- format resolution happens in `parseGlobalFlags` so conflicts are caught before dispatch, `FormatCfg` is populated in `Run()` before subcommand dispatch

V3 would rank second due to superior test thoroughness and the `NewFormatConfig` convenience constructor, but its `string`-returning Formatter design is a structural disadvantage for future tasks. V1 ranks last due to missing two acceptance criteria entirely.
