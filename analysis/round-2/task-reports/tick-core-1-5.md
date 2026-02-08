# Task tick-core-1-5: CLI Framework & tick init

## Task Summary

This task establishes the CLI entry point and first command for Tick. It requires:

1. **CLI entry point** (`main.go`) with `os.Args` subcommand dispatch
2. **Subcommand routing**: parse first non-flag argument as subcommand; unknown subcommands return error to stderr with exit code 1
3. **Global flags**: `--quiet`/`-q`, `--verbose`/`-v`, `--toon`, `--pretty`, `--json`
4. **TTY detection** on stdout using `os.Stdout.Stat()` to check `ModeCharDevice`. Non-TTY defaults to TOON, TTY defaults to pretty. Flags override.
5. **Error handling**: all errors to stderr with `Error: ` prefix; exit 0 success, exit 1 error; handlers return errors, main exits
6. **`tick init` command**: create `.tick/` directory (0755) with empty `tasks.jsonl` (0644, 0 bytes), do NOT create `cache.db`, print `Initialized tick in <abs-path>/.tick/`, quiet mode suppresses output
7. **`.tick/` directory discovery**: walk up from cwd to filesystem root looking for `.tick/`. Error if not found.
8. **Edge cases**: already initialized (error even if corrupted), unwritable directory (surface OS error), unknown subcommand (specific error message), no subcommand (usage with exit 0)

### Acceptance Criteria (from plan)
- `tick init` creates `.tick/` directory with empty `tasks.jsonl`
- `tick init` does not create `cache.db`
- `tick init` prints confirmation with absolute path
- `tick init` with `--quiet` produces no output on success
- `tick init` when `.tick/` exists returns error to stderr with exit code 1
- All errors written to stderr with `Error: ` prefix
- Exit code 0 for success, 1 for errors
- Global flags parsed: `--quiet`, `--verbose`, `--toon`, `--pretty`, `--json`
- TTY detection on stdout selects default output format
- `.tick/` directory discovery walks up from cwd
- Unknown subcommands return error with exit code 1

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| `tick init` creates `.tick/` with empty `tasks.jsonl` | PASS - `runInit()` calls `os.Mkdir(tickDir, 0755)` then `os.WriteFile(jsonlPath, []byte{}, 0644)` | PASS - `runInit()` calls `os.MkdirAll(tickDir, 0755)` then `os.WriteFile(jsonlPath, []byte{}, 0644)` |
| `tick init` does not create `cache.db` | PASS - No cache.db creation in init code; tested explicitly | PASS - No cache.db creation in init code; tested explicitly |
| `tick init` prints confirmation with absolute path | PASS - `fmt.Fprintf(a.stdout, "Initialized tick in %s/\n", tickDir)` | PASS - `fmt.Fprintf(a.Stdout, "Initialized tick in %s/\n", tickDir)` |
| `tick init` with `--quiet` produces no output | PASS - `if !a.config.Quiet` guard; tested with both `--quiet` and `-q` | PASS - `if !a.Quiet` guard; tested with both `--quiet` and `-q` |
| `tick init` when `.tick/` exists returns error to stderr with exit 1 | PASS - Returns `fmt.Errorf("Tick already initialized in this directory")`, main handles exit; integration test verifies exit code 1 | PASS - Returns same error, `Run()` returns int 1 directly; unit test verifies exit code 1 |
| All errors to stderr with `Error: ` prefix | PASS - `main.go` does `fmt.Fprintf(os.Stderr, "Error: %s\n", err)`; integration test verifies | PASS - `writeError()` method does `fmt.Fprintf(a.Stderr, "Error: %s\n", err.Error())`; unit test verifies |
| Exit code 0 success, 1 errors | PASS - `main.go` calls `os.Exit(1)` on error, implicit 0 on success | PASS - `Run()` returns int directly; `main.go` calls `os.Exit(app.Run(os.Args))` |
| Global flags parsed | PASS - All 5 flags + 2 short forms parsed in `parseGlobalFlags()` loop | PASS - All 5 flags + 2 short forms parsed in `parseGlobalFlags()` loop |
| TTY detection selects default format | PASS - `detectTTY()` package-level function checks `os.Stdout.Stat()` | PASS - `detectTTY()` method checks `a.Stdout` type-asserted to `*os.File` |
| `.tick/` discovery walks up from cwd | PASS - `DiscoverTickDir()` walks up with `filepath.Dir()` loop | PASS - `DiscoverTickDir()` walks up with `filepath.Dir()` loop |
| Unknown subcommands return error with exit 1 | PASS - `default` case in switch returns `fmt.Errorf("Unknown command '%s'...")` | PASS - `default` case calls `writeError()` and returns 1 |

## Implementation Comparison

### Approach

Both versions follow a very similar architectural pattern: a central `App` struct in `internal/cli/` with a `Run()` method that parses global flags and dispatches subcommands. The key structural differences are in the API surface and error propagation model.

#### V2: Error-returning `Run()` with private fields

V2's `App` struct uses private fields with a constructor:

```go
// internal/cli/app.go
type App struct {
    config  Config
    workDir string
    stdout  io.Writer
    stderr  io.Writer
}

func NewApp() *App {
    return &App{
        stdout: os.Stdout,
        stderr: os.Stderr,
    }
}
```

`Run()` returns `error`, leaving exit code handling to `main.go`:

```go
// cmd/tick/main.go
func main() {
    app := cli.NewApp()
    if err := app.Run(os.Args); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %s\n", err)
        os.Exit(1)
    }
}
```

A separate `Config` struct groups global flags:

```go
type Config struct {
    Quiet        bool
    Verbose      bool
    OutputFormat OutputFormat
}
```

#### V4: Integer-returning `Run()` with exported fields

V4's `App` struct uses exported fields, no constructor:

```go
// internal/cli/cli.go
type App struct {
    Stdout io.Writer
    Stderr io.Writer
    Dir    string

    Quiet        bool
    Verbose      bool
    OutputFormat OutputFormat
    IsTTY        bool
}
```

`Run()` returns `int` (the exit code), handling error formatting internally via `writeError()`:

```go
// cmd/tick/main.go
func main() {
    app := &cli.App{
        Stdout: os.Stdout,
        Stderr: os.Stderr,
        Dir:    ".",
    }
    if wd, err := os.Getwd(); err == nil {
        app.Dir = wd
    }
    os.Exit(app.Run(os.Args))
}
```

**Key difference**: V4's `main.go` resolves the working directory itself, while V2 defers it to `Run()` (falling back to `os.Getwd()` if `workDir` is empty). V4's `main.go` is slightly more explicit about what happens if `Getwd()` fails (falls back to `"."`), while V2 would propagate the error.

#### Error propagation model

This is the most significant architectural difference:

- **V2**: `Run()` returns `error`. The `main()` function is responsible for formatting (`"Error: %s\n"`) and calling `os.Exit(1)`. This means `init` errors bubble up naturally as Go errors.
- **V4**: `Run()` returns `int`. The `writeError()` method handles formatting internally. Each subcommand dispatch does `a.writeError(err); return 1`. This means `Run()` is self-contained regarding I/O but the error formatting responsibility is split between the `App` internals.

V2's approach is more idiomatic Go -- errors returned from functions, formatted at the boundary. V4's approach is more self-contained but duplicates the `writeError()`/`return 1` pattern in multiple places within `Run()`.

#### TTY Detection

**V2** uses a package-level function that always checks `os.Stdout`:

```go
func detectTTY() bool {
    fi, err := os.Stdout.Stat()
    if err != nil {
        return false
    }
    return fi.Mode()&os.ModeCharDevice != 0
}
```

**V4** uses a method that type-asserts `a.Stdout` to `*os.File`:

```go
func (a *App) detectTTY() {
    a.IsTTY = false
    a.OutputFormat = FormatTOON

    if f, ok := a.Stdout.(*os.File); ok {
        info, err := f.Stat()
        if err == nil && (info.Mode()&os.ModeCharDevice) != 0 {
            a.IsTTY = true
            a.OutputFormat = FormatPretty
        }
    }
}
```

**V4's approach is genuinely better** here. It works against the configured `Stdout` writer rather than always hitting `os.Stdout`. This makes it properly testable -- when `Stdout` is a `bytes.Buffer`, it correctly falls through as non-TTY. V2's `detectTTY()` always checks the real `os.Stdout` regardless of what `a.stdout` is set to, which means TTY detection is effectively untestable with injected writers. Both versions end up working in tests (where stdout is piped), but V4's design is cleaner and more correct.

#### init command

**V2** uses `os.Mkdir()`:
```go
if err := os.Mkdir(tickDir, 0755); err != nil {
    return fmt.Errorf("Could not create .tick/ directory: %w", err)
}
```

**V4** uses `os.MkdirAll()`:
```go
if err := os.MkdirAll(tickDir, 0755); err != nil {
    return fmt.Errorf("failed to create .tick/ directory: %w", err)
}
```

Since `.tick/` is always one level deep from an existing cwd, `os.Mkdir` is technically more correct -- it will fail if the parent doesn't exist, which should never happen and would be a meaningful error. `os.MkdirAll` would silently succeed in edge cases where the parent was somehow missing.

**V4 adds cleanup on failure** -- if `WriteFile` fails after creating the directory, it calls `os.RemoveAll(tickDir)`. V2 does not clean up, leaving a `.tick/` directory without `tasks.jsonl`. This is a genuine improvement in V4.

#### Subcommand argument passing

**V2** `runInit()` takes no arguments: `func (a *App) runInit() error`
**V4** `runInit()` accepts `args []string`: `func (a *App) runInit(args []string) error`

V4's approach is more extensible -- `init` could later accept subcommand-specific flags. Neither version currently uses subcommand args.

#### parseGlobalFlags remaining args

Both versions use the same pattern -- iterate through args, match flags, return on first non-flag. But V4 appends `args[i:]` (the rest of the args from the first non-flag onward) to `remaining`, while V2 just returns the single arg string. This means V4 properly preserves subcommand arguments, while V2 discards anything after the subcommand name.

**V2:**
```go
default:
    return arg, nil  // returns only the subcommand name
```

**V4:**
```go
default:
    remaining = append(remaining, args[i:]...)
    return remaining, nil  // returns subcommand + all its args
```

V4's approach is genuinely better for extensibility.

### Code Quality

#### Naming

- **V2**: `App`, `Config`, `NewApp()`, `runInit()`, `parseGlobalFlags()`, `printUsage()`, `detectTTY()`, `DiscoverTickDir()`. Uses `config.Quiet`, `config.Verbose`, `config.OutputFormat`. File name: `app.go`.
- **V4**: `App`, no separate config struct. Uses `a.Quiet`, `a.Verbose`, `a.OutputFormat`. File name: `cli.go`. Adds `writeError()` helper. Adds `IsTTY` field.

V2's separate `Config` struct provides better encapsulation. V4's flat structure on `App` is simpler but mixes configuration concerns with runtime state (`IsTTY`).

#### Error messages

**V2** uses capitalized error messages:
```go
return fmt.Errorf("Could not create .tick/ directory: %w", err)
return fmt.Errorf("Could not determine absolute path: %w", err)
return fmt.Errorf("Tick already initialized in this directory")
```

**V4** uses lowercase (Go convention) for some, capitalized for others:
```go
return fmt.Errorf("failed to create .tick/ directory: %w", err)
return fmt.Errorf("failed to resolve absolute path: %w", err)
return fmt.Errorf("Tick already initialized in this directory")  // inconsistent
```

Neither is fully consistent. Go convention is lowercase error messages (since they may be wrapped), but the spec says `Error: ` prefix goes on stderr output. V2 is internally consistent (always capitalized). V4 is inconsistent (mixed case).

However, the spec explicitly requires: `"Error: Could not create .tick/ directory: <os error>"` -- V2 matches this exactly, V4 does not (uses `"failed to create..."`).

#### Error handling in discover.go

**V2:**
```go
return "", errors.New("Not a tick project (no .tick directory found)")
```

**V4:**
```go
return "", fmt.Errorf("Not a tick project (no .tick directory found)")
```

Both produce the same result. V2 uses `errors.New` (slightly more idiomatic when no formatting is needed). V4 uses `fmt.Errorf` (unnecessary overhead for a static string).

#### Package doc comment

**V2** puts the package doc on `discover.go`:
```go
// Package cli implements the command-line interface for Tick.
package cli
```

**V4** puts it on `cli.go`:
```go
// Package cli provides the command-line interface for Tick.
// It handles subcommand dispatch, global flags, TTY detection, and error formatting.
package cli
```

V4's package comment is more descriptive.

#### Usage output

**V2** uses a single multi-line string literal:
```go
usage := `Usage: tick <command> [options]
...`
fmt.Fprint(a.stdout, usage)
```

**V4** uses multiple `fmt.Fprintln` calls:
```go
fmt.Fprintln(a.Stdout, "Usage: tick <command> [options]")
fmt.Fprintln(a.Stdout, "")
...
```

V2's approach is more readable and efficient (single write). V4's approach has more overhead (multiple syscalls) and is harder to read.

### Test Quality

#### V2 Test Functions

**`cmd/tick/main_test.go`** (integration tests -- builds binary):
1. `TestMainIntegration/it returns exit code 1 when .tick/ already exists` -- verifies `*exec.ExitError` exit code 1 and `Error: ` prefix
2. `TestMainIntegration/it returns exit code 0 on successful init` -- verifies no error and "Initialized tick in" output
3. `TestMainIntegration/it writes error messages to stderr, not stdout` -- separates stdout/stderr, verifies empty stdout and `Error: ` prefix on stderr
4. `TestMainIntegration/it returns exit code 1 for unknown subcommand` -- verifies exit code 1 and `Error: Unknown command 'nonexistent'` message
5. `TestMainIntegration/it returns exit code 0 with no subcommand (prints usage)` -- verifies exit 0 and `Usage:` in output
6. `TestMainIntegration/it prefixes all error messages with Error:` -- verifies `Error: ` prefix on stderr for bad command

**`internal/cli/app_test.go`** (unit tests):
1. `TestAppRouting/it routes unknown subcommands to error` -- exact error message match
2. `TestAppRouting/it prints basic usage with no subcommand` -- checks `Usage:` and `init` in output
3. `TestAppRouting/it parses --quiet global flag`
4. `TestAppRouting/it parses -q global flag`
5. `TestAppRouting/it parses --verbose global flag`
6. `TestAppRouting/it parses -v global flag`
7. `TestAppRouting/it parses --toon global flag` -- verifies `FormatTOON`
8. `TestAppRouting/it parses --pretty global flag` -- verifies `FormatPretty`
9. `TestAppRouting/it parses --json global flag` -- verifies `FormatJSON`
10. `TestTTYDetection/it detects TTY vs non-TTY on stdout` -- verifies `detectTTY()` returns false in pipe
11. `TestTTYDetection/it defaults to TOON format when not a TTY` -- verifies `FormatTOON` after `Run()`

**`internal/cli/discover_test.go`**:
1. `TestDiscoverTickDir/it discovers .tick/ directory by walking up from cwd` -- nested dir finds root `.tick/`
2. `TestDiscoverTickDir/it finds .tick/ in the starting directory itself`
3. `TestDiscoverTickDir/it errors when no .tick/ directory found (not a tick project)` -- exact message match
4. `TestDiscoverTickDir/it stops at the first .tick/ match walking up` -- two `.tick/` dirs at different levels, verifies closest found

**`internal/cli/init_test.go`**:
1. `TestInitCommand/it creates .tick/ directory in current working directory` -- `os.Stat` + `IsDir()`
2. `TestInitCommand/it creates empty tasks.jsonl inside .tick/` -- `os.Stat` + `Size() == 0`
3. `TestInitCommand/it does not create cache.db at init time` -- `os.IsNotExist()`
4. `TestInitCommand/it prints confirmation with absolute path on success` -- exact string match with `filepath.Abs`
5. `TestInitCommand/it prints nothing with --quiet flag on success` -- empty stdout
6. `TestInitCommand/it prints nothing with -q flag on success` -- empty stdout with short flag
7. `TestInitCommand/it errors when .tick/ already exists` -- contains "already initialized"
8. `TestInitCommand/it errors when .tick/ already exists even if corrupted (missing tasks.jsonl)` -- empty `.tick/` with no `tasks.jsonl`
9. `TestInitCommand/it writes error messages to stderr, not stdout` -- empty stdout on error
10. `TestInitCommand/it surfaces OS error when directory is not writable` -- `chmod 0555`, verifies "Could not create .tick/ directory"

**V2 total: 27 test cases** (6 integration + 11 app + 4 discover + 10 init)

#### V4 Test Functions

**No `cmd/tick/main_test.go`** -- V4 has no integration tests.

**`internal/cli/cli_test.go`** (unit tests):
1. `TestCLI_UnknownSubcommand/it routes unknown subcommands to error` -- verifies exit 1, `Error: ` prefix, `Unknown command 'foobar'`, `Run 'tick help' for usage`, empty stdout
2. `TestCLI_NoSubcommand/it prints basic usage with exit code 0 when no subcommand given` -- verifies exit 0, non-empty stdout, contains "usage"
3. `TestCLI_GlobalFlagsParsed/it parses --quiet flag before subcommand` -- verifies exit 0, empty stdout
4. `TestCLI_GlobalFlagsParsed/it parses --verbose flag` -- verifies exit 0
5. `TestCLI_GlobalFlagsParsed/it parses -v short flag for verbose` -- verifies exit 0
6. `TestCLI_GlobalFlagsParsed/it parses --toon flag` -- verifies exit 0
7. `TestCLI_GlobalFlagsParsed/it parses --pretty flag` -- verifies exit 0
8. `TestCLI_GlobalFlagsParsed/it parses --json flag` -- verifies exit 0
9. `TestCLI_TTYDetection/it detects TTY vs non-TTY on stdout` -- verifies `IsTTY == false` and `FormatTOON` for `bytes.Buffer`
10. `TestCLI_OutputFormatFlagOverride/it overrides TTY-detected format with --toon flag` -- verifies `FormatTOON`
11. `TestCLI_OutputFormatFlagOverride/it overrides TTY-detected format with --pretty flag` -- verifies `FormatPretty`
12. `TestCLI_OutputFormatFlagOverride/it overrides TTY-detected format with --json flag` -- verifies `FormatJSON`

**`internal/cli/discover_test.go`**:
1. `TestDiscoverTickDir/it discovers .tick/ directory by walking up from cwd` -- nested dir finds root `.tick/`
2. `TestDiscoverTickDir/it finds .tick/ in current directory`
3. `TestDiscoverTickDir/it errors when no .tick/ directory found (not a tick project)` -- checks both parts of message

**`internal/cli/init_test.go`**:
1. `TestInit_CreatesTickDirectory/it creates .tick/ directory in current working directory`
2. `TestInit_CreatesEmptyTasksJSONL/it creates empty tasks.jsonl inside .tick/`
3. `TestInit_DoesNotCreateCacheDB/it does not create cache.db at init time` -- also checks `os.IsNotExist` explicitly
4. `TestInit_PrintsConfirmationWithAbsolutePath/it prints confirmation with absolute path on success`
5. `TestInit_QuietFlagProducesNoOutput/it prints nothing with --quiet flag on success` -- also verifies stderr is empty
6. `TestInit_QuietFlagProducesNoOutput/it also accepts -q short flag`
7. `TestInit_ErrorWhenAlreadyExists/it errors when .tick/ already exists` -- verifies exit 1, `Error: ` prefix, "already initialized", empty stdout
8. `TestInit_ErrorWhenAlreadyExists/it returns exit code 1 when .tick/ already exists`
9. `TestInit_WritesErrorsToStderr/it writes error messages to stderr, not stdout` -- verifies empty stdout, non-empty stderr, `Error: ` prefix

**V4 total: 24 test cases** (0 integration + 12 cli + 3 discover + 9 init)

#### Test Coverage Comparison

| Test Aspect | V2 | V4 |
|-------------|-----|-----|
| Integration tests (binary build) | YES (6 tests) | NO |
| Unknown subcommand error | YES (unit + integration) | YES (unit) |
| No subcommand usage | YES (unit + integration) | YES (unit) |
| --quiet / -q flag | YES (both tested separately) | YES (both tested) |
| --verbose / -v flag | YES (both tested) | YES (both tested) |
| --toon flag | YES (verifies OutputFormat value) | YES (verifies OutputFormat value) |
| --pretty flag | YES (verifies OutputFormat value) | YES (verifies OutputFormat value) |
| --json flag | YES (verifies OutputFormat value) | YES (verifies OutputFormat value) |
| TTY detection | YES (detects non-TTY in pipe) | YES (detects non-TTY for bytes.Buffer) |
| Default TOON for non-TTY | YES | YES |
| Format flag overrides TTY default | NO explicit test | YES (3 tests) |
| .tick/ directory created | YES | YES |
| tasks.jsonl created empty | YES | YES |
| cache.db not created | YES | YES |
| Confirmation with abs path | YES | YES |
| Quiet suppresses output | YES | YES |
| Already initialized error | YES | YES |
| Corrupted .tick/ (missing tasks.jsonl) | YES | NO |
| Unwritable directory | YES | NO |
| Errors to stderr not stdout | YES (unit + integration) | YES (unit) |
| Exit code 1 on error | YES (integration verifies real exit code) | YES (unit verifies return int) |
| Discovery walks up | YES | YES |
| Discovery in current dir | YES | YES |
| Discovery not found | YES | YES |
| Discovery stops at first match | YES | NO |
| Error prefix verification | YES (multiple tests) | YES (multiple tests) |

**V2 unique coverage**: integration tests (real binary exit codes), corrupted `.tick/` edge case, unwritable directory edge case, discovery stops at first match (two nested `.tick/` dirs).

**V4 unique coverage**: format flag override tests (explicitly verify that `--toon`/`--pretty`/`--json` override the TTY-detected default).

#### Test Style

**V2** uses:
- Individual subtests under parent `t.Run()` groups (`TestAppRouting`, `TestTTYDetection`, `TestInitCommand`, `TestDiscoverTickDir`)
- In `main_test.go`: full integration tests building the binary with `go build` and using `exec.Command`
- Assertions via direct comparison, `strings.Contains`, `strings.HasPrefix`
- Private field injection: `app.workDir = dir`, `app.stdout = &stdout` (possible due to same-package tests)

**V4** uses:
- Top-level test functions per concern (`TestInit_CreatesTickDirectory`, `TestInit_CreatesEmptyTasksJSONL`, etc.) with a single subtest each
- No integration tests
- `bytes.Buffer` for stdout/stderr capture (explicit `var stdout, stderr bytes.Buffer`)
- Exported field injection: `app.Stdout = &stdout`, `app.Dir = dir`
- Assertions via `strings.Contains`, `strings.HasPrefix`, equality checks

V2's integration tests are a significant advantage -- they verify the actual binary behavior including real exit codes, real stderr separation, and the `Error: ` prefix being applied by `main()`. V4 only tests the `Run()` method return values and internal buffer contents.

V4's test organization (one top-level function per concern) is arguably cleaner for discoverability but creates many small test functions rather than logical groupings.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed | 10 (8 Go + 2 docs) | 9 (7 Go + 2 docs) |
| Lines added | 860 | 800 |
| Impl files | 4 (`main.go`, `app.go`, `discover.go`, `init.go`) | 4 (`main.go`, `cli.go`, `discover.go`, `init.go`) |
| Impl LOC | 220 (17+130+32+41) | 232 (23+135+31+43) |
| Test files | 4 (`main_test.go`, `app_test.go`, `discover_test.go`, `init_test.go`) | 3 (`cli_test.go`, `discover_test.go`, `init_test.go`) |
| Test LOC | 637 (168+173+95+201) | 564 (246+69+249) |
| Test functions | 27 subtests across 5 top-level functions | 24 subtests across 10 top-level functions |

## Verdict

**V2 is the better implementation of this task.**

The primary reasons:

1. **Integration tests**: V2 includes a full integration test suite (`main_test.go`, 168 lines) that builds the actual binary and tests real exit codes, real stderr separation, and the complete error formatting pipeline. V4 has no integration tests at all. This is a significant gap -- V4 never verifies that the `main()` function correctly calls `os.Exit()` with the right code, or that the `Error: ` prefix actually appears in the real binary output.

2. **Better edge case coverage**: V2 tests the corrupted `.tick/` edge case (directory exists but no `tasks.jsonl`), the unwritable directory edge case (`chmod 0555`), and the discovery "stops at first match" behavior with nested `.tick/` directories. V4 misses all three.

3. **More idiomatic error propagation**: V2's `Run()` returns `error`, with `main()` responsible for formatting and exit codes. This follows the Go convention of returning errors from functions and handling them at the boundary. V4's `Run()` returns `int` which conflates error handling with exit code management inside the library.

4. **Spec-matching error messages**: V2's error messages (`"Could not create .tick/ directory: ..."`) exactly match the spec. V4 uses `"failed to create .tick/ directory: ..."` which deviates from the spec.

5. **Encapsulation**: V2's private fields with `NewApp()` constructor provide better encapsulation than V4's exported fields. Tests use same-package access which is fine, but external callers cannot accidentally mutate `config` or `workDir`.

V4 has a few genuine advantages: the `detectTTY()` method correctly operates on the configured `Stdout` writer (rather than always `os.Stdout`); it preserves subcommand arguments through `parseGlobalFlags()`; it cleans up on partial failure in `runInit()`; and it has explicit format-flag-override tests. But these are smaller wins compared to V2's integration test coverage and closer spec adherence.
