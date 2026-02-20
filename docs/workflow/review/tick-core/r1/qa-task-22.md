TASK: Verbose output & edge case hardening (tick-core-4-6)

ACCEPTANCE CRITERIA:
- [ ] VerboseLogger writes stderr only when Verbose true
- [ ] Key operations instrumented (cache, lock, hash, write, format)
- [ ] All lines `verbose:` prefixed
- [ ] Zero verbose on stdout
- [ ] --quiet + --verbose works correctly
- [ ] Piping captures only formatted output
- [ ] No output when verbose off

STATUS: Complete

SPEC CONTEXT: The specification defines `--verbose` / `-v` as "More detail (useful for debugging)" under Verbosity Flags. Errors already go to stderr per spec; verbose follows the same stderr pattern. This is critical for agent pipelines where stdout must remain parseable (TOON/JSON) and verbose debug detail must not contaminate piped output.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/verbose.go` (VerboseLogger struct, Log method, storeOpts helper)
  - `/Users/leeovery/Code/tick/internal/cli/format.go:64-71` (FormatConfig with Logger field)
  - `/Users/leeovery/Code/tick/internal/cli/app.go:44-63` (verbose logger creation and format resolution logging)
  - `/Users/leeovery/Code/tick/internal/cli/helpers.go:34-40` (openStore passes storeOpts for verbose)
  - `/Users/leeovery/Code/tick/internal/storage/store.go:29-31,44-50,75-80` (WithVerbose option, verbose() helper)
  - `/Users/leeovery/Code/tick/internal/storage/store.go:95-121` (lock acquire/release verbose)
  - `/Users/leeovery/Code/tick/internal/storage/store.go:159-165` (write verbose)
  - `/Users/leeovery/Code/tick/internal/storage/store.go:194-223` (rebuild verbose)
  - `/Users/leeovery/Code/tick/internal/storage/store.go:302-307` (hash/freshness verbose)
- Notes:
  - VerboseLogger uses nil-receiver pattern for no-op when disabled -- clean design.
  - All commands go through `openStore(dir, fc)` which calls `storeOpts(fc)` -- verbose is consistently wired.
  - Format resolution is logged in `app.go:52-63` when verbose is on.
  - Key operations instrumented: lock acquire/release (shared & exclusive), JSONL atomic write, cache rebuild, hash comparison, JSONL read, cache.db deletion, hash update, format resolution.
  - All verbose output writes to `a.Stderr` via `NewVerboseLogger(a.Stderr)` -- never to stdout.
  - The `"verbose: "` prefix is hardcoded in `Log()` at `verbose.go:27`.

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/internal/cli/verbose_test.go` -- 7 test cases covering all acceptance criteria:
    1. "it writes cache/lock/hash/format verbose to stderr" -- verifies key operation messages and prefix
    2. "it writes nothing to stderr when verbose off" (unit) -- nil receiver no-op
    3. "it does not write verbose to stdout" -- separate stdout/stderr buffers
    4. "it allows quiet + verbose simultaneously" -- integration test with App, verifies quiet stdout (IDs only) and verbose stderr
    5. "it works with each format flag without contamination" -- tests --toon, --pretty, --json with --verbose; checks no verbose in stdout, all prefixed in stderr
    6. "it produces clean piped output with verbose enabled" -- IsTTY=false, verifies TOON output clean, verbose on stderr
    7. "it writes nothing to stderr when verbose off" (integration) -- full App run without --verbose, asserts stderr empty
    8. "it prefixes all lines with verbose:" -- verifies prefix on all lines
  - `/Users/leeovery/Code/tick/internal/storage/store_test.go:678` -- "it logs verbose messages during rebuild" verifies exact message sequence from Store layer
  - `/Users/leeovery/Code/tick/internal/cli/rebuild_test.go:251` -- "it logs rebuild steps with --verbose" end-to-end rebuild verbose test
  - `/Users/leeovery/Code/tick/internal/cli/format_test.go:130,147,160` -- FormatConfig propagation tests for verbose flag
  - `/Users/leeovery/Code/tick/internal/cli/cli_test.go:292-307,370-378` -- parseArgs tests for --verbose flag parsing
- Notes: Good coverage breadth -- unit tests for VerboseLogger, integration tests through App, store-level tests, format config propagation tests, and flag parsing tests. All specified test scenarios from the task are covered. Tests verify behavior (stderr vs stdout streams) not just implementation. The duplicate test name "it writes nothing to stderr when verbose off" (lines 43 and 180) is a minor concern but they test different things (nil receiver vs full App integration).

CODE QUALITY:
- Project conventions: Followed -- table-driven subtests, explicit error handling, t.Helper pattern used where applicable
- SOLID principles: Good -- VerboseLogger has single responsibility (write prefixed lines to a writer). Store accepts verbose as an option (open/closed principle via functional options). The io.Writer dependency injection is clean.
- Complexity: Low -- VerboseLogger is a thin wrapper (4 lines in Log), storeOpts is simple conditional. No branching complexity.
- Modern idioms: Yes -- nil-receiver pattern for no-op logger is idiomatic Go. Functional options for Store configuration. io.Writer abstraction for testability.
- Readability: Good -- clear naming, concise implementation, well-documented exported types and functions
- Issues: None significant

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Two test cases share the name "it writes nothing to stderr when verbose off" (lines 43 and 180 in verbose_test.go). While they test different scenarios (nil receiver unit test vs full App integration), distinct names would improve clarity. For example, rename the first to "it is a no-op on nil receiver" or similar.
- Phase 7 task tick-core-7-4 mentions removing a dead `VerboseLog` function. It does not exist in the current codebase, suggesting it was either already removed or was never committed. No action needed here.
