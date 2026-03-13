TASK: cli-enhancements-2-4 -- List/ready/blocked filtering by --type

ACCEPTANCE CRITERIA:
- `--type <value>` on `list`, `ready`, `blocked` filters by single value (normalized before matching)

STATUS: Complete

SPEC CONTEXT:
Spec section "Filtering" under Task Types states: `--type <value>` on `list`, `ready`, `blocked` -- single value filter only. No comma-separated, no multiple flags. Filter input normalized (trimmed, lowercased) before matching. Allowed values: bug, feature, task, chore (closed set -- anything else errors).

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/list.go:24` -- `Type string` field on `ListFilter` struct
  - `/Users/leeovery/Code/tick/internal/cli/list.go:67-72` -- `--type` flag parsed in `parseListFlags`, calls `task.NormalizeType()` (trim + lowercase)
  - `/Users/leeovery/Code/tick/internal/cli/list.go:126-130` -- Post-parse validation via `task.ValidateType(f.Type)` against closed set
  - `/Users/leeovery/Code/tick/internal/cli/list.go:284-287` -- `buildListQuery` adds `t.type = ?` WHERE clause when Type is set
  - `/Users/leeovery/Code/tick/internal/cli/app.go:179-189` -- `handleReady` prepends `--ready` to subArgs, passes through `parseListFlags` -- type filter inherited
  - `/Users/leeovery/Code/tick/internal/cli/app.go:192-202` -- `handleBlocked` prepends `--blocked` to subArgs, passes through `parseListFlags` -- type filter inherited
  - `/Users/leeovery/Code/tick/internal/task/task.go:247-249` -- `NormalizeType` trims and lowercases
  - `/Users/leeovery/Code/tick/internal/task/task.go:234-244` -- `ValidateType` checks against closed set `[bug, feature, task, chore]`
- Notes: Implementation is clean and correct. The `ready` and `blocked` commands share `parseListFlags` and `RunList` with the list command, so type filtering works uniformly across all three commands without code duplication. Normalization happens before validation, matching spec requirements. Single-value only -- no comma-separation or multi-flag logic for type (unlike tags), which aligns with the spec rationale about AND being meaningless for a single-value field.

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/internal/cli/list_filter_test.go:404-426` -- Filters list by `--type bug`, verifies matching task appears, non-matching type and untyped tasks excluded
  - `/Users/leeovery/Code/tick/internal/cli/list_filter_test.go:428-446` -- Normalizes `--type BUG` to lowercase before matching
  - `/Users/leeovery/Code/tick/internal/cli/list_filter_test.go:448-462` -- Invalid type value `epic` returns error with valid types listed
  - `/Users/leeovery/Code/tick/internal/cli/list_filter_test.go:464-479` -- No matching tasks returns "No tasks found." (not an error)
  - `/Users/leeovery/Code/tick/internal/cli/list_filter_test.go:481-499` -- Filters ready tasks by `--type feature`
  - `/Users/leeovery/Code/tick/internal/cli/list_filter_test.go:501-520` -- Filters blocked tasks by `--type chore`
  - `/Users/leeovery/Code/tick/internal/cli/list_filter_test.go:522-544` -- Combines `--type` with `--status` filter
  - `/Users/leeovery/Code/tick/internal/cli/list_filter_test.go:546-568` -- Combines `--type` with `--priority` filter
  - `/Users/leeovery/Code/tick/internal/cli/list_filter_test.go:570-592` -- No `--type` specified returns all tasks (typed and untyped)
  - `/Users/leeovery/Code/tick/internal/cli/list_filter_test.go:594-605` -- `--type` with no value errors
  - `/Users/leeovery/Code/tick/internal/cli/list_filter_test.go:744-772` -- Combines `--count` with `--type` filter
- Notes: Both plan edge cases (invalid type value, no matching tasks) are covered. Tests span all three commands (list, ready, blocked). Combination tests with --status, --priority, and --count verify composability. Tests are focused, each verifying a distinct behavior. No over-testing detected -- each test covers a unique scenario. Tests would fail if the feature broke (e.g., if type filtering was removed from buildListQuery, multiple tests would fail).

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests with "it does X" naming, t.Helper on helpers, t.TempDir for isolation, error wrapping with %w.
- SOLID principles: Good. ListFilter struct cleanly holds filter state. parseListFlags handles parsing, buildListQuery handles SQL construction -- single responsibility maintained. Type normalization and validation delegated to task package (dependency inversion on domain logic).
- Complexity: Low. The parseListFlags switch is straightforward linear parsing. buildListQuery appends conditions sequentially. No nested logic or complex branching.
- Modern idioms: Yes. Parameterized SQL queries prevent injection. Uses append-based condition building rather than string concatenation for WHERE clauses.
- Readability: Good. Clear field names (ListFilter.Type), inline comments where needed, consistent flag parsing pattern across all flags.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
