TASK: Add schema version constant and store in metadata during rebuild

ACCEPTANCE CRITERIA:
- const schemaVersion = 1 exists in internal/storage/cache.go
- Rebuild() stores schema_version in the metadata table inside the same transaction as jsonl_hash
- SchemaVersion() returns the stored version from the metadata table
- SchemaVersion() returns 0, nil when the row is missing (no error)
- CurrentSchemaVersion() returns the compiled-in constant
- All existing tests in internal/storage/ continue to pass

STATUS: Complete

SPEC CONTEXT: The spec describes a bug where SQLite cache schema changes (e.g., new columns) break existing users because `CREATE TABLE IF NOT EXISTS` silently succeeds on old schemas. The fix is a schema version constant stored in metadata, checked early in `ensureFresh()`. This task covers the foundation: declaring the constant, storing it in metadata during rebuild, and providing query functions. The check-and-rebuild-on-mismatch logic is task 1-2.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/storage/cache.go:14` -- `const schemaVersion = 1`
  - `/Users/leeovery/Code/tick/internal/storage/cache.go:226-231` -- Rebuild stores schema_version in metadata via `INSERT OR REPLACE` in the same transaction as jsonl_hash
  - `/Users/leeovery/Code/tick/internal/storage/cache.go:258-272` -- `SchemaVersion()` reads from metadata, returns 0/nil on missing row
  - `/Users/leeovery/Code/tick/internal/storage/cache.go:275-277` -- `CurrentSchemaVersion()` returns the compiled-in constant
- Notes: Implementation precisely matches all acceptance criteria. The schema_version is stored as a string (via `fmt.Sprintf("%d", schemaVersion)`) and parsed back with `strconv.Atoi` -- consistent with how the metadata table stores all values as TEXT. The `INSERT OR REPLACE` approach is correct given the `key TEXT PRIMARY KEY` constraint on the metadata table.

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/internal/storage/cache_test.go:1120` -- "it stores schema_version in metadata after rebuild": Calls Rebuild then queries raw SQL to verify the metadata row exists with value "1". Directly verifies the acceptance criterion.
  - `/Users/leeovery/Code/tick/internal/storage/cache_test.go:1144` -- "it returns current schema version via SchemaVersion()": Calls Rebuild then SchemaVersion(), verifies return value is 1. Tests the public API path.
  - `/Users/leeovery/Code/tick/internal/storage/cache_test.go:1167` -- "it returns 0 when schema_version row is missing": Opens cache without Rebuild, calls SchemaVersion(), verifies 0 with nil error. Covers the pre-versioning cache edge case from the spec.
  - `/Users/leeovery/Code/tick/internal/storage/cache_test.go:1187` -- "it stores schema_version in the same transaction as jsonl_hash": Does a valid rebuild, then a failing rebuild (duplicate task IDs), then verifies both schema_version and jsonl_hash remain from the valid rebuild. Proves transactional atomicity.
  - `/Users/leeovery/Code/tick/internal/storage/cache_test.go:1259` -- "it returns compiled-in version via CurrentSchemaVersion()": Verifies CurrentSchemaVersion() == 1. Simple but important for completeness.
- Notes: All four expected tests from the plan are present plus a bonus test for CurrentSchemaVersion(). Tests are focused, each verifying a distinct behavior. The transaction atomicity test is well-designed -- it uses duplicate primary keys to force a rollback, then checks that both metadata values from the prior successful rebuild survive. No over-testing detected; each test covers a unique acceptance criterion.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, t.TempDir for isolation, "it does X" naming convention, error wrapping with fmt.Errorf("%w"), no testify.
- SOLID principles: Good. SchemaVersion() and CurrentSchemaVersion() are small focused functions. The constant is package-private (unexported), the accessor is exported -- clean separation of internal state vs public API.
- Complexity: Low. SchemaVersion() has straightforward linear flow: query -> handle ErrNoRows -> parse. No branching complexity.
- Modern idioms: Yes. Uses `sql.ErrNoRows` comparison (not errors.Is, but consistent with existing IsFresh pattern in the same file at line 246). Uses strconv.Atoi for parsing.
- Readability: Good. Clear doc comments on all exported functions. The comment on SchemaVersion() explicitly documents the 0/nil return for missing rows. The Rebuild() section has a clear comment "Store the schema version" separating it from the hash storage.
- Issues: None significant.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `SchemaVersion()` method and the existing `IsFresh()` method both compare against `sql.ErrNoRows` using `==` rather than `errors.Is()`. This is safe with database/sql (ErrNoRows is a sentinel value, not wrapped), but using `errors.Is()` would be more idiomatic Go and future-proof. This is a pre-existing pattern, not introduced by this task.
- The schema version is stored as a formatted string (`fmt.Sprintf("%d", schemaVersion)`) and parsed back with `strconv.Atoi`. An alternative would be `strconv.Itoa(schemaVersion)` which is slightly more idiomatic for int-to-string. Trivial.
