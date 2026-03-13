TASK: Reproduce bug and build flag metadata with central validator

ACCEPTANCE CRITERIA:
- ValidateFlags returns nil for valid flag combinations on every command
- ValidateFlags returns error with exact format: unknown flag "{flag}" for "{command}". Run 'tick help {helpCmd}' for usage.
- Value-taking flags correctly cause next argument to be skipped
- Boolean flags do not cause next argument to be skipped
- Global flags (--quiet, -q, --verbose, -v, --toon, --pretty, --json, --help, -h) accepted on any command
- Short flags like -x rejected; -f accepted on remove
- Two-level commands use fully-qualified name in error, parent in help reference
- commandFlags map covers all 20 commands

STATUS: Complete

SPEC CONTEXT: The spec requires that all commands reject unrecognised flags with a clear error message in the format `unknown flag "{flag}" for "{command}". Run 'tick help {helpCmd}' for usage.` Global flags must pass through. Two-level commands (dep add/remove, note add/remove) use the fully-qualified name in the error but the parent command in the help reference. Flag metadata must distinguish boolean vs value-taking flags so the validator skips values correctly. 20 commands must be covered; version and help are excluded.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/flags.go:1-157
- Notes:
  - FlagDef struct (line 9-12) with TakesValue boolean -- clean, minimal metadata
  - globalFlagSet (lines 20-30) contains all 9 global flags (--quiet, -q, --verbose, -v, --toon, --pretty, --json, --help, -h)
  - commandFlags (lines 33-90) defines 18 commands statically; init() (lines 92-95) derives ready/blocked from list via copyFlagsExcept, totaling 20 commands
  - ValidateFlags (lines 115-146) iterates args, skips non-flag args, skips numeric values (e.g. "-1"), accepts global flags, validates against command flags, and skips next arg for value-taking flags
  - helpCommand (lines 151-156) splits on space to return parent for two-level commands
  - Error format exactly matches spec: `unknown flag %q for %q. Run 'tick help %s' for usage.`
  - Numeric value guard (lines 125-127) prevents negative numbers like "-1" from being treated as flags -- good edge case handling not in spec but necessary for correctness
  - copyFlagsExcept (lines 98-107) prevents drift between ready/blocked and list flag sets

TESTS:
- Status: Adequate
- Coverage:
  - flags_test.go: Unit tests for ValidateFlags -- nil for no flags, nil for known flags, error for unknown flag on dep add (bug repro), -f accepted on remove, helpCommand tests
  - flag_validation_test.go: Comprehensive coverage -- all 7 commands with flags validated (create, update, list, ready, blocked, remove, migrate), all 13 no-flag commands validated, flag count assertions, ready rejects --ready, blocked rejects --blocked, global flags accepted on all 20 commands, global flags mixed with command flags, value-taking flag skipping (6 subtests including value-that-looks-like-flag, consecutive value-taking, known flag in value position, global flag in value position, all create value-taking flags, all update value-taking flags), boolean flag does not skip next arg, drift-detection test (TestCommandFlagsMatchHelp)
  - unknown_flag_test.go: Integration tests through App.Run() -- all no-flag commands reject --unknown, commands with flags reject wrong flags, two-level commands use correct error format, short flag -x rejection, bug report scenario with real tasks, known flags accepted through dispatch, excluded commands (version/help) bypass validation, global flags not rejected through dispatch
- Notes: Tests are well-structured and cover all acceptance criteria, edge cases, and the original bug scenario. The test suite is thorough without being redundant -- unit tests cover the pure function, integration tests verify wiring through App.Run().

CODE QUALITY:
- Project conventions: Followed -- stdlib testing only, t.Run subtests, t.TempDir for isolation, t.Helper on helpers, error wrapping with fmt.Errorf, no testify
- SOLID principles: Good -- ValidateFlags is a pure function with injected flags map (testable), FlagDef is minimal, single responsibility for helpCommand, copyFlagsExcept prevents drift
- Complexity: Low -- ValidateFlags is a single-pass loop with clear branching; cyclomatic complexity is reasonable
- Modern idioms: Yes -- map literals, range loops, string operations all idiomatic Go
- Readability: Good -- well-commented, exported doc comments on all public types/functions, clear variable names, logical grouping (types, globals, commandFlags, init, helpers, validator)
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The numeric value guard (lines 125-127 of flags.go) for values like "-1" is a smart addition not explicitly in the spec, but prevents false positives when negative numbers are used as priority values. Well done.
- The `init()` function deriving ready/blocked flag sets from list via copyFlagsExcept is a good DRY pattern that also serves as a structural invariant preventing drift.
