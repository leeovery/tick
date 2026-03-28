TASK: Command Wiring and Formatter Interface

ACCEPTANCE CRITERIA:
- commandFlags contains "dep tree" with empty flag set
- qualifyCommand("dep", ["tree", "tick-abc123"]) returns ("dep tree", ["tick-abc123"])
- Formatter interface includes FormatDepTree(DepTreeResult) string
- All three formatters + StubFormatter compile with new method
- tick dep tree dispatches without error (placeholder returns nil)
- tick dep tree --unknown returns unknown flag error
- tick help dep mentions tree
- TestCommandFlagsMatchHelp passes
- All existing tests pass

STATUS: Complete

SPEC CONTEXT: The specification defines `tick dep tree [id]` as a new subcommand of `dep` with no command-specific flags (only global flags). It extends the existing `dep add/remove` pattern. The Formatter interface must include FormatDepTree for all three format implementations (Pretty, Toon, JSON).

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/flags.go:76` — `"dep tree": {}` in commandFlags
  - `internal/cli/app.go:375` — `qualifyCommand` switch case extended with `"tree"`
  - `internal/cli/dep.go:30-31` — `case "tree":` in handleDep dispatches to RunDepTree
  - `internal/cli/dep_tree.go:12-63` — RunDepTree handler (fully implemented, not just a placeholder)
  - `internal/cli/format.go:149-180` — DepTreeTask, DepTreeNode, DepTreeResult types defined
  - `internal/cli/format.go:202` — FormatDepTree method on Formatter interface
  - `internal/cli/format.go:227` — FormatDepTree stub on baseFormatter
  - `internal/cli/format.go:275` — FormatDepTree stub on StubFormatter
  - `internal/cli/pretty_formatter.go:307` — FormatDepTree on PrettyFormatter (fully implemented, not a stub)
  - `internal/cli/toon_formatter.go:175` — FormatDepTree on ToonFormatter (fully implemented, not a stub)
  - `internal/cli/json_formatter.go:360` — FormatDepTree on JSONFormatter (fully implemented, not a stub)
  - `internal/cli/help.go:152-161` — dep help entry updated with tree subcommand
- Notes: The plan described Task 1 as creating stubs on all formatters, but the implementations here are complete (from later tasks). The wiring itself is correct. The DepTreeResult struct omits the `Mode string` field specified in the plan, using `Target *DepTreeTask` for mode discrimination instead. This is a reasonable design decision that avoids stringly-typed mode switching. The dep.go usage strings were also updated to include "tree".

TESTS:
- Status: Adequate
- Coverage:
  - "it qualifies dep tree as a two-level command" — verifies qualifyCommand returns correct cmd and rest args
  - "it qualifies dep tree with no args" — verifies qualifyCommand with empty args
  - "it rejects unknown flag on dep tree" — verifies ValidateFlags error message includes "dep tree" and help reference
  - "it accepts global flags on dep tree" — verifies --quiet is accepted (plan says --verbose; both are global flags, same behavior)
  - "it dispatches dep tree without error" — verifies App.Run exits 0
  - "it shows tree in dep help text" — verifies help output contains "tree"
  - "it shows <add|remove|tree> in dep help text" — additional test verifying exact usage pattern (in help_test.go:134-142)
  - "it does not break existing dep add/remove dispatch" — verifies qualifyCommand still works for add/remove
  - "dep tree" appears in noFlagCommands list in flag_validation_test.go:138 (existing drift-detection test covers it)
  - "dep tree" appears in commands list in TestGlobalFlagsAcceptedOnAnyCommand (flag_validation_test.go:191)
- Notes: All seven specified test names from the plan are present. Additional tests exist in the same file for later tasks (RunDepTree handler tests) which verify the full dispatch path. Test coverage is thorough without being redundant.

CODE QUALITY:
- Project conventions: Followed — handler follows the RunXxx pattern with (dir, fc, fmtr, args, stdout) signature, uses openStore/ReadTasks for read-only access, uses Formatter interface for output
- SOLID principles: Good — single responsibility maintained, interface extended minimally, formatters implement through composition (baseFormatter embedding)
- Complexity: Low — straightforward switch case additions, clean data type definitions
- Modern idioms: Yes — Go interface satisfaction via compile-time checks (var _ Formatter = (*XxxFormatter)(nil))
- Readability: Good — clear naming, consistent with existing patterns, good doc comments
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The qualifyCommand switch case "add", "remove", "tree" applies to both "dep" and "note" parents. Running `tick note tree` would produce "note tree" as the qualified command, which is not in commandFlags. The error path still works correctly (ValidateFlags passes for empty args, then handleNote returns "unknown note sub-command 'tree'"), but the error could be confusing if someone passes flags (e.g., `tick note tree --foo` would say `unknown flag "--foo" for "note tree"`). This is a minor pre-existing pattern issue, not introduced by this task.
- The DepTreeResult struct deviates from the plan by omitting the `Mode string` field, using structural discrimination (`Target != nil`) instead. This is arguably better design but differs from the plan specification.
