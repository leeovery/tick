TASK: Use task.Status type and constants instead of raw status strings

ACCEPTANCE CRITERIA:
- MigratedTask.Status is task.Status not string
- No raw status string literals remain in migrate.go, engine.go, or beads.go maps
- No task.Status() cast in store_creator.go
- All tests compile and pass

STATUS: Issues Found

SPEC CONTEXT: The migration pipeline normalizes external tool data into tick's schema. Status values must align with the task package's Status type (open, in_progress, done, cancelled). Type-safety prevents silent drift between the task package and migration package.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/migrate/migrate.go:32 -- MigratedTask.Status is task.Status (was string)
  - /Users/leeovery/Code/tick/internal/migrate/migrate.go:13-18 -- validStatuses map uses task.StatusOpen, task.StatusInProgress, task.StatusDone, task.StatusCancelled
  - /Users/leeovery/Code/tick/internal/migrate/engine.go:10-13 -- completedStatuses map uses task.StatusDone, task.StatusCancelled with task.Status key type
  - /Users/leeovery/Code/tick/internal/migrate/beads/beads.go:19-23 -- statusMap values are task.StatusOpen, task.StatusInProgress, task.StatusDone
  - /Users/leeovery/Code/tick/internal/migrate/store_creator.go:52-54 -- No task.Status() cast; direct comparison mt.Status == "" works because task.Status underlying type is string
- Notes:
  - All four acceptance criteria for production code are met.
  - beads.go:99 uses task.Status("(invalid)") for malformed JSON sentinel entries. This is intentional -- it creates a deliberately invalid status that will fail validation, making malformed lines visible as failures. This is acceptable usage since it's constructing a known-invalid value on purpose, not representing a real tick status.

TESTS:
- Status: Minor gap (non-blocking)
- Coverage: Existing tests for Validate, Engine.Run, StoreTaskCreator.CreateTask, filterPending, and beads provider all compile and use task.Status constants throughout. Invalid-status tests correctly use task.Status("completed") / task.Status("invalid_status") / task.Status("invalid") to exercise the validation path.
- Notes:
  - /Users/leeovery/Code/tick/internal/migrate/dry_run_creator_test.go:28 uses raw string literal Status: "done" instead of task.StatusDone. The file does not import the task package. Go accepts this due to implicit conversion from untyped string to task.Status, so the test compiles and runs correctly. However, the task's "Do" list item 6 says "Update all tests that construct MigratedTask with string status values to use task.Status constants." This is a minor omission -- functionally equivalent but inconsistent with the stated goal of eliminating raw status strings from test code.
  - The invalid-status validation test at migrate_test.go:57 correctly verifies that an unrecognized status is rejected.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing, t.Run subtests, t.Helper on helpers, error wrapping with %w
- SOLID principles: Good -- task.Status type enforces single source of truth for status values; compile-time checks ensure interface satisfaction
- Complexity: Low -- simple map lookups and direct field comparison
- Modern idioms: Yes -- typed string constants, map-based validation, nil pointer for optional values
- Readability: Good -- clear intent, well-commented maps, descriptive variable names
- Issues: None in production code

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- /Users/leeovery/Code/tick/internal/migrate/dry_run_creator_test.go:28 should use task.StatusDone instead of raw string "done" and add task package import, for consistency with the task's stated goal of eliminating all raw status string literals from test code.
