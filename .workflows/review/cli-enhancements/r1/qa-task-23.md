TASK: cli-enhancements-4-8 -- Note remove subcommand

ACCEPTANCE CRITERIA:
- `tick note remove <id> <index>` removes by 1-based position; index validated >= 1 and <= note count
- Removing a note updates the task's `Updated` timestamp

EDGE CASES (from plan):
- index 0
- index exceeding note count
- negative index
- non-integer index
- task with no notes

STATUS: Complete

SPEC CONTEXT:
The specification (Notes section) states: "`tick note remove <id> <index>` -- remove by 1-based position". Validation: "Note remove index must be >= 1 and <= number of existing notes; out-of-bounds errors". Also: "Adding or removing a note updates the task's Updated timestamp".

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/note.go:98-149 (RunNoteRemove function)
- Routing: /Users/leeovery/Code/tick/internal/cli/note.go:31 (handleNote routes "remove" to RunNoteRemove)
- App dispatch: /Users/leeovery/Code/tick/internal/cli/app.go:106-107 (routes "note" command)
- Help text: /Users/leeovery/Code/tick/internal/cli/help.go:134-140 (documents note add/remove usage)
- Notes:
  - Args parsing correctly requires exactly 2 positional args (ID and index), with separate error messages for each missing arg (lines 99-104)
  - Index parsed via strconv.Atoi with clear error for non-integer input (lines 108-111)
  - Lower bound validated: index < 1 errors with descriptive message (lines 112-114)
  - Partial ID resolution via store.ResolveID (line 122)
  - Inside Mutate callback: checks for zero notes (line 130-132), upper bound (line 133-135), performs 1-based to 0-based conversion (line 136), removes via slice append pattern (line 137)
  - Updated timestamp set to time.Now().UTC().Truncate(time.Second) (line 138) -- matches the pattern used in note add
  - After mutation, outputs result via outputMutationResult (line 148) which shows the updated task detail
  - No drift from plan or specification

TESTS:
- Status: Adequate
- Location: /Users/leeovery/Code/tick/internal/cli/note_test.go:232-509 (TestNoteRemove)
- Coverage:
  - Happy path: removes note at index 1 (line 235), removes at last index (line 267), removes middle note preserving order (line 303)
  - Edge case -- index 0: tested (line 339), verifies "must be >= 1" error
  - Edge case -- negative index: tested (line 358), verifies "must be >= 1" error
  - Edge case -- index exceeding note count: tested (line 377), verifies "out of range" and "has 2 note(s)" messages
  - Edge case -- non-integer index: tested (line 400), verifies "invalid index" error
  - Edge case -- task with no notes: tested (line 419), verifies "task has no notes to remove" error
  - Task not found: tested (line 435)
  - Partial ID resolution: tested (line 447)
  - Updated timestamp verification: tested (line 479), confirms timestamp is set to current time
- All edge cases from the plan are covered. Tests verify behavior, not implementation details. Each test is focused on a single concern. No redundant tests.
- Notes: The three happy-path tests (first, middle, last) are valuable -- they verify that slice manipulation preserves correct ordering for all positions, not just one.

CODE QUALITY:
- Project conventions: Followed
  - Handler signature matches the project pattern: `Run<Command>(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error`
  - Error wrapping with fmt.Errorf throughout
  - Tests use stdlib testing only, t.Run subtests, "it does X" naming, t.TempDir via setupTickProjectWithTasks
  - Uses `--pretty` implicitly via IsTTY=true in test helper (matches project convention for tests)
- SOLID principles: Good
  - Single responsibility: RunNoteRemove does one thing -- parse args, validate, mutate, output
  - Mutation logic cleanly separated inside Store.Mutate callback
  - ID resolution delegated to store.ResolveID
  - ID normalization delegated to task.NormalizeID
- Complexity: Low
  - Straightforward linear flow: parse args -> validate -> open store -> resolve ID -> mutate -> output
  - Single loop in Mutate callback to find task by ID (necessary pattern used across all commands)
- Modern idioms: Yes
  - Uses strconv.Atoi for integer parsing
  - Slice manipulation via append pattern (standard Go idiom for removing an element)
  - time.Now().UTC().Truncate(time.Second) for consistent timestamp handling
- Readability: Good
  - Clear variable names (rawID, index, idx for 0-based)
  - Descriptive error messages include context (actual index, note count)
  - Comment on RunNoteRemove explains what it does
- Issues: None

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- The missing-ID and missing-index error paths (lines 99-104) are validated before the store is opened, which is efficient -- no store/lock overhead for obvious input errors. Good.
- The `index < 1` check (line 112) happens before the store is opened, while `index > len(notes)` (line 133) necessarily happens inside Mutate. This is the correct separation since the upper bound depends on runtime data.
