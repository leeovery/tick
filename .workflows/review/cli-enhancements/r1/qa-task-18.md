TASK: cli-enhancements-4-3 -- Create and update with --refs and --clear-refs flags

ACCEPTANCE CRITERIA:
- `--refs <comma-separated>` on `create` and `update` sets/replaces refs
- `--clear-refs` on `update` removes all refs
- `--refs` and `--clear-refs` are mutually exclusive
- Empty `--refs` value errors

STATUS: Complete

SPEC CONTEXT:
Spec (lines 109-113) requires `--refs <comma-separated>` on create and update to set/replace all refs, `--clear-refs` on update to remove all, mutual exclusivity, and empty `--refs` value erroring. Refs are validated as non-empty, no commas, no whitespace, max 200 chars, max 10 per task, with silent deduplication. Refs are not filterable on list commands.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/create.go:86-92` -- `--refs` flag parsing in `parseCreateArgs`
  - `/Users/leeovery/Code/tick/internal/cli/create.go:145-151` -- `validateRefsFlag` call in `RunCreate`
  - `/Users/leeovery/Code/tick/internal/cli/create.go:212` -- `Refs: opts.refs` in Task construction
  - `/Users/leeovery/Code/tick/internal/cli/update.go:96-104` -- `--refs` and `--clear-refs` parsing in `parseUpdateArgs`
  - `/Users/leeovery/Code/tick/internal/cli/update.go:184-193` -- mutual exclusivity check and `validateRefsFlag` in `RunUpdate`
  - `/Users/leeovery/Code/tick/internal/cli/update.go:282-286` -- applying `--clear-refs` or `--refs` to task
  - `/Users/leeovery/Code/tick/internal/cli/update.go:33` -- `hasChanges()` includes `refs != nil` and `clearRefs`
  - `/Users/leeovery/Code/tick/internal/cli/helpers.go:84-96` -- `validateRefsFlag` helper
  - `/Users/leeovery/Code/tick/internal/cli/help.go:45` -- `--refs` documented for create
  - `/Users/leeovery/Code/tick/internal/cli/help.go:89-90` -- `--refs` and `--clear-refs` documented for update
- Notes: Implementation matches spec and acceptance criteria exactly. The `validateRefsFlag` helper handles deduplication, empty check, and validation delegation to `task.ValidateRefs`. Error messages guide users (e.g., "use --clear-refs to remove all refs" on update, "omit the flag to leave refs unset" on create). The `--refs` flag is correctly listed in the "at least one flag required" error message for update (line 137).

TESTS:
- Status: Adequate
- Coverage:
  - Create tests (`/Users/leeovery/Code/tick/internal/cli/create_test.go`):
    - Line 982: creates task with `--refs gh-123,JIRA-456` -- verifies refs persisted correctly
    - Line 1003: creates task without `--refs` -- verifies empty refs
    - Line 1018: errors on empty `--refs` value -- verifies error mentions `--refs`
    - Line 1029: deduplicates refs on create -- verifies dedup from 3 to 2
    - Line 1041: rejects invalid ref with whitespace -- verifies error mentions whitespace
  - Update tests (`/Users/leeovery/Code/tick/internal/cli/update_test.go`):
    - Line 878: updates refs with `--refs new-ref` -- verifies replacement
    - Line 899: clears refs with `--clear-refs` -- verifies refs cleared
    - Line 917: errors on `--refs` and `--clear-refs` together -- verifies "mutually exclusive" error
    - Line 933: errors on empty `--refs` value -- verifies error mentions `--clear-refs` and refs unchanged
    - Line 955: succeeds with `--clear-refs` on task with no refs (idempotent) -- edge case
- Notes: All four specified edge cases from the plan are covered: (1) --refs and --clear-refs together, (2) empty --refs value, (3) --refs with duplicates, (4) --clear-refs on task with no refs. Tests verify both persistence and error messages. Tests are behavioral and focused.

CODE QUALITY:
- Project conventions: Followed. Uses pointer types for optional fields in updateOpts (`*[]string`), follows the established pattern from tags, uses `t.Run()` subtests with "it does X" naming, stdlib testing only.
- SOLID principles: Good. `validateRefsFlag` in helpers.go follows SRP and DRY -- extracted shared validation logic used by both create.go and update.go. The helper delegates to `task.DeduplicateRefs` and `task.ValidateRefs` cleanly.
- Complexity: Low. Flag parsing, validation, and application are linear and straightforward.
- Modern idioms: Yes. Idiomatic Go error handling with `fmt.Errorf`, proper use of pointer types for distinguishing "not set" from "set to empty".
- Readability: Good. Code mirrors the pattern established by `--tags`/`--clear-tags` exactly, making it easy to follow for anyone who understands one to understand the other.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The structural parallel between tags and refs validation (validateTagsFlag/validateRefsFlag) has been noted in analysis cycles as a low-severity duplication. The current approach is clean and direct.
