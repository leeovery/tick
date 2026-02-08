# Task tick-core-4-6: Verbose Output & Edge Case Hardening

## Task Summary

Implement `--verbose` / `-v` across all commands. Debug detail (cache rebuild, lock, hash comparison) must go to stderr only. Piped output stays clean. Quiet + verbose = silent stdout, debug stderr.

**Acceptance Criteria:**
1. VerboseLogger writes stderr only when Verbose true
2. Key operations instrumented (cache, lock, hash, write, format)
3. All lines `verbose:` prefixed
4. Zero verbose on stdout
5. `--quiet` + `--verbose` works correctly
6. Piping captures only formatted output
7. No output when verbose off

## Acceptance Criteria Compliance

| Criterion | V2 | V4 |
|-----------|-----|-----|
| VerboseLogger writes stderr only when Verbose true | PASS -- `VerboseLogger` struct with `enabled` bool gate; initialized with `a.stderr` in `app.go:67` | PASS -- `VerboseLogger` struct with `enabled` bool gate; initialized with `a.Stderr` in `cli.go:50` |
| Key operations instrumented (cache, lock, hash, write, format) | PASS -- store.go logs lock acquire/release, freshness/hash checks, cache rebuild, atomic write, task counts; app.go logs format resolution | PASS -- store.go logs lock acquire/release, hash comparison, cache freshness, atomic write; cli.go logs format resolution via `formatName()` |
| All lines `verbose:` prefixed | PASS -- `fmt.Fprintf(v.w, "verbose: %s\n", msg)` in `verbose.go:28` | PASS -- `fmt.Fprintf(v.w, "verbose: %s\n", msg)` in `verbose.go:27` |
| Zero verbose on stdout | PASS -- Logger writes to `a.stderr`; store receives logger interface that writes to same stderr | PASS -- Logger writes to `a.Stderr`; store receives `LogFunc` that writes to same stderr |
| `--quiet` + `--verbose` works correctly | PASS -- orthogonal flags; quiet suppresses stdout formatting, verbose writes to stderr independently | PASS -- orthogonal flags; quiet suppresses stdout formatting, verbose writes to stderr independently |
| Piping captures only formatted output | PASS -- verbose always on stderr; tested with multi-task list checking no verbose lines on stdout | PASS -- verbose always on stderr; tested with multi-task list checking no verbose lines on stdout |
| No output when verbose off | PASS -- `enabled` check at top of `Log()` returns immediately; test verifies zero stderr content | PASS -- `enabled` check at top of `Log()` returns immediately; test verifies zero stderr content |

## Implementation Comparison

### Approach

Both versions follow the same high-level architecture: a `VerboseLogger` struct in `internal/cli/verbose.go` that wraps an `io.Writer` with an `enabled` flag, an integration point in the main App struct, and instrumentation calls in the Store layer.

**VerboseLogger API**

V2 uses a simple `Log(msg string)` signature:
```go
// V2 verbose.go:25
func (v *VerboseLogger) Log(msg string) {
    if !v.enabled { return }
    fmt.Fprintf(v.w, "verbose: %s\n", msg)
}
```

V4 uses a printf-style `Log(format string, args ...interface{})` signature:
```go
// V4 verbose.go:23
func (v *VerboseLogger) Log(format string, args ...interface{}) {
    if !v.enabled { return }
    msg := fmt.Sprintf(format, args...)
    fmt.Fprintf(v.w, "verbose: %s\n", msg)
}
```

V4's variadic approach is genuinely better -- it avoids forcing callers to pre-format with `fmt.Sprintf()`. V2 callers must do `a.verbose.Log(fmt.Sprintf("freshness: read %d tasks from JSONL", len(tasks)))` whereas V4 callers do `s.vlog("acquiring exclusive lock on %s", s.lockPath)`. This is more ergonomic and follows Go conventions (e.g., `log.Printf`, `fmt.Errorf`).

**Store-CLI Integration**

V2 defines a `Logger` interface in the store package and uses dependency injection via `SetLogger`:
```go
// V2 storage/store.go
type Logger interface {
    Log(msg string)
}
func (s *Store) SetLogger(l Logger) { s.logger = l }
func (s *Store) logVerbose(msg string) {
    if s.logger != nil { s.logger.Log(msg) }
}
```

V4 uses an exported function field on the Store struct:
```go
// V4 store/store.go
LogFunc func(format string, args ...interface{})
func (s *Store) vlog(format string, args ...interface{}) {
    if s.LogFunc != nil { s.LogFunc(format, args...) }
}
```

V2's interface approach is more idiomatic Go -- it decouples the store from the logger implementation and allows any struct satisfying `Logger` to be plugged in. V4's function field is simpler and more pragmatic -- less boilerplate, no interface to define, and the store doesn't need to know about the logger type at all. Both are reasonable; V2 is more extensible, V4 is more concise.

**App-level Wiring**

V2 creates the logger in `app.go:67` and wires it via `store.SetLogger(a.verbose)`:
```go
// V2 app.go:67-68
a.verbose = NewVerboseLogger(a.stderr, a.config.Verbose)
// V2 app.go:197-199 (newStore helper)
store.SetLogger(a.verbose)
```

V4 creates the logger in `cli.go:50` and wires it via direct field assignment:
```go
// V4 cli.go:50
a.vlog = NewVerboseLogger(a.Stderr, a.Verbose)
// V4 cli.go:175 (openStore helper)
s.LogFunc = a.vlog.Log
```

V2 names the field `verbose`, V4 names it `vlog`. V4's naming is more concise. V2's helper method is named `newStore`, V4's is `openStore`. Both centralize store creation and logger wiring, which is clean.

**Fallback for Pre-Init Logging**

V2 includes a fallback `logVerbose` method on App that handles the case where `a.verbose` is nil (pre-Run initialization):
```go
// V2 app.go:188-196
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

V4 has no such fallback. This is a minor edge case -- in practice the logger is always initialized before any logging occurs, so V2's fallback is defensive but unnecessary.

**Store Logging Messages**

V2 uses descriptive category prefixes in log messages (e.g., `"lock: acquiring exclusive lock"`, `"freshness: checking cache hash"`, `"cache: freshness check complete"`, `"write: atomic write to tasks.jsonl"`). These have a consistent `category: action` format.

V4 uses plain English descriptions without category prefixes (e.g., `"acquiring exclusive lock on %s"`, `"checking cache freshness via hash comparison"`, `"cache is fresh"`, `"atomic write to %s"`). V4 includes actual path values in the messages via format strings.

V2's approach is more structured and grep-friendly. V4's approach includes more contextual data (paths). Different but equivalent in utility.

**Files Modified**

V2 modifies 10 files in `internal/`: `app.go`, `create.go`, `dep.go`, `list.go`, `show.go`, `transition.go`, `update.go`, `verbose.go` (new), `verbose_test.go` (new), `storage/store.go`.

V4 modifies 13 files in `internal/`: `blocked.go`, `cli.go`, `create.go`, `dep.go`, `format.go`, `list.go`, `ready.go`, `show.go`, `transition.go`, `update.go`, `verbose.go` (new), `verbose_test.go` (new), `store/store.go`.

V4 touches more files because it also updates `blocked.go`, `ready.go`, and `format.go` (adding `formatName` helper). V2 doesn't need to update ready/blocked separately because they route through `runList`.

### Code Quality

**Go Idioms**

V2's `Logger` interface in the store package follows the "accept interfaces, return structs" Go idiom. The interface is small (single method) which is idiomatic.

V4's `LogFunc` function field is a valid Go pattern (used in stdlib's `http.Server.ErrorLog`) but less common for this use case.

**Naming**

V2: `VerboseLogger`, `verbose`, `logVerbose`, `SetLogger`, `newStore`
V4: `VerboseLogger`, `vlog`, `Log`, `LogFunc`, `openStore`, `formatName`

V4's `vlog` is terse; V2's `verbose` is clearer. V4's `openStore` better communicates intent than V2's `newStore` (the store already exists on disk; we're opening it).

**Error Handling**

Both versions have identical error handling patterns -- errors propagate up from store operations. Neither version adds verbose-specific error handling, which is correct since logging failures should be silent.

**DRY**

Both versions centralize store creation into a single helper method (`newStore`/`openStore`) to avoid repeating the logger wiring code across all command handlers. This is clean.

V4 adds a `formatName()` function in `format.go` to convert the format enum to a string name for verbose logging. V2 uses `fmt.Sprintf("format resolved: %s", a.FormatCfg.Format)` which relies on the `OutputFormat` type being a string alias (`OutputFormat = "toon"`), so it works without a conversion function.

**Type Safety**

V2's `Logger` interface ensures compile-time type checking -- the `VerboseLogger` must implement `Log(msg string)`. V4's `LogFunc` field has the function signature checked at assignment time but offers no interface contract.

### Test Quality

**V2 Test Functions (259 lines):**

1. `TestVerboseLogger` (unit tests for the logger itself):
   - `"it writes verbose-prefixed messages when enabled"` -- checks output contains `verbose:` and the message
   - `"it writes nothing when disabled"` -- checks empty output
   - `"it prefixes every line with verbose:"` -- multi-message prefix check

2. `TestVerboseOutput` (integration tests via full App.Run):
   - `"it writes cache/lock/hash/format verbose to stderr"` -- checks stderr contains `verbose:` and all four categories (format, lock, freshness, cache)
   - `"it writes nothing to stderr when verbose off"` -- checks no `verbose:` in stderr
   - `"it does not write verbose to stdout"` -- checks no `verbose:` in stdout
   - `"it allows quiet + verbose simultaneously"` -- checks stdout has no verbose, stderr has verbose
   - `"it works with each format flag without contamination"` -- table-driven across toon/pretty/json; checks stdout clean, stderr has verbose
   - `"it produces clean piped output with verbose enabled"` -- two tasks, checks no verbose lines on stdout, verifies task data present
   - `"it logs verbose for mutations too"` -- runs `create`, checks stderr mentions lock and write

**V4 Test Functions (223 lines):**

1. `TestVerbose_WritesToStderr`:
   - `"it writes cache/lock/hash/format verbose to stderr"` -- checks all four categories individually (lock, hash, cache/fresh, format)

2. `TestVerbose_WritesNothingWhenOff`:
   - `"it writes nothing to stderr when verbose off"` -- checks stderr is completely empty (stricter than V2 which only checks for `verbose:` prefix)

3. `TestVerbose_DoesNotWriteToStdout`:
   - `"it does not write verbose to stdout"` -- checks no `verbose:` in stdout

4. `TestVerbose_AllowsQuietPlusVerbose`:
   - `"it allows quiet + verbose simultaneously"` -- checks no format headers on stdout, verbose on stderr

5. `TestVerbose_WorksWithEachFormatFlag`:
   - `"it works with each format flag without contamination"` -- table-driven across toon/pretty/json

6. `TestVerbose_CleanPipedOutput`:
   - `"it produces clean piped output with verbose enabled"` -- two tasks, checks no verbose on stdout, asserts exactly 3 lines (header + 2 tasks)

7. `TestVerbose_AllLinesPrefixed`:
   - `"it prefixes all verbose lines with verbose:"` -- iterates all stderr lines ensuring each starts with `verbose:`

**Test Coverage Comparison:**

| Edge Case | V2 | V4 |
|-----------|-----|-----|
| Logger unit test (enabled writes) | YES | NO (integration only) |
| Logger unit test (disabled silent) | YES | NO (integration only) |
| Logger unit test (prefix on all lines) | YES | NO (covered in integration) |
| Stderr contains all categories | YES | YES |
| No output when verbose off | YES | YES (stricter: checks empty stderr) |
| No verbose on stdout | YES | YES |
| Quiet + verbose orthogonal | YES | YES |
| Format flags don't contaminate | YES (table-driven) | YES (table-driven) |
| Clean piped output | YES | YES (stricter: asserts exact line count) |
| All stderr lines prefixed | NO (unit test covers prefix, not integration) | YES (integration-level check) |
| Mutation verbose logging | YES (`create` command) | NO |
| Hash mentioned in verbose | NO (checks freshness, not hash directly) | YES (explicit `hasHash` check) |

V2 has 10 test cases (3 unit + 7 integration). V4 has 7 test cases (all integration).

V2 includes unit tests for VerboseLogger in isolation, which is good practice for testing the building block independently. V4 only tests through integration, meaning the VerboseLogger struct is never tested in isolation.

V2 tests mutation verbose output (via `create`), which V4 does not. This is a meaningful gap -- V4 doesn't verify that write operations emit verbose logging.

V4 has a dedicated `TestVerbose_AllLinesPrefixed` that verifies at the integration level that every single line on stderr starts with `verbose:`. V2 only checks this at the unit level.

V4 explicitly checks for "hash" in verbose output; V2 checks for "freshness" instead. Both adequately verify cache/hash instrumentation.

V4's `TestVerbose_WritesNothingWhenOff` asserts `stderr.String() != ""` which is stricter -- it ensures absolutely nothing is on stderr, not just no `verbose:` prefix. V2 only checks `!strings.Contains(stderr.String(), "verbose:")`.

**Test Setup:**

V2 uses `setupTickDirWithContent` (writes raw JSONL) and `setupInitializedTickDir` (for mutations). Tests use `strings.Builder` for stdout/stderr buffers.

V4 uses `setupInitializedDirWithTasks` (writes structured `task.Task` objects). Tests use `bytes.Buffer` for stdout/stderr buffers.

V4's setup is more type-safe (uses `task.Task` structs instead of raw JSON strings). V2's raw JSON setup is more fragile but also tests the full parse path.

**App Construction:**

V2: `app := NewApp()` then sets fields on the private struct (`app.workDir = dir`).
V4: `app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}` -- direct struct literal.

V4's approach is cleaner for tests since all fields are exported.

## Diff Stats

| Metric | V2 | V4 |
|--------|-----|-----|
| Files changed (internal/) | 10 | 13 |
| Lines added (internal/) | 363 | 327 |
| Lines deleted (internal/) | 17 | 22 |
| Impl LOC (verbose.go) | 29 | 28 |
| Impl LOC (store changes) | ~43 added | ~34 added |
| Impl LOC (app/cli changes) | ~29 added | ~21 added + 14 in format.go |
| Test LOC (verbose_test.go) | 259 | 223 |
| Test functions | 10 (3 unit + 7 integration) | 7 (all integration) |

## Verdict

**V2 is slightly better overall**, primarily due to its more comprehensive test suite.

Key differentiators:

1. **Test coverage**: V2 has 10 tests vs V4's 7. V2 includes unit tests for VerboseLogger in isolation (3 tests) and crucially tests mutation verbose output (`create` command), which V4 omits entirely. This is a meaningful gap -- the task requires instrumenting write operations, and V4 doesn't verify that.

2. **Logger API**: V4's `Log(format string, args ...interface{})` signature is genuinely better than V2's `Log(msg string)` -- it eliminates boilerplate `fmt.Sprintf` calls at every call site in `store.go`. This is a clear win for V4.

3. **Store integration**: V2's `Logger` interface is more idiomatic Go. V4's `LogFunc` function field is simpler but less extensible. Both work correctly.

4. **Log message structure**: V2's `category: action` format (`"lock: acquiring exclusive lock"`) is more structured and grep-friendly. V4's messages include actual path values which is useful for debugging. Different tradeoffs, roughly equivalent.

5. **Test strictness**: V4's "verbose off" test is stricter (checks empty stderr vs checking no `verbose:` prefix). V4 also has a dedicated integration-level "all lines prefixed" test. These are genuine improvements.

The deciding factor is test coverage breadth. V2 tests both reads (list) and writes (create) with verbose, tests the VerboseLogger in isolation, and has more edge cases covered. V4's logger API is the superior design choice, but the test gap (no mutation testing) is a more significant shortcoming for a task that explicitly requires instrumenting "cache rebuild, lock, hash comparison" across all operations.
