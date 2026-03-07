TASK: cli-enhancements-2-2 -- Add Type column to SQLite schema and Cache.Rebuild

ACCEPTANCE CRITERIA:
- SQLite `tasks` table has `type TEXT` column
- Populated during `Cache.Rebuild()`

STATUS: Complete

SPEC CONTEXT: The specification states that task type storage in SQLite should be a TEXT column on the tasks table, populated during Cache.Rebuild(). The type field is optional (empty string means unset), with allowed values: bug, feature, task, chore.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/storage/cache.go:23` -- `type TEXT` column in `schemaSQL` CREATE TABLE statement
  - `/Users/leeovery/Code/tick/internal/storage/cache.go:124` -- INSERT statement includes `type` in column list
  - `/Users/leeovery/Code/tick/internal/storage/cache.go:167-169` -- Empty type maps to nil pointer (SQL NULL), non-empty type maps to string value
  - `/Users/leeovery/Code/tick/internal/storage/cache.go:182` -- `typeStr` passed as 6th parameter matching column position
- Notes: Implementation correctly handles the nullable semantics -- empty Go string becomes SQL NULL via nil *string pointer. The column position in the INSERT matches the VALUES placeholder ordering.

TESTS:
- Status: Adequate
- Coverage:
  - `TestCacheSchema` (cache_test.go:15) -- verifies `type` column exists in tasks table schema via PRAGMA table_info
  - `TestCacheRebuild/"it rebuilds cache from parsed tasks"` (cache_test.go:214) -- full round-trip test includes Type="feature", scans it back as sql.NullString and verifies value
  - `TestCacheRebuild/"it stores type value in SQLite after rebuild"` (cache_test.go:353) -- dedicated test: creates task with Type="bug", rebuilds, queries back and verifies the value
  - `TestCacheRebuild/"it stores NULL for empty type after rebuild"` (cache_test.go:390) -- dedicated test: creates task with no type, rebuilds, verifies the column is SQL NULL
- Notes: Good coverage of both the set and unset cases. The round-trip test also validates type alongside all other fields. Tests are focused and not redundant -- the dedicated type tests verify specific behaviors (value vs NULL) while the round-trip test confirms integration with the full field set. No edge cases were specified for this task, and none are meaningfully missing given the simplicity of the column addition.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, t.TempDir for isolation, t.Helper on helpers. Error wrapping with fmt.Errorf and %w throughout.
- SOLID principles: Good. Cache.Rebuild has a single responsibility (repopulate cache from task data). The nullable mapping pattern (empty string -> nil *string -> SQL NULL) is consistent with how description, parent, and closed are handled.
- Complexity: Low. The type handling in Rebuild is 3 lines of straightforward nil-pointer mapping, identical to the existing pattern for description and parent.
- Modern idioms: Yes. Uses sql.NullString for nullable scanning in tests, prepared statements for batch inserts, proper transaction with deferred rollback.
- Readability: Good. The type handling follows the same pattern as adjacent fields (description at lines 173-175, parent at lines 162-164), making the code predictable and scannable.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
