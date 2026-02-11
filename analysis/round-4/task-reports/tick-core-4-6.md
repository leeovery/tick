# Task tick-core-4-6: Verbose output & edge case hardening

## Task Summary

Implement `--verbose` / `-v` across all commands. Debug detail (cache rebuild, lock, hash comparison) goes to stderr only. Piped output stays clean. Quiet + verbose = silent stdout, debug stderr.

Requirements:
- `VerboseLogger` wrapping `fmt.Fprintf(os.Stderr, ...)`. No-op when verbose off.
- Instrument: freshness detection, cache rebuild, lock acquire/release, atomic write, format resolution.
- All lines prefixed `verbose:` for grep-ability.
- Quiet + verbose: orthogonal (different streams). Both active = silent stdout, debug stderr.
- Pipe safety: `tick list --verbose | wc -l` counts only task lines.

Acceptance Criteria:
1. VerboseLogger writes stderr only when Verbose true
2. Key operations instrumented (cache, lock, hash, write, format)
3. All lines `verbose:` prefixed
4. Zero verbose on stdout
5. --quiet + --verbose works correctly
6. Piping captures only formatted output
7. No output when verbose off

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| VerboseLogger writes stderr only when Verbose true | PASS -- `VerboseLogger` in `engine` package uses `enabled` bool field; all calls check `if !v.enabled { return }` | PASS -- `VerboseLogger` in `cli` package uses nil-receiver pattern; `if vl == nil { return }`. Store uses `func(msg string)` callback, nil-checked via `if s.verboseLog != nil` |
| Key operations instrumented (cache, lock, hash, write, format) | PASS -- Lock acquire (exclusive/shared), lock release, cache freshness check, cache fresh/stale, cache rebuild, atomic write, format resolution all instrumented | PASS -- Lock acquire (exclusive/shared), lock acquired, lock release, hash match (yes/no), cache fresh, cache rebuild, writing JSONL atomically, rebuilding cache, format resolution all instrumented |
| All lines `verbose:` prefixed | PASS -- `VerboseLogger.Log` and `Logf` both format with `"verbose: %s\n"` prefix | PASS -- `VerboseLogger.Log` formats with `"verbose: %s\n"` prefix; `VerboseLog` standalone function also updated to add prefix |
| Zero verbose on stdout | PASS -- VerboseLogger always writes to the writer passed at construction (stderr); never touches stdout | PASS -- Same pattern; VerboseLogger constructed with `a.Stderr`; store callback delegates to same |
| --quiet + --verbose works correctly | PASS -- Tested in `TestVerbose/"it allows quiet + verbose simultaneously"` | PASS -- Tested in `TestVerboseLogger/"it allows quiet + verbose simultaneously"` |
| Piping captures only formatted output | PASS -- Tested with `--toon` flag; verifies no `verbose:` lines in stdout and tasks present | PASS -- Tested with `IsTTY: false` pipe simulation; verifies no `verbose:` in stdout and at least 2 lines |
| No output when verbose off | PASS -- Tested at both engine (store) and CLI integration level | PASS -- Tested at both VerboseLogger unit level and full App integration level |

## Implementation Comparison

### Approach

**V5: VerboseLogger as a concrete struct in `engine` package, passed as whole object**

V5 places `VerboseLogger` in `internal/engine/verbose.go` as a struct with two fields:
```go
type VerboseLogger struct {
    w       io.Writer
    enabled bool
}
```
The logger is created via `NewVerboseLogger(w io.Writer, enabled bool)` and provides both `Log(msg string)` and `Logf(format string, args ...interface{})` methods. The no-op check is `if !v.enabled { return }`.

The `Store` holds a `*VerboseLogger` field directly:
```go
type Store struct {
    // ...
    verbose *VerboseLogger
}
```
Initialized with a no-op default in `NewStore`:
```go
verbose: NewVerboseLogger(nil, false),
```
The store option `WithVerbose(vl *VerboseLogger)` passes the entire struct. The CLI creates it via `Context.newVerboseLogger()`:
```go
func (c *Context) newVerboseLogger() *engine.VerboseLogger {
    return engine.NewVerboseLogger(c.Stderr, c.Verbose)
}
```
Store calls use `s.verbose.Log(...)` directly.

**V6: VerboseLogger as a nil-safe struct in `cli` package, store receives a `func(msg string)` callback**

V6 places `VerboseLogger` in `internal/cli/verbose.go` with a single field:
```go
type VerboseLogger struct {
    w io.Writer
}
```
No `enabled` field -- the no-op behavior is achieved by nil-receiver safety:
```go
func (vl *VerboseLogger) Log(msg string) {
    if vl == nil {
        return
    }
    fmt.Fprintf(vl.w, "verbose: %s\n", msg)
}
```
The logger is only created when verbose is actually enabled (`app.go` line 37-39):
```go
if fc.Verbose {
    fc.Logger = NewVerboseLogger(a.Stderr)
}
```

The `Store` in `internal/storage/store.go` does NOT depend on the VerboseLogger type. Instead it holds a function:
```go
verboseLog func(msg string)
```
With a helper method:
```go
func (s *Store) verbose(msg string) {
    if s.verboseLog != nil {
        s.verboseLog(msg)
    }
}
```
The bridge is in `storeOpts`:
```go
func storeOpts(fc FormatConfig) []storage.StoreOption {
    if fc.Logger == nil {
        return nil
    }
    return []storage.StoreOption{
        storage.WithVerbose(fc.Logger.Log),
    }
}
```

V6 also modifies the pre-existing `VerboseLog` standalone function in `format.go` to add the `verbose:` prefix, whereas V5 does not touch this function (it does not appear to exist in V5's codebase structure).

Additionally, V6 adds a `Logger *VerboseLogger` field to `FormatConfig`, which propagates the logger through the command dispatch path.

**Key architectural difference:** V5 couples `engine.Store` to `engine.VerboseLogger` (same package). V6 decouples `storage.Store` from `cli.VerboseLogger` via a `func(msg string)` callback, achieving better separation of concerns -- the storage layer has no knowledge of the CLI's verbose logger type.

### Code Quality

**V5:**
- VerboseLogger provides both `Log` and `Logf` methods. `Logf` is used in `Rebuild()`: `s.verbose.Logf("rebuilding cache (%d tasks)", len(tasks))`. This is a convenience but adds surface area.
- Default initialization in `NewStore` creates a disabled logger: `verbose: NewVerboseLogger(nil, false)`. This means `s.verbose.Log(...)` is always safe without nil checks, but a `VerboseLogger` object is allocated even when verbose is off.
- The `Context.storeOpts()` always includes the verbose option even when verbose is off (the disabled logger is still passed). Slightly wasteful but safe.
- Lock acquisition is factored into `acquireExclusive()`/`acquireShared()` helper functions that return `unlock func()`. Clean pattern.
- Verbose logs in store: `"lock acquired (exclusive)"`, `"lock acquired (shared)"`, `"lock released"`, `"cache freshness check"`, `"cache is fresh"`, `"cache rebuild (stale or missing)"`, `"atomic write to tasks.jsonl"`.
- Format resolution logged in `cli.go` `Run()` function only when `ctx.Verbose` is true, using a fresh logger instance.
- All exported functions and types are documented.

**V6:**
- VerboseLogger only provides `Log`. No `Logf`. Format strings are constructed at the call site: `s.verbose(fmt.Sprintf("rebuilding cache with %d tasks", len(tasks)))`.
- Nil-receiver pattern is idiomatic Go and avoids allocating a disabled logger.
- `storeOpts(fc)` returns nil when `fc.Logger == nil`, avoiding unnecessary option application.
- Store uses inline lock management (no helper functions); lock/unlock with deferred closures directly in `Mutate`/`Query`.
- Store verbose messages are more granular: `"acquiring exclusive lock"`, `"lock acquired"`, `"lock released"`, `"hash match: yes"`, `"hash match: no"`, `"rebuilding cache from JSONL"`, `"writing JSONL atomically"`, `"cache is fresh"`.
- Modifies the existing `VerboseLog` standalone function to add `verbose:` prefix. This retroactively fixes any pre-existing callers to use the prefix.
- `FormatConfig` gains a `Logger *VerboseLogger` field, enabling the verbose logger to flow through the entire command dispatch without additional plumbing.
- All exported functions and types are documented.

**V6's decoupled approach is architecturally cleaner.** The storage package should not need to know about a CLI-level logger type. The `func(msg string)` callback pattern is a standard Go idiom for optional logging (used by `log.Logger`, `testing.T.Logf`, etc.).

**V5's approach is simpler within its own package** -- calling `s.verbose.Log(...)` reads slightly more naturally than `s.verbose(...)`, and having `Logf` avoids `fmt.Sprintf` at call sites.

### Test Quality

**V5 Test Functions:**

`internal/engine/verbose_test.go` -- `TestVerboseLogger`:
1. `"it writes to writer with verbose prefix when verbose is true"` -- verifies exact output `"verbose: cache rebuild started\n"`
2. `"it writes nothing when verbose is false"` -- verifies `buf.Len() != 0`
3. `"it supports formatted output"` -- tests `Logf` with format args
4. `"it writes nothing for formatted output when verbose is false"` -- Logf no-op check
5. `"it handles nil writer gracefully when verbose is false"` -- nil writer doesn't panic

`internal/engine/store_verbose_test.go` -- `TestStoreVerbose`:
1. `"it logs lock/cache/hash/write operations when verbose is on"` -- Mutate flow, checks 5 expected phrases
2. `"it logs lock/cache/hash operations during query when verbose is on"` -- Query flow, checks 3 expected phrases
3. `"it writes nothing when verbose is off"` -- Mutate with disabled logger, empty buffer
4. `"it logs hash comparison on freshness check"` -- Two queries, second shows "fresh"
5. `"it prefixes all lines with verbose:"` -- Iterates all output lines, checks prefix

`internal/cli/verbose_test.go` -- `TestVerbose`:
1. `"it writes cache/lock/hash/format verbose to stderr"` -- Full CLI integration, 4 expected phrases
2. `"it writes nothing to stderr when verbose off"` -- No `verbose:` in stderr
3. `"it does not write verbose to stdout"` -- No `verbose:` in stdout
4. `"it allows quiet + verbose simultaneously"` -- Quiet stdout (ID only) + verbose stderr
5. `"it works with each format flag without contamination"` -- Subtests for --toon, --pretty, --json; checks format correctness and no verbose on stdout
6. `"it produces clean piped output with verbose enabled"` -- Two tasks, --toon, no verbose in stdout, tasks present
7. `"it logs verbose during mutation commands"` -- `create` command, checks exclusive lock and atomic write logs

**Total V5 tests: 17** (5 unit + 5 store integration + 7 CLI integration)

**V6 Test Functions:**

`internal/cli/verbose_test.go` -- `TestVerboseLogger`:
1. `"it writes cache/lock/hash/format verbose to stderr"` -- Unit test, manually calls `Log` 5 times, checks exact output lines
2. `"it writes nothing to stderr when verbose off"` -- Nil receiver test, no panic
3. `"it does not write verbose to stdout"` -- VerboseLogger writes to stderr buffer, stdout stays empty
4. `"it allows quiet + verbose simultaneously"` -- Full App integration: `--quiet --verbose list`, verifies quiet stdout and verbose stderr
5. `"it works with each format flag without contamination"` -- Subtests for --toon, --pretty, --json via App.Run
6. `"it produces clean piped output with verbose enabled"` -- Two tasks, `IsTTY: false`, no verbose in stdout
7. `"it writes nothing to stderr when verbose off"` (duplicate name, line 180) -- Full App integration, no --verbose flag, stderr empty
8. `"it prefixes all lines with verbose:"` -- Unit test, 3 manual Log calls, checks prefix

**Total V6 tests: 8** (in a single file)

**V6 also modifies `format_test.go`:** Updates the existing `TestVerboseToStderrOnly` test to expect `"verbose: debug info\n"` instead of `"debug info\n"`.

**Test gap analysis:**

| Test coverage area | V5 | V6 |
|---|---|---|
| VerboseLogger unit tests (Log/Logf) | 5 tests | 3 tests (no Logf tests -- V6 has no Logf) |
| Store-level verbose integration | 5 dedicated tests in `store_verbose_test.go` | None -- no store-level verbose tests |
| CLI integration tests | 7 tests using `Run()` function | 5 tests using `App.Run()` |
| Mutation command verbose | Tested (`create` command) | Not tested at integration level |
| Nil writer safety | Tested (verbose=false with nil writer) | Tested (nil receiver) |
| Hash comparison (fresh vs stale) | Tested (two sequential queries) | Not explicitly tested at store level |
| Prefix enforcement on all lines | Tested at both store and CLI level | Tested at unit level only |

**V5 has significantly better test depth** with 17 total tests across three layers (unit, store integration, CLI integration). V5's `store_verbose_test.go` is notable -- it directly tests verbose output during real `Mutate` and `Query` operations with actual file I/O, which V6 completely lacks. V6's CLI integration tests are solid but skip the storage layer entirely.

V6 has a gap: it does not test that the `storeOpts` bridge actually results in verbose output flowing from storage operations through to stderr. The unit tests manually call `Log` and check output, but don't verify the end-to-end path through `storage.Store -> verboseLog callback -> VerboseLogger.Log -> stderr`.

### Skill Compliance

| Skill Constraint | V5 | V6 |
|---|---|---|
| Use gofmt/golangci-lint | Assumed compliant (code is well-formatted) | Assumed compliant |
| Handle all errors explicitly | PASS -- no naked returns | PASS -- no naked returns |
| Write table-driven tests with subtests | PARTIAL -- Format contamination test uses table-driven subtests; others are individual subtests but not table-driven | PARTIAL -- Format contamination test uses table-driven approach; others are individual subtests |
| Document all exported functions/types | PASS -- `VerboseLogger`, `NewVerboseLogger`, `Log`, `Logf`, `WithVerbose` all documented | PASS -- `VerboseLogger`, `NewVerboseLogger`, `Log`, `WithVerbose` all documented |
| Propagate errors with fmt.Errorf("%w", err) | PASS -- seen in store.go | PASS -- seen in store.go |
| No panic for normal error handling | PASS | PASS |
| No goroutines without lifecycle management | N/A | N/A |
| No ignored errors without justification | PASS -- `_ = fl.Unlock()` is standard for unlock cleanup | PASS -- same pattern |
| No hardcoded configuration | PASS -- verbose is flag-controlled | PASS -- verbose is flag-controlled |

Both versions comply well with skill constraints. Neither version uses fully table-driven tests for the majority of test cases, but the spec-required tests have specific names that map 1:1 to spec lines, making table-driven format less natural.

### Spec-vs-Convention Conflicts

**VerboseLogger placement:**
The spec says `VerboseLogger wrapping fmt.Fprintf(os.Stderr, ...)`. V5 places it in `engine` (the storage package), V6 places it in `cli`. Both achieve the spec requirement. V6's placement in `cli` is arguably more correct since verbose logging is a CLI concern, and the storage layer receives only a callback. V5 placing it in `engine` creates a tighter coupling but keeps things simpler.

**`Logf` method:**
The spec does not mention `Logf`. V5 adds it as a convenience. V6 omits it. Neither is wrong; V5's choice adds minimal surface area for a real use case (formatted rebuild messages).

**VerboseLog standalone function:**
V6 modifies the pre-existing `VerboseLog(w io.Writer, verbose bool, msg string)` function in `format.go` to add the `verbose:` prefix. This is a retroactive fix to ensure all verbose output across the codebase uses the prefix. V5 does not touch this function (it doesn't exist in V5's format.go, suggesting V5's codebase structure differs slightly). This is a subtle but important difference -- V6 ensures consistency of the `verbose:` prefix across both the new VerboseLogger and any pre-existing verbose output paths.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | 12 | 12 |
| Lines added | 521 | 328 |
| Lines removed | 8 | 14 |
| New files | 3 (`verbose.go`, `verbose_test.go`, `store_verbose_test.go`) | 2 (`verbose.go`, `verbose_test.go`) |
| Test functions | 17 | 8 (+1 modified in `format_test.go`) |
| VerboseLogger location | `internal/engine` | `internal/cli` |
| Store verbose mechanism | `*VerboseLogger` field | `func(msg string)` callback |
| VerboseLogger methods | `Log`, `Logf` | `Log` only |
| No-op mechanism | `enabled bool` field check | nil-receiver check |

## Verdict

Both implementations fully satisfy all seven acceptance criteria. The functional behavior is equivalent -- both produce `verbose:`-prefixed debug output on stderr, support quiet+verbose orthogonality, and keep piped stdout clean.

**V6 has the better architecture.** The `func(msg string)` callback decouples `storage.Store` from the CLI's logger type, which is the right separation of concerns. The nil-receiver pattern for no-op is more idiomatic Go than allocating a disabled logger struct. The retroactive fix to `VerboseLog` in `format.go` shows better codebase awareness.

**V5 has significantly better test coverage.** With 17 tests across three layers versus V6's 8, and critically including `store_verbose_test.go` which validates the verbose output actually flows through real storage operations, V5 provides much stronger confidence. V6's complete absence of store-level verbose tests means the `storeOpts` bridge and the `func(msg string)` callback path are only tested indirectly through full CLI integration tests.

**Overall: Close call, slight edge to V5.** While V6's architecture is cleaner, V5's test suite is substantially more thorough. In a production Go codebase following the skill's mandate for comprehensive testing, the test depth difference is significant enough to outweigh the architectural elegance. V6's store-level test gap is a real risk -- a refactor breaking the callback wiring would not be caught by V6's tests.
