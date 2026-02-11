# Task tick-core-4-1: Formatter abstraction & TTY-based format selection

## Task Summary

Define a `Formatter` interface, implement TTY detection, resolve format from flags vs auto-detection with conflict handling, and wire the chosen formatter into CLI dispatch. This is the foundation for tasks 4-2 through 4-4 (concrete Toon, Pretty, and JSON formatters).

**Required elements:**
- `Formatter` interface with methods: `FormatTaskList`, `FormatTaskDetail`, `FormatTransition`, `FormatDepChange`, `FormatStats`, `FormatMessage`
- `Format` enum: `FormatToon`, `FormatPretty`, `FormatJSON`
- `DetectTTY()`: `os.Stdout.Stat()` -> check `ModeCharDevice`. Stat failure -> default non-TTY.
- `ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY)`: >1 flag -> error; 1 flag -> that format; 0 flags + TTY -> Pretty; 0 flags + no TTY -> Toon
- `FormatConfig` struct: Format, Quiet, Verbose -- passed to all handlers
- Stub formatter as placeholder (concrete formatters in 4-2 through 4-4)
- `--verbose` to stderr only, never contaminates stdout

**Acceptance Criteria:**
1. Formatter interface covers all command output types
2. Format enum with 3 constants
3. TTY detection works correctly
4. ResolveFormat handles all flag/TTY combos
5. Conflicting flags -> error
6. FormatConfig wired into CLI dispatch
7. Verbose to stderr only
8. Stat failure handled gracefully

## Acceptance Criteria Compliance

| Criterion | V5 | V6 |
|-----------|-----|-----|
| Formatter interface covers all command output types | PASS -- interface at `format.go:12-25` has all 6 methods (`FormatTaskList`, `FormatTaskDetail`, `FormatTransition`, `FormatDepChange`, `FormatStats`, `FormatMessage`) | PASS -- interface at `format.go:116-129` has all 6 methods with matching names |
| Format enum with 3 constants | PASS -- `OutputFormat` type with `FormatToon`, `FormatPretty`, `FormatJSON` at `cli.go:14-23` | PASS -- `Format` type with `FormatToon`, `FormatPretty`, `FormatJSON` at `format.go:13-19` |
| TTY detection works correctly | PASS -- `DetectTTY` at `format.go:37-43` checks `ModeCharDevice`, returns false on error | PASS -- `DetectTTY` at `format.go:24-30` checks `ModeCharDevice`, returns false on error |
| ResolveFormat handles all flag/TTY combos | PASS -- `ResolveFormat` at `format.go:48-77` handles all combos, returns error on conflict | PASS -- `ResolveFormat` at `format.go:34-62` handles all combos, returns error on conflict |
| Conflicting flags -> error | PASS -- counts flags, returns error if count > 1 | PASS -- counts flags, returns error if count > 1 |
| FormatConfig wired into CLI dispatch | PASS -- `FormatCfg()` method on `Context` at `cli.go:38-44`, format resolved in `parseArgs` at `cli.go:157-163` | PASS -- `NewFormatConfig` at `format.go:74-84` called in `App.Run` at `app.go:30-34`, passed to every handler |
| Verbose to stderr only | PASS -- verbose logging via `VerboseLogger` through `engine.NewVerboseLogger(c.Stderr, c.Verbose)`, tested in `verbose_test.go` | PASS -- verbose via `VerboseLogger` at `verbose.go:12-28`, writes to stderr, tested extensively in `verbose_test.go` |
| Stat failure handled gracefully | PASS -- `DetectTTY` returns false on stat error (line 39-40) | PASS -- `DetectTTY` returns false on stat error (line 26-28) |

## Implementation Comparison

### Approach

**V5** keeps the existing `Run()` free-function architecture. It:
1. Moves `isTerminal` from `cmd/tick/main.go` to `format.go` as the exported `DetectTTY`.
2. Keeps the `OutputFormat` type and enum constants in `cli.go` (unchanged from prior task).
3. Extracts format resolution from inline logic in `parseArgs` to a new `ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY bool)` function.
4. Adds a `FormatConfig` struct and a `FormatCfg()` method on `*Context`.
5. Adds a `Formatter` interface using `io.Writer` + typed data structs as method parameters.
6. Provides both a `StubFormatter` (unused) and a `newFormatter` factory that returns concrete `ToonFormatter`/`PrettyFormatter`/`JSONFormatter`. The `Fmt` field on `Context` is set during `parseArgs`.

The V5 `Formatter` interface methods accept an `io.Writer` as the first argument and return `error`:
```go
type Formatter interface {
    FormatTaskList(w io.Writer, rows []TaskRow) error
    FormatTaskDetail(w io.Writer, data *showData) error
    FormatTransition(w io.Writer, data *TransitionData) error
    FormatDepChange(w io.Writer, data *DepChangeData) error
    FormatStats(w io.Writer, data *StatsData) error
    FormatMessage(w io.Writer, msg string)
}
```

**V6** uses an `App` struct-based architecture (already established in prior tasks). It:
1. Moves `IsTerminal` from `app.go` to `format.go` as the exported `DetectTTY`.
2. Renames the type from `OutputFormat` to `Format`, and renames constants from `FormatHuman`/`FormatTOON` to `FormatPretty`/`FormatToon` (matching the spec exactly).
3. Creates `ResolveFormat(flags globalFlags, isTTY bool)` taking the struct directly instead of individual booleans.
4. Adds `FormatConfig` struct with a `Logger *VerboseLogger` field, plus a `NewFormatConfig` constructor.
5. Defines additional data types: `RelatedTask`, `TaskDetail`, `Stats` for typed formatter arguments.
6. Uses a `Formatter` interface that returns `string` instead of writing to `io.Writer`:
```go
type Formatter interface {
    FormatTaskList(tasks []task.Task) string
    FormatTaskDetail(detail TaskDetail) string
    FormatTransition(id string, oldStatus string, newStatus string) string
    FormatDepChange(action string, taskID string, depID string) string
    FormatStats(stats Stats) string
    FormatMessage(msg string) string
}
```
7. Provides a `baseFormatter` struct for shared text-based transition/dep logic (DRY pattern).
8. `FormatConfig` and `Formatter` are both passed explicitly to every handler method.

**Key structural difference in `ResolveFormat` signature:**

V5 uses individual boolean parameters:
```go
func ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY bool) (OutputFormat, error)
```

V6 uses the `globalFlags` struct:
```go
func ResolveFormat(flags globalFlags, isTTY bool) (Format, error)
```

V6's approach couples `ResolveFormat` to the CLI layer (since `globalFlags` is unexported), which means it cannot be used by external callers. V5's approach with plain booleans is more reusable and aligns better with the spec's explicit `ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY)` signature.

**Key structural difference in `Formatter` return types:**

V5 returns `error` and writes to `io.Writer`, giving callers control over output destination. V6 returns `string`, pushing the write responsibility to the caller. The V6 approach is simpler but less flexible -- errors during formatting (e.g., write failures) are not propagated. V5's approach is more Go-idiomatic for I/O operations.

**Key structural difference in Formatter parameter types:**

V5 uses custom types (`TaskRow`, `*showData`, `*TransitionData`, etc.) specific to presentation. V6 uses `[]task.Task` directly (domain type) plus custom `TaskDetail` and `Stats` types. V6's `FormatTransition` and `FormatDepChange` use individual string parameters rather than a data struct.

### Code Quality

**Type naming:**

V5 keeps `OutputFormat` (pre-existing name). The spec calls for a `Format` enum. V6 renames the type to `Format`, matching the spec precisely.

**Enum constant naming:**

V5 uses `FormatToon`, `FormatPretty`, `FormatJSON` -- matching the spec. V6 also uses `FormatToon`, `FormatPretty`, `FormatJSON` -- matching the spec. V6 had to rename from `FormatHuman`/`FormatTOON` (the prior codebase naming) to match the spec. V5's prior codebase already had the correct names.

**Error handling:**

Both versions handle the conflicting-flags error identically:
```go
// V5 format.go:60-61
if count > 1 {
    return 0, fmt.Errorf("only one format flag may be specified (--toon, --pretty, --json)")
}

// V6 format.go:45-46
if count > 1 {
    return 0, fmt.Errorf("only one format flag (--toon, --pretty, --json) may be specified")
}
```

Slightly different message wording but functionally identical.

**DRY patterns:**

V6 introduces a `baseFormatter` struct (format.go:133) for shared `FormatTransition` and `FormatDepChange` implementations that `ToonFormatter` and `PrettyFormatter` embed. V5 uses standalone helper functions (`formatTransitionText`, `formatDepChangeText`, `formatMessageText`) at format.go:93-118. Both achieve DRY. V6's embedding approach is more idiomatic Go (composition via embedding), while V5's standalone functions are simpler and equally clear.

**Exported vs unexported:**

V5 exports `DetectTTY`, `ResolveFormat`, `FormatConfig`, `Formatter`, `StubFormatter`, and the `FormatCfg` method. `newFormatter` is unexported.

V6 exports `DetectTTY`, `ResolveFormat`, `FormatConfig`, `NewFormatConfig`, `Formatter`, `StubFormatter`, `NewFormatter`, `VerboseLog`/`VerboseLogger`, `RelatedTask`, `TaskDetail`, `Stats`. More surface area, but the additional types (`RelatedTask`, `TaskDetail`, `Stats`) provide type safety for formatter arguments.

**Documentation:**

Both versions document all exported types and functions with Go doc comments. Quality is equivalent.

**Context/wiring design:**

V5 embeds `Fmt Formatter` directly in `Context` (cli.go:33), set during `parseArgs`. Handlers access it via `ctx.Fmt`. This is a clean single-dispatch design.

V6 passes `FormatConfig` and `Formatter` as separate parameters to every handler:
```go
func (a *App) handleInit(fc FormatConfig, fmtr Formatter, _ []string) error
```
This is more explicit but creates verbose handler signatures. Each handler receives both `fc` and `fmtr` separately.

### Test Quality

**V5 test functions in `format_test.go` (176 lines):**
1. `TestDetectTTY/it detects TTY vs non-TTY` -- pipe detection
2. `TestDetectTTY/it defaults to non-TTY on stat failure` -- closed file
3. `TestResolveFormat/it defaults to Toon when non-TTY` -- no flags, non-TTY
4. `TestResolveFormat/it defaults to Pretty when TTY` -- no flags, TTY
5. `TestResolveFormat/it returns FormatToon when toon flag set` -- toon flag
6. `TestResolveFormat/it returns FormatPretty when pretty flag set` -- pretty flag
7. `TestResolveFormat/it returns FormatJSON when json flag set` -- json flag
8. `TestResolveFormat/it errors when multiple format flags set` -- table-driven with 4 conflict combos (toon+pretty, toon+json, pretty+json, all three)
9. `TestFormatConfig/it propagates quiet and verbose in FormatConfig` -- struct field check
10. `TestConflictingFormatFlagsIntegration/it errors when multiple format flags passed via CLI` -- full CLI integration
11. `TestFormatConfigWiredIntoContext/it creates FormatConfig from Context` -- `FormatCfg()` method

**V5 verbose tests in `verbose_test.go` (191 lines):**
1. `TestVerbose/it writes cache/lock/hash/format verbose to stderr` -- integration test
2. `TestVerbose/it writes nothing to stderr when verbose off` -- integration test
3. `TestVerbose/it does not write verbose to stdout` -- contamination check
4. `TestVerbose/it allows quiet + verbose simultaneously` -- orthogonality
5. `TestVerbose/it works with each format flag without contamination` -- table-driven (toon, pretty, json)
6. `TestVerbose/it produces clean piped output with verbose enabled` -- multi-task pipe test
7. `TestVerbose/it logs verbose during mutation commands` -- create command verbose

**V5 total: 18 test cases across 2 files**

**V6 test functions in `format_test.go` (376 lines):**
1. `TestFormatEnum/it defines three distinct format constants` -- uniqueness check
2. `TestDetectTTY/it detects non-TTY for a pipe` -- pipe detection
3. `TestDetectTTY/it defaults to non-TTY on stat failure` -- closed file
4. `TestResolveFormat/it defaults to Toon when non-TTY` -- no flags, non-TTY
5. `TestResolveFormat/it defaults to Pretty when TTY` -- no flags, TTY
6. `TestResolveFormat/it returns correct format for each flag override` -- table-driven with 6 combos (toon+TTY, toon+nonTTY, pretty+nonTTY, pretty+TTY, json+TTY, json+nonTTY)
7. `TestResolveFormat/it errors when multiple format flags set` -- table-driven with 4 conflict combos
8. `TestFormatConfig/it propagates quiet and verbose in FormatConfig` -- struct check
9. `TestFormatConfig/it defaults quiet and verbose to false` -- zero-value check
10. `TestNewFormatConfig/it builds FormatConfig from flags and TTY detection` -- constructor
11. `TestNewFormatConfig/it returns error from conflicting flags` -- constructor error path
12. `TestFormatterInterface/it satisfies Formatter interface with stub` -- all 6 methods exercised
13. `TestTaskDetailStruct/it holds task with related context for show output` -- populated struct
14. `TestTaskDetailStruct/it works with empty related slices` -- zero-value safety
15. `TestStatsStruct/it holds all stat fields with correct types` -- all fields verified
16. `TestStatsStruct/it defaults to zero values` -- zero-value safety
17. `TestFormatterInterfaceCompileCheck` -- compile-time check
18. `TestCLIDispatchRejectsConflictingFlags/it errors before dispatch when multiple format flags set` -- integration (checks exact stderr message)

**V6 tests in `cli_test.go` (modified, relevant portion):**
19. `TestTTYDetection/it detects pipe as non-TTY` -- pipe (updated from prior)
20. `TestTTYDetection/it defaults to Toon when not TTY` -- (updated)
21. `TestTTYDetection/it defaults to Pretty when TTY` -- (updated)
22. `TestTTYDetection/it overrides with --toon flag` -- (updated)
23. `TestTTYDetection/it overrides with --pretty flag` -- (updated)
24. `TestTTYDetection/it overrides with --json flag` -- (updated)

**V6 verbose tests in `verbose_test.go` (218 lines):**
25. `TestVerboseLogger/it writes cache/lock/hash/format verbose to stderr` -- unit test
26. `TestVerboseLogger/it writes nothing to stderr when verbose off` -- nil receiver
27. `TestVerboseLogger/it does not write verbose to stdout` -- contamination
28. `TestVerboseLogger/it allows quiet + verbose simultaneously` -- integration
29. `TestVerboseLogger/it works with each format flag without contamination` -- integration (3 flags)
30. `TestVerboseLogger/it produces clean piped output with verbose enabled` -- integration
31. `TestVerboseLogger/it writes nothing to stderr when verbose off` (second one, integration) -- app-level
32. `TestVerboseLogger/it prefixes all lines with verbose:` -- prefix check

**V6 total: ~32 test cases across 3 files**

**Test coverage diff:**

V6 has significantly more test coverage:
- V6 tests the `Format` enum uniqueness (V5 does not).
- V6 tests `FormatConfig` zero-value defaults (V5 does not).
- V6 tests `NewFormatConfig` constructor separately (V5 does not have this constructor).
- V6 exercises all 6 `StubFormatter` methods with return value assertions (V5 does not test StubFormatter -- the diff's `format_test.go` has no StubFormatter test).
- V6 tests `TaskDetail` and `Stats` data structures (V5 does not define these in this task).
- V6's `TestResolveFormat` flag override subtable covers 6 combos (each flag x both TTY states). V5 tests each flag individually but only with one TTY state.
- V6's `TestCLIDispatchRejectsConflictingFlags` checks exact stderr message content. V5's integration test only checks stderr is non-empty.
- V6 has compile-time interface check via `var _ Formatter = (*StubFormatter)(nil)` both in source and test.

V5 has one test V6 lacks:
- `TestVerbose/it logs verbose during mutation commands` -- tests verbose during `create` command.

**Test style:**

Both use table-driven tests where appropriate (conflict flags, format overrides). Both use `t.Run` subtests. V6 is more thorough in table-driven approach (6 override combos in a single table vs V5's 3 individual subtests for overrides).

### Skill Compliance

| Constraint | V5 | V6 |
|------------|-----|-----|
| Use gofmt and golangci-lint on all code | PASS -- code is properly formatted | PASS -- code is properly formatted |
| Handle all errors explicitly (no naked returns) | PASS -- all errors handled in ResolveFormat, DetectTTY | PASS -- all errors handled in ResolveFormat, DetectTTY, NewFormatConfig |
| Write table-driven tests with subtests | PASS -- table-driven for conflict flags (4 combos); individual subtests elsewhere | PASS -- table-driven for conflict flags (4 combos) and flag overrides (6 combos); individual subtests elsewhere |
| Document all exported functions, types, and packages | PASS -- all exported items documented | PASS -- all exported items documented |
| Propagate errors with fmt.Errorf("%w", err) | N/A -- no error wrapping needed in this task's new code | N/A -- no error wrapping needed in this task's new code |
| Ignore errors (avoid _ assignment without justification) | PASS -- no ignored errors | PASS -- no ignored errors |
| Use panic for normal error handling | PASS -- no panics | PASS -- no panics |
| Hardcode configuration | PASS -- no hardcoded config; all via flags/TTY | PASS -- no hardcoded config; all via flags/TTY |

### Spec-vs-Convention Conflicts

**1. `ResolveFormat` signature**

- **Spec says:** `ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY)`
- **Convention:** Go often passes structs for multiple related parameters (especially when they come from the same source).
- **V5 chose:** Individual booleans -- matches spec verbatim.
- **V6 chose:** `ResolveFormat(flags globalFlags, isTTY bool)` -- deviates from spec, uses struct.
- **Assessment:** V5's choice is more faithful to the spec and also more reusable (not coupled to `globalFlags`). V6's choice reduces parameter count but couples the function to internal CLI plumbing. Both are reasonable, but V5's is better for this specific case since the spec explicitly defined the parameter list.

**2. Formatter method signatures**

- **Spec says:** Methods `FormatTaskList`, `FormatTaskDetail`, `FormatTransition`, `FormatDepChange`, `FormatStats`, `FormatMessage` (no return/param details specified).
- **Go convention:** I/O operations should return `error`; methods that write should accept `io.Writer`.
- **V5 chose:** `FormatX(w io.Writer, data T) error` -- follows Go I/O convention.
- **V6 chose:** `FormatX(data T) string` -- simpler, returns formatted string.
- **Assessment:** The spec does not dictate signatures. V5's `io.Writer` pattern is more idiomatic for Go formatters (aligns with `fmt.Fprint*`, `encoding/json.Encoder`, etc.). V6's string-return pattern is simpler but shifts error handling burden to callers and prevents streaming for large outputs. For a CLI tool, both are adequate. V5 is marginally more Go-idiomatic.

**3. Format type name**

- **Spec says:** "`Format` enum"
- **V5 chose:** Kept `OutputFormat` (pre-existing name from earlier task).
- **V6 chose:** Renamed to `Format`.
- **Assessment:** V6 matches the spec exactly. V5's `OutputFormat` is descriptive but does not match the spec. Minor issue since the spec is about the concept, not necessarily the exact Go type name.

## Diff Stats

| Metric | V5 | V6 |
|--------|-----|-----|
| Files changed | 6 | 7 |
| Lines added | 328 | 611 |
| Impl LOC (format.go) | 119 | 183 |
| Test LOC (format_test.go) | 176 | 376 |
| Test LOC (verbose_test.go) | 191 | 218 |
| Test functions (format_test.go) | 11 | 18 |
| Test functions (verbose_test.go) | 7 | 8 |
| Test functions (cli_test.go modified) | 0 | 6 |
| Total test functions | 18 | 32 |

## Verdict

**V6 is the better implementation of this task**, though both versions pass all acceptance criteria.

The key advantages of V6:

1. **Spec fidelity:** V6 renames the type to `Format` (matching spec) and the constants to `FormatToon`/`FormatPretty`/`FormatJSON`. V5 keeps the pre-existing `OutputFormat` name, which does not match the spec's "`Format` enum" language.

2. **Test thoroughness:** V6 has 32 test cases vs V5's 18. V6 covers format enum uniqueness, `FormatConfig` zero-value defaults, `NewFormatConfig` constructor paths, `StubFormatter` method return values, `TaskDetail` and `Stats` data types, and 6 flag-override combos (each flag with both TTY states). V5's test suite is adequate but notably thinner.

3. **Typed formatter data:** V6 defines `RelatedTask`, `TaskDetail`, and `Stats` types for formatter parameters, providing type safety for downstream tasks 4-2 through 4-4. V5 uses `interface{}` less (it has typed structs too, like `*showData`, `*TransitionData`), but these are defined elsewhere. V6 co-locates the formatter data types with the interface.

4. **Compile-time interface check:** V6 includes `var _ Formatter = (*StubFormatter)(nil)` ensuring compile-time verification. V5 does not.

5. **`baseFormatter` embedding:** V6's `baseFormatter` pattern is more idiomatic Go composition than V5's standalone helper functions, though both achieve DRY.

V5 has two notable advantages:

1. **`ResolveFormat` signature:** V5's `ResolveFormat(toonFlag, prettyFlag, jsonFlag, isTTY bool)` matches the spec verbatim and is more reusable. V6's coupling to `globalFlags` is less flexible.

2. **`io.Writer` Formatter pattern:** V5's `FormatX(w io.Writer, data T) error` is more idiomatic Go for I/O operations and enables streaming/error propagation. V6's string-return pattern is simpler but less flexible.

Despite these two design advantages in V5, V6's significantly superior test coverage, closer spec naming compliance, and additional type-safe data structures make it the stronger implementation overall. The `ResolveFormat` signature difference is the most substantive V5 advantage, but it is outweighed by V6's other improvements.
