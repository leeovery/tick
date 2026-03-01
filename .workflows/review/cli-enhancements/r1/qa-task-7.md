TASK: cli-enhancements-2-3 -- Create and update commands with --type and --clear-type flags

ACCEPTANCE CRITERIA:
- `--type <value>` on `create` and `update` sets/replaces type; validated against closed set (bug, feature, task, chore); case-insensitive input normalized to lowercase
- `--clear-type` on `update` removes the type; mutually exclusive with `--type`; empty `--type` value errors

STATUS: Complete

SPEC CONTEXT:
The spec defines `--type <value>` on `create` and `update` to set/replace the type, validated against the closed set {bug, feature, task, chore}. Input is case-insensitive, trimmed, and stored lowercase. `--clear-type` on `update` explicitly removes the type. `--type` and `--clear-type` are mutually exclusive. Empty `--type` value must error (protective against accidental erasure).

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/create.go:72-78` -- `--type` flag parsing in `parseCreateArgs`, sets `opts.taskType` and `opts.hasType`
  - `/Users/leeovery/Code/tick/internal/cli/create.go:130-135` -- Type validation in `RunCreate` via `validateTypeFlag`
  - `/Users/leeovery/Code/tick/internal/cli/create.go:210` -- Type assigned to new task struct
  - `/Users/leeovery/Code/tick/internal/cli/update.go:23-24` -- `taskType *string` and `clearType bool` in `updateOpts`
  - `/Users/leeovery/Code/tick/internal/cli/update.go:78-86` -- `--type` and `--clear-type` parsing in `parseUpdateArgs`
  - `/Users/leeovery/Code/tick/internal/cli/update.go:159-169` -- Mutual exclusion check and validation in `RunUpdate`
  - `/Users/leeovery/Code/tick/internal/cli/update.go:272-275` -- Type application during mutation (clear or set)
  - `/Users/leeovery/Code/tick/internal/cli/helpers.go:59-68` -- `validateTypeFlag` helper: normalizes, checks non-empty, validates against closed set
  - `/Users/leeovery/Code/tick/internal/task/task.go:231-258` -- Domain validation: `allowedTypes`, `ValidateType`, `NormalizeType`, `ValidateTypeNotEmpty`
- Notes: Implementation is clean and follows the established patterns for similar flag pairs (description/clear-description, tags/clear-tags). The `validateTypeFlag` helper in `helpers.go` is shared between create and update, following DRY. The `updateOpts.hasChanges()` at line 33 correctly includes `taskType` and `clearType`.

TESTS:
- Status: Adequate
- Coverage:
  - Create tests (`/Users/leeovery/Code/tick/internal/cli/create_test.go`):
    - Line 804: `--type bug` sets type correctly
    - Line 819: Case normalization (`FEATURE` -> `feature`)
    - Line 834: Optional (no `--type` leaves type empty)
    - Line 849: Empty `--type` value errors, mentions `--clear-type`
    - Line 860: Invalid type value (`epic`) errors
    - Line 871: Whitespace-only `--type` errors
  - Update tests (`/Users/leeovery/Code/tick/internal/cli/update_test.go`):
    - Line 633: `--type chore` sets type
    - Line 651: `--clear-type` removes type from a task that has one
    - Line 669: `--type` and `--clear-type` together errors with "mutually exclusive"
    - Line 685: Empty `--type` value on update errors, mentions `--clear-type`, verifies type unchanged
    - Line 707: Invalid type value on update errors, verifies type unchanged
  - Domain tests (`/Users/leeovery/Code/tick/internal/task/task_test.go`):
    - Line 562: All four allowed types pass validation
    - Line 574: Invalid type rejected with helpful error listing allowed values
    - Line 587: Empty type allowed (optional field)
    - Line 594: Mixed-case invalid type after normalization still rejected
    - Line 604: `NormalizeType` tests for various inputs (uppercase, whitespace, empty)
    - Line 628: `ValidateTypeNotEmpty` rejects empty and whitespace-only
  - Edge cases from plan covered: `--type` and `--clear-type` together (update_test.go:669), empty `--type` value (create_test.go:849, update_test.go:685)
- Notes: One minor gap: the update tests do not explicitly test case normalization on `--type` for update (e.g., `--type CHORE` normalizing to `chore`). However, the normalization logic is the same shared `validateTypeFlag` helper tested thoroughly via create tests and domain-level `NormalizeType` tests, so the risk is negligible. Tests are focused and not over-tested -- each test verifies a distinct behavior.

CODE QUALITY:
- Project conventions: Followed -- uses stdlib testing, `t.Run()` subtests, `t.Helper()`, `t.TempDir()`, error wrapping with `%w`, pointer types for optional fields in `updateOpts`, handler signature pattern
- SOLID principles: Good -- validation logic lives in the `task` package (domain), CLI-level validation helper `validateTypeFlag` composes domain functions, clear single responsibility
- Complexity: Low -- straightforward flag parsing and conditional application
- Modern idioms: Yes -- idiomatic Go patterns throughout
- Readability: Good -- clear naming (`validateTypeFlag`, `NormalizeType`, `ValidateTypeNotEmpty`), consistent with existing patterns for description/clear-description
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Consider adding a case-normalization test for `--type` on update (e.g., `--type CHORE`) for completeness, even though the shared helper is already tested. This is low risk since the code path is identical.
