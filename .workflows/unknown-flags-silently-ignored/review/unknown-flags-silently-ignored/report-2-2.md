TASK: Remove dead silent-skip logic from command parsers

ACCEPTANCE CRITERIA:
- No strings.HasPrefix(arg, "-") skip-and-continue logic exists in parseCreateArgs, parseUpdateArgs, parseDepArgs, RunNoteAdd, or parseRemoveArgs
- All existing tests pass (go test ./internal/cli/... and go test ./...)
- go vet ./... passes with no errors
- No unused imports remain after cleanup

STATUS: Complete

SPEC CONTEXT: The specification (Cleanup section) states: "The existing strings.HasPrefix(arg, '-') silent-skip logic in each command's parser can be removed -- unknown flags are caught before the handler is called." Phase 1 wired central validation via ValidateFlags() at all three dispatch points (app.go:57 for doctor, app.go:64 for migrate, app.go:101 for main switch), making per-parser skip logic dead code.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/cli/create.go (parseCreateArgs, lines 31-101): No HasPrefix skip logic. Default branch (line 93-97) assigns positional title only.
  - /Users/leeovery/Code/tick/internal/cli/update.go (parseUpdateArgs, lines 38-119): No HasPrefix skip logic. Default branch (line 111-116) assigns positional ID only.
  - /Users/leeovery/Code/tick/internal/cli/dep.go (parseDepArgs, lines 37-48): No HasPrefix skip logic. Simply copies all args as positional and extracts two IDs.
  - /Users/leeovery/Code/tick/internal/cli/note.go (RunNoteAdd, lines 39-51): No HasPrefix skip logic. All args treated as positional (first=ID, rest=text).
  - /Users/leeovery/Code/tick/internal/cli/remove.go (parseRemoveArgs, lines 20-39): No HasPrefix skip logic. Switch handles --force/-f explicitly, default adds to ID list.
- Notes: A codebase-wide grep for `strings.HasPrefix(arg, "-")` in non-test files shows only two occurrences, both in the Phase 1 central validation infrastructure: app.go:348 (pre-subcommand unknown flag rejection in parseArgs) and flags.go:120 (ValidateFlags central checker). Neither is dead skip logic.

TESTS:
- Status: Adequate
- Coverage: This task's acceptance criteria specify that all existing tests pass after skip removal -- no new dedicated tests are required for this cleanup task. The existing test suite across create_test.go, update_test.go, dep_test.go, note_test.go, and remove_test.go validates the functional behavior of these parsers. The flag_validation_test.go and flags_test.go cover the central validation that replaced the dead skip logic. Comprehensive regression tests for unknown flag rejection are covered by the sibling task (unknown-flags-silently-ignored-2-2).
- Notes: Test adequacy is appropriate for a dead-code removal task. The tests that matter are the existing functional tests (verifying no regressions) and the Phase 1 validation tests (verifying the replacement mechanism works).

CODE QUALITY:
- Project conventions: Followed. All parsers follow the established handler signature and pattern. No deviation from the DI-via-struct or functional-options patterns.
- SOLID principles: Good. Each parser has a single responsibility (parsing its command's args). The central validation (SRP) is in flags.go, not duplicated per-parser.
- Complexity: Low. All five parsers are straightforward iteration/switch patterns.
- Modern idioms: Yes. Standard Go patterns throughout.
- Readability: Good. Each parser is well-commented (e.g., "Positional argument: title (first one wins)").
- Issues: None.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- (none)
