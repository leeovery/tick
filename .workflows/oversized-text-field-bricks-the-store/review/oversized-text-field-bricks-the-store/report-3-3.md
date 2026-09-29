TASK: Migrate Skips An Issue Whose Description Exceeds The Cap (oversized-text-field-bricks-the-store-3-3, tick-277d5a)

ACCEPTANCE CRITERIA:
- A `.beads/issues.jsonl` holding, in order, an issue whose description is exactly 50,000 characters, one whose description is 50,001 characters and an ordinary issue: `tick migrate --from beads` completes the import as it does today when an issue is skipped, reporting `Done: 2 imported, 1 failed`. The first and third issues are in `tasks.jsonl`, the first with its full 50,000-character description (§6.4, §8.5)
- Same run: the second issue's skip line and its entry under `Failures:` give a reason naming the description field, the 50,000-character limit and the submitted length, 50,001, and saying nothing was saved. `tasks.jsonl` holds no task for it, neither whole nor truncated (§6.3, §6.4, §8.5)
- An issue whose description is exactly 50,000 characters wrapped in leading and trailing whitespace imports, with the description trimmed. An issue whose description is 50,001 characters wrapped likewise is skipped, and its reason reports 50,001 (§6.1, §6.3)
- `tick migrate --from beads --dry-run` on the same source reports the same skip with the same reason and the same summary, and writes nothing (§6.4, §8.5)

STATUS: complete

SPEC CONTEXT: §6.1 caps a description at 50,000 Unicode characters counted after trimming edge whitespace. §6.2 requires every route that sets a description (create, update, migrate) to enforce it. §6.3 requires the refusal to name the field, the limit, the submitted (trimmed) length and that nothing was saved; on migrate it is the skipped issue's reason. §6.4 requires migrate to skip an over-cap issue through the same path as any other invalid issue, never truncate, never abort the rest, and have --dry-run report the same refusal because validation precedes the no-op creator. §8.5 lists the migrate boundary tests.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/migrate/migrate.go:40-51 — `MigratedTask.Validate` ends with `return task.ValidateDescription(mt.Description)` (line 50), after the title, status and priority checks.
  - internal/task/task.go:204-210 — `ValidateDescription` counts `utf8.RuneCountInString(strings.TrimSpace(desc))` against `maxDescriptionLen` (50000, task.go:35) and returns `description is %d characters, over the %d-character limit; nothing was saved` — all four §6.3 elements.
  - internal/migrate/engine.go:71-78 — each issue is `Normalize()`d (trims description via `task.TrimDescription`, migrate.go:55-59) and then validated at line 74; a failure is recorded as a failed `Result` and the loop continues, so the rest of the import proceeds and nothing is truncated.
  - internal/migrate/presenter.go:22-29, 49-64 — the unchanged skip line `✗ Task: <title> (skipped: <reason>)` and `Failures:` entry carry the reason.
  - internal/cli/migrate.go:107-128 — dry run swaps only the creator; validation in `Engine.Run` precedes it, so --dry-run reports the identical skip.
- Notes: The reason is reported against the trimmed length because the engine normalises before validating (and `ValidateDescription` trims again, harmlessly). The wording uses plain digits ("50001", "50000") rather than the spec example's thousands separators; §6.3 leaves wording to the implementer and the four elements are present. The Phase 1 long-line beads test (internal/cli/migrate_test.go:778-826) uses a 12,000-character description, so it keeps importing under the cap as the task's context requires. No drift.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/migrate_test.go:828-913 `TestMigrateDescriptionCap` drives the real CLI (`runMigrate`) over a beads source of at-cap / over-cap / ordinary issues:
    - Criteria 1 and 2 (pad ""): asserts exit 0, the exact skip line and the exact `Failures:` entry with the full refusal text (lines 843-854), `Done: 2 imported, 1 failed`, exactly two persisted tasks in order "At cap", "Ordinary" (lines 879-885), the at-cap description equal to the full 50,000 characters (886-888), and that no persisted task carries the over-cap issue's "b" text, so neither a whole nor a truncated copy was written (889-893).
    - Criterion 3 (pad " \t\n"): the same assertions against whitespace-wrapped descriptions; the refusal constant fixes the reported length at 50001 (so an untrimmed count of 50007 would fail), and the at-cap description is asserted equal to the trimmed 50,000 characters.
    - Criterion 4 (lines 897-912): --dry-run asserts the same skip line, Failures entry and summary, and that `tasks.jsonl` is byte-identical before and after.
  - internal/migrate/migrate_test.go:127-146 `TestMigratedTaskDescriptionCap` pins the 50,000 / 50,001 boundary and the exact reason at the `Validate` level.
- Notes: Each test would fail if the check were removed (the over-cap issue would import and the summary, skip-line and persisted-count assertions would fail), if the count ignored trimming, or if the reason lost an element. The two layers are not redundant: the unit test pins `Validate`, the CLI test pins the engine skip path, the presenter output, persistence and dry run. Multibyte counting is not re-tested on the migrate route; it is the shared `task.ValidateDescription` counter, and neither this task's criteria nor §8.5's migrate bullet asks for it.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` subtests named "it ...", `t.Helper()` on helpers, `t.TempDir()` isolation via `setupTickProject`; the validation is reused from `internal/task`, not duplicated.
- SOLID principles: Good — the cap lives once in `task.ValidateDescription`; `MigratedTask.Validate` composes it; engine and presenter unchanged.
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
