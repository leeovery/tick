TASK: cli-enhancements-1-1 -- ResolveID method in storage layer

ACCEPTANCE CRITERIA:
- ResolveID(prefix) method in storage layer queries WHERE id LIKE 'tick-{prefix}%' and returns the full ID
- Both tick-a3f and a3f input forms accepted; tick- prefix stripped if present, input lowercased before matching
- Exact full-ID match (10-char input tick- + 6 hex) returns immediately without prefix collision check
- Minimum 3 hex chars required for prefix matching; fewer returns a validation error
- Ambiguous prefix (2+ matches) returns error listing all matching IDs
- Zero matches returns "not found" error
- All commands accepting task IDs resolve through ResolveID

STATUS: Complete

SPEC CONTEXT:
The specification (Partial ID Matching section) defines resolution rules for prefix-based task ID lookup. Both tick-a3f and a3f forms are accepted. Exact full-ID match takes priority (10-char tick- + 6 hex returns immediately). Minimum 3 hex chars for prefix matching. Ambiguity (2+ matches) returns error listing matches. Zero matches returns "not found". Case-insensitive. Implementation lives in storage layer as ResolveID(prefix). All commands resolve first, then proceed with the full ID. Applies to positional args, --parent, --blocked-by, --blocks.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/storage/store.go:285-353
- Notes:
  - Method signature: `ResolveID(input string) (string, error)` on `*Store`
  - Strips `tick-` prefix case-insensitively via `strings.ToLower` + `strings.HasPrefix` (line 292-296)
  - Minimum 3 hex chars enforced (line 299-301)
  - Exact full-ID match: when hex is 6 chars, tries `SELECT id FROM tasks WHERE id = ?` first (line 309-316)
  - Prefix search uses `SELECT id FROM tasks WHERE id LIKE ? ORDER BY id` with `fullID + "%"` (line 320)
  - Switch on match count: 0 -> not-found error with original input preserved, 1 -> returns match, 2+ -> ambiguous error listing all matching IDs (lines 338-345)
  - Uses `s.Query()` for shared locking and cache freshness (line 305)
  - All acceptance criteria are directly addressed in the implementation

INTEGRATION (acceptance criterion 7 -- all commands resolve through ResolveID):
- show.go:46 -- positional ID
- transition.go:24 -- start/done/cancel/reopen positional ID
- update.go:209 -- positional ID
- update.go:214 -- --parent flag
- update.go:221 -- --blocks flag (iterates slice)
- create.go:162 -- --parent flag
- create.go:168 -- --blocked-by flag (iterates slice)
- create.go:174 -- --blocks flag (iterates slice)
- dep.go:71-75 -- dep add: both taskID and blockedByID
- dep.go:148-152 -- dep rm: both taskID and blockedByID
- note.go:68 -- note add positional ID
- note.go:122 -- note remove positional ID
- app.go:246 -- remove command (iterates rawIDs)
- list.go:157 -- --parent filter flag
- All commands accepting task IDs resolve through ResolveID: Confirmed

TESTS:
- Status: Adequate
- Coverage: All edge cases from the plan are tested in /Users/leeovery/Code/tick/internal/storage/resolve_id_test.go
  - Unique 3-char prefix resolution (line 21)
  - tick- prefix stripping (line 38)
  - Mixed-case normalization -- lowercase input (line 55)
  - Mixed-case normalization -- uppercase TICK- prefix (line 72)
  - Exact full-ID match bypasses ambiguity check (line 89)
  - Prefix shorter than 3 hex chars -- 2 chars (line 108)
  - Prefix shorter than 3 hex chars -- 1 char (line 126)
  - Ambiguous prefix listing all matching IDs (line 144)
  - Zero matches -- not-found error (line 168)
  - 4-char prefix resolves uniquely (line 186)
  - 5-char prefix resolves uniquely (line 203)
  - 6-char non-existent ID fallback to prefix search (line 220)
  - Empty string input (line 246)
  - tick- prefix leaving fewer than 3 hex chars (line 264)
  - Original input preserved in not-found error message (line 282)
- Notes:
  - Tests use a well-designed fixture with two tasks sharing the "a3f" prefix (tick-a3f1b2, tick-a3f1b3) and one distinct task (tick-b12345), enabling both ambiguity and unique-match testing
  - Each test creates a fresh temp dir via setupTickDirWithTasks, ensuring isolation
  - Tests verify exact error messages, not just error presence
  - No over-testing detected -- each test targets a distinct behavior/edge case
  - Would fail if the feature broke

CODE QUALITY:
- Project conventions: Followed
  - Uses stdlib testing only (no testify)
  - t.Run subtests with "it does X" naming
  - t.TempDir for isolation
  - t.Helper on test helpers
  - Error wrapping with fmt.Errorf("context: %w", err)
  - Functional options pattern consistent with Store design
- SOLID principles: Good
  - Single responsibility: ResolveID does one thing (resolve input to canonical ID)
  - Uses Store.Query for shared locking/cache freshness rather than duplicating that logic
  - Method is on Store (correct layer per spec: storage layer)
- Complexity: Low
  - Linear flow: normalize -> validate -> exact match -> prefix search -> switch on count
  - Single Query call wrapping the entire DB interaction (consolidated per Phase 5 task cli-enhancements-5-5)
- Modern idioms: Yes
  - strings.ToLower + strings.HasPrefix for case-insensitive prefix handling
  - errors.New for simple errors, fmt.Errorf for formatted errors
  - Proper defer on rows.Close()
  - rows.Err() checked after iteration
- Readability: Good
  - Clear variable names (originalInput, lower, hex, fullID, matches)
  - Comments document the logic at each step
  - Method doc comment explains the full contract
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The test file creates a new Store for every subtest (15 subtests, 15 stores). This is fine for isolation but could be slightly more efficient with a shared setup. Not a concern given test runtime is negligible.
- The `hex` variable name in ResolveID is slightly misleading -- it is not validated as actual hex characters. However, since IDs are generated internally with hex chars, this is acceptable; the method is intentionally permissive on input format and relies on the DB query to determine validity.
