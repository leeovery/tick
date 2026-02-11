# Task tick-core-1-5: CLI framework & tick init

## Task Summary

This task introduces the CLI entry point (`cmd/tick/main.go`), subcommand dispatch, global flag parsing (`--quiet`, `--verbose`, `--toon`, `--pretty`, `--json`), TTY detection, error handling conventions (all errors to stderr with `Error: ` prefix, exit code 0/1), the `tick init` command (creates `.tick/` with empty `tasks.jsonl`), and a `.tick/` directory discovery helper that walks up from cwd.

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|---|---|---|
| `tick init` creates `.tick/` with empty `tasks.jsonl` | PASS | PASS |
| `tick init` does not create `cache.db` | PASS | PASS |
| `tick init` prints confirmation with absolute path | PASS | PASS |
| `tick init` with `--quiet` produces no output on success | PASS | PASS |
| `tick init` when `.tick/` exists returns error to stderr with exit code 1 | PASS | PASS |
| All errors written to stderr with `Error: ` prefix | PASS | PASS |
| Exit code 0 for success, 1 for errors | PASS | PASS |
| Global flags parsed: `--quiet`, `--verbose`, `--toon`, `--pretty`, `--json` | PASS | PASS |
| TTY detection on stdout selects default output format | PASS | PASS |
| `.tick/` directory discovery walks up from cwd | PASS | PASS |
| Unknown subcommands return error with exit code 1 | PASS | PASS |

Both versions satisfy all 11 acceptance criteria.

## Implementation Comparison

### Approach

**V5: Functional package-level `Run()` with `Context` struct**

V5 uses a package-level function `cli.Run()` that accepts all dependencies as parameters:

```go
// internal/cli/cli.go (V5)
func Run(args []string, workDir string, stdout, stderr io.Writer, isTTY bool) int {
    ctx, subcmd, err := parseArgs(args, workDir, stdout, stderr, isTTY)
    // ...
}
```

`main.go` does TTY detection and cwd resolution before calling `Run`:

```go
// cmd/tick/main.go (V5)
func main() {
    isTTY := isTerminal(os.Stdout)
    cwd, err := os.Getwd()
    if err != nil {
        os.Stderr.WriteString("Error: " + err.Error() + "\n")
        os.Exit(1)
    }
    code := cli.Run(os.Args, cwd, os.Stdout, os.Stderr, isTTY)
    os.Exit(code)
}
```

Command dispatch uses a `map[string]func(*Context) error`:

```go
// internal/cli/cli.go (V5)
var commands = map[string]func(*Context) error{
    "init": runInit,
}
```

The `Context` struct carries all parsed state:

```go
type Context struct {
    WorkDir string
    Stdout  io.Writer
    Stderr  io.Writer
    Quiet   bool
    Verbose bool
    Format  OutputFormat
    Args    []string
}
```

**V6: Struct-based `App` with method receiver**

V6 uses an `App` struct with an exported `Run` method:

```go
// internal/cli/app.go (V6)
type App struct {
    Stdout io.Writer
    Stderr io.Writer
    Getwd  func() (string, error)
    IsTTY  bool
}

func (a *App) Run(args []string) int {
    flags, subcmd, subArgs := parseArgs(args[1:])
    // ...
}
```

`main.go` constructs `App` with injected dependencies:

```go
// cmd/tick/main.go (V6)
func main() {
    app := &cli.App{
        Stdout: os.Stdout,
        Stderr: os.Stderr,
        Getwd:  os.Getwd,
        IsTTY:  cli.IsTerminal(os.Stdout),
    }
    os.Exit(app.Run(os.Args))
}
```

Command dispatch uses a `switch` statement, and `Getwd` is lazy (called at dispatch time, not upfront):

```go
// internal/cli/app.go (V6)
switch subcmd {
case "init":
    err = a.handleInit(flags, subArgs)
default:
    fmt.Fprintf(a.Stderr, "Error: Unknown command '%s'. Run 'tick help' for usage.\n", subcmd)
    return 1
}
```

The init handler is a separate exported function `RunInit()` that takes primitive args:

```go
// internal/cli/init.go (V6)
func RunInit(dir string, quiet bool, stdout io.Writer) error {
```

**Key architectural difference**: V5 resolves cwd eagerly in `main()` and passes it as a string. V6 injects `Getwd` as a function, deferring resolution until the handler needs it. This makes V6 more testable (can mock `Getwd` failure) but slightly more indirect.

### Code Quality

**Go idioms**

V5 uses a pure-functional approach (package-level function, no receiver). This is idiomatic for simple CLIs. The `Context` pattern is a well-known Go idiom.

V6 uses an object-oriented approach (struct with method receiver). Also idiomatic Go. The `App` struct pattern is common in larger CLIs (e.g., Kong, Cobra internal patterns).

**Naming**

| Aspect | V5 | V6 |
|---|---|---|
| Main file | `cli.go` | `app.go` |
| Entry function | `Run()` (package-level) | `App.Run()` (method) |
| Init handler | `runInit` (unexported) | `RunInit` (exported) + `handleInit` (unexported bridge) |
| Format constants | `FormatToon`, `FormatPretty`, `FormatJSON` | `FormatHuman`, `FormatTOON`, `FormatJSON` |
| Error message | `"Tick already initialized in this directory"` | `"tick already initialized in %s"` (includes abs path) |
| Discovery error | `"Not a tick project ..."` (capital N) | `"not a tick project ..."` (lowercase) |

V6's error casing is more Go-idiomatic (errors should not be capitalized per `go vet` / Effective Go). V5 uses `"Not a tick project"` with a capital N and `"Tick already initialized"` with capital T -- both violate Go error conventions.

V6's `FormatHuman` is arguably better than V5's `FormatPretty` for self-documentation, and `FormatTOON` (all-caps) is more correct as an acronym.

V6 includes the absolute path in the "already initialized" error (`"tick already initialized in %s"`), which is more informative than V5's generic `"Tick already initialized in this directory"`.

**Error handling**

V5 `init.go` error for "not writable":
```go
return fmt.Errorf("creating .tick directory: %w", err)
```

V6 `init.go` error for "not writable":
```go
return fmt.Errorf("could not create .tick/ directory: %w", err)
```

The spec says: `"Could not create .tick/ directory: <os error>"`. V6 matches the spec error prefix exactly. V5 does not.

V5 `discover.go` uses `fmt.Errorf` for the not-found error:
```go
return "", fmt.Errorf("Not a tick project (no .tick directory found)")
```

V6 uses `errors.New` (more appropriate since there is no wrapping):
```go
return "", errors.New("not a tick project (no .tick directory found)")
```

V6 is correct here -- `errors.New` is preferred when not wrapping.

**Unknown flag handling**

V5 returns an error for unknown flags before the subcommand:
```go
case strings.HasPrefix(arg, "-"):
    return nil, "", fmt.Errorf("unknown flag '%s'", arg)
```

V6 silently skips unknown flags before the subcommand:
```go
if strings.HasPrefix(arg, "-") {
    // Unknown flag before subcommand -- skip
    continue
}
```

The spec says nothing about unknown *flags* (only unknown *subcommands*). V6's silent skip is arguably more flexible but could mask typos. V5's strict rejection is safer.

**Global flags after subcommand**

V5 does NOT parse global flags after the subcommand -- once `foundCmd` is true, all remaining args go into `cmdArgs`:
```go
for _, arg := range remaining {
    if foundCmd {
        cmdArgs = append(cmdArgs, arg)
        continue
    }
    // ...
}
```

V6 continues extracting global flags even after the subcommand:
```go
for _, arg := range args {
    if applyGlobalFlag(&flags, arg) {
        continue
    }
    // ...
}
```

This means `tick init --quiet` works in V6 but NOT in V5 (the `--quiet` would be passed as a subcommand arg). V6 explicitly tests this behavior.

**DRY / Separation**

V6 separates flag application into `applyGlobalFlag()`:
```go
func applyGlobalFlag(flags *globalFlags, arg string) bool {
    switch arg {
    case "--quiet", "-q":
        flags.quiet = true
    // ...
    }
}
```

V5 inlines all flag parsing in `parseArgs`. V6 is slightly more modular.

V6 also separates `RunInit` as an exported function (testable independently), while V5's `runInit` is unexported and tightly coupled to `*Context`.

**Type safety**

V5 uses an exported `Context` struct with exported fields -- any code in the package can modify it. V6 uses an unexported `globalFlags` struct, keeping flag state internal.

V5's `OutputFormat` is resolved eagerly during arg parsing and stored in `Context.Format`. V6 stores raw flag booleans and provides `ResolveFormat()` to derive the format lazily, which is cleaner for testing.

**Usage output**

V5 prints a comprehensive usage listing all future commands (create, update, start, done, cancel, reopen, list, show, ready, blocked, dep, stats, doctor, rebuild). V6 prints only the currently implemented command (`init`). V6 is more conservative and avoids advertising unimplemented features.

### Test Quality

**V5 Test Functions and Subtests** (`internal/cli/cli_test.go`, 333 lines):

1. `TestInit`
   - `"it creates .tick/ directory in current working directory"` (line 12)
   - `"it creates empty tasks.jsonl inside .tick/"` (line 28)
   - `"it does not create cache.db at init time"` (line 43)
   - `"it prints confirmation with absolute path on success"` (line 56)
   - `"it prints nothing with --quiet flag on success"` (line 73)
   - `"it errors when .tick/ already exists"` (line 87)
   - `"it returns exit code 1 when .tick/ already exists"` (line 104)
   - `"it writes error messages to stderr, not stdout"` (line 117)
   - `"it errors even when .tick/ exists but is corrupted (missing tasks.jsonl)"` (line 134)
   - `"it accepts -q shorthand for --quiet"` (line 149)

2. `TestDiscoverTickDir`
   - `"it discovers .tick/ directory by walking up from cwd"` (line 165)
   - `"it errors when no .tick/ directory found (not a tick project)"` (line 186)
   - `"it finds .tick/ in the starting directory itself"` (line 199)

3. `TestSubcommandRouting`
   - `"it routes unknown subcommands to error"` (line 215)
   - `"it prints basic usage with no subcommand and exits 0"` (line 231)

4. `TestTTYDetection`
   - `"it detects TTY vs non-TTY on stdout"` (line 247) -- minimal; just checks both paths don't crash

5. `TestGlobalFlags`
   - `"it parses --verbose flag"` (line 269)
   - `"it parses -v shorthand for --verbose"` (line 281)
   - `"it parses --toon flag"` (line 293)
   - `"it parses --pretty flag"` (line 305)
   - `"it parses --json flag"` (line 317)

**Total V5: 5 top-level test functions, 20 subtests.**

---

**V6 Test Functions and Subtests** (`internal/cli/cli_test.go`, 487 lines):

1. `TestInit`
   - `"it creates .tick/ directory in current working directory"` (line 12)
   - `"it creates empty tasks.jsonl inside .tick/"` (line 28)
   - `"it does not create cache.db at init time"` (line 49)
   - `"it prints confirmation with absolute path on success"` (line 67)
   - `"it prints nothing with --quiet flag on success"` (line 82)
   - `"it errors when .tick/ already exists"` (line 95)
   - `"it returns exit code 1 when .tick/ already exists"` (line 113)
   - `"it writes error messages to stderr, not stdout"` (line 131)

2. `TestDispatch`
   - `"it routes unknown subcommands to error"` (line 158)
   - `"it prints usage with exit code 0 when no subcommand given"` (line 177)
   - `"it passes --quiet flag to init command via dispatch"` (line 196)
   - `"it passes -q short flag to init command via dispatch"` (line 213)
   - `"it accepts global flags after subcommand"` (line 230)

3. `TestParseArgs`
   - `"it parses all global flags"` (table-driven, line 251)
     - Sub-cases: `"--quiet"`, `"-q short form"`, `"--verbose"`, `"-v short form"`, `"--toon"`, `"--pretty"`, `"--json"` (7 sub-sub-tests)

4. `TestParseArgsGlobalFlagsAfterSubcommand`
   - `"it extracts global flags after the subcommand"` (line 333)
   - `"it extracts global flags from both before and after the subcommand"` (line 346)
   - `"it keeps non-global args in subArgs"` (line 362)

5. `TestDiscoverTickDir`
   - `"it discovers .tick/ directory by walking up from cwd"` (line 379)
   - `"it errors when no .tick/ directory found (not a tick project)"` (line 403)

6. `TestTTYDetection`
   - `"it detects TTY vs non-TTY on stdout"` (line 416) -- uses `os.Pipe()` to verify non-TTY
   - `"it defaults to TOON when not TTY"` (line 428)
   - `"it defaults to human-readable when TTY"` (line 435)
   - `"it overrides with --toon flag"` (line 442)
   - `"it overrides with --pretty flag"` (line 449)
   - `"it overrides with --json flag"` (line 456)

**Total V6: 6 top-level test functions, 25 subtests (32 including table-driven sub-cases).**

---

**Comparison**

| Aspect | V5 | V6 |
|---|---|---|
| Total subtests | 20 | 25 (32 with table sub-cases) |
| Table-driven tests | 0 | 1 (`TestParseArgs`) |
| Tests `RunInit` directly | No (always via `Run`) | Yes (most init tests call `RunInit` directly) |
| Tests `parseArgs` directly | No | Yes (`TestParseArgs`, `TestParseArgsGlobalFlagsAfterSubcommand`) |
| Tests TTY format resolution | No (just smoke test) | Yes (6 subtests for `ResolveFormat`) |
| Tests global flags after subcmd | No | Yes (3 subtests) |
| Tests corrupted .tick/ | Yes | No |
| Tests "finds .tick/ in starting dir itself" | Yes | No |
| Tests -q shorthand for --quiet | Yes (in TestInit) | Yes (in TestDispatch) |
| Error message content assertions | Partial (checks `Error: ` prefix) | Exact (checks full error string) |
| TTY detection with actual pipe | No | Yes (`os.Pipe()`) |

V5 has the unique `"it errors even when .tick/ exists but is corrupted"` and `"it finds .tick/ in the starting directory itself"` subtests. V6 misses these edge cases.

V6 has significantly stronger coverage of flag parsing (table-driven `TestParseArgs`, global-flags-after-subcommand tests), TTY format resolution (`ResolveFormat` tests with each flag override), and dispatch integration (tests `--quiet` and `-q` through the full `App.Run` path separately).

V6 tests `RunInit` as a standalone function (better unit isolation) whereas V5 always routes through the full `Run()` pipeline.

V6's TTY test creates an actual `os.Pipe()` to verify non-TTY detection. V5's TTY test only calls `Run()` with `isTTY=false` and `isTTY=true` and checks that neither crashes -- it never actually tests format selection.

V6 makes exact assertions on error message content in most tests. V5's `"it writes error messages to stderr"` test only checks `strings.HasPrefix(stderr.String(), "Error: ")` without verifying the full message.

### Skill Compliance

| Constraint | V5 | V6 |
|---|---|---|
| Handle all errors explicitly (no naked returns) | PASS | PASS |
| Write table-driven tests with subtests | FAIL -- no table-driven tests | PASS -- `TestParseArgs` uses table-driven |
| Document all exported functions, types, and packages | PASS | PASS |
| Propagate errors with `fmt.Errorf("%w", err)` | PASS | PASS |
| Error messages not capitalized (Go convention) | FAIL -- `"Not a tick project"`, `"Tick already initialized"` | PASS -- lowercase errors |
| Use `errors.New` when not wrapping | FAIL -- uses `fmt.Errorf` without `%w` in `discover.go` | PASS -- uses `errors.New` in `discover.go` |
| No panic for normal error handling | PASS | PASS |
| No hardcoded configuration | PASS | PASS |

### Spec-vs-Convention Conflicts

1. **Spec error message casing**: The spec says `"Error: Not a tick project (no .tick directory found)"`. V5 matches this literally with a capital "N" in the error value. V6 uses lowercase `"not a tick project"`, which follows Go convention but technically differs from the spec's exact wording. However, since the `Error: ` prefix is added at the dispatch level (both versions prepend `"Error: "` via `fmt.Fprintf`), the difference is in the error *value* -- Go conventions say error strings should not be capitalized. V6 prioritizes Go convention over spec literal text.

2. **Spec error for already initialized**: Spec says `"Error: Tick already initialized..."` is implied but not specified verbatim. V5 says `"Tick already initialized in this directory"` (generic). V6 says `"tick already initialized in <abs-path>"` (specific, lowercase). V6 is more useful and more Go-idiomatic.

3. **Spec error for creation failure**: Spec says `"Error: Could not create .tick/ directory: <os error>"`. V5 says `"creating .tick directory: <os error>"` -- does NOT match spec. V6 says `"could not create .tick/ directory: <os error>"` -- matches spec (modulo Go-convention lowercase).

4. **Unknown flags**: Spec does not mention unknown flag handling. V5 errors on unknown flags; V6 silently skips them. Neither is wrong per spec.

5. **Global flags after subcommand**: Spec does not specify flag position behavior. V6 allows global flags after the subcommand (like `git`); V5 does not. V6's behavior is more user-friendly.

## Diff Stats

| Metric | V5 | V6 |
|---|---|---|
| Files changed (code only) | 4 | 4 |
| Implementation lines added | 251 (main 28 + cli 150 + init 40 + discover 33) | 246 (main 18 + app 157 + init 40 + discover 31) |
| Test lines added | 333 | 487 |
| Total lines added | 584 | 733 |
| Top-level test functions | 5 | 6 |
| Subtests | 20 | 25 (32 with table sub-cases) |
| Table-driven test groups | 0 | 1 |
| Extra files | -- | `.gitignore` (adds `/tick` binary ignore) |

## Verdict

**V6 is the stronger implementation.** While both versions satisfy all 11 acceptance criteria, V6 is superior in several dimensions:

1. **Architecture**: V6's `App` struct with injected `Getwd` is more testable and extensible than V5's flat parameter list. The exported `RunInit` enables isolated unit testing.

2. **Go idioms**: V6 follows Go error conventions (lowercase errors, `errors.New` for non-wrapping), uses table-driven tests, and keeps internal state unexported (`globalFlags`). V5 violates Go error casing conventions.

3. **Spec compliance**: V6 matches the spec's error message for creation failure (`"could not create .tick/ directory: ..."`) while V5 does not. V6 includes the absolute path in the already-initialized error.

4. **Test coverage**: V6 has 60% more test lines (487 vs 333), more subtests (25 vs 20), table-driven flag parsing tests, direct unit tests of `parseArgs` and `ResolveFormat`, actual `os.Pipe()` TTY verification, and tests for global flags after the subcommand. V5's TTY test is effectively a smoke test that verifies nothing about format selection.

5. **Flag handling**: V6 supports global flags after the subcommand (like `git`), which is a better UX. V6 also separates `applyGlobalFlag` for reusability.

V5's two unique edge-case tests (corrupted `.tick/` and "finds .tick/ in starting directory") are notable but do not outweigh V6's systematic advantages. V6 also thoughtfully avoids advertising unimplemented commands in its usage output.

The only area where V5 could be considered preferable is its strict unknown-flag handling (errors instead of silently skipping), which prevents user typos from going unnoticed. However, this is a design choice, not a deficiency.
