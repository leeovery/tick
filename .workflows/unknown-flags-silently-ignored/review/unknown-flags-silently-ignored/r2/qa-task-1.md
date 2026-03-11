TASK: Normalize dep rm to dep remove and remove --from=value syntax

ACCEPTANCE CRITERIA:
- tick dep remove tick-aaa tick-bbb removes the dependency (same behaviour as old dep rm)
- tick dep rm tick-aaa tick-bbb returns exit code 1 with error containing "unknown dep sub-command 'rm'"
- tick migrate --from=beads fails (equals-sign syntax no longer supported)
- tick migrate --from beads still works (space-separated)
- All existing dep tests pass with updated sub-command name
- Help text for dep command shows <add|remove> not <add|rm>

STATUS: Complete

SPEC CONTEXT: The spec requires normalizing dep rm to dep remove for consistency with note remove and top-level remove. The --from=value equals-sign syntax must be removed from parseMigrateArgs so only --from value (space-separated) is supported, eliminating a special case the central validator would need to handle.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/cli/dep.go:25-33 - switch routes to "add" and "remove" only; default case returns "unknown dep sub-command" error
  - /Users/leeovery/Code/tick/internal/cli/dep.go:19 - usage string shows <add|remove>
  - /Users/leeovery/Code/tick/internal/cli/dep.go:128-194 - RunDepRemove function (renamed from old RunDepRm)
  - /Users/leeovery/Code/tick/internal/cli/migrate.go:47-67 - parseMigrateArgs only handles "--from" exact match (no equals-sign parsing)
  - /Users/leeovery/Code/tick/internal/cli/help.go:153 - help registry shows Usage: "tick dep <add|remove> <task-id> <blocked-by-id>"
  - /Users/leeovery/Code/tick/internal/cli/flags.go:75-76 - commandFlags registry uses "dep remove" not "dep rm"
  - /Users/leeovery/Code/tick/internal/cli/app.go:374-376 - qualifyCommand recognizes "add" and "remove" (not "rm")
- Notes: Implementation is clean and consistent. The dep.go switch, help registry, commandFlags, and qualifyCommand all use "remove" exclusively.

TESTS:
- Status: Adequate
- Coverage:
  - "it removes an existing dependency" (dep_test.go:364) - verifies dep remove works with exit code 0 and persistence
  - "it returns unknown sub-command error for dep rm" (dep_test.go:878-900) - verifies dep rm returns exit 1 with correct error message
  - "it accepts --from value (space-separated) on migrate" (migrate_test.go:352-361) - verifies space-separated --from works
  - "it rejects --from=value (equals-sign) on migrate" (migrate_test.go:363-375) - verifies equals-sign syntax fails with exit 1
  - "it shows <add|remove> in dep help text" (help_test.go:134-142) - verifies help text contains <add|remove>
  - All existing dep tests updated to use "remove" (dep_test.go TestDepRemove, TestDepRemovePartialID)
  - All existing dep add tests still reference "add" (dep_test.go TestDepAdd, TestDepAddPartialID)
- Notes: All five specified tests are present and correctly verify the acceptance criteria. Tests are focused and not redundant.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, t.Helper on helpers, error wrapping with %w, handler signature pattern matches existing convention.
- SOLID principles: Good. handleDep has single responsibility (routing). parseDepArgs handles argument parsing. RunDepRemove handles business logic. Clean separation.
- Complexity: Low. The dep.go switch is straightforward. The parseMigrateArgs loop is simple with no nested conditionals.
- Modern idioms: Yes. Standard Go patterns used throughout.
- Readability: Good. Function names are clear (RunDepRemove), error messages are descriptive ("unknown dep sub-command 'rm'"), usage strings guide the user.
- Issues: None.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- The --from=beads rejection path goes through ValidateFlags (since "--from=beads" does not match "--from" in commandFlags) rather than parseMigrateArgs. The error message will say "unknown flag" rather than something --from-specific. This is functionally correct and arguably better (consistent error format), but the user might be confused since "--from=beads" looks like a valid flag. The test correctly verifies the behavior as-is.
