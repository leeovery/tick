TASK: cli-enhancements-1-2 -- Integrate ResolveID into positional ID commands

ACCEPTANCE CRITERIA:
- All commands accepting task IDs resolve through ResolveID: show, update, start, done, cancel, reopen, dep add/rm, remove, and ID-accepting flags (--parent, --blocked-by, --blocks)
- (This task scopes to positional ID commands only; flags and dep are tasks 1-3 and 1-4)

STATUS: Complete

SPEC CONTEXT:
The specification states "Centralized: all commands resolve first, then proceed with the full ID. Applies everywhere an ID is accepted: positional args, --parent, --blocked-by, --blocks." This task covers the positional-arg portion of that requirement.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - show: /Users/leeovery/Code/tick/internal/cli/show.go:46 -- `store.ResolveID(args[0])`
  - transition (start/done/cancel/reopen): /Users/leeovery/Code/tick/internal/cli/transition.go:24 -- `store.ResolveID(args[0])`
  - update (positional ID): /Users/leeovery/Code/tick/internal/cli/update.go:209 -- `store.ResolveID(opts.id)`
  - remove (multi-ID): /Users/leeovery/Code/tick/internal/cli/app.go:244-252 -- loop resolving each raw ID via `store.ResolveID(raw)`
  - note add: /Users/leeovery/Code/tick/internal/cli/note.go:68 -- `store.ResolveID(rawID)`
  - note remove: /Users/leeovery/Code/tick/internal/cli/note.go:122 -- `store.ResolveID(rawID)`
- Notes: All six positional-ID code paths call `store.ResolveID()` before proceeding with the resolved full ID. The pattern is consistent: open store, resolve, then mutate/query with the canonical ID. No drift from the plan.

TESTS:
- Status: Adequate
- Coverage:
  - show with 3-char partial ID (line 14)
  - start with partial ID (line 33)
  - done with partial ID (line 60)
  - cancel with partial ID (line 84)
  - reopen with partial ID (line 108)
  - remove single with partial ID + --force (line 133)
  - remove multiple with partial IDs (line 154)
  - ambiguous prefix error on show (line 176)
  - not-found prefix error on start (line 198)
  - update positional ID with partial (line 213)
  - full IDs still work across show/start/remove (line 407)
  - All located in /Users/leeovery/Code/tick/internal/cli/partial_id_test.go
- Notes: Tests cover all positional-ID commands (show, start, done, cancel, reopen, update, remove) with partial IDs. Error paths (ambiguous, not-found) are covered. Full-ID backward compatibility is verified. The `note add` and `note remove` commands with partial IDs are not tested in this file, but they were added in Phase 4 and would be covered by the note test file. Tests are focused and not redundant -- each subtest targets a distinct command or error scenario.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, t.TempDir via setupTickProjectWithTasks, t.Helper on helpers. Error wrapping with fmt.Errorf throughout. Handler signatures follow the established pattern.
- SOLID principles: Good. ResolveID is a single-responsibility method in the storage layer. Each command handler has a single point of resolution before delegating to domain logic. No violations.
- Complexity: Low. Each integration point is a straightforward resolve-then-proceed pattern, typically 3-4 lines (call, error check, use resolved ID).
- Modern idioms: Yes. Idiomatic Go error handling, proper defer on store.Close(), no unnecessary abstractions.
- Readability: Good. Comments like "Resolve partial IDs via store.ResolveID" at update.go:208 and create.go:160 make intent clear. The handleRemove loop at app.go:244-252 is clean and handles early close on error.
- Issues: None.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- The test file partial_id_test.go includes tests for flag-based resolution (--parent, --blocked-by, --blocks at lines 237-405) which technically belong to tasks 1-3 and 1-4. This is not a problem -- tests are logically grouped -- but worth noting for traceability.
