# Phase 4: Output Formats

## Task Scorecard

| Task | Winner | Key Differentiator |
|------|--------|--------------------|
| 4-1  | V5 wins | Stricter `DetectTTY(*os.File)` signature, working `StubFormatter.FormatMessage`, derived `FormatConfig` via method instead of stored redundant state, exact error message assertions in tests |
| 4-2  | V5 wins | Exhaustive write-error checking (37 sites vs 0), efficient `escapeField` fast-path, shared text helpers for DRY across formatters, exact full-output test assertions, strict spec compliance (no scope creep) |
| 4-3  | V5 wins | Right-aligned PRI numeric column, exact string comparison tests, clean quiet separation from formatter interface, shared text helpers, correct `truncateTitle` for small maxWidth values |
| 4-4  | V4 wins | Type-safe concrete Formatter interface vs V5's `interface{}` anti-pattern, `FormatMessage` returns `error`, nil-slice and special-character edge case testing, table-driven valid-JSON test |
| 4-5  | V5 wins | Typed struct parameters for formatter methods, consistent quiet handling at command level, `TaskRow` DTO decoupling, shared `formatTransitionText`/`formatDepChangeText` helpers |
| 4-6  | V5 wins | 17 tests across 3 layers (unit/store/CLI) vs 7 CLI-only tests, functional options pattern (`WithVerbose`), default no-op logger eliminating nil guards, `VerboseLogger` in `engine` package for correct layering |

## Phase-Level Patterns

### Architecture

The two implementations diverge sharply on Formatter interface design, which is the defining architectural decision of this phase.

**V4** uses a decomposed-parameter Formatter interface where each method receives its data as individual arguments or value types: `FormatTransition(w io.Writer, id string, oldStatus string, newStatus string) error`. This provides compile-time type safety from task 4-1 onward -- every call site and every formatter implementation is statically checked. V4 also pushes `quiet bool` into `FormatTaskList` and `FormatDepChange` signatures, making the formatter responsible for quiet-mode behavior. The `App` struct stores `FormatCfg` as a public field, creating redundant state that could drift from the underlying flags.

**V5** initially used `interface{}` parameters on all Formatter methods (tasks 4-1 through 4-4), requiring runtime type assertions in every implementation. This was a significant anti-pattern that V5 acknowledged and fixed in a later cleanup task (T6-7), replacing `interface{}` with concrete types like `*TransitionData`, `*DepChangeData`, and `*StatsData`. After that fix, V5's interface uses struct-pointer parameters that are both type-safe and extensible -- adding a field to `TransitionData` requires no interface signature change. V5 handles quiet at the command level, keeping formatters purely about rendering. `FormatConfig` is derived on demand via `ctx.FormatCfg()`.

The quiet-handling divergence has cascading effects. V4's approach means every formatter must implement quiet logic identically for `FormatTaskList` and `FormatDepChange`, duplicating the concern. V5's approach keeps formatters focused on a single responsibility. By task 4-5, V5's clean separation shows its value: the integration is simpler because formatters never need to know about quiet mode.

V5's shared helper functions (`formatTransitionText`, `formatDepChangeText`, `formatMessageText`) emerge in task 4-2 and prove their value through tasks 4-3 and 4-5 -- Toon and Pretty formatters delegate to identical plain-text implementations with zero duplication. V4 implements the same logic independently in each formatter.

For verbose logging (task 4-6), the architectural split widens. V4 uses a public `LogFunc` field on `Store` -- functional but loosely typed. V5 uses the functional options pattern (`WithVerbose`) with a default no-op logger, which is considered best-practice Go for optional configuration and eliminates nil guards entirely. V5 also places `VerboseLogger` in the `engine` package rather than `cli`, maintaining correct dependency direction.

### Code Quality

**Error handling** is the most consistent differentiator. V5's TOON formatter (task 4-2) checks all 37 write-error return values from `fmt.Fprintf`/`fmt.Fprintln`. V4's TOON formatter checks zero write errors. V4's Pretty formatter (task 4-3) also ignores all write errors. V5's Pretty formatter is inconsistent -- checking errors in `FormatTaskList` but not in `FormatTaskDetail` or `FormatStats`. Neither version achieves perfect write-error hygiene, but V5 is substantially closer.

However, V5's `FormatMessage` returns no error (void signature), which means write failures are silently lost. V4's `FormatMessage` consistently returns `error` across all formatters. This is V4's most persistent code quality advantage across the phase.

**Escaping efficiency** in the TOON formatter (task 4-2) strongly favors V5. V5's `escapeField` has a fast-path that returns immediately for strings without special characters and only escapes title fields. V4's `toonEscapeValue` marshals a full TOON document for every single field value, including IDs and timestamps that provably never need escaping.

**DRY principles**: V5 extracts shared text formatting helpers in `format.go` starting from task 4-2. V4 duplicates identical transition/dep/message formatting logic in each formatter independently. V5's `resolveFormatter`/`newFormatter` and shared helpers reduce the total code footprint while maintaining clarity.

**Naming and idiom**: V5 generally follows more idiomatic Go conventions -- `switch` for multi-branch dispatch (task 4-1), separate `Log`/`Logf` methods mirroring stdlib (task 4-6), unexported fields on internal structs. V4 uses sequential `if/else-if` chains and a single `Log(format, args...)` method.

### Test Quality

Both versions provide thorough test coverage, but they diverge in assertion strategy and layering.

**V5's dominant assertion pattern is exact full-output comparison** (`buf.String() != expected`). This appears in tasks 4-2 (TOON formatter), 4-3 (Pretty formatter), and propagates through the integration tests. Any change to formatting -- extra space, wrong alignment, missing newline -- immediately causes failure. V4 relies more heavily on `strings.Contains` checks, which can pass even if the output has extra or malformed content.

**V4's test advantages are concentrated in specific edge cases.** Task 4-4 (JSON formatter) is where V4 tests most shine: nil-slice testing (the Go nil-to-null gotcha), special character encoding (embedded quotes and newlines), and a genuinely table-driven valid-JSON test iterating over all six formatter methods. These are meaningful coverage gaps in V5's JSON tests.

**V5's layered testing strategy** in task 4-6 (verbose) is notably superior: 17 tests across unit (VerboseLogger isolation), store-integration (instrumentation verification), and CLI-integration (end-to-end verbose flow) layers. V4 has only 7 CLI-integration tests with no unit or store-level verbose coverage. V5's approach catches bugs at the appropriate abstraction level.

**Table-driven tests remain underused by both versions.** V4 uses them in task 4-4 (valid-JSON test) and task 4-2 (escaping test with 3 sub-cases). V5 uses them for the format-flag test in task 4-6 and the update test in task 4-5. Neither version adopts table-driven tests as the default pattern, representing a shared golang-pro skill compliance gap.

### Spec Compliance

Both versions satisfy the core acceptance criteria across all six tasks. The differences are in strictness and scope creep.

**V5 is strictly spec-compliant across the phase.** No extra fields, no unsolicited data, no deviations from the plan's output formats.

**V4 adds `ParentTitle` (`parent_title`) to task detail output** in tasks 4-2 (TOON formatter), 4-3 (Pretty formatter), 4-4 (JSON formatter), and 4-5 (integration). This field is not specified in any task plan. While arguably useful, it constitutes scope creep that could cause issues for consumers doing strict schema validation against spec-defined output.

Both versions have PARTIAL compliance on the "verbose to stderr only" requirement from task 4-1 -- neither addresses it in that commit, deferring to task 4-6 where both fully implement it.

### golang-pro Skill Compliance

**Error handling**: V5 is stronger for write-error checking (task 4-2's 37 checked sites) but weaker for `FormatMessage` (void return, `//nolint:errcheck` in JSON formatter). V4 is weaker for write-error checking (systematically ignored) but stronger for `FormatMessage` (returns error). Overall, V5's approach is closer to the spirit of the rule since write errors affect more methods and more code paths than the single `FormatMessage` case.

**Table-driven tests**: Both PARTIAL. Neither adopts this as the default pattern. V4 has a slight edge in task 4-4 with a proper table-driven valid-JSON test.

**Documentation**: Both PASS consistently across all tasks. All exported types, functions, and methods have doc comments.

**Error propagation with `%w`**: Both PASS where applicable.

**No panic, no ignored errors**: V5's `//nolint:errcheck` on `FormatMessage` in the JSON formatter (task 4-4) is a deliberate suppression that technically violates the "no ignored errors" rule. V4's systematic ignoring of `fmt.Fprintf` return values (tasks 4-2, 4-3) is a more pervasive violation but without explicit suppression comments, making it less visible.

**Reflection/`interface{}` avoidance**: V5's use of `interface{}` in the Formatter interface (tasks 4-1 through 4-4) is a notable violation. The golang-pro skill says to avoid reflection without performance justification. Runtime type assertions are reflection-adjacent and were later fixed in V5's cleanup phase. V4 avoids this entirely with concrete types from the start.

## Phase Verdict

**Winner**: V5 wins 5 of 6 tasks (4-1, 4-2, 4-3, 4-5, 4-6 vs V4's 4-4)

V5 demonstrates stronger engineering across the output formatting phase on three systemic dimensions: interface extensibility, separation of concerns, and test rigor.

The most impactful pattern is V5's interface design trajectory. While V5's initial `interface{}` parameters (tasks 4-1 through 4-4) were a genuine anti-pattern -- and the reason V4 wins task 4-4 -- the rest of V5's design choices compound in value across the phase. Typed struct parameters (`*TransitionData`, `*DepChangeData`) make the interface extensible without signature changes. Clean quiet separation keeps formatters focused on rendering. Shared text helpers eliminate duplication between Toon and Pretty formatters. The `TaskRow` DTO provides proper decoupling between internal query results and the formatting layer. These choices reduce the total integration cost in task 4-5 and produce a more maintainable architecture overall.

V4's strongest contribution is the type-safe Formatter interface from task 4-1 onward. Getting concrete types right from the start means V4 never incurs the `interface{}` technical debt that V5 later cleaned up. V4 also maintains better discipline on `FormatMessage` error returns, provides more thorough edge case testing in the JSON formatter (nil slices, special characters), and offers more granular verbose instrumentation (24 log points vs 13). These are real advantages that reflect careful attention to correctness and debuggability.

However, V5's advantages are more structural. The write-error checking in task 4-2 (37 checked sites vs zero) represents a fundamental quality gap. The exact full-output test assertions catch formatting regressions that V4's substring checks would miss. The layered test strategy in task 4-6 (unit + store + CLI) provides coverage at appropriate abstraction levels. The functional options pattern and default no-op logger in task 4-6 are canonical Go patterns that V4's public `LogFunc` field does not match. Collectively, these advantages span correctness, maintainability, and idiomatic Go practice -- the dimensions that matter most for a formatting subsystem that every command in the CLI depends on.
