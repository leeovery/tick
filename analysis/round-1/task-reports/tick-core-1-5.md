# Task tick-core-1-5: CLI framework & tick init

## Task Summary

This task requires building the CLI entry point and the first command (`tick init`) for the tick project management tool. The specification mandates:

1. A `main.go` entry point with `os.Args` subcommand dispatch
2. Subcommand routing (first non-flag argument = subcommand, unknown = error)
3. Global flags: `--quiet`/`-q`, `--verbose`/`-v`, `--toon`, `--pretty`, `--json`
4. TTY detection on stdout (`os.Stdout.Stat()` checking `ModeCharDevice`)
5. Error handling: all errors to stderr with `Error: ` prefix, exit 0/1
6. `tick init`: create `.tick/` (0755) + empty `tasks.jsonl` (0644), no `cache.db`
7. Confirmation message: `Initialized tick in <absolute-path>/.tick/`
8. `--quiet` suppresses success output
9. `.tick/` directory discovery: walk up from cwd to root
10. Already-initialized detection (even corrupted `.tick/` counts)
11. Unknown subcommand error: `Error: Unknown command '<name>'. Run 'tick help' for usage.`
12. No subcommand: print basic usage with exit code 0

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

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| `tick init` creates `.tick/` + empty `tasks.jsonl` | PASS - tested in `TestInitCommand/"creates .tick directory"` and `"creates empty tasks.jsonl"` | PASS - tested in `TestInitCommand/"it creates .tick/ directory"` and `"it creates empty tasks.jsonl"` | PASS - tested in `TestInitCommand/"it creates .tick/ directory"` and `"it creates empty tasks.jsonl"` |
| `tick init` does not create `cache.db` | PASS - tested in `TestInitCommand/"does not create cache.db"` | PASS - tested in `TestInitCommand/"it does not create cache.db"` | PASS - tested in `TestInitCommand/"it does not create cache.db"` |
| Prints confirmation with absolute path | PASS - uses `filepath.Abs(tickDir)` in `cmdInit`, tested | PASS - uses `filepath.Abs(a.workDir)` in `runInit`, tested | PARTIAL - uses `a.Cwd` directly (no `filepath.Abs` call), relies on `Cwd` being absolute already. Works in tests because `t.TempDir()` returns absolute paths, but no normalization if relative `Cwd` is provided. |
| `--quiet` produces no output on success | PASS - checks `a.opts.Quiet`, tested | PASS - checks `a.config.Quiet`, tested | PASS - checks `a.flags.Quiet`, tested |
| `.tick/` exists returns error to stderr with exit 1 | PASS - tested in `"errors when .tick already exists"` | PASS - tested in `"it errors when .tick/ already exists"` | PASS - tested in `"it errors when .tick/ already exists"` |
| All errors to stderr with `Error: ` prefix | PASS - `fmt.Fprintf(a.stderr, "Error: %s\n", err)` in `Run()` | PARTIAL - errors returned from `Run()` have no `Error: ` prefix; `main.go` adds the prefix. Internal `runInit` returns bare errors. Unknown subcommand error message includes `Error:` but only when formatted by `main.go`. | PASS - all error writes use `fmt.Fprintf(a.Stderr, "Error: ...")`  directly in handler methods |
| Exit 0 success, 1 errors | PASS - `Run()` returns int | PARTIAL - `Run()` returns `error`, `main.go` exits 1 on error. Unit tests can't verify exit codes directly. Integration tests (`main_test.go`) verify via binary. | PASS - `Run()` returns int directly |
| Global flags: --quiet, --verbose, --toon, --pretty, --json | PASS - all parsed in `parseGlobalFlags`. Stored as individual bools (`Toon`, `Pretty`, `JSON`). | PASS - all parsed in `parseGlobalFlags`. Stored as `OutputFormat` enum type. | PASS - all parsed in `ParseGlobalFlags` (exported). Stored as `OutputFormat` string. |
| TTY detection on stdout | FAIL - no TTY detection implemented. No `os.Stdout.Stat()` or `ModeCharDevice` check anywhere. | PASS - `detectTTY()` calls `os.Stdout.Stat()` and checks `ModeCharDevice`. Applied in `Run()` when no format flag set. | PASS - `IsTTY(io.Writer)` checks writer via `*os.File` assertion, then `Stat()` + `ModeCharDevice`. `DefaultOutputFormat()` uses it. |
| `.tick/` directory discovery walks up from cwd | PASS - `FindTickDir()` implemented and tested | PASS - `DiscoverTickDir()` implemented in separate file, tested with 4 subcases | PASS - `DiscoverTickDir()` implemented inline in `cli.go`, tested with 3 subcases |
| Unknown subcommands return error with exit 1 | PASS - tested in `TestUnknownSubcommand` | PASS - tested in both `TestAppRouting` and `TestMainIntegration` | PASS - tested in `TestUnknownSubcommand` |

## Implementation Comparison

### Approach

#### File Organization

**V1**: Monolithic - everything in 2 Go files plus main:
- `cmd/tick/main.go` (18 lines)
- `internal/cli/cli.go` (169 lines) - App, GlobalOpts, init, FindTickDir, parseGlobalFlags, printUsage
- `internal/cli/cli_test.go` (194 lines)

**V2**: Highly decomposed - 8 Go files plus main:
- `cmd/tick/main.go` (17 lines)
- `cmd/tick/main_test.go` (168 lines) - integration tests building actual binary
- `internal/cli/app.go` (130 lines) - App, Config, OutputFormat, TTY detection
- `internal/cli/app_test.go` (173 lines) - routing and flag tests
- `internal/cli/discover.go` (32 lines)
- `internal/cli/discover_test.go` (95 lines)
- `internal/cli/init.go` (41 lines)
- `internal/cli/init_test.go` (201 lines)

**V3**: Monolithic like V1 - everything in 2 Go files plus main:
- `cmd/tick/main.go` (25 lines)
- `internal/cli/cli.go` (174 lines) - App, GlobalFlags, init, DiscoverTickDir, IsTTY, ParseGlobalFlags, printUsage, DefaultOutputFormat
- `internal/cli/cli_test.go` (475 lines)

#### App struct design

**V1** uses unexported fields with a constructor:
```go
type App struct {
    stdout io.Writer
    stderr io.Writer
    opts   GlobalOpts
}

func NewApp(stdout, stderr io.Writer) *App {
    return &App{stdout: stdout, stderr: stderr}
}
```
`Run()` takes `(args []string, workDir string)` - workDir is passed per-call, not stored on the struct.

**V2** uses unexported fields with a constructor, but stores workDir:
```go
type App struct {
    config  Config
    workDir string
    stdout  io.Writer
    stderr  io.Writer
}

func NewApp() *App {
    return &App{stdout: os.Stdout, stderr: os.Stderr}
}
```
`Run()` takes only `(args []string)` and returns `error` (not int). Tests override `app.workDir` and `app.stdout` directly (unexported fields accessed from same package).

**V3** uses exported fields with struct literal construction:
```go
type App struct {
    Stdout io.Writer
    Stderr io.Writer
    Cwd    string
    flags  GlobalFlags
}
```
No constructor. Tests create `&App{Stdout: &stdout, Stderr: &stderr, Cwd: dir}` directly. `Run()` takes `(args []string)` and returns `int`.

#### Error handling strategy

**V1**: `Run()` returns `int` (exit code). The `cmdInit` method returns `error`, and `Run()` formats it to stderr with `Error: ` prefix:
```go
if err != nil {
    fmt.Fprintf(a.stderr, "Error: %s\n", err)
    return 1
}
```
Unknown subcommand errors bypass this and write directly to stderr.

**V2**: `Run()` returns `error`. The `main.go` is responsible for formatting and exit:
```go
func main() {
    app := cli.NewApp()
    if err := app.Run(os.Args); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %s\n", err)
        os.Exit(1)
    }
}
```
This means `runInit()` also returns `error`. The unknown subcommand error message includes `"Unknown command '%s'. Run 'tick help' for usage."` which gets wrapped with `Error: ` by main. This is a clean separation but means the CLI layer doesn't directly control the `Error: ` prefix.

**V3**: `Run()` returns `int` (exit code). The `runInit()` method also returns `int`. All error formatting happens inline in each handler:
```go
func (a *App) runInit() int {
    // ...
    if _, err := os.Stat(tickDir); err == nil {
        fmt.Fprintf(a.Stderr, "Error: Tick already initialized in this directory\n")
        return 1
    }
    // ...
}
```
No centralized error formatting - each handler writes `Error: ` prefix itself.

#### Global flags model

**V1** - `GlobalOpts` with individual booleans:
```go
type GlobalOpts struct {
    Quiet   bool
    Verbose bool
    Toon    bool
    Pretty  bool
    JSON    bool
}
```
Output format flags stored as separate booleans. No unified format concept. No TTY detection.

**V2** - `Config` with typed `OutputFormat`:
```go
type OutputFormat string
const (
    FormatTOON   OutputFormat = "toon"
    FormatPretty OutputFormat = "pretty"
    FormatJSON   OutputFormat = "json"
)
type Config struct {
    Quiet        bool
    Verbose      bool
    OutputFormat OutputFormat
}
```
Typed constants, TTY detection integrated into `Run()`:
```go
if a.config.OutputFormat == "" {
    if detectTTY() {
        a.config.OutputFormat = FormatPretty
    } else {
        a.config.OutputFormat = FormatTOON
    }
}
```

**V3** - `GlobalFlags` with string OutputFormat:
```go
type GlobalFlags struct {
    Quiet        bool
    Verbose      bool
    OutputFormat string // "toon", "pretty", "json", or "" for auto-detect
}
```
String-based format, TTY detection is lazy via `DefaultOutputFormat()`:
```go
func (a *App) DefaultOutputFormat() string {
    if a.flags.OutputFormat != "" {
        return a.flags.OutputFormat
    }
    if IsTTY(a.Stdout) {
        return "pretty"
    }
    return "toon"
}
```

#### Flag parsing approach

**V1** iterates with index, handles unknown flags specially:
```go
default:
    if strings.HasPrefix(arg, "-") {
        // Unknown flag -- pass through to subcommand
        remaining = append(remaining, args[i:]...)
        return subcmd, remaining
    }
```
This means if an unknown `-flag` is encountered, everything from that point on is passed to the subcommand. This is a reasonable approach for future extensibility.

**V2** iterates with index, stops at first non-flag (subcommand):
```go
default:
    // First non-flag argument is the subcommand
    return arg, nil
```
Everything after the subcommand is lost (no remaining args returned). Flags must come before the subcommand.

**V3** uses exported `ParseGlobalFlags()` that iterates all args, stripping flags and keeping non-flags:
```go
func ParseGlobalFlags(args []string) (GlobalFlags, []string) {
    for _, arg := range args {
        switch arg {
        case "-q", "--quiet":
            flags.Quiet = true
        // ...
        default:
            remaining = append(remaining, arg)
        }
    }
    return flags, remaining
}
```
Flags can appear anywhere (before or after subcommand). This is the most flexible approach.

#### TTY detection

**V1**: Not implemented at all. This is a clear gap against the acceptance criteria.

**V2**: Package-level function hardcoded to `os.Stdout`:
```go
func detectTTY() bool {
    fi, err := os.Stdout.Stat()
    if err != nil {
        return false
    }
    return fi.Mode()&os.ModeCharDevice != 0
}
```
Called eagerly in `Run()`. Not injectable/testable beyond the fact that tests run in pipes.

**V3**: Takes an `io.Writer` parameter, making it testable:
```go
func IsTTY(w io.Writer) bool {
    f, ok := w.(*os.File)
    if !ok {
        return false
    }
    info, err := f.Stat()
    if err != nil {
        return false
    }
    return info.Mode()&os.ModeCharDevice != 0
}
```
Called lazily via `DefaultOutputFormat()`. More testable because non-`*os.File` writers (like `bytes.Buffer`) correctly return false.

#### Directory discovery

**V1** - `FindTickDir(startDir string)` uses `filepath.Abs` to normalize first:
```go
func FindTickDir(startDir string) (string, error) {
    dir, err := filepath.Abs(startDir)
    if err != nil {
        return "", fmt.Errorf("resolving path: %w", err)
    }
    // walk up...
}
```

**V2** - `DiscoverTickDir(startDir string)` also uses `filepath.Abs`:
```go
func DiscoverTickDir(startDir string) (string, error) {
    dir, err := filepath.Abs(startDir)
    if err != nil {
        return "", err
    }
    // walk up...
}
```
Error message: `"Not a tick project (no .tick directory found)"` (capital N).

**V3** - `DiscoverTickDir(startDir string)` does NOT normalize with `filepath.Abs`:
```go
func DiscoverTickDir(startDir string) (string, error) {
    dir := startDir
    // walk up...
}
```
Error message: `"not a tick project (no .tick directory found)"` (lowercase n).

#### Init command - absolute path handling

**V1**: Uses `filepath.Abs(tickDir)` at output time:
```go
absDir, _ := filepath.Abs(tickDir)
fmt.Fprintf(a.stdout, "Initialized tick in %s/\n", absDir)
```

**V2**: Uses `filepath.Abs(a.workDir)` at start of `runInit`:
```go
absDir, err := filepath.Abs(a.workDir)
if err != nil {
    return fmt.Errorf("Could not determine absolute path: %w", err)
}
tickDir := filepath.Join(absDir, ".tick")
```

**V3**: Relies on `a.Cwd` being absolute already:
```go
tickDir := filepath.Join(a.Cwd, ".tick")
// ...
fmt.Fprintf(a.Stdout, "Initialized tick in %s/\n", tickDir)
```
No `filepath.Abs` call. If `Cwd` is relative, the output would be relative too.

#### Init cleanup on failure

**V1** cleans up `.tick/` if `tasks.jsonl` creation fails:
```go
if err := os.WriteFile(jsonlPath, []byte(""), 0644); err != nil {
    os.RemoveAll(tickDir)
    return fmt.Errorf("Could not create tasks.jsonl: %w", err)
}
```

**V2** and **V3** do not clean up on `tasks.jsonl` failure. They leave the `.tick/` directory behind.

#### Usage message

**V1** lists all future commands (create, list, show, start, done, cancel, reopen, update, dep, ready, blocked, stats, rebuild) - 14 commands total.

**V2** and **V3** only list the `init` command that is actually implemented, plus global flags documentation.

### Code Quality

#### Go Idioms and Type Safety

**V2** has the strongest type safety with typed `OutputFormat` constants:
```go
type OutputFormat string
const (
    FormatTOON   OutputFormat = "toon"
    FormatPretty OutputFormat = "pretty"
    FormatJSON   OutputFormat = "json"
)
```
This prevents typos and provides IDE autocompletion.

**V3** uses plain strings (`"toon"`, `"pretty"`, `"json"`) which are more error-prone.

**V1** uses separate booleans (`Toon bool`, `Pretty bool`, `JSON bool`) which allows mutually exclusive flags to be simultaneously true - a bug waiting to happen.

#### Naming

**V1**: `GlobalOpts`, `cmdInit`, `FindTickDir` - inconsistent prefix style.

**V2**: `Config`, `runInit`, `DiscoverTickDir` - consistent naming. The `run` prefix for command handlers is idiomatic.

**V3**: `GlobalFlags`, `runInit`, `DiscoverTickDir`, `ParseGlobalFlags`, `IsTTY`, `DefaultOutputFormat` - most exported functions. `ParseGlobalFlags` being exported allows testing in isolation.

#### DRY

**V3** has the most repetitive test code - every test creates:
```go
var stdout, stderr bytes.Buffer
app := &App{
    Stdout: &stdout,
    Stderr: &stderr,
    Cwd:    dir,
}
```
This is repeated 12+ times. A helper would reduce boilerplate.

**V2** tests also repeat `app := NewApp(); app.workDir = t.TempDir()` but it's more concise.

**V1** tests repeat `var stdout, stderr bytes.Buffer; app := NewApp(&stdout, &stderr)` which is medium verbosity.

#### Error Handling

**V3** has the most thorough `os.Stat` error handling in `runInit`:
```go
if _, err := os.Stat(tickDir); err == nil {
    fmt.Fprintf(a.Stderr, "Error: Tick already initialized in this directory\n")
    return 1
} else if !os.IsNotExist(err) {
    fmt.Fprintf(a.Stderr, "Error: Could not check .tick/ directory: %v\n", err)
    return 1
}
```
The `else if !os.IsNotExist(err)` branch handles the case where `os.Stat` fails for a reason other than "not exists" (e.g., permissions). **V1** and **V2** don't handle this case.

#### File creation method

**V1** uses `os.WriteFile(jsonlPath, []byte(""), 0644)` - passing an empty string as bytes.
**V2** uses `os.WriteFile(jsonlPath, []byte{}, 0644)` - passing an empty byte slice.
**V3** uses `os.OpenFile(jsonlPath, os.O_CREATE|os.O_WRONLY, 0644)` then `f.Close()` - more explicit but more verbose.

All are functionally equivalent.

### Test Quality

#### V1 Test Functions (internal/cli/cli_test.go - 194 lines)

1. `TestInitCommand/"creates .tick directory in target directory"` - verifies directory creation + IsDir
2. `TestInitCommand/"creates empty tasks.jsonl inside .tick"` - verifies file exists + size 0
3. `TestInitCommand/"does not create cache.db at init time"` - verifies cache.db absent
4. `TestInitCommand/"prints confirmation with absolute path"` - exact string match
5. `TestInitCommand/"prints nothing with --quiet flag"` - verifies stdout empty
6. `TestInitCommand/"errors when .tick already exists"` - checks exit code 1 + error message contains "Tick already initialized"
7. `TestInitCommand/"writes error messages to stderr not stdout"` - verifies stdout empty, stderr non-empty
8. `TestFindTickDir/"discovers .tick directory by walking up from cwd"` - nested dir walk-up
9. `TestFindTickDir/"errors when no .tick directory found"` - error on missing
10. `TestUnknownSubcommand/"routes unknown subcommands to error"` - exit code 1 + "Unknown command"
11. `TestGlobalFlags/"parses --quiet flag"` - exit code 0 only
12. `TestGlobalFlags/"parses -q short form"` - exit code 0 only

**V1 total: 12 test cases across 4 top-level functions**

#### V2 Test Functions

**internal/cli/app_test.go (173 lines):**
1. `TestAppRouting/"it routes unknown subcommands to error"` - exact error message match
2. `TestAppRouting/"it prints basic usage with no subcommand"` - checks "Usage:" and "init"
3. `TestAppRouting/"it parses --quiet global flag"` - asserts `config.Quiet == true`
4. `TestAppRouting/"it parses -q global flag"` - asserts `config.Quiet == true`
5. `TestAppRouting/"it parses --verbose global flag"` - asserts `config.Verbose == true`
6. `TestAppRouting/"it parses -v global flag"` - asserts `config.Verbose == true`
7. `TestAppRouting/"it parses --toon global flag"` - asserts `OutputFormat == FormatTOON`
8. `TestAppRouting/"it parses --pretty global flag"` - asserts `OutputFormat == FormatPretty`
9. `TestAppRouting/"it parses --json global flag"` - asserts `OutputFormat == FormatJSON`
10. `TestTTYDetection/"it detects TTY vs non-TTY on stdout"` - verifies `detectTTY() == false` in pipe
11. `TestTTYDetection/"it defaults to TOON format when not a TTY"` - verifies `OutputFormat == FormatTOON`

**internal/cli/discover_test.go (95 lines):**
12. `TestDiscoverTickDir/"it discovers .tick/ directory by walking up from cwd"` - nested walk-up
13. `TestDiscoverTickDir/"it finds .tick/ in the starting directory itself"` - same-dir discovery
14. `TestDiscoverTickDir/"it errors when no .tick/ directory found"` - exact error message
15. `TestDiscoverTickDir/"it stops at the first .tick/ match walking up"` - two `.tick/` dirs at different levels, verifies nearest wins

**internal/cli/init_test.go (201 lines):**
16. `TestInitCommand/"it creates .tick/ directory in current working directory"` - stat + IsDir
17. `TestInitCommand/"it creates empty tasks.jsonl inside .tick/"` - stat + size 0
18. `TestInitCommand/"it does not create cache.db at init time"` - os.IsNotExist check
19. `TestInitCommand/"it prints confirmation with absolute path on success"` - exact string match
20. `TestInitCommand/"it prints nothing with --quiet flag on success"` - stdout empty
21. `TestInitCommand/"it prints nothing with -q flag on success"` - stdout empty (short flag)
22. `TestInitCommand/"it errors when .tick/ already exists"` - error message contains "already initialized"
23. `TestInitCommand/"it errors when .tick/ already exists even if corrupted"` - no tasks.jsonl inside .tick
24. `TestInitCommand/"it writes error messages to stderr, not stdout"` - stdout empty on error
25. `TestInitCommand/"it surfaces OS error when directory is not writable"` - read-only dir, checks error message

**cmd/tick/main_test.go (168 lines):**
26. `TestMainIntegration/"it returns exit code 1 when .tick/ already exists"` - binary execution, exit code check
27. `TestMainIntegration/"it returns exit code 0 on successful init"` - binary execution
28. `TestMainIntegration/"it writes error messages to stderr, not stdout"` - binary with separate stdout/stderr
29. `TestMainIntegration/"it returns exit code 1 for unknown subcommand"` - binary, exact error message
30. `TestMainIntegration/"it returns exit code 0 with no subcommand (prints usage)"` - binary, usage output
31. `TestMainIntegration/"it prefixes all error messages with Error:"` - binary, stderr prefix check

**V2 total: 31 test cases across 6 top-level functions in 4 test files**

#### V3 Test Functions (internal/cli/cli_test.go - 475 lines)

1. `TestInitCommand/"it creates .tick/ directory in current working directory"` - stat + IsDir
2. `TestInitCommand/"it creates empty tasks.jsonl inside .tick/"` - ReadFile + len check
3. `TestInitCommand/"it does not create cache.db at init time"` - os.IsNotExist check
4. `TestInitCommand/"it prints confirmation with absolute path on success"` - exact string match
5. `TestInitCommand/"it prints nothing with --quiet flag on success"` - stdout empty (flag after subcommand)
6. `TestInitCommand/"it prints nothing with -q flag on success"` - stdout empty (flag before subcommand)
7. `TestInitCommand/"it errors when .tick/ already exists"` - exit code 1 + Error prefix + "already initialized"
8. `TestInitCommand/"it returns exit code 1 when .tick/ already exists"` - exit code 1 specifically
9. `TestInitCommand/"it writes error messages to stderr, not stdout"` - stdout empty, stderr non-empty
10. `TestInitCommand/"it errors when cannot create .tick directory (unwritable parent)"` - read-only dir, exit 1 + error message
11. `TestDiscoverTickDir/"it discovers .tick/ directory by walking up from cwd"` - nested walk-up
12. `TestDiscoverTickDir/"it finds .tick/ in current directory"` - same-dir discovery
13. `TestDiscoverTickDir/"it errors when no .tick/ directory found"` - exact error message
14. `TestUnknownSubcommand/"it routes unknown subcommands to error"` - exact error string match
15. `TestNoSubcommand/"it prints usage when no subcommand provided"` - exit 0 + "Usage:" + "init"
16. `TestTTYDetection/"it detects TTY vs non-TTY on stdout"` - bytes.Buffer is not TTY
17. `TestTTYDetection/"it sets default output format based on TTY detection"` - non-TTY defaults to "toon"
18. `TestGlobalFlags/"it parses --quiet flag"` - flag value + remaining args check
19. `TestGlobalFlags/"it parses -q flag"` - flag value + remaining args check
20. `TestGlobalFlags/"it parses --verbose flag"` - flag value + remaining args check
21. `TestGlobalFlags/"it parses -v flag"` - flag value + remaining args check
22. `TestGlobalFlags/"it parses --toon flag"` - OutputFormat value
23. `TestGlobalFlags/"it parses --pretty flag"` - OutputFormat value
24. `TestGlobalFlags/"it parses --json flag"` - OutputFormat value
25. `TestGlobalFlags/"flags can appear after subcommand"` - positional flexibility
26. `TestGlobalFlags/"multiple flags can be combined"` - `-q -v --toon` all parsed

**V3 total: 26 test cases across 6 top-level functions in 1 test file**

#### Test Coverage Comparison

**Tests unique to V1:**
- None (V1 has the minimal subset)

**Tests unique to V2:**
- Corrupted `.tick/` (missing `tasks.jsonl`) still counts as initialized
- Unwritable directory error message
- `DiscoverTickDir` stops at first (nearest) `.tick/` match
- `-q` short flag tested separately for init
- Full integration tests building the actual binary (6 tests verifying real exit codes, stderr separation)

**Tests unique to V3:**
- `ParseGlobalFlags` tested in isolation (not via `Run()`)
- Flags after subcommand (`"tick", "init", "--quiet"`)
- Multiple flags combined (`-q -v --toon`)
- `DiscoverTickDir` finds `.tick/` in current directory (also in V2)
- Unwritable directory error message

**Tests in all 3:**
- Creates `.tick/` directory
- Creates empty `tasks.jsonl`
- No `cache.db` at init
- Confirmation with absolute path
- `--quiet` suppresses output
- Error when `.tick/` already exists
- Error messages to stderr not stdout
- Walk-up discovery from nested dir
- Error when no `.tick/` found
- Unknown subcommand error
- `--quiet` flag parsing
- `-q` flag parsing

**Tests missing from V1 but present in V2/V3:**
- `--verbose`/`-v` flag parsing (V1 only tests `--quiet`/`-q`)
- `--toon`/`--pretty`/`--json` flag parsing
- TTY detection
- No subcommand prints usage (V1 doesn't test this)
- Unwritable directory handling
- `.tick/` found in current dir (not just walk-up)

**Tests missing from all versions:**
- `os.Mkdir` with mode 0755 verification (permissions test)
- `tasks.jsonl` with mode 0644 verification (permissions test)
- Symlinked `.tick/` directory behavior

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed (Go only) | 3 | 8 | 3 |
| Lines added (Go only) | 381 | 857 | 674 |
| Impl LOC (non-test Go) | 187 (18 + 169) | 220 (17 + 130 + 32 + 41) | 199 (25 + 174) |
| Test LOC | 194 | 637 (168 + 173 + 95 + 201) | 475 |
| Test functions | 12 | 31 | 26 |

## Verdict

**V3 is the best implementation overall**, with V2 as a close second.

**Rationale:**

1. **Completeness**: V3 implements all acceptance criteria including TTY detection (which V1 entirely omits). V2 also implements TTY detection but V3's `IsTTY(io.Writer)` is more testable than V2's `detectTTY()` which is hardcoded to `os.Stdout`.

2. **Error handling**: V3 has the most thorough `os.Stat` error handling with the `else if !os.IsNotExist(err)` branch in `runInit()`. Neither V1 nor V2 handles the case where `os.Stat` fails for reasons other than "not exists".

3. **API design**: V3's `Run()` returning `int` (exit code) is the most direct mapping to CLI semantics. V2's `Run()` returning `error` requires `main.go` to add the `Error: ` prefix, creating a two-location responsibility for error formatting. V3's exported `ParseGlobalFlags` is independently testable.

4. **Flag flexibility**: V3's `ParseGlobalFlags` strips flags from anywhere in the args list, allowing `tick init --quiet` and `tick --quiet init` to both work. V2's `parseGlobalFlags` stops at the first non-flag, so flags must precede the subcommand. V3 tests this explicitly with `"flags can appear after subcommand"`.

5. **Test quality**: V3 has 26 test cases covering TTY detection, all flag variants (including `--toon`, `--pretty`, `--json`), multiple combined flags, and flags-after-subcommand. V2 has more test cases (31) but this is partly because it has integration tests that build the binary -- valuable but slow. V1 has only 12 tests and skips most flag and TTY tests.

**V3's weaknesses:**
- No `filepath.Abs` call in `runInit` -- relies on `Cwd` being absolute, which is fragile.
- No cleanup of `.tick/` on `tasks.jsonl` creation failure (V1 has this).
- `OutputFormat` as a plain string rather than a typed constant (V2 is stronger here).
- No integration tests (V2 uniquely has these).
- Committed a compiled `tick` binary in the diff (the `tick` file at 2.4MB).
- `DiscoverTickDir` doesn't normalize with `filepath.Abs` unlike V1 and V2.

**V2's unique strengths:**
- Best file organization (separate files per concern).
- Typed `OutputFormat` constants.
- Integration tests building the real binary verify actual exit codes and stderr separation.
- Corrupted `.tick/` test case.
- `DiscoverTickDir` "stops at first match" test with multiple `.tick/` dirs.
- Normalizes paths with `filepath.Abs` in both `runInit` and `DiscoverTickDir`.

**V1's weaknesses:**
- Missing TTY detection entirely (acceptance criteria failure).
- Fewest tests (12).
- No `--verbose`, `--toon`, `--pretty`, `--json` flag tests.
- Separate booleans for format flags (no unified format concept).
- Usage message lists unimplemented commands (premature).

If forced to pick one implementation to merge as-is, **V3** provides the most complete and testable implementation despite its minor weaknesses. However, the ideal implementation would combine V3's TTY detection approach and flag flexibility with V2's typed `OutputFormat`, `filepath.Abs` normalization, integration tests, and file organization.
