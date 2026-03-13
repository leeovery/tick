TASK: Provider Contract & Migration Types (migration-1-1)

ACCEPTANCE CRITERIA:
- [x] `MigratedTask` struct exists with fields matching tick's task schema
- [x] `Provider` interface exists with `Name()` and `Tasks()` methods
- [x] `Result` struct exists with `Title`, `Success`, and `Err` fields
- [x] Validation rejects empty/whitespace-only titles
- [x] Validation rejects invalid status values while accepting all four valid ones plus empty
- [x] Validation rejects priority outside 0-4 range
- [x] A mock provider satisfies the `Provider` interface (proven by test)
- [x] Types are self-contained within the migrate package

STATUS: Complete

SPEC CONTEXT: The specification defines a plugin/strategy architecture (Provider -> Normalize -> Insert) where providers map source data to a normalized format mirroring tick's task schema. Title is the only required field. Fields with no tick equivalent are discarded by the provider. This task establishes the contract types that all subsequent migration tasks depend on.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/migrate/migrate.go:1-69
- Notes:
  - `MigratedTask` struct (line 30-38) has all specified fields: Title, Status, Priority (*int for nil-distinguishable), Description, Created, Updated, Closed
  - `Provider` interface (line 57-62) has `Name() string` and `Tasks() ([]MigratedTask, error)` exactly as specified
  - `Result` struct (line 65-69) has Title, Success, Err fields as specified
  - `Validate()` method (line 43-54) checks empty/whitespace title, invalid status, and priority range 0-4
  - `Status` field uses `task.Status` type rather than raw string, which is a refinement from phase 3 task migration-3-4 that is already integrated. This is a forward-compatible improvement, not drift.
  - `validStatuses` map (line 13-18) uses task package constants, which is clean and maintainable
  - `FallbackTitle` constant (line 21) added for untitled tasks -- used by engine, not part of original task spec but a reasonable addition
  - Priority uses `*int` pointer semantics to distinguish "not provided" from zero, exactly as the plan recommended

TESTS:
- Status: Adequate
- Coverage: All 10 tests specified in the plan are present, plus 2 additional meaningful tests
- Test file: /Users/leeovery/Code/tick/internal/migrate/migrate_test.go:1-217
- Tests present:
  1. "MigratedTask with only title is valid" (line 12)
  2. "MigratedTask with all fields populated is valid" (line 20)
  3. "MigratedTask with empty title is invalid" (line 40)
  4. "MigratedTask with whitespace-only title is invalid" (line 48)
  5. "MigratedTask with invalid status is rejected" (line 56)
  6. "MigratedTask with valid status values are accepted" -- tests all four statuses as subtests (line 64)
  7. "MigratedTask with empty status is valid (defaults applied later)" (line 77)
  8. "MigratedTask with priority out of range is rejected" -- tests -1 and 5 (line 85)
  9. "MigratedTask with priority in range is accepted" -- tests 0 and 4 boundaries (line 105)
  10. "Provider interface is implementable by a mock" -- compile-time check plus runtime verification (line 128)
  11. "Provider mock can return errors" -- additional test for error path (line 155) -- useful, not redundant
  12. "Result captures success outcome" and "Result captures failure outcome" (line 174-198) -- verify Result struct fields work correctly
- Notes: The extra tests (provider error path, Result struct verification) are meaningful and non-redundant. They verify the contract types are usable, not just compilable. No over-testing concern.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib `testing` only, `t.Run()` subtests, table-driven tests for priority cases. Error wrapping with `fmt.Errorf`. Matches CLAUDE.md patterns.
- SOLID principles: Good. Single responsibility -- migrate.go defines only types and validation. Interface segregation -- Provider is minimal (2 methods). Open/closed -- new providers implement the interface without modifying existing code.
- Complexity: Low. Validate() has straightforward linear checks with early returns. No nested logic.
- Modern idioms: Yes. Pointer semantics for optional int (`*int` for Priority), map-based validation lookup, `strings.TrimSpace` for whitespace handling.
- Readability: Good. All exported types and methods have doc comments. Field comments explain semantics (e.g., "nil means not provided"). Constants named clearly (minPriority, maxPriority).
- Security: N/A for this task (types and validation only)
- Performance: N/A (no I/O or iteration concerns at this level)
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `FallbackTitle` constant is defined in this file but only consumed by the engine (engine.go:74). This is a reasonable location since it is a package-level concern, but could also live in engine.go if the package grows. No action needed.
- The `Status` field type is `task.Status` rather than `string` as originally planned. This is a deliberate improvement from phase 3 (migration-3-4) and is cleaner. No concern.
