TASK: tick update command (tick-core-2-3)

ACCEPTANCE CRITERIA:
- [ ] All five flags work correctly (--title, --description, --priority, --parent, --blocks)
- [ ] Multiple flags combinable in single command
- [ ] `updated` refreshed on every update
- [ ] No flags -> error with exit code 1
- [ ] Missing/not-found ID -> error with exit code 1
- [ ] Invalid values -> error with exit code 1, no mutation
- [ ] Output shows full task details; `--quiet` outputs ID only
- [ ] Input IDs normalized to lowercase
- [ ] Mutation persisted through storage engine

STATUS: Complete

SPEC CONTEXT: `tick update <id>` with --title, --description, --priority, --parent, --blocks. At least one flag required. Cannot change id/status/created/blocked_by. `--blocks` is inverse of `--blocked-by` (adds this task's ID to target's `blocked_by`). Output like `tick show`; `--quiet` outputs ID only. Title validation: trim, 500 max, no newlines. Priority 0-4. Parent must exist, no self-ref. Blocks IDs must exist.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/update.go (full command, 202 lines)
- Supporting helpers: /Users/leeovery/Code/tick/internal/cli/helpers.go:16-77 (outputMutationResult, openStore, parseCommaSeparatedIDs, applyBlocks)
- Command registration: /Users/leeovery/Code/tick/internal/cli/app.go:74-75 (case "update"), :141-147 (handleUpdate)
- Notes:
  - All five flags parsed correctly via `parseUpdateArgs` with pointer types for optional distinction
  - `hasChanges()` correctly checks all five flags including blocks
  - Title validated via `task.ValidateTitle` and trimmed via `task.TrimTitle` before mutation
  - Priority validated via `task.ValidatePriority` before mutation
  - Parent existence check, self-ref check, and blocks ID existence checks inside Mutate callback
  - `--parent ""` clears parent (sets empty string); `--description ""` clears description
  - `--blocks` delegates to `applyBlocks` which handles deduplication and timestamp refresh on targets
  - Dependency validation (cycle + child-blocked-by-parent) runs after `applyBlocks` but before returning from Mutate; on error the full mutation is rolled back (no persist)
  - `updated` timestamp set to `time.Now().UTC().Truncate(time.Second)` on every change
  - Output via `outputMutationResult` shared with create command (quiet=ID only, else full show format)
  - IDs normalized to lowercase via `task.NormalizeID` on positional ID and `--parent` value; `--blocks` normalized via `parseCommaSeparatedIDs`

TESTS:
- Status: Adequate
- Coverage: 22 test cases covering all 19 spec-listed tests plus 3 additional edge cases
- Test file: /Users/leeovery/Code/tick/internal/cli/update_test.go (595 lines)
- Tests present:
  - "it updates title with --title flag" (line 29)
  - "it updates description with --description flag" (line 50)
  - "it clears description with --description empty string" (line 68)
  - "it updates priority with --priority flag" (line 86)
  - "it updates parent with --parent flag" (line 104)
  - "it clears parent with --parent empty string" (line 129)
  - "it updates blocks with --blocks flag" (line 154)
  - "it updates multiple fields in a single command" (line 183)
  - "it refreshes updated timestamp on any change" (line 222, also checks created unchanged)
  - "it outputs full task details on success" (line 244, checks for ID:/Status:/Priority: fields)
  - "it outputs only task ID with --quiet flag" (line 274)
  - "it errors when no flags are provided" (line 292, checks exit 1 and stderr mentions --title)
  - "it errors when task ID is missing" (line 312)
  - "it errors when task ID is not found" (line 324)
  - "it errors on invalid title (empty/500/newlines)" (line 336, table-driven: empty, whitespace-only, 501 chars, newline)
  - "it errors on invalid priority (outside 0-4)" (line 366, table-driven: -1, 5, 100)
  - "it errors on non-existent parent/blocks IDs" (line 395, table-driven for both)
  - "it errors on self-referencing parent" (line 423)
  - "it normalizes input IDs to lowercase" (line 439, tests uppercase ID + uppercase parent)
  - "it persists changes via atomic write" (line 464, reads back from disk)
  - "it rejects --blocks that would create child-blocked-by-parent dependency" (line 486, verifies no mutation persisted)
  - "it does not duplicate blocked_by when --blocks with existing dependency" (line 522)
  - "it rejects --blocks that would create a cycle" (line 556, verifies no mutation persisted)
- Additional helper tests in /Users/leeovery/Code/tick/internal/cli/helpers_test.go: applyBlocks (7 subtests), parseCommaSeparatedIDs (7 subtests), outputMutationResult (3 subtests)
- Notes:
  - Tests are behavior-focused and verify persistence via disk reads
  - Table-driven tests used appropriately for validation error cases
  - The `--blocks` test at line 154 always pairs with `--title`; no test verifies `--blocks` as sole flag. Minor gap but `hasChanges()` handles it and the blocks behavior itself is verified.
  - No over-testing observed; each test verifies a distinct behavior

CODE QUALITY:
- Project conventions: Followed. Table-driven tests with subtests, explicit error handling, t.Helper() on helpers, consistent test helper pattern (runUpdate, setupTickProjectWithTasks, readPersistedTasks)
- SOLID principles: Good. updateOpts struct is a clean value object. RunUpdate has single responsibility (parse, validate, mutate, output). Mutation logic delegated to store.Mutate. Output delegated to shared outputMutationResult.
- Complexity: Low. Linear flow in RunUpdate. parseUpdateArgs is a simple switch-based parser. No deeply nested logic.
- Modern idioms: Yes. Pointer types for optional flag distinction is idiomatic Go. Error wrapping where appropriate.
- Readability: Good. Well-commented, clear function names, struct fields self-documenting.
- Issues: None blocking.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `--blocks` flag is never tested in isolation (always paired with `--title`). Adding a test with `--blocks` as the sole flag would close a minor gap, though `hasChanges()` does handle it correctly.
- Unknown flags (line 78-79) are silently skipped. If a user types `--status open` it would be silently ignored. Consider emitting an error for unrecognized flags to prevent confusion, though this is a design decision that may affect global flag handling.
- The `--parent` flag normalizes via `task.NormalizeID` after a `strings.TrimSpace` (line 69), but `--title` and `--description` values are not trimmed at the parse level (title is trimmed later via `task.TrimTitle`). This is fine but the asymmetry is worth noting.
