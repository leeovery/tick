# Task 4-6: Verbose output & edge case hardening

## Task Plan Summary

The task requires implementing `--verbose` / `-v` across all commands. Key requirements:

1. **VerboseLogger** wrapping `fmt.Fprintf(os.Stderr, ...)` -- no-op when verbose is off.
2. **Instrument** key operations: freshness detection, cache rebuild, lock acquire/release, atomic write, format resolution.
3. All verbose lines **prefixed `verbose:`** for grep-ability.
4. **Quiet + verbose**: orthogonal (different streams). Both active = silent stdout, debug stderr.
5. **Pipe safety**: `tick list --verbose | wc -l` counts only task lines (zero verbose contamination on stdout).
6. Six specified tests plus edge cases for stderr-only, quiet+verbose orthogonality, format-flag compatibility, and zero-output when disabled.

Acceptance criteria: VerboseLogger writes stderr only when Verbose true; key operations instrumented (cache, lock, hash, write, format); all lines `verbose:` prefixed; zero verbose on stdout; --quiet + --verbose works correctly; piping captures only formatted output; no output when verbose off.

---

## V4 Implementation

### Architecture & Design

V4 places the `VerboseLogger` type in `internal/cli/verbose.go` (package `cli`). It is a simple struct with an `io.Writer` and `enabled` bool. The single method `Log(format string, args ...interface{})` does a `fmt.Sprintf` then `fmt.Fprintf` with the `verbose:` prefix.

**Wiring approach**: The CLI `App` struct in `cli.go` gains a `vlog *VerboseLogger` field (line 32). During `Run()`, after flag parsing, `a.vlog = NewVerboseLogger(a.Stderr, a.Verbose)` is created (line 50). The CLI immediately logs format resolution: `a.vlog.Log("format resolved to %s", formatName(a.OutputFormat))` (line 62).

For the storage layer, V4 uses a **callback injection pattern**: `Store` in `internal/store/store.go` exposes a public field `LogFunc func(format string, args ...interface{})` (line 33). The helper method `s.vlog(format, args...)` calls `LogFunc` if non-nil (lines 66-70). The CLI's `openStore()` method wires this up:

```go
func (a *App) openStore(tickDir string) (*store.Store, error) {
    s, err := store.NewStore(tickDir)
    if err != nil {
        return nil, err
    }
    if a.vlog != nil {
        s.LogFunc = a.vlog.Log
    }
    return s, nil
}
```

All commands that previously called `store.NewStore(tickDir)` directly are refactored to use `a.openStore(tickDir)` instead (9 files changed: blocked.go, create.go, dep.go, list.go, ready.go, show.go, transition.go, update.go plus the store import removal).

**Instrumentation points** (24 `s.vlog` calls in V4 store.go):
- Mutate: acquiring exclusive lock, lock acquired, lock released (defer), checking cache freshness via hash comparison, cache is fresh, atomic write to path, atomic write complete, rebuilding cache with new hash, cache rebuild complete.
- Rebuild: acquiring exclusive lock, lock acquired, lock released, deleting existing cache.db, reading JSONL, parsed N tasks, rebuilding cache with N tasks, cache rebuild complete/hash updated.
- Query: acquiring shared lock, shared lock acquired, shared lock released (defer), checking cache freshness via hash comparison, cache is fresh.

The store's lock acquire/release was refactored from `defer fl.Unlock()` to `defer func() { fl.Unlock(); s.vlog("...released") }()` to inject the release log.

### Code Quality

- **VerboseLogger** is 28 lines, minimal and clean. Single method `Log` combines `Sprintf` + `Fprintf` -- slightly inefficient (two allocations) but perfectly adequate.
- The public `LogFunc` field on `Store` is a simple but somewhat unidiomatic Go approach. Public mutable function fields create a wider API surface than necessary and can lead to accidental nil dereferences if `vlog()` guard is removed.
- The `vlog()` helper on `Store` properly guards against nil `LogFunc`.
- `formatName()` is a clean switch helper in `format.go`.
- The `openStore()` centralization is good refactoring -- eliminates the `store` import from 8 files.
- Error wrapping in store.go uses `%w` consistently.
- Lock messages include full paths (e.g., `"acquiring exclusive lock on %s"` with `s.lockPath`) -- useful for debugging.

### Test Coverage

V4 has **7 test functions** in `cli/verbose_test.go`, all integration-level tests running through the full `App.Run()`:

1. `TestVerbose_WritesToStderr` -- checks "verbose:" present + lock/hash/cache/format keywords with loose assertions (`strings.Contains` on individual keywords).
2. `TestVerbose_WritesNothingWhenOff` -- asserts `stderr.String() != ""` produces no output.
3. `TestVerbose_DoesNotWriteToStdout` -- checks stdout doesn't contain "verbose:".
4. `TestVerbose_AllowsQuietPlusVerbose` -- checks quiet suppresses format headers but verbose still writes stderr.
5. `TestVerbose_WorksWithEachFormatFlag` -- loops over `--toon`, `--pretty`, `--json`; checks no verbose on stdout, has verbose on stderr.
6. `TestVerbose_CleanPipedOutput` -- 2 tasks, checks no verbose lines leaked to stdout, expects 3 stdout lines (TOON header + 2 data).
7. `TestVerbose_AllLinesPrefixed` -- iterates all stderr lines and asserts `HasPrefix "verbose:"`.

**No unit tests for VerboseLogger itself.** No tests at the store level specifically for verbose instrumentation. All tests are integration tests through the full CLI path.

Test setup uses manual task construction with explicit `time.Date(2026, 1, 19, ...)`, `Created`, `Updated` fields.

### Spec Compliance

| Acceptance Criterion | Met? | Notes |
|---|---|---|
| VerboseLogger writes stderr only when Verbose true | Yes | Guard check in `Log()` |
| Key operations instrumented (cache, lock, hash, write, format) | Yes | 24 vlog calls in store, 1 in cli for format |
| All lines `verbose:` prefixed | Yes | Enforced by `Log()` method and tested |
| Zero verbose on stdout | Yes | VerboseLogger writes to Stderr only |
| --quiet + --verbose works correctly | Yes | Tested; orthogonal streams |
| Piping captures only formatted output | Yes | Tested with line count assertion |
| No output when verbose off | Yes | Tested |

V4 fully satisfies all acceptance criteria. The instrumentation is thorough -- 24 store log points cover lock, hash, cache, atomic write, and the format resolution log in CLI.

### golang-pro Skill Compliance

| Rule | Compliance | Notes |
|---|---|---|
| Handle all errors explicitly | Pass | All error paths handled |
| Write table-driven tests with subtests | Partial | Tests use subtests but are not table-driven; each test is a separate function with a single `t.Run` |
| Document all exported functions/types | Pass | `VerboseLogger`, `NewVerboseLogger`, `Log` all documented |
| Propagate errors with `fmt.Errorf("%w", err)` | Pass | Consistent in store.go |
| No panic for normal error handling | Pass | |
| No ignored errors without justification | Partial | `fl.Unlock()` return value ignored in defer (acceptable for file locks) |

---

## V5 Implementation

### Architecture & Design

V5 places the `VerboseLogger` in `internal/engine/verbose.go` (package `engine`), not in `cli`. It provides **two methods**: `Log(msg string)` for plain messages and `Logf(format string, args ...interface{})` for formatted ones. This is a more Go-idiomatic API separation (compare to `log.Print` vs `log.Printf`).

**Wiring approach**: V5 uses the **functional options pattern**. `Store` has a private `verbose *VerboseLogger` field. A new `WithVerbose(vl *VerboseLogger) Option` is provided. The store constructor initializes a **default no-op logger**: `verbose: NewVerboseLogger(nil, false)` (line 82), then applies options. This means the store never needs nil guards -- the verbose field is always non-nil with a safe no-op default.

The CLI's `Context` struct gains two helper methods:

```go
func (c *Context) newVerboseLogger() *engine.VerboseLogger {
    return engine.NewVerboseLogger(c.Stderr, c.Verbose)
}

func (c *Context) storeOpts() []engine.Option {
    return []engine.Option{
        engine.WithVerbose(c.newVerboseLogger()),
    }
}
```

All commands use `engine.NewStore(tickDir, ctx.storeOpts()...)` -- the verbose logger is threaded through via the options pattern. Format resolution is logged conditionally:

```go
if ctx.Verbose {
    vl := ctx.newVerboseLogger()
    formatName := formatLabel(ctx.Format)
    vl.Logf("format resolved: %s", formatName)
}
```

**Instrumentation points** (13 `s.verbose.Log`/`Logf` calls in V5 store.go):
- `acquireExclusive()`: "lock acquired (exclusive)" + "lock released" in unlock closure.
- `acquireShared()`: "lock acquired (shared)" + "lock released" in unlock closure.
- `ensureFresh()`: "cache freshness check", then either "cache is fresh" or "cache rebuild (stale or missing)".
- `Mutate()`: "atomic write to tasks.jsonl".
- `Rebuild()`: "deleting existing cache", "reading tasks.jsonl", "rebuilding cache (N tasks)" via `Logf`, "hash updated".

V5 has fewer instrumentation points than V4 (13 vs 24). Notably, V5 does NOT log "acquiring lock" (only "lock acquired"), does not log "atomic write complete", does not log "cache rebuild complete" after mutation, and does not include file paths in lock messages.

The lock acquire/release is cleanly factored into `acquireExclusive()` / `acquireShared()` helper methods (pre-existing in V5's architecture), making the verbose instrumentation naturally centralized rather than duplicated.

### Code Quality

- **VerboseLogger** is 37 lines with two methods (`Log` and `Logf`). The split mirrors Go stdlib's `log.Print`/`log.Printf` convention -- more idiomatic.
- The **default no-op pattern** (`verbose: NewVerboseLogger(nil, false)`) eliminates all nil checks. This is cleaner than V4's `if s.LogFunc != nil` guard.
- The **functional options pattern** (`WithVerbose`) is considered best-practice Go for optional configuration. It avoids public mutable fields.
- `VerboseLogger` lives in `engine` package, making it available to the storage layer without a cross-package dependency on `cli`. This is better architectural layering -- `cli` depends on `engine`, not the reverse.
- Lock messages distinguish "(exclusive)" vs "(shared)" -- useful for diagnosing concurrent access issues.
- The format resolution log is gated by `if ctx.Verbose` before even creating the logger. This is slightly redundant (the logger gates internally) but avoids creating a throwaway logger object.

### Test Coverage

V5 has **three test files**:

1. **`engine/verbose_test.go`** (5 subtests) -- **unit tests for VerboseLogger**:
   - "it writes to writer with verbose prefix when verbose is true"
   - "it writes nothing when verbose is false"
   - "it supports formatted output" (tests `Logf`)
   - "it writes nothing for formatted output when verbose is false"
   - "it handles nil writer gracefully when verbose is false" -- important edge case preventing panics

2. **`engine/store_verbose_test.go`** (5 subtests) -- **store-level integration tests**:
   - "it logs lock/cache/hash/write operations when verbose is on" (Mutate) -- checks exact phrases like `"verbose: lock acquired (exclusive)"`, `"verbose: cache freshness check"`, `"verbose: cache rebuild"`, `"verbose: atomic write"`, `"verbose: lock released"`
   - "it logs lock/cache/hash operations during query when verbose is on" (Query)
   - "it writes nothing when verbose is off"
   - "it logs hash comparison on freshness check" -- runs two queries, resets buffer between them, verifies second shows "cache is fresh"
   - "it prefixes all lines with verbose:" -- iterates lines asserting prefix

3. **`cli/verbose_test.go`** (7 subtests) -- **end-to-end CLI integration tests**:
   - "it writes cache/lock/hash/format verbose to stderr" -- checks exact phrases
   - "it writes nothing to stderr when verbose off"
   - "it does not write verbose to stdout"
   - "it allows quiet + verbose simultaneously" -- asserts quiet output is exactly the ID
   - "it works with each format flag without contamination" -- includes format-specific validators (TOON checks `"tasks["`, Pretty checks `"ID"`, JSON checks `"["`)
   - "it produces clean piped output with verbose enabled" -- checks no verbose leak + task IDs present
   - "it logs verbose during mutation commands" -- tests `create` command, checks exclusive lock + atomic write logs

Total: **17 test cases** across 3 files, spanning unit, store integration, and CLI integration layers.

### Spec Compliance

| Acceptance Criterion | Met? | Notes |
|---|---|---|
| VerboseLogger writes stderr only when Verbose true | Yes | Guard check in `Log()`/`Logf()` |
| Key operations instrumented (cache, lock, hash, write, format) | Yes | 13 store calls + 1 CLI format log |
| All lines `verbose:` prefixed | Yes | Enforced by both `Log()` and `Logf()` methods, tested at all 3 layers |
| Zero verbose on stdout | Yes | VerboseLogger writes to Stderr only |
| --quiet + --verbose works correctly | Yes | Tested with exact ID assertion |
| Piping captures only formatted output | Yes | Tested |
| No output when verbose off | Yes | Tested at unit, store, and CLI levels |

V5 fully satisfies all acceptance criteria. Slightly fewer instrumentation points than V4 but all spec-required categories are covered.

### golang-pro Skill Compliance

| Rule | Compliance | Notes |
|---|---|---|
| Handle all errors explicitly | Pass | All error paths handled |
| Write table-driven tests with subtests | Partial | Format flag test uses a struct slice (table-driven); other tests are individual subtests |
| Document all exported functions/types | Pass | `VerboseLogger`, `NewVerboseLogger`, `Log`, `Logf`, `WithVerbose` all documented |
| Propagate errors with `fmt.Errorf("%w", err)` | Pass | |
| No panic for normal error handling | Pass | Nil writer edge case explicitly tested |
| No ignored errors without justification | Pass | `fl.Unlock()` ignored via `_ =` (explicit) |

---

## Comparative Analysis

### Where V4 is Better

1. **More thorough instrumentation**. V4 has 24 verbose log calls in the store vs V5's 13. V4 logs "acquiring" the lock (before attempt) AND "acquired" (after success), while V5 only logs "acquired". V4 logs "atomic write complete" as a separate event. V4 includes full file paths in lock messages (`"acquiring exclusive lock on %s", s.lockPath`), which is more useful for debugging in environments with multiple tick directories. V4 logs "rebuilding cache with new hash" and "cache rebuild complete" as distinct events in the Mutate flow, giving finer-grained visibility.

2. **More verbose log detail**. V4 messages like `"checking cache freshness via hash comparison"` are more descriptive than V5's `"cache freshness check"`. V4's `"parsed %d tasks from JSONL"` in Rebuild provides quantitative data. V4's `"acquiring exclusive lock on /path/..."` makes it clear WHICH lock file is being targeted.

### Where V5 is Better

1. **Significantly better test coverage**. V5 has 17 test cases across 3 layers (unit, store, CLI) vs V4's 7 integration-only tests. V5 unit-tests the `VerboseLogger` in isolation including the nil-writer edge case. V5 has dedicated store-level verbose tests that verify instrumentation independent of CLI wiring. V5's CLI tests include a mutation test (`create` command) that V4 lacks, verifying verbose works for write operations not just reads.

2. **Better architectural design**. The functional options pattern (`WithVerbose`) is canonical Go for optional configuration. It avoids exposing mutable public fields. The default no-op logger (`NewVerboseLogger(nil, false)`) eliminates nil checks entirely -- the store never needs to guard against a nil verbose logger.

3. **Better package layering**. `VerboseLogger` lives in `engine` (the storage layer package), not in `cli`. This means the storage layer owns its own logging abstraction with no reverse dependency on the CLI package. V4 puts `VerboseLogger` in `cli` and passes a `func` into the store -- the store accepts any `func(string, ...interface{})`, which is flexible but loosely typed.

4. **More idiomatic Go API**. Separate `Log(msg)` and `Logf(format, args...)` methods mirror `log.Print`/`log.Printf`. V4's single `Log(format, args...)` is less explicit about whether formatting is expected.

5. **Stronger test assertions**. V5's CLI tests check exact verbose phrases like `"verbose: lock acquired (exclusive)"` rather than V4's loose keyword checks (`strings.Contains(stderrStr, "lock")`). V5's quiet+verbose test asserts the exact stdout content (`"tick-aaaaaa"`) rather than just checking headers are absent. V5's format-flag test includes format-specific validators that verify the actual format output is correct.

6. **Lock type differentiation**. V5's messages distinguish `"lock acquired (exclusive)"` from `"lock acquired (shared)"`, which is valuable for debugging concurrent access issues. V4 logs `"exclusive lock acquired"` and `"shared lock acquired"` which is functionally equivalent but less consistent in format.

7. **Hash comparison detail in tests**. V5's `store_verbose_test.go` has a test that runs two queries (first triggers rebuild, second shows fresh) and verifies the hash comparison flow, demonstrating the cache freshness path more rigorously.

### Differences That Are Neutral

1. **Package naming**: V4 uses `internal/store` while V5 uses `internal/engine` for the storage layer. This is a pre-existing architectural difference unrelated to this task.

2. **Verbose prefix format**: V4 uses `"verbose: "` (with space after colon). V5 also uses `"verbose: "`. Both comply with the spec's `verbose:` prefix requirement.

3. **Format resolution logging location**: V4 logs unconditionally via `a.vlog.Log(...)` (the no-op handles disabled state). V5 gates with `if ctx.Verbose` before creating a logger. Both produce identical behavior.

4. **Test helper functions**: V4 uses `setupInitializedDirWithTasks(t, tasks)` with manual Task construction. V5 uses `initTickProjectWithTasks(t, []task.Task{tk})` with `task.NewTask()`. Different naming but equivalent functionality.

---

## Verdict

**V5 wins.**

The decisive factors are:

1. **Test quality is substantially better**. V5's 17 tests across 3 layers (unit, store, CLI) vs V4's 7 CLI-only tests represents a meaningful coverage advantage. The unit tests for `VerboseLogger` (including nil-writer safety) and dedicated store-level verbose tests catch bugs that V4's integration-only approach could miss. V5's mutation-path test (`create` command) covers a code path V4 never tests.

2. **The functional options pattern is the right Go idiom**. V4's public `LogFunc` field is functional but not idiomatic. V5's `WithVerbose(vl)` option is the standard Go approach for optional configuration, is used consistently with the pre-existing `WithLockTimeout` option, and avoids exposing mutable state on the Store type.

3. **The default no-op logger is a superior design**. By initializing `verbose: NewVerboseLogger(nil, false)` in the constructor, V5 eliminates all nil guards. V4 requires the `vlog()` helper to check `if s.LogFunc != nil` every call. This is a small but meaningful reduction in defensive code.

4. **Package placement is architecturally cleaner**. Putting `VerboseLogger` in `engine` rather than `cli` means the storage layer owns its logging contract. V4's approach of passing a bare function from `cli` to `store` works but creates an implicit coupling with no type safety.

V4's advantage in instrumentation thoroughness (24 vs 13 log points) is real but modest. The additional log points (e.g., "acquiring" before lock attempt, "complete" after writes) provide marginally more debugging detail. However, V5 covers all spec-required categories and the extra V4 points are nice-to-have rather than spec-required. The architectural and testing advantages of V5 outweigh this difference.
