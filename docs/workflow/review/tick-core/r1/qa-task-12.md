TASK: tick dep add & tick dep rm commands

ACCEPTANCE CRITERIA:
- dep add adds dependency and outputs confirmation
- dep rm removes dependency and outputs confirmation
- Non-existent IDs return error
- Duplicate/missing dep return error
- Self-ref, cycle, child-blocked-by-parent return error
- IDs normalized to lowercase
- --quiet suppresses output
- updated timestamp refreshed
- Persisted through storage engine

STATUS: Complete

SPEC CONTEXT: Spec defines `tick dep add <task_id> <blocked_by_id>` and `tick dep rm <task_id> <blocked_by_id>` with argument order "task first, dependency second". Output: "Dependency added: tick-c3d4 blocked by tick-a1b2" / "Dependency removed: tick-c3d4 no longer blocked by tick-a1b2". --quiet suppresses output. Validation timing: reject invalid dependencies at write time. rm does not validate blocked_by_id exists as a task -- only checks array membership (supports removing stale refs).

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/dep.go:1-182
- Notes: Clean implementation. `handleDep` routes to `RunDepAdd`/`RunDepRm` via switch on sub-subcommand. `parseDepArgs` extracts two positional IDs, skipping flag-like args, normalizes via `task.NormalizeID`. Both commands use `store.Mutate` for atomic read-modify-write with exclusive locking. `dep add` validates: task exists, blocked_by exists, no duplicate, delegates to `task.ValidateDependency` for self-ref/cycle/child-blocked-by-parent. `dep rm` validates: task exists, blocked_by_id present in array (does NOT require blocked_by_id to exist as a task -- matches spec edge case). Output via `fmtr.FormatDepChange`, suppressed when `fc.Quiet` is true. Timestamps updated via `time.Now().UTC().Truncate(time.Second)`.

TESTS:
- Status: Adequate
- Coverage: All 15 planned test cases are covered across TestDepAdd (13 subtests) and TestDepRm (10 subtests) plus TestDepNoSubcommand (1 subtest). Specifically:
  - "it adds a dependency between two existing tasks" -- verifies persistence of blocked_by
  - "it removes an existing dependency" -- verifies blocked_by emptied
  - "it outputs confirmation on success (add/rm)" -- exact string match against spec format
  - "it updates task's updated timestamp" -- before/after bracket check for both add and rm
  - "it errors when task_id not found (add/rm)" -- checks exit 1 + "not found" in stderr
  - "it errors when blocked_by_id not found (add)" -- exit 1 + "not found"
  - "it errors on duplicate dependency (add)" -- exit 1 + "already"
  - "it errors when dependency not found (rm)" -- exit 1 + "not a dependency"
  - "it errors on self-reference (add)" -- exit 1 + "cycle"
  - "it errors when add creates cycle" -- A blocked by B, add B blocked by A
  - "it errors when add creates child-blocked-by-parent" -- child + parent setup
  - "it normalizes IDs to lowercase" -- uppercase input, lowercase in persisted data and output
  - "it suppresses output with --quiet" -- empty stdout and stderr
  - "it errors when fewer than two IDs provided" -- one ID and zero IDs cases
  - "it persists via atomic write" -- reads back from disk to confirm
  - "rm does not validate blocked_by_id exists as a task" -- stale ref removal works
- Notes: Tests are thorough and well-structured. Each test sets up its own isolated project directory. Tests verify both behavior (exit codes, output) and persistence (reading back from JSONL). No over-testing observed -- each test covers a distinct scenario.

CODE QUALITY:
- Project conventions: Followed. Table-driven subtests via t.Run, proper t.Helper usage in test helper, error handling follows Go idioms, exported functions documented.
- SOLID principles: Good. Single responsibility maintained -- dep.go handles CLI concerns, task/dependency.go handles validation logic. Dependency validation is delegated cleanly to the task package.
- Complexity: Low. Linear flows in both RunDepAdd and RunDepRm. The Mutate callback is straightforward with sequential validation steps.
- Modern idioms: Yes. Uses Go range loops, slice manipulation, proper defer for store.Close().
- Readability: Good. Code is well-commented, function names are descriptive, flow is easy to follow.
- Issues: None significant.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Minor: The `parseDepArgs` function silently skips any flag-like args (starting with "-") that are not recognized. This is intentional for --quiet passthrough but could mask typos in flags. Acceptable tradeoff for simplicity.
- Minor: Task lookup in both RunDepAdd and RunDepRm uses linear scan. Fine for expected scale (hundreds of tasks) but could be noted if scale assumptions change.
