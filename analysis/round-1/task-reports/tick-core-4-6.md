# Task tick-core-4-6: Verbose output & edge case hardening

## Task Summary

Implement `--verbose` / `-v` across all commands. Debug detail (cache rebuild, lock, hash comparison) to stderr only. Piped output stays clean. Quiet + verbose = silent stdout, debug stderr.

Requires a `VerboseLogger` wrapping `fmt.Fprintf(os.Stderr, ...)` that is a no-op when verbose is off. Instrumented operations: freshness detection, cache rebuild, lock acquire/release, atomic write, format resolution. All lines prefixed `verbose:` for grep-ability. Quiet + verbose are orthogonal (different streams).

### Acceptance Criteria

1. VerboseLogger writes stderr only when Verbose true
2. Key operations instrumented (cache, lock, hash, write, format)
3. All lines `verbose:` prefixed
4. Zero verbose on stdout
5. `--quiet` + `--verbose` works correctly
6. Piping captures only formatted output
7. No output when verbose off

## Acceptance Criteria Compliance

| Criterion | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| VerboseLogger writes stderr only when Verbose true | PASS -- `VerboseLogger` struct with `enabled` bool, no-op when false | PASS -- `VerboseLogger` struct with `enabled` bool, no-op when false | PARTIAL -- No dedicated `VerboseLogger` type; reuses existing `WriteVerbose` method on `App` with prefix change from `[verbose]` to `verbose:` |
| Key operations instrumented (cache, lock, hash, write, format) | PASS -- lock acquire/release, cache fresh/stale, atomic write via `store.logf`; format resolution in `cli.go` | PASS -- lock acquire/release (including release in defer), cache freshness, atomic write, format resolved; all via `store.logVerbose` and `app.logVerbose` | PASS -- All commands manually annotated with `WriteVerbose` calls for store open, lock acquire, cache freshness, atomic write, lock release |
| All lines `verbose:` prefixed | PASS -- `fmt.Fprintf(v.w, "verbose: "+format+"\n", args...)` | PASS -- `fmt.Fprintf(v.w, "verbose: %s\n", msg)` | PASS -- `fmt.Fprintf(a.Stderr, "verbose: %s\n", msg)` |
| Zero verbose on stdout | PASS -- Logger writes to `a.stderr` only | PASS -- Logger writes to `a.stderr` only | PASS -- `WriteVerbose` writes to `a.Stderr` only |
| `--quiet` + `--verbose` works correctly | PASS -- Tested: quiet gives IDs on stdout, verbose still on stderr | PASS -- Tested: quiet gives IDs on stdout, verbose still on stderr | PASS -- Tested: quiet gives IDs on stdout, verbose still on stderr |
| Piping captures only formatted output | PASS -- Tested: stdout has no `verbose:` lines | PASS -- Tested: stdout has no `verbose:` lines | PASS -- Tested: stdout has no `verbose:` lines |
| No output when verbose off | PASS -- Tested: no `verbose:` on stderr when flag absent | PASS -- Tested: no `verbose:` on stderr when flag absent | PASS -- Tested: no `verbose:` on stderr when flag absent |

## Implementation Comparison

### Approach

All three versions achieve the same goal -- verbose debug logging to stderr with `verbose:` prefix -- but with notably different architectural strategies.

#### V1: Dedicated type + Store injection via `LogFunc`

V1 creates a new `VerboseLogger` struct in `internal/cli/verbose.go` (27 LOC) and a `LogFunc` type alias in `internal/storage/store.go`. The logger is variadic-format-capable:

```go
// internal/cli/verbose.go (V1)
func (v *VerboseLogger) Log(format string, args ...any) {
    if !v.enabled {
        return
    }
    fmt.Fprintf(v.w, "verbose: "+format+"\n", args...)
}
```

The `Store` receives logging via a function type:

```go
// internal/storage/store.go (V1)
type LogFunc func(format string, args ...any)

func (s *Store) SetLogger(fn LogFunc) {
    s.log = fn
}

func (s *Store) logf(format string, args ...any) {
    if s.log != nil {
        s.log(format, args...)
    }
}
```

The wiring happens through a new `openStore` helper on `App`:

```go
// internal/cli/cli.go (V1)
func (a *App) openStore(tickDir string) (*storage.Store, error) {
    store, err := storage.NewStore(tickDir)
    if err != nil {
        return nil, fmt.Errorf("opening store: %w", err)
    }
    if a.verbose != nil {
        store.SetLogger(a.verbose.Log)
    }
    return store, nil
}
```

All command handlers (`create.go`, `dep.go`, `list.go`, `ready.go`, `transition.go`, `update.go`) are refactored to call `a.openStore(tickDir)` instead of `storage.NewStore(tickDir)`, and the `storage` import is removed from each. This is a clean centralization of store creation.

Format resolution is logged conditionally:

```go
// internal/cli/cli.go (V1)
formatName := [...]string{"toon", "pretty", "json"}[format]
if a.opts.Toon || a.opts.Pretty || a.opts.JSON {
    a.verbose.Log("format=%s (flag override)", formatName)
} else {
    a.verbose.Log("format=%s (auto-detected, tty=%v)", formatName, a.isTTY)
}
```

#### V2: Dedicated type + Store injection via `Logger` interface

V2 also creates a `VerboseLogger` in `internal/cli/verbose.go` (29 LOC) but uses a simpler string-only signature:

```go
// internal/cli/verbose.go (V2)
func (v *VerboseLogger) Log(msg string) {
    if !v.enabled {
        return
    }
    fmt.Fprintf(v.w, "verbose: %s\n", msg)
}
```

Critically, V2 defines a proper Go **interface** in the storage package:

```go
// internal/storage/store.go (V2)
type Logger interface {
    Log(msg string)
}

func (s *Store) SetLogger(l Logger) {
    s.logger = l
}

func (s *Store) logVerbose(msg string) {
    if s.logger != nil {
        s.logger.Log(msg)
    }
}
```

This is the most Go-idiomatic approach: the `Store` depends on a small interface, and `VerboseLogger` satisfies it implicitly. The `App` wires them through a `newStore` helper (similar to V1's `openStore`):

```go
// internal/cli/app.go (V2)
func (a *App) newStore(tickDir string) (*storage.Store, error) {
    store, err := storage.NewStore(tickDir)
    if err != nil {
        return nil, err
    }
    if a.verbose != nil {
        store.SetLogger(a.verbose)
    }
    return store, nil
}
```

V2 also has a `logVerbose` fallback method on `App` for pre-initialization safety:

```go
// internal/cli/app.go (V2)
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

V2's store instrumentation is the most comprehensive, instrumenting **both** `Mutate` and `Query` paths inside `store.go` itself. Lock release is logged via defer:

```go
// internal/storage/store.go (V2, Mutate)
s.logVerbose("lock: acquiring exclusive lock")
// ...
s.logVerbose("lock: exclusive lock acquired")
defer func() {
    fl.Unlock()
    s.logVerbose("lock: exclusive lock released")
}()
```

Format resolution logging:

```go
// internal/cli/app.go (V2)
a.verbose.Log(fmt.Sprintf("format resolved: %s", a.FormatCfg.Format))
```

#### V3: No new type -- reuses existing `WriteVerbose` with prefix fix

V3 takes a fundamentally different approach: **no new `VerboseLogger` type** and **no changes to `store.go`**. Instead, it:

1. Changes the existing `WriteVerbose` method's prefix from `[verbose]` to `verbose:`:

```go
// internal/cli/cli.go (V3)
func (a *App) WriteVerbose(format string, args ...interface{}) {
    if !a.formatConfig.Verbose {
        return
    }
    msg := fmt.Sprintf(format, args...)
    fmt.Fprintf(a.Stderr, "verbose: %s\n", msg)
}
```

2. Manually adds `WriteVerbose` calls in **every command handler** at the CLI layer:

```go
// internal/cli/create.go (V3)
a.WriteVerbose("store open %s", tickDir)
// ...
a.WriteVerbose("lock acquire exclusive")
a.WriteVerbose("cache freshness check")
err = store.Mutate(func(tasks []task.Task) ([]task.Task, error) {
    // ...
})
a.WriteVerbose("atomic write complete")
a.WriteVerbose("lock release")
```

This is repeated for `list.go`, `ready.go`, `show.go`, `blocked.go`, `transition.go`, `update.go`, and `dep.go` (both `runDepAdd` and `runDepRm`).

V3 also updates `format_test.go` to fix 4 existing test assertions that referenced the old `[verbose]` prefix.

### Code Quality

#### Go Idioms

**V2 is the most idiomatic**. It defines a small `Logger` interface in the storage package, which the `VerboseLogger` satisfies implicitly. This follows the Go convention of "accept interfaces, return structs" and enables testability/mockability.

**V1 uses a function type** (`LogFunc func(format string, args ...any)`), which is acceptable in Go but less discoverable than an interface. It does have the advantage of variadic formatting at the log call site (`s.logf("cache stale, rebuilding (%d tasks)", len(tasks))`).

**V3 uses no abstraction** at the storage layer. Logging calls are sprinkled in the CLI layer only. This means the store operations (lock acquire, cache check, atomic write) are logged **before and after** the call, not **inside** the actual operation. The verbose messages are "aspirational" -- they log "lock acquire exclusive" before `store.Mutate()` is called, but the lock acquisition actually happens inside `Mutate`. If `Mutate` fails to acquire the lock, the verbose output would be misleading.

#### Naming

- V1: `VerboseLogger.Log(format, args...)`, `store.logf(format, args...)`, `openStore` -- clear, consistent
- V2: `VerboseLogger.Log(msg)`, `store.logVerbose(msg)`, `newStore`, `logVerbose` on App -- clear, consistent
- V3: `WriteVerbose(format, args...)` on App -- reuses existing naming convention; no new types

#### Error Handling

V1 wraps the store creation error: `fmt.Errorf("opening store: %w", err)`. All command handlers become simpler: `return err` instead of `return fmt.Errorf("opening store: %w", err)`.

V2 does not wrap the error in `newStore`: `return nil, err`. Command handlers remain `return err`.

V3 does not change store creation at all -- each command still calls `storage.NewStore(tickDir)` directly.

#### DRY

V1 and V2 centralize store creation into a single helper (`openStore`/`newStore`), eliminating duplicate `storage.NewStore` + `SetLogger` calls across 6-7 handlers. V1 additionally removes the `storage` import from each handler file.

V3 is the least DRY. It manually adds 4-6 `WriteVerbose` calls per command handler, duplicating nearly identical verbose logging patterns across 8 files. The "store open", "lock acquire", "cache freshness check", "atomic write complete", "lock release" sequence is copy-pasted. There is no centralization.

#### Type Safety

V2's interface approach is the strongest for type safety and future evolution (easy to swap logger implementations). V1's function type is also strongly typed. V3's approach has no type-level contracts.

### Test Quality

#### V1 Test Functions (162 LOC, file: `internal/cli/verbose_test.go`)

Unit tests (`TestVerboseLogger`):
1. `"writes to writer when enabled"` -- confirms `verbose: test message hello` appears in buffer
2. `"writes nothing when disabled"` -- confirms zero output when `enabled=false`
3. `"prefixes all lines with verbose:"` -- two Log calls, splits on newline, checks each prefix

Integration tests (`TestVerboseIntegration`):
4. `"writes verbose to stderr not stdout"` -- creates task, runs `--verbose list`, checks stderr has `verbose:`, stdout does not
5. `"writes nothing to stderr when verbose off"` -- runs without `--verbose`, checks no `verbose:` on stderr
6. `"allows quiet + verbose simultaneously"` -- `--quiet --verbose list`, verifies stderr has `verbose:`, stdout has only IDs (`tick-` prefix)
7. `"verbose with --json keeps stdout clean"` -- `--verbose --json list`, checks no `verbose:` on stdout, JSON array present, stderr has `verbose:`
8. `"logs format resolution"` -- checks stderr contains `format=`
9. `"logs lock and cache operations"` -- checks stderr contains `lock` and `cache`
10. `"piped output is clean with verbose"` -- checks no line in stdout starts with `verbose:`

**Total: 10 test functions (3 unit + 7 integration)**

#### V2 Test Functions (259 LOC, file: `internal/cli/verbose_test.go`)

Unit tests (`TestVerboseLogger`):
1. `"it writes verbose-prefixed messages when enabled"` -- checks `verbose:` and message content
2. `"it writes nothing when disabled"` -- checks empty output
3. `"it prefixes every line with verbose:"` -- two messages, checks each line prefix

Integration tests (`TestVerboseOutput`):
4. `"it writes cache/lock/hash/format verbose to stderr"` -- sets up JSONL directly, runs `--verbose list`, checks for `verbose:`, `format`, `lock`, `freshness`, `cache` keywords
5. `"it writes nothing to stderr when verbose off"` -- runs without `--verbose`, checks no `verbose:`
6. `"it does not write verbose to stdout"` -- `--verbose list`, confirms no `verbose:` on stdout
7. `"it allows quiet + verbose simultaneously"` -- `--quiet --verbose list`, checks IDs only on stdout, `verbose:` on stderr
8. `"it works with each format flag without contamination"` -- sub-tests for `--toon`, `--pretty`, `--json`; each checks stdout clean, stderr has `verbose:`
9. `"it produces clean piped output with verbose enabled"` -- two tasks, `--verbose --toon list`, checks no `verbose:` on stdout, task data present
10. `"it logs verbose for mutations too"` -- runs `--verbose create "Test task"`, checks `verbose:` on stderr, mentions `lock` and `write`

**Total: 10 test functions (3 unit + 7 integration), but test 8 has 3 sub-tests, so effectively 12 test cases**

#### V3 Test Functions (253 LOC, file: `internal/cli/verbose_test.go`)

All integration tests (no unit tests -- there is no separate `VerboseLogger` to unit test):
1. `TestVerbose_WritesDebugToStderr` -- `--verbose --pretty list`, checks `verbose:`, `format`, `lock`, `store`, `cache` on stderr
2. `TestVerbose_WritesNothingWhenOff` -- no `--verbose`, confirms no `verbose:` on stderr
3. `TestVerbose_NeverWritesToStdout` -- `--verbose --pretty list`, confirms no `verbose:` on stdout
4. `TestVerbose_QuietPlusVerboseSimultaneously` -- `--quiet --verbose --pretty list`, checks IDs on stdout, `verbose:` on stderr
5. `TestVerbose_WorksWithEachFormatFlag` -- sub-tests for `--toon` (checks `tasks[`), `--pretty` (checks `ID`), `--json` (checks `"id"`); all verify no `verbose:` on stdout, `verbose:` on stderr
6. `TestVerbose_CleanPipedOutput` -- two tasks, `--verbose list`, checks no `verbose:` lines on stdout
7. `TestVerbose_AllLinesPrefixed` -- `--verbose --pretty list`, iterates all stderr lines, checks each starts with `verbose:`
8. `TestVerbose_CreateCommandInstrumented` -- `--verbose --pretty create "Test task"`, checks `verbose:` and `store` on stderr, no `verbose:` on stdout

**Total: 8 top-level test functions, but test 5 has 3 sub-tests, so effectively 10 test cases**

Additionally, V3 updates 4 assertions in `internal/cli/format_test.go` to match the new `verbose:` prefix (changing from `[verbose]` checks).

#### Test Coverage Gaps

| Edge Case | V1 | V2 | V3 |
|-----------|-----|-----|-----|
| VerboseLogger unit: enabled writes | YES | YES | N/A (no type) |
| VerboseLogger unit: disabled silent | YES | YES | N/A (no type) |
| VerboseLogger unit: prefix check | YES | YES | N/A (no type) |
| Integration: stderr has verbose | YES | YES | YES |
| Integration: stderr silent when off | YES | YES | YES |
| Integration: stdout never has verbose | YES | YES | YES |
| Integration: quiet + verbose | YES | YES | YES |
| Integration: each format flag clean | JSON only | toon+pretty+json | toon+pretty+json |
| Integration: piped output clean | YES | YES | YES |
| Integration: format resolution logged | YES | YES | YES (in WritesDebugToStderr) |
| Integration: lock/cache logged | YES | YES | YES |
| Integration: mutation commands | NO | YES (create) | YES (create) |
| All verbose lines prefixed | Unit only | Unit only | Integration-level |
| `--verbose` prefix is exactly `verbose: ` | YES | YES | YES |

**V1 gap**: Does not test mutation commands (create/update/transition) with verbose. Only tests read path (list).

**V2 gap**: None significant. Has the most comprehensive coverage.

**V3 gap**: No unit tests for the verbose mechanism itself (since there is no separate type). Relies entirely on integration tests. Also, the format_test.go changes are fixing pre-existing tests, not adding new ones.

## Diff Stats

| Metric | V1 | V2 | V3 |
|--------|-----|-----|-----|
| Files changed (impl) | 10 | 10 | 12 (14 total, 2 docs) |
| Lines added (impl) | 260 | 363 | 304 |
| Lines removed (impl) | 25 | 15 | 5 |
| Impl LOC (new/changed) | 27 (verbose.go) + ~71 (cli.go+store.go changes) = ~98 | 29 (verbose.go) + ~68 (app.go+store.go changes) = ~97 | ~51 (cli changes across 9 files) |
| Test LOC | 162 | 259 | 253 (+8 format_test.go fixes) |
| Test functions | 10 (3 unit + 7 integration) | 10 (3 unit + 7 integration, 12 cases) | 8 (all integration, 10 cases) |

## Verdict

**V2 is the best implementation.**

**Architecture**: V2 uses a Go interface (`Logger`) for the store-to-logger contract, which is the most idiomatic Go pattern. The `VerboseLogger` satisfies this interface implicitly, and any future logger could be swapped in. V1's function type is acceptable but less discoverable. V3 has no abstraction at the storage layer.

**Correctness of instrumentation**: V2 and V1 place logging calls **inside** `store.go`, meaning the verbose messages accurately reflect what is happening (e.g., `"lock: exclusive lock acquired"` is logged immediately after the lock is actually acquired). V3 places all logging in the CLI layer **outside** the store calls, which means messages like `"lock acquire exclusive"` are logged before `store.Mutate()` is called, not when the lock is actually acquired. This is semantically misleading -- if the lock acquisition fails, the user still sees `"lock acquire exclusive"` in verbose output.

**DRY**: V1 and V2 centralize store creation into a helper method, eliminating duplicate code. V3 copy-pastes 4-6 `WriteVerbose` calls across 8+ command handlers with identical patterns.

**Test coverage**: V2 has the most comprehensive tests (259 LOC), including mutation commands, all three format flags, and unit tests. V1 is close but lacks mutation command testing. V3 is adequate but lacks unit tests due to having no standalone type.

**Defensive coding**: V2's `logVerbose` fallback on `App` (which handles the pre-initialization case) shows extra care. V2 also logs lock **release** via `defer`, which V1 does not.

V1 is a close second -- it has clean architecture and good test coverage. The `LogFunc` approach is slightly less Go-idiomatic than V2's interface but perfectly functional. V3 is the weakest due to the lack of storage-layer integration and the repetitive manual instrumentation pattern.
