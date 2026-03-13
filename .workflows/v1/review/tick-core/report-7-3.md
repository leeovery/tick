TASK: CLI framework & tick init

ACCEPTANCE CRITERIA:
- tick init creates .tick/ directory with empty tasks.jsonl
- tick init does not create cache.db
- tick init prints confirmation with absolute path
- tick init with --quiet produces no output on success
- tick init when .tick/ exists returns error to stderr with exit code 1
- All errors written to stderr with "Error: " prefix
- Exit code 0 for success, 1 for errors
- Global flags parsed: --quiet, --verbose, --toon, --pretty, --json
- TTY detection on stdout selects default output format
- .tick/ directory discovery walks up from cwd
- Unknown subcommands return error with exit code 1

STATUS: Complete

SPEC CONTEXT:
The specification defines `tick init` as creating `.tick/` directory with empty `tasks.jsonl`, SQLite cache created on first operation (not at init). Error handling: exit 0 = success, 1 = error. All errors to stderr. TTY detection determines output format (TOON for non-TTY/pipe, human-readable for terminal). Global flags: --quiet/-q, --verbose/-v, --toon, --pretty, --json. Unknown commands return error. No subcommand prints usage with exit 0.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/cmd/tick/main.go:1-18 (entry point, TTY detection, App wiring)
  - /Users/leeovery/Code/tick/internal/cli/app.go:1-267 (App struct, Run dispatcher, parseArgs, global flag parsing, subcommand routing, printUsage, handleInit)
  - /Users/leeovery/Code/tick/internal/cli/init.go:1-41 (RunInit: creates .tick/ dir, empty tasks.jsonl, prints confirmation, --quiet suppression)
  - /Users/leeovery/Code/tick/internal/cli/discover.go:1-31 (DiscoverTickDir: walks up from cwd to filesystem root)
  - /Users/leeovery/Code/tick/internal/cli/format.go:1-184 (Format enum, DetectTTY, ResolveFormat, FormatConfig, Formatter interface, NewFormatter factory)
- Notes:
  - All acceptance criteria are met in the implementation.
  - Error messages use "Error: " prefix via fmt.Fprintf(a.Stderr, "Error: %s\n", err) at app.go:40,94.
  - Unknown subcommands handled at app.go:88-91 with correct message format matching spec.
  - No subcommand prints usage at app.go:24-27 with exit code 0.
  - Global flags (--quiet/-q, --verbose/-v, --toon, --pretty, --json) parsed by applyGlobalFlag at app.go:250-266.
  - TTY detection via DetectTTY(os.Stdout) at format.go:24-30 using os.ModeCharDevice check.
  - Format resolution via ResolveFormat at format.go:34-62: non-TTY defaults to TOON, TTY defaults to Pretty, flags override.
  - Conflicting format flags (more than one) rejected with error.
  - DiscoverTickDir at discover.go:11-31 walks up from startDir to filesystem root, returning error if no .tick/ found.
  - Minor note: the spec says init error should be "Tick already initialized in this directory" but implementation uses "tick already initialized in <abs-path>" (lowercase, includes path). This is actually better for agent consumption and the spec says "Error with message" without mandating exact text. The task plan says "already initialized: if .tick/ exists, error with exit code 1" which is satisfied. Not a blocking issue.

TESTS:
- Status: Adequate
- Coverage:
  - /Users/leeovery/Code/tick/internal/cli/cli_test.go:10-165 - TestInit covers all 7 init-specific test cases from the plan:
    - "it creates .tick/ directory in current working directory" (line 11)
    - "it creates empty tasks.jsonl inside .tick/" (line 30)
    - "it does not create cache.db at init time" (line 52)
    - "it prints confirmation with absolute path on success" (line 71)
    - "it prints nothing with --quiet flag on success" (line 87)
    - "it errors when .tick/ already exists" (line 101)
    - "it returns exit code 1 when .tick/ already exists" (line 121)
    - "it writes error messages to stderr, not stdout" (line 140)
  - /Users/leeovery/Code/tick/internal/cli/cli_test.go:167-261 - TestDispatch covers:
    - "it routes unknown subcommands to error" (line 168)
    - "it prints usage with exit code 0 when no subcommand given" (line 189)
    - "--quiet flag passing via dispatch" (lines 209-261, including -q short form and flags after subcommand)
  - /Users/leeovery/Code/tick/internal/cli/cli_test.go:264-398 - TestParseArgs covers all global flag parsing
  - /Users/leeovery/Code/tick/internal/cli/cli_test.go:400-436 - TestDiscoverTickDir covers:
    - "it discovers .tick/ directory by walking up from cwd" (line 401)
    - "it errors when no .tick/ directory found" (line 425)
  - /Users/leeovery/Code/tick/internal/cli/cli_test.go:438-502 - TestTTYDetection covers:
    - Non-TTY detection via pipe
    - Default format selection (TOON for non-TTY, Pretty for TTY)
    - Flag overrides (--toon, --pretty, --json)
  - /Users/leeovery/Code/tick/internal/cli/format_test.go - Additional format tests: enum distinctness, DetectTTY edge cases, ResolveFormat comprehensive table-driven tests, conflicting flags, FormatConfig construction, NewFormatConfig
  - All 12 test cases from the plan are present and accounted for
- Notes: Tests are well-structured, use table-driven patterns where appropriate, and test behavior not implementation details. No over-testing observed. Each test verifies a distinct acceptance criterion or edge case.

CODE QUALITY:
- Project conventions: Followed. Table-driven tests, exported functions documented, errors wrapped with fmt.Errorf and %w, clear package structure (cmd/tick/main.go entry, internal/cli/ for CLI logic).
- SOLID principles: Good. App struct uses dependency injection (Stdout, Stderr, Getwd, IsTTY). Formatter interface with clean segregation. Single responsibility: init.go handles init, discover.go handles discovery, format.go handles format resolution, app.go handles dispatch.
- Complexity: Low. RunInit is a straightforward linear sequence. parseArgs is a simple single-pass loop. DiscoverTickDir is a clean upward walk. No complex branching.
- Modern idioms: Yes. Uses filepath.Abs, os.Stat for existence checks, io.Writer for testability, t.TempDir() for test cleanup.
- Readability: Good. Clear function names, short functions, well-commented. The parseArgs/applyGlobalFlag separation is clean.
- Issues: None significant.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The spec says init error message should be "Tick already initialized in this directory" but the implementation uses "tick already initialized in <abs-path>" (lowercase 't', includes absolute path). The implementation is arguably more informative. The task plan does not prescribe exact wording; it says to error when .tick/ already exists, which is satisfied.
- The printUsage at app.go:203-208 only lists "init" as a command. This is acceptable for Phase 1 but later phases should update it. Not a concern for this task.
- The spec edge case mentions "No parent directory / not writable: surface OS error as 'Error: Could not create .tick/ directory: <os error>'" -- the implementation at init.go:26-28 uses the message "could not create .tick/ directory: <os error>" (lowercase 'c'). When combined with the "Error: " prefix in app.go:94, the full output becomes "Error: could not create .tick/ directory: ..." which differs slightly from the spec's "Error: Could not create .tick/ directory: ...". This is a very minor capitalization difference and is non-blocking.
