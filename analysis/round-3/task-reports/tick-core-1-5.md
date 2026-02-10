# Task Report: tick-core-1-5 -- CLI framework & tick init

## 1. Task Summary

**Goal:** Create the CLI entry point (`main.go`) with subcommand dispatch using Go's standard library (no third-party CLI framework), global flag parsing, TTY detection, error handling conventions, and implement `tick init` -- the command that creates `.tick/` with an empty `tasks.jsonl`. Also implement `.tick/` directory discovery by walking up from cwd.

### Acceptance Criteria (from plan)

1. `tick init` creates `.tick/` directory with empty `tasks.jsonl`
2. `tick init` does not create `cache.db`
3. `tick init` prints confirmation with absolute path
4. `tick init` with `--quiet` produces no output on success
5. `tick init` when `.tick/` exists returns error to stderr with exit code 1
6. All errors written to stderr with `Error: ` prefix
7. Exit code 0 for success, 1 for errors
8. Global flags parsed: `--quiet`, `--verbose`, `--toon`, `--pretty`, `--json`
9. TTY detection on stdout selects default output format
10. `.tick/` directory discovery walks up from cwd
11. Unknown subcommands return error with exit code 1

### Specified Edge Cases

- Already initialized: error even if corrupted (missing `tasks.jsonl`)
- Unwritable parent directory: surface as `Error: Could not create .tick/ directory: <os error>`
- Directory discovery: walk up to filesystem root, stop at first `.tick/` match
- Unknown subcommand: `Error: Unknown command '<name>'. Run 'tick help' for usage.` (exit 1)
- No subcommand: print basic usage with exit code 0

### Specified Tests (12 total)

1. "it creates .tick/ directory in current working directory"
2. "it creates empty tasks.jsonl inside .tick/"
3. "it does not create cache.db at init time"
4. "it prints confirmation with absolute path on success"
5. "it prints nothing with --quiet flag on success"
6. "it errors when .tick/ already exists"
7. "it returns exit code 1 when .tick/ already exists"
8. "it writes error messages to stderr, not stdout"
9. "it discovers .tick/ directory by walking up from cwd"
10. "it errors when no .tick/ directory found (not a tick project)"
11. "it routes unknown subcommands to error"
12. "it detects TTY vs non-TTY on stdout"

---

## 2. Acceptance Criteria Compliance

| # | Criterion | V4 | V5 | Notes |
|---|-----------|----|----|-------|
| 1 | `tick init` creates `.tick/` + empty `tasks.jsonl` | PASS | PASS | Both create directory and 0-byte file. V4 uses `os.MkdirAll`, V5 uses `os.Mkdir`. |
| 2 | Does not create `cache.db` | PASS | PASS | Both tested explicitly. |
| 3 | Prints confirmation with absolute path | PASS | PASS | Both output `"Initialized tick in <abs-path>/.tick/"`. V4 resolves abs path early in `runInit`; V5 resolves it after creation for the message only. |
| 4 | `--quiet` produces no output | PASS | PASS | Both check `Quiet` flag and skip output. Both test `--quiet` and `-q` shorthand. |
| 5 | Error when `.tick/` exists (stderr, exit 1) | PASS | PASS | Both return `"Tick already initialized in this directory"`. |
| 6 | All errors to stderr with `Error: ` prefix | PASS | PASS | V4: `writeError()` method. V5: inline `fmt.Fprintf(stderr, "Error: %s\n", err)` in `Run`. |
| 7 | Exit code 0/1 | PASS | PASS | Both return int from Run, `main` calls `os.Exit`. |
| 8 | Global flags parsed | PASS | PASS | Both parse all 5 flags + short forms `-q` and `-v`. V5 additionally detects unknown flags (`strings.HasPrefix(arg, "-")` returns error); V4 silently treats unknown flags as subcommand. |
| 9 | TTY detection | PASS | PASS | V4: `DetectTTY(io.Writer)` checks for `*os.File` then `ModeCharDevice`. V5: `DetectTTY(*os.File)` checks `ModeCharDevice` directly. Both use `ResolveFormat` for TTY-to-format mapping. |
| 10 | `.tick/` directory discovery | PASS | PASS | Virtually identical `DiscoverTickDir` in both. |
| 11 | Unknown subcommands return error (exit 1) | PASS | PASS | Both produce `"Error: Unknown command '<name>'. Run 'tick help' for usage."`. V4 uses switch/default; V5 uses map lookup. |

---

## 3. Implementation Comparison

### 3.1 Approach

#### Architecture

**V4** uses a **struct-based approach** with an `App` struct:

```go
type App struct {
    Stdout       io.Writer
    Stderr       io.Writer
    Dir          string
    Quiet        bool
    Verbose      bool
    OutputFormat Format
    IsTTY        bool
    Formatter    Formatter
    vlog         *VerboseLogger
}
```

Methods are attached to `*App`: `app.Run(args)`, `app.runInit(subArgs)`, `app.writeError(err)`, `app.printUsage()`, `app.openStore(tickDir)`. The `main.go` constructs an `App` and calls `os.Exit(app.Run(os.Args))`.

**V5** uses a **function-based approach** with a `Context` struct and a top-level `Run` function:

```go
func Run(args []string, workDir string, stdout, stderr io.Writer, isTTY bool) int
```

The `Context` struct carries parsed state:

```go
type Context struct {
    WorkDir string
    Stdout  io.Writer
    Stderr  io.Writer
    Quiet   bool
    Verbose bool
    Format  OutputFormat
    Fmt     Formatter
    Args    []string
}
```

Command handlers are functions receiving `*Context`: `runInit(ctx *Context) error`. Commands are registered in a `map[string]func(*Context) error` called `commands`.

**Assessment:** Both are valid Go idioms. V4's struct approach couples state and behavior together (OOP-style). V5's function approach with a context parameter is more idiomatic Go -- it separates the dispatch mechanism from the command handlers and makes the dependency injection explicit. The `commands` map in V5 is more extensible than V4's switch statement, but both work.

#### main.go

**V4** (23 lines): Creates `App` struct, resolves `os.Getwd()` with fallback to `"."`, calls `app.Run(os.Args)`.

```go
app := &cli.App{
    Stdout: os.Stdout,
    Stderr: os.Stderr,
    Dir:    ".",
}
if wd, err := os.Getwd(); err == nil {
    app.Dir = wd
}
os.Exit(app.Run(os.Args))
```

**V5** (28 lines): Calls `DetectTTY` separately, gets cwd (exits on failure), calls `cli.Run(...)`.

```go
isTTY := cli.DetectTTY(os.Stdout)
cwd, err := os.Getwd()
if err != nil {
    os.Stderr.WriteString("Error: " + err.Error() + "\n")
    os.Exit(1)
}
code := cli.Run(os.Args, cwd, os.Stdout, os.Stderr, isTTY)
os.Exit(code)
```

**Assessment:** V4 silently falls back to `"."` if `Getwd` fails, which could cause confusing errors later. V5 exits immediately with an error. V5 is more correct here. V5 also detects TTY in `main.go` before calling `Run`, while V4 detects inside `Run` -- V5's approach is cleaner for testing because `isTTY` is just a bool parameter.

#### Subcommand Dispatch

**V4:** Switch statement in `Run` (lines 64-134) with a separate case for each command, each following the pattern:
```go
case "init":
    if err := a.runInit(subArgs); err != nil {
        a.writeError(err)
        return 1
    }
    return 0
```

**V5:** Map-based dispatch (lines 97-112):
```go
var commands = map[string]func(*Context) error{
    "init":    runInit,
    "create":  runCreate,
    ...
}
```
With a single dispatch point:
```go
handler, ok := commands[subcmd]
if !ok {
    fmt.Fprintf(stderr, "Error: Unknown command '%s'. Run 'tick help' for usage.\n", subcmd)
    return 1
}
if err := handler(ctx); err != nil { ... }
```

**Assessment:** V5's map-based dispatch eliminates the repeated boilerplate. The switch in V4 is 70 lines of repetitive code. V5 is clearly more maintainable and idiomatic.

#### Flag Parsing

**V4:** `parseGlobalFlags` processes args in a `for` loop. When it hits a non-flag argument, it appends `args[i:]` (all remaining args) to `remaining` and exits. This means global flags can only appear BEFORE the subcommand.

**V5:** `parseArgs` iterates all args. Once the subcommand is found (`foundCmd = true`), subsequent args are collected as `cmdArgs`. It also rejects unknown flags with `strings.HasPrefix(arg, "-")`.

```go
case strings.HasPrefix(arg, "-"):
    return nil, "", fmt.Errorf("unknown flag '%s'", arg)
```

**Assessment:** V5's unknown flag detection is a robustness improvement not specified by the task but defensively sound. V4 would silently treat `--typo` as the subcommand name, producing `"Unknown command '--typo'"`. V5 gives a clearer error: `"unknown flag '--typo'"`.

#### init Implementation

**V4** (`init.go`, 43 lines):
- Resolves absolute path from `a.Dir` using `filepath.Abs`
- Uses `os.MkdirAll` to create `.tick/`
- Creates empty `tasks.jsonl` with `os.WriteFile`
- On `WriteFile` failure: **cleans up** by removing `.tick/` with `os.RemoveAll`
- Outputs via `a.Formatter.FormatMessage(a.Stdout, msg)` (returns error)

**V5** (`init.go`, 40 lines):
- Uses `ctx.WorkDir` directly (already absolute from `main.go`)
- Uses `os.Mkdir` (not `MkdirAll`) to create `.tick/`
- Creates empty `tasks.jsonl` with `os.WriteFile`
- No cleanup on `WriteFile` failure
- Resolves absolute path only for the confirmation message (via `filepath.Abs`)
- Outputs via `ctx.Fmt.FormatMessage(ctx.Stdout, ...)` (no return value)

**Assessment:** V4's use of `os.MkdirAll` vs V5's `os.Mkdir` -- `Mkdir` is more appropriate here since we're only creating one directory level, and `MkdirAll` would silently succeed if intermediate directories are missing (not relevant for `.tick/` but conceptually wrong). V4's cleanup-on-failure is a nice defensive measure that V5 lacks, though the practical scenario is very unlikely (creating a file in a directory we just created). V5 does a second `filepath.Abs` call for the message which is slightly redundant since `WorkDir` should already be absolute, but more robust.

#### DiscoverTickDir

Both implementations are virtually identical:

```go
// V4
func DiscoverTickDir(startDir string) (string, error) {
    dir, err := filepath.Abs(startDir)
    // ...
    for {
        candidate := filepath.Join(dir, ".tick")
        info, err := os.Stat(candidate)
        if err == nil && info.IsDir() { return candidate, nil }
        parent := filepath.Dir(dir)
        if parent == dir { return "", fmt.Errorf("Not a tick project (no .tick directory found)") }
        dir = parent
    }
}
```

V5 is the same except it uses `break` instead of returning directly from the root check, then returns the error after the loop. Functionally identical.

#### Error Message for Directory Creation Failure

The spec says: `"Error: Could not create .tick/ directory: <os error>"`. Neither version uses this exact message. V4 uses `"failed to create .tick/ directory: %w"`. V5 uses `"creating .tick directory: %w"`. Both deviate from the spec's exact phrasing, but V4 is closer.

#### DetectTTY

**V4:** `DetectTTY(w io.Writer) bool` -- accepts `io.Writer`, type-asserts to `*os.File`, then checks `ModeCharDevice`.

**V5:** `DetectTTY(f *os.File) bool` -- accepts `*os.File` directly. No type assertion needed.

**Assessment:** V5's approach is more type-safe. The function is only meaningful for `*os.File`, so accepting that directly is correct. V4's approach means the caller can pass any `io.Writer` and get `false`, which could hide bugs.

### 3.2 Code Quality

| Aspect | V4 | V5 |
|--------|----|----|
| **Line count (task files only)** | 796 lines across 7 files | 584 lines across 5 files |
| **File organization** | Separate files: `cli.go`, `discover.go`, `discover_test.go`, `init.go`, `init_test.go` | Fewer files: `cli.go`, `discover.go`, `init.go`. Tests consolidated in `cli_test.go` |
| **Boilerplate** | Significant: 70-line switch statement with repeated error/return pattern | Minimal: map-based dispatch, single handler path |
| **Error wrapping** | Uses `%w` verb consistently | Uses `%w` verb consistently |
| **Doc comments** | All exported functions documented | All exported functions documented |
| **Code duplication** | Each switch case repeats the err-check-writeError-return pattern | Single dispatch path eliminates duplication |
| **Idiomatic Go** | Acceptable but verbose | More idiomatic (function-based, context passing, map dispatch) |

### 3.3 Test Quality

#### V4 Test Functions (across 3 test files)

**cli_test.go** (242 lines, 10 subtests):
- `TestCLI_UnknownSubcommand/it routes unknown subcommands to error`
- `TestCLI_NoSubcommand/it prints basic usage with exit code 0 when no subcommand given`
- `TestCLI_GlobalFlagsParsed/it parses --quiet flag before subcommand`
- `TestCLI_GlobalFlagsParsed/it parses --verbose flag`
- `TestCLI_GlobalFlagsParsed/it parses -v short flag for verbose`
- `TestCLI_GlobalFlagsParsed/it parses --toon flag`
- `TestCLI_GlobalFlagsParsed/it parses --pretty flag`
- `TestCLI_GlobalFlagsParsed/it parses --json flag`
- `TestCLI_TTYDetection/it detects non-TTY and defaults to Toon via Run`
- `TestCLI_OutputFormatFlagOverride/it overrides TTY-detected format with --toon flag`
- `TestCLI_OutputFormatFlagOverride/it overrides TTY-detected format with --pretty flag`
- `TestCLI_OutputFormatFlagOverride/it overrides TTY-detected format with --json flag`

**discover_test.go** (69 lines, 3 subtests):
- `TestDiscoverTickDir/it discovers .tick/ directory by walking up from cwd`
- `TestDiscoverTickDir/it finds .tick/ in current directory`
- `TestDiscoverTickDir/it errors when no .tick/ directory found (not a tick project)`

**init_test.go** (249 lines, 9 subtests):
- `TestInit_CreatesTickDirectory/it creates .tick/ directory in current working directory`
- `TestInit_CreatesEmptyTasksJSONL/it creates empty tasks.jsonl inside .tick/`
- `TestInit_DoesNotCreateCacheDB/it does not create cache.db at init time`
- `TestInit_PrintsConfirmationWithAbsolutePath/it prints confirmation with absolute path on success`
- `TestInit_QuietFlagProducesNoOutput/it prints nothing with --quiet flag on success`
- `TestInit_QuietFlagProducesNoOutput/it also accepts -q short flag`
- `TestInit_ErrorWhenAlreadyExists/it errors when .tick/ already exists`
- `TestInit_ErrorWhenAlreadyExists/it returns exit code 1 when .tick/ already exists`
- `TestInit_WritesErrorsToStderr/it writes error messages to stderr, not stdout`

**V4 total: 22 subtests across 3 files**

#### V5 Test Functions (single cli_test.go, 362 lines)

**TestInit** (8 subtests):
- `it creates .tick/ directory in current working directory`
- `it creates empty tasks.jsonl inside .tick/`
- `it does not create cache.db at init time`
- `it prints confirmation with absolute path on success`
- `it prints nothing with --quiet flag on success`
- `it errors when .tick/ already exists`
- `it returns exit code 1 when .tick/ already exists`
- `it writes error messages to stderr, not stdout`
- `it errors even when .tick/ exists but is corrupted (missing tasks.jsonl)` **(UNIQUE)**
- `it accepts -q shorthand for --quiet`

**TestDiscoverTickDir** (3 subtests):
- `it discovers .tick/ directory by walking up from cwd`
- `it errors when no .tick/ directory found (not a tick project)`
- `it finds .tick/ in the starting directory itself`

**TestSubcommandRouting** (4 subtests):
- `it routes unknown subcommands to error`
- `it prints basic usage with no subcommand and exits 0`
- `it does not advertise doctor command in help output` **(UNIQUE)**
- `it returns unknown command error for doctor` **(UNIQUE)**

**TestTTYDetection** (1 subtest):
- `it detects TTY vs non-TTY on stdout`

**TestGlobalFlags** (5 subtests):
- `it parses --verbose flag`
- `it parses -v shorthand for --verbose`
- `it parses --toon flag`
- `it parses --pretty flag`
- `it parses --json flag`

**V5 total: 21 subtests in 1 file**

#### Test Gap Analysis

| Test | V4 | V5 |
|------|----|----|
| Corrupted `.tick/` (edge case from spec) | MISSING | PRESENT |
| `--quiet` on init (long flag) | PRESENT | PRESENT |
| `-q` shorthand | PRESENT | PRESENT |
| Format flag override verifies `OutputFormat` field | PRESENT (checks `app.OutputFormat`) | MISSING (only checks exit code) |
| Non-TTY defaults to Toon (verifies format field) | PRESENT (checks `app.OutputFormat`) | PARTIAL (tests both TTY values but can't check internal format) |
| Doctor command rejection | MISSING | PRESENT |
| Doctor not in help output | MISSING | PRESENT |

**Key test differences:**

1. **V4 verifies internal state** (e.g., `app.OutputFormat == FormatToon`) because it has access to the `App` struct after calling `Run`. V5 cannot do this because `Run` is a function that returns only an int -- the `Context` is local. This means V5's format-flag tests only verify "no crash" rather than "correct format selected."

2. **V5 tests the corrupted `.tick/` edge case** which is explicitly specified in the plan: "Even a corrupted `.tick/` (missing `tasks.jsonl`) counts as 'already initialized'." V4 misses this.

3. **V5 has doctor-command tests** which are not part of this task's spec but guard against regression from an unimplemented feature.

4. **V4 spreads tests across 3 files** matching the source file organization (cli_test, discover_test, init_test). V5 consolidates everything into `cli_test.go`. V4's approach is more conventional Go practice (test files mirror source files).

### 3.4 Skill Compliance

| Skill Requirement | V4 | V5 |
|-------------------|----|----|
| Table-driven tests with subtests | PARTIAL -- uses subtests (`t.Run`) but not table-driven | PARTIAL -- uses subtests but not table-driven |
| Document all exported functions | PASS | PASS |
| Handle all errors explicitly | PASS | PASS |
| Propagate errors with `fmt.Errorf("%w", err)` | PASS | PASS |
| No panic for error handling | PASS | PASS |
| No ignored errors without justification | PASS (one `os.RemoveAll` result ignored in cleanup, acceptable) | PASS |

Neither version uses table-driven tests for the flag parsing or init scenarios, which would have been more idiomatic. Both versions use `t.Run` subtests extensively.

### 3.5 Spec-vs-Convention Conflicts

1. **Error message for directory creation failure:** The spec says `"Error: Could not create .tick/ directory: <os error>"`. V4 uses `"failed to create .tick/ directory: <wrapped-error>"`. V5 uses `"creating .tick directory: <wrapped-error>"`. Both deviate. V4 is closer but still not exact.

2. **`os.Mkdir` vs `os.MkdirAll`:** The spec says "Create `.tick/` directory (mode 0755)". This implies a single directory creation. V5's `os.Mkdir` is the literal interpretation. V4's `os.MkdirAll` is overly permissive but functionally equivalent since `.tick/` is always a single level deep.

3. **Confirmation message format:** Spec says `"Initialized tick in <absolute-path>/.tick/"`. Both match this exactly.

4. **No subcommand behavior:** Spec says "No subcommand: print basic usage (list of commands) with exit code 0". Both comply. V5 prints a more comprehensive usage listing (all commands including create, update, start, etc.) while V4 only lists "init". V5's approach is forward-looking but arguably goes beyond the task scope.

---

## 4. Diff Stats

| Metric | V4 | V5 |
|--------|----|----|
| Files changed (task-relevant) | 7 | 5 |
| Lines added (task-relevant) | 796 | 584 |
| Lines deleted | 0 | 0 |
| Implementation files | 4 (`main.go`, `cli.go`, `discover.go`, `init.go`) | 4 (`main.go`, `cli.go`, `discover.go`, `init.go`) |
| Test files | 3 (`cli_test.go`, `discover_test.go`, `init_test.go`) | 1 (`cli_test.go`) |
| Implementation LOC | 232 | 211 |
| Test LOC | 564 | 362 |
| Test subtests | 22 | 21 |

---

## 5. Verdict

**V5 is the stronger implementation**, with specific advantages:

1. **Architecture:** V5's function-based approach with `Context` and map-based command dispatch is more idiomatic Go and substantially less boilerplate than V4's 70-line switch statement. The `commands` map pattern is a well-known Go idiom for CLI dispatch.

2. **main.go correctness:** V5 properly fails on `os.Getwd()` error rather than silently falling back to `"."`, which prevents downstream confusion.

3. **DetectTTY signature:** V5's `DetectTTY(*os.File)` is more type-safe than V4's `DetectTTY(io.Writer)`.

4. **Edge case coverage:** V5 tests the spec-required "corrupted `.tick/`" edge case that V4 misses entirely. V5 also validates unknown flag handling, which V4 silently misroutes.

5. **`os.Mkdir` vs `os.MkdirAll`:** V5's use of `os.Mkdir` is more correct for the stated purpose.

6. **Code economy:** V5 achieves equivalent functionality in 584 lines vs V4's 796 lines (-27%), primarily by eliminating dispatch boilerplate.

**V4 has two advantages:**

1. **Cleanup on failure:** V4's `os.RemoveAll(tickDir)` when `WriteFile` fails is a defensive measure V5 lacks. This is a minor but genuine improvement.

2. **Format flag verification:** V4's tests can verify the `OutputFormat` field was set correctly (e.g., `app.OutputFormat == FormatToon`) because of the struct-based design. V5's tests can only verify "no crash" for format flags because `Context` is not returned from `Run`.

3. **Test file organization:** V4 mirrors source files with test files (`init.go` -> `init_test.go`), which is conventional Go practice. V5 consolidates all tests in `cli_test.go`, which is acceptable for a small package but less navigable as the codebase grows.

These V4 advantages are minor relative to V5's architectural and correctness improvements. The corrupted `.tick/` edge case gap in V4 is a spec compliance issue, and the unknown-flag behavior difference (V5 rejects, V4 misroutes) demonstrates better defensive coding in V5.

**Winner: V5**
