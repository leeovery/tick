TASK: Integrate formatters into all commands (tick-core-4-5)

ACCEPTANCE CRITERIA:
- All commands output via Formatter
- --quiet overrides per spec (ID for mutations, nothing for transitions/deps/messages)
- Empty list correct per format
- TTY auto-detection end-to-end
- Flag overrides work for all commands
- Errors remain plain text stderr
- Format resolved once in dispatcher

STATUS: Complete

SPEC CONTEXT: The spec defines per-command output: create/update = show format (FormatTaskDetail), transitions = plain line (FormatTransition), deps = confirmation line (FormatDepChange), list = table (FormatTaskList), init/rebuild = message (FormatMessage). All TTY-aware with --quiet override. TTY detection selects Pretty (terminal) vs TOON (pipe/redirect). --toon/--pretty/--json flags override auto-detection. --quiet suppresses non-essential output: create/update output ID only, transitions/deps/messages produce nothing.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/cli/app.go:37-49 - Format resolved once in dispatcher (NewFormatConfig + NewFormatter)
  - /Users/leeovery/Code/tick/internal/cli/app.go:65-91 - All commands receive fc and fmtr
  - /Users/leeovery/Code/tick/internal/cli/create.go:173 - create uses outputMutationResult (FormatTaskDetail / quiet=ID only)
  - /Users/leeovery/Code/tick/internal/cli/update.go:201 - update uses outputMutationResult (FormatTaskDetail / quiet=ID only)
  - /Users/leeovery/Code/tick/internal/cli/transition.go:45-47 - transition uses FormatTransition, quiet=no output
  - /Users/leeovery/Code/tick/internal/cli/dep.go:118-119 - dep add uses FormatDepChange, quiet=no output
  - /Users/leeovery/Code/tick/internal/cli/dep.go:177-178 - dep rm uses FormatDepChange, quiet=no output
  - /Users/leeovery/Code/tick/internal/cli/list.go:157-165 - list uses FormatTaskList, quiet=IDs only
  - /Users/leeovery/Code/tick/internal/cli/show.go:49-56 - show uses FormatTaskDetail, quiet=ID only
  - /Users/leeovery/Code/tick/internal/cli/init.go:35-37 - init uses FormatMessage, quiet=no output
  - /Users/leeovery/Code/tick/internal/cli/rebuild.go:24-27 - rebuild uses FormatMessage, quiet=no output
  - /Users/leeovery/Code/tick/internal/cli/stats.go:12-14 - stats returns early when quiet (no output)
  - /Users/leeovery/Code/tick/internal/cli/helpers.go:16-29 - outputMutationResult shared helper for create/update
  - /Users/leeovery/Code/tick/internal/cli/app.go:93-94 - Errors go to stderr as plain text
- Notes: All commands consistently route through the Formatter interface. Format resolution occurs exactly once in the dispatcher (app.go:38-49). The doctor and migrate commands correctly bypass the format machinery (app.go:29-35). The implementation cleanly matches the spec.

TESTS:
- Status: Adequate
- Coverage:
  - /Users/leeovery/Code/tick/internal/cli/format_integration_test.go:16-81 - create in toon/pretty/json formats
  - /Users/leeovery/Code/tick/internal/cli/format_integration_test.go:83-152 - transitions in toon/pretty/json formats
  - /Users/leeovery/Code/tick/internal/cli/format_integration_test.go:154-221 - dep confirmations in toon/pretty/json formats
  - /Users/leeovery/Code/tick/internal/cli/format_integration_test.go:223-287 - list in toon/pretty/json formats
  - /Users/leeovery/Code/tick/internal/cli/format_integration_test.go:289-352 - show in toon/pretty/json formats
  - /Users/leeovery/Code/tick/internal/cli/format_integration_test.go:354-414 - init in toon/pretty/json formats
  - /Users/leeovery/Code/tick/internal/cli/format_integration_test.go:416-541 - --quiet overrides for create (ID only), transition (nothing), dep add (nothing), list (IDs only), show (ID only), init (nothing)
  - /Users/leeovery/Code/tick/internal/cli/format_integration_test.go:543-599 - empty list per format (toon zero-count, pretty "No tasks found.", json [])
  - /Users/leeovery/Code/tick/internal/cli/format_integration_test.go:601-643 - TTY auto-detection (non-TTY=toon, TTY=pretty)
  - /Users/leeovery/Code/tick/internal/cli/format_integration_test.go:645-703 - flag overrides (--toon, --pretty, --json)
  - /Users/leeovery/Code/tick/internal/cli/format_integration_test.go:706-726 - --quiet + --json (quiet wins, no JSON wrapping)
  - /Users/leeovery/Code/tick/internal/cli/format_integration_test.go:728-748 - errors remain plain text to stderr
  - /Users/leeovery/Code/tick/internal/cli/rebuild_test.go:210-227 - rebuild confirmation output
  - /Users/leeovery/Code/tick/internal/cli/rebuild_test.go:229-249 - rebuild --quiet suppresses output
  - /Users/leeovery/Code/tick/internal/cli/stats_test.go:328-342 - stats --quiet suppresses output
  - /Users/leeovery/Code/tick/internal/cli/update_test.go:244-290 - update output (full details and --quiet ID only)
  - /Users/leeovery/Code/tick/internal/cli/dep_test.go:507 - dep rm --quiet tested
- Notes: All 9 planned test scenarios from the task are covered. The format_integration_test covers init in each format but not rebuild in each format as a separate test. However, rebuild and init both use the same FormatMessage code path, and rebuild's output is tested in rebuild_test.go (line 224: exact string match "Cache rebuilt: 3 tasks\n" when using TOON format). The test coverage is thorough without being redundant.

CODE QUALITY:
- Project conventions: Followed - table-driven tests, explicit error handling, proper use of interfaces
- SOLID principles: Good - Formatter interface with separate implementations (OCP), single responsibility per handler, dependency inversion via Formatter interface
- Complexity: Low - each command handler has a simple flow: open store, mutate/query, format output. The quiet check is a simple boolean guard.
- Modern idioms: Yes - idiomatic Go patterns throughout, proper use of io.Writer for testability
- Readability: Good - outputMutationResult helper in helpers.go reduces duplication between create and update. The dispatcher in app.go is clear and sequential.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The format_integration_test does not explicitly test rebuild in each format (toon/pretty/json). While this shares the FormatMessage path with init (which is tested), adding explicit rebuild format tests would provide completeness. Low priority since the code path is identical.
- The stats command suppresses all output with --quiet (returns nil immediately at line 13-14 of stats.go). The spec does not specify quiet behavior for stats explicitly, but this is a reasonable interpretation.
