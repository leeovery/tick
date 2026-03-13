TASK: tick-core-4-1 - Formatter abstraction & TTY-based format selection

ACCEPTANCE CRITERIA:
- [ ] Formatter interface covers all command output types
- [ ] Format enum with 3 constants
- [ ] TTY detection works correctly
- [ ] ResolveFormat handles all flag/TTY combos
- [ ] Conflicting flags -> error
- [ ] FormatConfig wired into CLI dispatch
- [ ] Verbose to stderr only
- [ ] Stat failure handled gracefully

STATUS: Complete

SPEC CONTEXT:
The specification defines TTY-based auto-detection: no TTY (pipe/redirect) -> TOON format, TTY (terminal) -> human-readable. Flags --toon, --pretty, --json override auto-detection. --verbose writes to stderr only (never contaminates stdout). --quiet suppresses non-essential output. These are orthogonal to format selection.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - Format enum (3 constants): `/Users/leeovery/Code/tick/internal/cli/format.go:11-20`
  - DetectTTY function: `/Users/leeovery/Code/tick/internal/cli/format.go:24-30`
  - ResolveFormat function: `/Users/leeovery/Code/tick/internal/cli/format.go:34-62`
  - FormatConfig struct: `/Users/leeovery/Code/tick/internal/cli/format.go:65-71`
  - NewFormatConfig builder: `/Users/leeovery/Code/tick/internal/cli/format.go:74-84`
  - Formatter interface (6 methods): `/Users/leeovery/Code/tick/internal/cli/format.go:116-129`
  - StubFormatter placeholder: `/Users/leeovery/Code/tick/internal/cli/format.go:148-171`
  - NewFormatter factory: `/Users/leeovery/Code/tick/internal/cli/format.go:174-183`
  - baseFormatter shared text implementations: `/Users/leeovery/Code/tick/internal/cli/format.go:131-146`
  - Supporting types (RelatedTask, TaskDetail, Stats): `/Users/leeovery/Code/tick/internal/cli/format.go:87-112`
  - CLI dispatch wiring: `/Users/leeovery/Code/tick/internal/cli/app.go:37-63`
  - VerboseLogger (stderr-only): `/Users/leeovery/Code/tick/internal/cli/verbose.go:1-39`
  - main.go TTY detection at startup: `/Users/leeovery/Code/tick/cmd/tick/main.go:15`
- Notes:
  - All acceptance criteria are met.
  - Formatter interface has 6 methods: FormatTaskList, FormatTaskDetail, FormatTransition, FormatDepChange, FormatStats, FormatMessage -- covering all command output types per spec (list, show/create/update, transitions, dep changes, stats, general messages).
  - Format enum defines FormatToon=0, FormatPretty=1, FormatJSON=2 via iota.
  - DetectTTY uses os.File.Stat() checking ModeCharDevice; returns false on stat failure.
  - ResolveFormat counts flags, errors on >1, returns flag match if 1, falls back to TTY detection (Pretty for TTY, Toon for non-TTY).
  - FormatConfig properly carries Format, Quiet, Verbose, and optional Logger.
  - CLI dispatch in App.Run calls NewFormatConfig, creates formatter via NewFormatter, and passes both to all command handlers.
  - VerboseLogger writes to stderr with "verbose: " prefix; nil-receiver safe.
  - Conflicting flags are caught before dispatch (error + exit code 1).

TESTS:
- Status: Adequate
- Coverage:
  - `TestFormatEnum` - verifies 3 distinct constants: `/Users/leeovery/Code/tick/internal/cli/format_test.go:10-24`
  - `TestDetectTTY` - pipe detection + stat failure default: `/Users/leeovery/Code/tick/internal/cli/format_test.go:26-54`
  - `TestResolveFormat` - defaults (Toon for non-TTY, Pretty for TTY), all flag overrides (6 combos), conflicting flags (4 combos): `/Users/leeovery/Code/tick/internal/cli/format_test.go:56-127`
  - `TestFormatConfig` - propagates quiet/verbose, defaults to false: `/Users/leeovery/Code/tick/internal/cli/format_test.go:129-156`
  - `TestNewFormatConfig` - builds from flags+TTY, returns error on conflict: `/Users/leeovery/Code/tick/internal/cli/format_test.go:158-183`
  - `TestFormatterInterface` - verifies StubFormatter satisfies interface: `/Users/leeovery/Code/tick/internal/cli/format_test.go:185-241`
  - `TestCLIDispatchRejectsConflictingFlags` - end-to-end conflict rejection via App.Run: `/Users/leeovery/Code/tick/internal/cli/format_test.go:341-362`
  - `TestTTYDetection` in cli_test.go - additional TTY/format tests: `/Users/leeovery/Code/tick/internal/cli/cli_test.go:438-502`
  - `TestVerboseLogger` - verbose to stderr, nil-receiver no-op, no stdout contamination, quiet+verbose orthogonal, format-independent: `/Users/leeovery/Code/tick/internal/cli/verbose_test.go:12-218`
  - Format integration tests covering all formats across all command types: `/Users/leeovery/Code/tick/internal/cli/format_integration_test.go:1-749`
  - baseFormatter tests for transition/dep change text output: `/Users/leeovery/Code/tick/internal/cli/base_formatter_test.go:1-96`
- Notes:
  - All 6 specified test cases from the task are covered. Some tests in cli_test.go duplicate coverage from format_test.go (TestTTYDetection repeats the same ResolveFormat checks), but this is minor and not excessive.
  - Edge cases are well covered: stat failure, all conflicting flag combinations, quiet+verbose interaction, verbose across all format flags.
  - Tests verify behavior not implementation details.

CODE QUALITY:
- Project conventions: Followed. Table-driven tests used throughout. Exported functions documented. Error handling explicit. Compile-time interface verification present (`var _ Formatter = (*StubFormatter)(nil)`).
- SOLID principles: Good. Formatter interface is well-segregated with clear single responsibility. Format resolution is separated from detection. baseFormatter uses composition for shared behavior. Dependency inversion via Formatter interface allows pluggable implementations.
- Complexity: Low. ResolveFormat is simple flag counting + conditional. DetectTTY is 4 lines. No deep nesting or complex control flow.
- Modern idioms: Yes. iota for enum, nil-receiver safety on VerboseLogger, io.Writer injection for testability, functional option-style logger wiring.
- Readability: Good. Clear naming (DetectTTY, ResolveFormat, FormatConfig). Well-structured with logical grouping. Comments explain intent.
- Issues: None significant.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Minor test duplication: `TestTTYDetection` in cli_test.go largely repeats what `TestResolveFormat` in format_test.go already covers (defaults and flag overrides). Could consolidate but not worth the churn.
- The `StubFormatter` is retained alongside real formatters (ToonFormatter, PrettyFormatter, JSONFormatter). It is still referenced by compile-time check. Since `NewFormatter` never returns it, it is effectively dead code -- could be removed, but it serves as documentation of the interface pattern and costs nothing.
