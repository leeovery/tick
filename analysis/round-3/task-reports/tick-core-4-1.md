# Task 4-1: Formatter abstraction & TTY-based format selection

## Task Plan Summary

This task establishes the formatting foundation for the tick CLI. It requires:

1. **`Formatter` interface** with six methods: `FormatTaskList`, `FormatTaskDetail`, `FormatTransition`, `FormatDepChange`, `FormatStats`, `FormatMessage`
2. **`Format` enum** with three constants: `FormatToon`, `FormatPretty`, `FormatJSON`
3. **`DetectTTY()`** using `os.Stdout.Stat()` to check `ModeCharDevice`; stat failure defaults to non-TTY
4. **`ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY)`** that errors on >1 flag, returns the flag's format if exactly one is set, or auto-detects (TTY=Pretty, non-TTY=Toon)
5. **`FormatConfig` struct** with Format, Quiet, Verbose -- passed to all handlers
6. **Stub formatter** as placeholder for tasks 4-2 through 4-4
7. **`--verbose`** to stderr only
8. Six specific test cases covering TTY detection, defaults, flag overrides, conflicts, FormatConfig propagation, and stat failure handling

## V4 Implementation

### Architecture & Design

V4 uses a struct-based `App` architecture where the top-level `App` struct holds all state:

```go
// internal/cli/cli.go
type App struct {
    Stdout       io.Writer
    Stderr       io.Writer
    Dir          string
    Quiet        bool
    Verbose      bool
    OutputFormat Format
    IsTTY        bool
    FormatCfg    FormatConfig  // added in this task
}
```

**Format enum** is defined as `type Format int` with `iota` constants in `format.go`:
```go
type Format int
const (
    FormatToon Format = iota
    FormatPretty
    FormatJSON
)
```

**DetectTTY** accepts `io.Writer` and performs a type assertion to `*os.File`:
```go
func DetectTTY(w io.Writer) bool {
    f, ok := w.(*os.File)
    if !ok {
        return false
    }
    info, err := f.Stat()
    if err != nil {
        return false
    }
    return (info.Mode() & os.ModeCharDevice) != 0
}
```

**FormatConfig wiring** is done in `App.Run()` after flag parsing:
```go
a.FormatCfg = FormatConfig{
    Format:  a.OutputFormat,
    Quiet:   a.Quiet,
    Verbose: a.Verbose,
}
```

**Formatter interface** uses concrete types directly in method signatures:
```go
type Formatter interface {
    FormatTaskList(w io.Writer, rows []listRow, quiet bool) error
    FormatTaskDetail(w io.Writer, detail TaskDetail) error
    FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
    FormatDepChange(w io.Writer, taskID string, blockedByID string, action string, quiet bool) error
    FormatStats(w io.Writer, stats StatsData) error
    FormatMessage(w io.Writer, msg string) error
}
```

The interface uses inline parameters (e.g., `id string, oldStatus string, newStatus string`) rather than data objects. `FormatMessage` returns `error`.

**StubFormatter** returns nil for all methods including `FormatMessage`:
```go
func (f *StubFormatter) FormatMessage(w io.Writer, msg string) error {
    return nil
}
```

**Key refactoring**: The task removed the old `OutputFormat string` type (`FormatTOON = "toon"`) and the `detectTTY()` method from `App`, replacing them with the new `Format int` enum and the extracted `DetectTTY()` function. It also rewired `parseGlobalFlags` to collect boolean flags and defer to `ResolveFormat`.

**Data types**: V4 introduces `TaskDetail` and `StatsData` as empty placeholder structs with TODO comments:
```go
type TaskDetail struct{} // TODO: Fields will be added...
type StatsData struct{}  // TODO: Fields will be added...
```

### Code Quality

- Clean separation: all format-related code in `format.go`
- `DetectTTY` accepts `io.Writer` -- more flexible but relies on type assertion, which is a less explicit contract
- `FormatConfig` is stored directly on `App` as a public field (`FormatCfg`), coupling format config to the App struct
- `ResolveFormat` uses sequential if/else-if for flag counting and resolution rather than switch
- Naming: `FormatToon` (changed from `FormatTOON`) follows Go conventions
- The `FormatMessage` stub returning nil (doing nothing) means `init` output breaks -- running `tick init` with this commit produces no confirmation message

### Test Coverage

**format_test.go** (193 lines) contains:

1. `TestDetectTTY` -- 3 subtests:
   - Non-TTY for `bytes.Buffer` (non-`*os.File`)
   - Non-TTY for pipe `*os.File`
   - Non-TTY on stat failure (closed file descriptor)

2. `TestResolveFormat` -- 6 subtests:
   - Default Toon for non-TTY
   - Default Pretty for TTY
   - `--toon` flag override (with TTY=true, verifying override)
   - `--pretty` flag override (with non-TTY, verifying override)
   - `--json` flag override
   - Conflicting flags (table-driven with 4 sub-subtests: toon+pretty, toon+json, pretty+json, all three)

3. `TestFormatter` -- 1 subtest: stub implements interface and all methods callable without panic

4. `TestCLI_ConflictingFormatFlags` -- integration test via `App.Run`

5. `TestCLI_FormatConfigWired` -- verifies FormatConfig fields after `App.Run` with `--quiet --verbose --json`

**cli_test.go** changes: Updated `TestCLI_TTYDetection` to test through `Run` instead of calling `detectTTY` directly. Updated format constant references from `FormatTOON` to `FormatToon`. Format string changed from `%q` to `%v` (int enum instead of string).

### Spec Compliance

| Acceptance Criterion | Status |
|---|---|
| Formatter interface covers all command output types | PASS -- 6 methods matching spec |
| Format enum with 3 constants | PASS -- `FormatToon`, `FormatPretty`, `FormatJSON` as `int` iota |
| TTY detection works correctly | PASS -- `DetectTTY` checks `ModeCharDevice` |
| ResolveFormat handles all flag/TTY combos | PASS -- all combos tested |
| Conflicting flags -> error | PASS -- integration test confirms |
| FormatConfig wired into CLI dispatch | PASS -- set in `App.Run` after flag parsing |
| Verbose to stderr only | PARTIAL -- not addressed in this commit (no VerboseLogger yet) |
| Stat failure handled gracefully | PASS -- returns false |

**Deviation**: `FormatMessage` returns `error` in V4's Formatter interface, but the spec doesn't specify whether it should. The StubFormatter.FormatMessage does nothing (returns nil), which breaks existing `init` functionality -- the confirmation message "Initialized tick in ..." would not appear.

### golang-pro Skill Compliance

| Rule | Status | Notes |
|---|---|---|
| Use gofmt and golangci-lint | ASSUMED PASS | Code appears formatted |
| Handle all errors explicitly | PASS | All error paths return or propagate |
| Write table-driven tests with subtests | PASS | Conflicting flags test is table-driven with subtests |
| Document all exported functions/types/packages | PASS | All exports have doc comments |
| Propagate errors with fmt.Errorf("%w", err) | N/A | No error wrapping in this task |
| No panic for normal error handling | PASS | Stat failure returns false gracefully |
| No _ assignment without justification | PASS | No ignored errors |

## V5 Implementation

### Architecture & Design

V5 uses a functional architecture with a top-level `Run` function and a `Context` struct:

```go
// internal/cli/cli.go
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

**Format enum** is defined as `type OutputFormat int` in `cli.go` (pre-existing):
```go
type OutputFormat int
const (
    FormatToon OutputFormat = iota
    FormatPretty
    FormatJSON
)
```

**DetectTTY** accepts `*os.File` directly (not `io.Writer`):
```go
func DetectTTY(f *os.File) bool {
    info, err := f.Stat()
    if err != nil {
        return false
    }
    return info.Mode()&os.ModeCharDevice != 0
}
```

**FormatConfig wiring** is via a method on Context:
```go
func (c *Context) FormatCfg() FormatConfig {
    return FormatConfig{
        Format:  c.Format,
        Quiet:   c.Quiet,
        Verbose: c.Verbose,
    }
}
```

**Formatter interface** uses `interface{}` for all data parameters:
```go
type Formatter interface {
    FormatTaskList(w io.Writer, data interface{}) error
    FormatTaskDetail(w io.Writer, data interface{}) error
    FormatTransition(w io.Writer, data interface{}) error
    FormatDepChange(w io.Writer, data interface{}) error
    FormatStats(w io.Writer, data interface{}) error
    FormatMessage(w io.Writer, msg string)
}
```

`FormatMessage` does NOT return error, matching a simpler contract for plain messages.

**StubFormatter**: `FormatMessage` actually writes output:
```go
func (f *StubFormatter) FormatMessage(w io.Writer, msg string) {
    fmt.Fprintln(w, msg)
}
```

All other stubs return nil.

**Key refactoring**: V5's pre-existing code already had `OutputFormat int` and the `Context`-based design. The task removed the inline TTY-default logic from `parseArgs`:
```go
// REMOVED:
if isTTY {
    ctx.Format = FormatPretty
} else {
    ctx.Format = FormatToon
}
```
And replaced per-flag format assignment with boolean tracking + `ResolveFormat` call.

V5 also moved `DetectTTY` from `cmd/tick/main.go` (local `isTerminal` function) into `internal/cli/format.go` as an exported function, and updated `main.go` to call `cli.DetectTTY(os.Stdout)`.

### Code Quality

- Clean separation: all format-related code in `format.go`
- `DetectTTY` accepts `*os.File` -- type-safe at the call site, no runtime type assertion needed; clearer contract
- `FormatConfig` accessed via `ctx.FormatCfg()` method -- lazy construction, no stored redundancy
- `ResolveFormat` uses `switch` for the format selection (more idiomatic Go for multi-branch dispatch):
  ```go
  switch {
  case toonFlag:
      return FormatToon, nil
  case prettyFlag:
      return FormatPretty, nil
  case jsonFlag:
      return FormatJSON, nil
  default:
      ...
  }
  ```
- Naming: `OutputFormat` (pre-existing) is more descriptive than V4's `Format`
- `StubFormatter.FormatMessage` actually writes, so `tick init` continues to produce output during the stub phase

**Weakness**: Using `interface{}` for Formatter method parameters loses compile-time type safety. Implementers must type-assert, which shifts errors from compile time to runtime. V5 later fixed this in commit `552624d` ("T6-7 -- replace interface{} Formatter parameters with type-safe signatures").

### Test Coverage

**format_test.go** (192 lines) contains:

1. `TestDetectTTY` -- 2 subtests:
   - Non-TTY for pipe (using `os.Pipe()`)
   - Non-TTY on stat failure (closed pipe)

2. `TestResolveFormat` -- 6 subtests:
   - Default Toon for non-TTY
   - Default Pretty for TTY
   - FormatToon with toon flag (TTY=true)
   - FormatPretty with pretty flag (non-TTY)
   - FormatJSON with json flag
   - Conflicting flags (table-driven, 4 sub-subtests) -- V5 also **asserts the exact error message**:
     ```go
     expected := "only one format flag may be specified (--toon, --pretty, --json)"
     if err.Error() != expected {
         t.Errorf("error = %q, want %q", err.Error(), expected)
     }
     ```

3. `TestFormatConfig` -- 1 subtest: propagates Quiet/Verbose

4. `TestStubFormatter` -- 2 subtests:
   - Interface compliance
   - `FormatMessage` produces actual output ("hello\n")

5. `TestConflictingFormatFlagsIntegration` -- integration test via `Run()`

6. `TestFormatConfigWiredIntoContext` -- verifies `ctx.FormatCfg()` returns correct values

**cli_test.go** (in V5's diff) has no changes to test files -- format tests were added purely in `format_test.go`.

### Spec Compliance

| Acceptance Criterion | Status |
|---|---|
| Formatter interface covers all command output types | PASS -- 6 methods matching spec |
| Format enum with 3 constants | PASS -- `FormatToon`, `FormatPretty`, `FormatJSON` (pre-existing) |
| TTY detection works correctly | PASS -- `DetectTTY` checks `ModeCharDevice` |
| ResolveFormat handles all flag/TTY combos | PASS -- all combos tested |
| Conflicting flags -> error | PASS -- integration test confirms |
| FormatConfig wired into CLI dispatch | PASS -- `FormatCfg()` method on Context |
| Verbose to stderr only | NOT ADDRESSED -- no verbose logger in this commit (was pre-existing in V5's codebase but not part of this diff) |
| Stat failure handled gracefully | PASS -- returns false |

**Deviation**: The `interface{}` parameters on the Formatter interface are a significant departure from the plan's intent of a type-safe abstraction. The plan says "FormatTaskList", "FormatTaskDetail" etc. which implies typed signatures. V5 later fixed this in a cleanup task (T6-7).

### golang-pro Skill Compliance

| Rule | Status | Notes |
|---|---|---|
| Use gofmt and golangci-lint | ASSUMED PASS | Code appears formatted |
| Handle all errors explicitly | PASS | All error paths return or propagate |
| Write table-driven tests with subtests | PASS | Conflicting flags test is table-driven |
| Document all exported functions/types/packages | PASS | All exports have doc comments |
| Propagate errors with fmt.Errorf("%w", err) | N/A | No error wrapping |
| No panic for normal error handling | PASS | Graceful fallbacks |
| No _ assignment without justification | PASS | No ignored errors |
| Use reflection without performance justification | CONCERN | `interface{}` parameters require runtime type assertions in implementers, which is a form of reflection-adjacent pattern |

## Comparative Analysis

### Where V4 is Better

1. **Type-safe Formatter interface**: V4's Formatter uses concrete types in all method signatures:
   ```go
   // V4
   FormatTaskList(w io.Writer, rows []listRow, quiet bool) error
   FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error
   ```
   vs V5:
   ```go
   // V5
   FormatTaskList(w io.Writer, data interface{}) error
   FormatTransition(w io.Writer, data interface{}) error
   ```
   V4's approach catches type mismatches at compile time. V5's `interface{}` approach defers errors to runtime and requires type assertions in every concrete formatter implementation. This is a significant code quality advantage for V4. (V5 later acknowledged this as technical debt and fixed it in T6-7.)

2. **More thorough DetectTTY tests**: V4 tests three scenarios:
   - `bytes.Buffer` (non-`*os.File` writer) -- tests the type assertion path
   - `os.Pipe()` (real `*os.File` but not a TTY) -- tests the `ModeCharDevice` path
   - Closed file descriptor -- tests the stat failure path

   V5 tests only two: pipe and closed pipe. V5's `DetectTTY` accepts `*os.File` so the `bytes.Buffer` test isn't applicable, but V5 has fewer edge case scenarios as a result.

3. **Placeholder data types defined early**: V4 introduces `TaskDetail{}` and `StatsData{}` as empty structs with TODO comments, providing compile-time placeholders that later tasks can fill in. V5 defines no data types at all in this commit (they're added later in the formatter tasks).

### Where V5 is Better

1. **Stricter DetectTTY signature**: V5's `DetectTTY(f *os.File) bool` accepts only `*os.File`, making the function's contract explicit. V4's `DetectTTY(w io.Writer) bool` accepts any `io.Writer` but internally type-asserts to `*os.File`, meaning non-file writers silently return false. V5's approach pushes the responsibility to the caller (in `main.go`: `cli.DetectTTY(os.Stdout)`), which is cleaner and more honest about what the function actually needs.

2. **FormatMessage that works during stub phase**: V5's `StubFormatter.FormatMessage` actually writes output:
   ```go
   func (f *StubFormatter) FormatMessage(w io.Writer, msg string) {
       fmt.Fprintln(w, msg)
   }
   ```
   V4's does nothing:
   ```go
   func (f *StubFormatter) FormatMessage(w io.Writer, msg string) error {
       return nil
   }
   ```
   This means V5 maintains working `init` output during the stub phase (tasks 4-1 through 4-4), while V4 would silently swallow init confirmation messages until a real formatter is wired in. V5's approach is more pragmatic.

3. **FormatMessage signature**: V5's `FormatMessage(w io.Writer, msg string)` (no error return) is simpler and more appropriate -- writing a simple string message to a writer is unlikely to fail in a way the caller can meaningfully handle. V4's `FormatMessage(...) error` adds ceremony without value.

4. **Error message exactness in tests**: V5 asserts the exact error message text for conflicting flags:
   ```go
   expected := "only one format flag may be specified (--toon, --pretty, --json)"
   if err.Error() != expected {
       t.Errorf("error = %q, want %q", err.Error(), expected)
   }
   ```
   V4 only checks `err == nil` vs `err != nil`. V5's approach is more rigorous.

5. **FormatConfig as a method, not stored state**: V5 derives `FormatConfig` on demand via `ctx.FormatCfg()`. V4 stores it as `App.FormatCfg`, creating redundant state that could drift from the underlying Quiet/Verbose/Format fields. V5's approach follows the "single source of truth" principle.

6. **main.go cleanup**: V5 moves `isTerminal` from `cmd/tick/main.go` into `cli.DetectTTY`, eliminating code duplication. V4 doesn't touch `main.go` at all because V4's `DetectTTY` accepts `io.Writer` and is called within `App.Run()` -- the TTY detection was already internal to App.

7. **ResolveFormat uses switch**: V5's `switch { case toonFlag: ... }` is more idiomatic Go for multi-branch dispatch than V4's sequential `if` statements.

### Differences That Are Neutral

1. **Type name**: V4 uses `Format` while V5 uses `OutputFormat`. Both are clear; `OutputFormat` is slightly more descriptive.

2. **Overall architecture** (App-based vs function-based): This is a pre-existing difference, not introduced by this task. V4 mutates `App` fields; V5 passes `Context` through. Both are valid Go patterns.

3. **Error message wording**: V4: `"only one format flag allowed; got multiple of --toon, --pretty, --json"`. V5: `"only one format flag may be specified (--toon, --pretty, --json)"`. Both are clear.

4. **Unknown flag handling**: V5's `parseArgs` rejects unknown flags with `strings.HasPrefix(arg, "-")`. V4's `parseGlobalFlags` treats unknown flags as the start of the subcommand arguments. Both behaviors are reasonable given their different architectural contexts.

5. **Test structure**: V4 has `TestFormatter` (stub test) inline in format_test.go. V5 has `TestStubFormatter` with an additional subtest for `FormatMessage` output. Both adequately test the stub.

## Verdict

**Winner: V5**, with caveats.

V5 wins on several design quality dimensions that matter for maintainability:

1. **`DetectTTY(*os.File)`** is the superior signature. It makes the contract explicit at the type level rather than hiding a type assertion. The caller in `main.go` passing `os.Stdout` directly is clean and obvious. V4's `io.Writer` parameter is misleading -- it suggests any writer works, but only `*os.File` actually enables TTY detection.

2. **Working StubFormatter.FormatMessage** is pragmatically important. During the multi-task development window (4-1 through 4-4), V5 maintains a functional CLI. V4's stub silently drops messages, which could confuse users or break integration tests that depend on init output.

3. **FormatConfig as derived state** (`ctx.FormatCfg()`) is cleaner than V4's stored redundant field. It eliminates the possibility of the config drifting from the underlying flags.

4. **Exact error message assertions** in tests make V5's test suite more precise and regression-resistant.

However, V4 has one significant advantage: **type-safe Formatter interface signatures**. This is a real and important difference. V5's `interface{}` parameters are a Go anti-pattern that V5 itself acknowledged by fixing in a later cleanup task (T6-7). V4 got this right from the start, which means V4's formatters (in subsequent tasks 4-2 through 4-4) would have had compile-time guarantees from day one.

The margin is narrow. V5's `interface{}` choice is the biggest blemish, but V5's other decisions (DetectTTY signature, working stub, derived FormatConfig, test precision) collectively outweigh it. V5 also demonstrated awareness of the `interface{}` problem by addressing it in its cleanup phase, suggesting a deliberate "get it working, then refine" approach rather than an oversight.

If forced to rank the factors:
- Type safety (V4 advantage): HIGH importance but was later fixed by V5
- DetectTTY contract (V5 advantage): MEDIUM importance, cleaner API surface
- Working stub (V5 advantage): MEDIUM importance, practical impact during development
- Test precision (V5 advantage): LOW-MEDIUM importance
- FormatConfig design (V5 advantage): LOW importance but shows better design sense

**V5 wins by a small margin**, primarily on API contract honesty and practical functionality, despite the `interface{}` misstep.
