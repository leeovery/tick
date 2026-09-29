TASK: Migrate Adopts The CLI's Title Rules (oversized-text-field-bricks-the-store-3-4, tick-2f337d)

ACCEPTANCE CRITERIA:
- A `.beads/issues.jsonl` holding an issue with a 501-character title, an issue whose title spans two lines, an issue with a title of exactly 500 characters and an ordinary issue: `tick migrate --from beads` skips the first two and imports the other two, reporting `Done: 2 imported, 2 failed` (§6.4, §8.5)
- Same run: the 501-character title's reason is `title exceeds maximum length of 500 characters`, and the two-line title's reason is `title must be a single line (no newlines)` — the CLI's current title refusal messages (§6.4, §7)
- An issue whose title is exactly 500 multibyte characters imports, because the title is counted in characters, not bytes (§1.2, §6.4)
- An issue whose title has line breaks only at its edges, such as `"Ship it\n"`, imports with the title `Ship it`, as the CLI accepts it once trimmed (§6.4)
- `tick migrate --from beads --dry-run` on the same source reports the same two skips with the same reasons (§6.4)

STATUS: complete

SPEC CONTEXT: §6.4 has migrate adopt the CLI's title rules: an issue whose title exceeds 500 characters or spans more than one line is skipped with its reason through the same validation branch as any other invalid issue, the rest import, and `--dry-run` reports the same refusals because validation runs before the dry run's no-op creator. §7 keeps the title refusal messages unchanged. §8.5 asks for the over-500 and multi-line cases to be tested alongside the description-cap migrate tests.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/migrate/migrate.go:40-51 — `MigratedTask.Validate` now calls `task.ValidateTitle(mt.Title)` in place of the old empty-only check, so the empty, multi-line and over-500 refusals and their wording are the CLI's own (internal/task/task.go:178-190).
  - internal/migrate/engine.go:71-78 — the engine runs `mt.Normalize()` (which applies `task.TrimTitle`, internal/migrate/migrate.go:55-59) before `mt.Validate()`, the same order `tick create` uses (internal/cli/create.go:123-126). So `"Ship it\n"` is trimmed to `Ship it` and accepted, and an interior newline is refused.
  - internal/cli/migrate.go:107-128 — dry-run swaps in `DryRunTaskCreator` behind the same engine, so validation, and with it the skip reasons, are identical in both modes.
- Notes: The characters-not-bytes count comes from `utf8.RuneCountInString` in `ValidateTitle`. The empty-title message is unchanged (`title is required and cannot be empty`), so the existing engine tests `it keeps today's error for a whitespace-only title` (internal/migrate/engine_test.go:992) and `it reports the trimmed title in a dry run` (:1035) still hold. The Phase 1 long-line beads test keeps its title short (internal/cli/migrate_test.go:783), so the new rule does not affect it. No drift from the plan.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/migrate_test.go:915-1019 `TestMigrateTitleRules` covers every criterion end to end through `App.Run`:
    - criteria 1 and 2 (:954): the four-issue source, the exact `✗ Task: … (skipped: <reason>)` line and the exact Failures entry for each skip, `Done: 2 imported, 2 failed`, and the two persisted titles in order.
    - criterion 5 (:973): the dry run gives the same assertions and leaves `tasks.jsonl` byte-for-byte unchanged.
    - criteria 3 and 4 (:995-996): a title of 500 multibyte `é` characters and `"Ship it\n"` persist as expected.
  - internal/migrate/migrate_test.go:148-182 `TestMigratedTaskTitleRules` pins the exact refusal strings at the `Validate` level: 501 characters, an interior `\n`, an interior `\r`, and 500 ASCII and 500 `é` characters accepted.
  - Reverting `Validate` to the empty-only check would import all four issues and fail both the summary and the skip-line assertions, so the tests would catch the feature breaking.
- Notes: The unit tests and the CLI tests overlap on the boundary cases, but at different layers (the refusal wording on `Validate`, and the import outcome through the command). This is not redundant.

CODE QUALITY:
- Project conventions: Followed (stdlib testing, `t.Run` subtests, `t.Helper()` on helpers, "it does X" naming, table-driven cases)
- SOLID principles: Good — migrate delegates to the domain's single title rule rather than duplicating it
- Complexity: Low
- Modern idioms: Yes
- Readability: Good
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
